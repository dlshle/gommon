package jobs

import (
	"log"
	"os"
	"sync"
	"time"
)

const (
	JOB_STATUS_WAITING     = 0
	JOB_STATUS_RUNNING     = 1
	JOB_STATUS_DONE        = 2
	JOB_STATUS_TERMINATING = 4

	EVICT_POLICY_CANCEL_LAST = 0

	FINISH_POLICY_REMOVE = 0
)

const (
	timeout_executor_strategy  = 0
	interval_executor_strategy = 1

	transit_WAITING_RUNNING     = 1
	transit_WAITING_DONE        = 2
	transit_WAITING_TERMINATING = 4
	transit_TERMINATING_DONE    = 42 // interval job only
	transit_RUNNING_DONE        = 12
	transit_RUNNING_TERMINATING = 14 // interval job only
	transit_RUNNING_WAITING     = 10 // interval job only
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
	jobPoolEvictStrategies[EVICT_POLICY_CANCEL_LAST] = func(p *JobPool, uuid int64) {
		job := p.jobMap[uuid]
		if job != nil {
			job.Status = JOB_STATUS_DONE
			delete(p.jobMap, uuid)
		} else {
			p.logger.Printf("WARN job[%d] does not exist!\n", uuid)
		}
	}
}

func initTransitStrategy() {
	jobPoolTransitStrategies[transit_WAITING_RUNNING] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d started running\n")
		p.jobMap[uuid].executor()
	}
	jobPoolTransitStrategies[transit_WAITING_DONE] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d is canceled\n")
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transit_WAITING_TERMINATING] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d is terminating after waiting...\n")
	}
	jobPoolTransitStrategies[transit_RUNNING_DONE] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d finished\n")
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transit_RUNNING_TERMINATING] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d is terminating after running...\n")
	}
	jobPoolTransitStrategies[transit_RUNNING_WAITING] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d interval done, onto the next interval...\n")
	}
	jobPoolTransitStrategies[transit_TERMINATING_DONE] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Task %d final interval done, task has been terminated\n")
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
}

func initPoolExecutorBuilder() {
	shouldTerminateJob := func(p *JobPool, uuid int64) bool {
		return p.GetStatus(uuid) == JOB_STATUS_TERMINATING
	}
	jobPoolExecutorBuildingStrategies[timeout_executor_strategy] = func(p *JobPool, task func(), duration time.Duration) *Job {
		uuid := time.Now().Unix()
		return NewJob(uuid, func() {
			time.Sleep(duration)
			if shouldTerminateJob(p, uuid) {
				p.transitJobStatus(uuid, JOB_STATUS_DONE)
				return
			}
			p.transitJobStatus(uuid, JOB_STATUS_RUNNING)
			task()
			p.transitJobStatus(uuid, JOB_STATUS_DONE)
		})
	}
	jobPoolExecutorBuildingStrategies[interval_executor_strategy] = func(p *JobPool, task func(), duration time.Duration) *Job {
		uuid := time.Now().Unix()
		return NewJob(uuid, func() {
			for {
				time.Sleep(duration)
				p.transitJobStatus(uuid, JOB_STATUS_RUNNING)
				if shouldTerminateJob(p, uuid) {
					p.transitJobStatus(uuid, JOB_STATUS_DONE)
					return
				}
				task()
				if shouldTerminateJob(p, uuid) {
					p.transitJobStatus(uuid, JOB_STATUS_DONE)
					return
				}
				p.transitJobStatus(uuid, JOB_STATUS_WAITING)
			}
			p.transitJobStatus(uuid, JOB_STATUS_DONE)
		})
	}
}

func initStatusStringMap() {
	statusStringMap[JOB_STATUS_WAITING] = "WAITING"
	statusStringMap[JOB_STATUS_RUNNING] = "RUNNING"
	statusStringMap[JOB_STATUS_DONE] = "DONE"
	statusStringMap[JOB_STATUS_TERMINATING] = "TERMINATING"
}

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
	logger       log.Logger
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
		return JOB_STATUS_DONE
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
	return p.scheduleJob(task, duration, timeout_executor_strategy)
}

func (p *JobPool) ScheduleIntervalJob(task func(), duration time.Duration) int64 {
	return p.scheduleJob(task, duration, interval_executor_strategy)
}

func (p *JobPool) CancelJob(uuid int64) bool {
	if p.HasJob(uuid) {
		p.logger.Printf("Can not find job %d\n", uuid)
		return false
	}
	p.transitJobStatus(uuid, JOB_STATUS_TERMINATING)
	return true
}
