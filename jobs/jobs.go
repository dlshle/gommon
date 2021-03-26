package jobs

import (
	"log"
	"os"
	"sync"
	"time"
)

const (
	JobStatusWaiting     = 0
	JobStatusRunning     = 1
	JobStatusDone        = 2
	JobStatusTerminating = 4

	EvictPolicyCancelLast = 0
)

const (
	timeoutExecutorStrategy  = 0
	intervalExecutorStrategy = 1

	transitWaitingRunning     = 1
	transitWaitingDone        = 2
	transitWaitingTerminating = 4
	transitTerminatingDone    = 42 // interval job only
	transitRunningDone        = 12
	transitRunningTerminating = 14 // interval job only
	transitRunningWaiting     = 10 // interval job only
)

var globalLogger = log.New(os.Stdout, "[Performance]", log.Ldate|log.Ltime|log.Lshortfile)

var jobPoolEvictStrategies = make(map[int]func(p *JobPool, uuid int64))
var jobPoolTransitStrategies = make(map[int]func(p *JobPool, uuid int64))
var jobPoolExecutorBuildingStrategies = make(map[int]func(p *JobPool, task func(), duration time.Duration) *Job)
var statusStringMap = make(map[int]string)

func init() {
	initPoolEvictStrategy()
	initPoolExecutorBuilder()
	initStatusStringMap()
	initTransitStrategy()
}

func initPoolEvictStrategy() {
	jobPoolEvictStrategies[EvictPolicyCancelLast] = func(p *JobPool, uuid int64) {
		job := p.jobMap[uuid]
		if job != nil {
			job.Status = JobStatusDone
			delete(p.jobMap, uuid)
		} else {
			p.logger.Printf("WARN job[%d] does not exist!\n", uuid)
		}
	}
}

func initTransitStrategy() {
	jobPoolTransitStrategies[transitWaitingRunning] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d started running\n")
		p.jobMap[uuid].executor()
	}
	jobPoolTransitStrategies[transitWaitingDone] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d is canceled\n")
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transitWaitingTerminating] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d is terminating after waiting...\n")
	}
	jobPoolTransitStrategies[transitRunningDone] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d finished\n")
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transitRunningTerminating] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d is terminating after running...\n")
	}
	jobPoolTransitStrategies[transitRunningWaiting] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d interval done, onto the next interval...\n")
	}
	jobPoolTransitStrategies[transitTerminatingDone] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d final interval done, task has been terminated\n")
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
}

func initPoolExecutorBuilder() {
	shouldTerminateJob := func(p *JobPool, uuid int64) bool {
		return p.GetStatus(uuid) == JobStatusTerminating
	}
	jobPoolExecutorBuildingStrategies[timeoutExecutorStrategy] = func(p *JobPool, task func(), duration time.Duration) *Job {
		uuid := time.Now().Unix()
		return NewJob(uuid, func() {
			time.Sleep(duration)
			if shouldTerminateJob(p, uuid) {
				p.transitJobStatus(uuid, JobStatusDone)
				return
			}
			p.transitJobStatus(uuid, JobStatusRunning)
			task()
			p.transitJobStatus(uuid, JobStatusDone)
		})
	}
	jobPoolExecutorBuildingStrategies[intervalExecutorStrategy] = func(p *JobPool, task func(), duration time.Duration) *Job {
		uuid := time.Now().Unix()
		return NewJob(uuid, func() {
			for {
				time.Sleep(duration)
				p.transitJobStatus(uuid, JobStatusRunning)
				if shouldTerminateJob(p, uuid) {
					p.transitJobStatus(uuid, JobStatusDone)
					return
				}
				task()
				if shouldTerminateJob(p, uuid) {
					p.transitJobStatus(uuid, JobStatusDone)
					return
				}
				p.transitJobStatus(uuid, JobStatusWaiting)
			}
			p.transitJobStatus(uuid, JobStatusDone)
		})
	}
}

func initStatusStringMap() {
	statusStringMap[JobStatusWaiting] = "WAITING"
	statusStringMap[JobStatusRunning] = "RUNNING"
	statusStringMap[JobStatusDone] = "DONE"
	statusStringMap[JobStatusTerminating] = "TERMINATING"
}

// --------------- Type and Interface Definitions & Implementations --------------- //

type Job struct {
	executor func()
	uuid     int64
	Status   int
}

func NewJob(uuid int64, executor func()) *Job {
	return &Job{executor, uuid, 0}
}

type JobPool struct {
	id           string
	jobMap       map[int64]*Job
	maxSize      uint
	evictPolicy  int
	finishPolicy int
	logger       *log.Logger
	*sync.RWMutex
}

type IJobPool interface {
	makeExecutor()
	scheduleJob(task func(), duration time.Duration, executorStrategy int) int64
	ScheduleTimeoutJob(task func(), duration time.Duration) int64
	ScheduleIntervalJob(task func(), duration time.Duration) int64
	CancelJob(uuid int64) bool
	HasJob(uuid int64) bool
	GetStatus(uuid int64) int
	setStatus(uuid int64, status int)
	transitJobStatus(uuid int64, status int)
}

func (p *JobPool) transitJobStatus(uuid int64, status int) {
	fromStatus := p.GetStatus(uuid)
	transitHandler := jobPoolTransitStrategies[fromStatus*10+status]
	if transitHandler == nil {
		p.logger.Printf("Invalid job status transit from %s to %s. Job will be canceled\n", statusStringMap[fromStatus], statusStringMap[status])
		p.CancelJob(uuid)
		return
	}
	p.setStatus(uuid, status)
	transitHandler(p, uuid)
}

func (p *JobPool) setStatus(uuid int64, status int) {
	p.RWMutex.Lock()
	defer p.RWMutex.Unlock()
	p.jobMap[uuid].Status = status
}

func (p *JobPool) HasJob(id int64) bool {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	return p.jobMap[id] != nil
}

func (p *JobPool) GetStatus(id int64) int {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	job := p.jobMap[id]
	if job == nil {
		return JobStatusDone
	} else {
		return job.Status
	}
}

func (p *JobPool) scheduleJob(task func(), duration time.Duration, executorStrategy int) int64 {
	uuid := time.Now().Unix()
	job := jobPoolExecutorBuildingStrategies[executorStrategy](p, task, duration)
	p.jobMap[job.uuid] = job
	p.logger.Printf("task %d has been scheduled\n", uuid)
	return job.uuid
}

func (p *JobPool) ScheduleTimeoutJob(task func(), duration time.Duration) int64 {
	return p.scheduleJob(task, duration, timeoutExecutorStrategy)
}

func (p *JobPool) ScheduleIntervalJob(task func(), duration time.Duration) int64 {
	return p.scheduleJob(task, duration, intervalExecutorStrategy)
}

func (p *JobPool) CancelJob(uuid int64) bool {
	if p.HasJob(uuid) {
		p.logger.Printf("Can not find job %d\n", uuid)
		return false
	}
	p.transitJobStatus(uuid, JobStatusTerminating)
	return true
}
