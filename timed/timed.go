package timed

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/dlshle/gommon/async"
	"github.com/dlshle/gommon/logger"
)

const (
	JobStatusWaiting     = 0
	JobStatusRunning     = 1
	JobStatusDone        = 2
	JobStatusTerminating = 4

	EvictPolicyCancelLast = 0

	MinPoolSize = 16
	MaxPoolSize = 1024 * 8
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

var jobPoolEvictStrategies = make(map[int]func(p *JobPool, uuid int64))
var jobPoolTransitStrategies = make(map[int]func(p *JobPool, uuid int64))
var jobExecutorBuildingStrategies = make(map[int]func(p *JobPool, Job func(), duration time.Duration) *Job)
var statusStringMap = make(map[int]string)
var globalPool = NewJobPool("default", 1024, true)

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
			delete(p.jobMap, uuid)
		} else {
			p.logger.Printf("WARN job[%d] does not exist!\n", uuid)
		}
	}
}

func initTransitStrategy() {
	jobPoolTransitStrategies[transitWaitingRunning] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d started running\n", uuid)
	}
	jobPoolTransitStrategies[transitWaitingDone] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d is canceled\n", uuid)
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transitWaitingTerminating] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d is terminating after waiting...\n", uuid)
	}
	jobPoolTransitStrategies[transitRunningDone] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d finished\n", uuid)
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transitRunningTerminating] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d is terminating after running...\n", uuid)
	}
	jobPoolTransitStrategies[transitRunningWaiting] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d interval done, onto the next interval...\n", uuid)
	}
	jobPoolTransitStrategies[transitTerminatingDone] = func(p *JobPool, uuid int64) {
		p.logger.Printf("Job %d final interval done, Job has been terminated\n", uuid)
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
}

func initPoolExecutorBuilder() {
	transitWithTerminateCheck := func(p *JobPool, uuid int64, status int) bool {
		if p.GetStatus(uuid) == JobStatusTerminating {
			p.logger.Printf("Job %d received terminating signal, will terminate the job.\n", uuid)
			p.transitJobStatus(uuid, JobStatusDone)
			return false
		}
		p.transitJobStatus(uuid, status)
		return true
	}
	jobExecutorBuildingStrategies[timeoutExecutorStrategy] = func(p *JobPool, Job func(), duration time.Duration) *Job {
		uuid := time.Now().Unix()
		return NewJob(uuid, func() {
			time.Sleep(duration)
			if !transitWithTerminateCheck(p, uuid, JobStatusRunning) {
				return
			}
			Job()
			p.transitJobStatus(uuid, JobStatusDone)
		})
	}
	jobExecutorBuildingStrategies[intervalExecutorStrategy] = func(p *JobPool, Job func(), duration time.Duration) *Job {
		uuid := time.Now().Unix()
		return NewJob(uuid, func() {
			for {
				time.Sleep(duration)
				if !transitWithTerminateCheck(p, uuid, JobStatusRunning) {
					return
				}
				Job()
				if !transitWithTerminateCheck(p, uuid, JobStatusWaiting) {
					return
				}
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
	return &Job{executor, uuid, JobStatusWaiting}
}

type JobPool struct {
	id           string
	jobMap       map[int64]*Job
	maxSize      int
	evictPolicy  int
	finishPolicy int
	logger       *logger.SimpleLogger
	*sync.RWMutex
}

type IJobPool interface {
	makeExecutor()
	scheduleJob(Job func(), duration time.Duration, executorStrategy int, runAsync bool) int64
	ScheduleTimeoutJob(Job func(), duration time.Duration) int64
	ScheduleAsyncTimeoutJob(Job func(), duration time.Duration) int64
	ScheduleIntervalJob(Job func(), duration time.Duration) int64
	ScheduleAsyncIntervalJob(Job func(), duration time.Duration) int64
	CancelJob(uuid int64) bool
	HasJob(uuid int64) bool
	GetStatus(uuid int64) int
	setStatus(uuid int64, status int)
	transitJobStatus(uuid int64, status int)
	Size() int
	Verbose(use bool)
}

func NewJobPool(id string, maxSize int, verbose bool) *JobPool {
	if maxSize < MinPoolSize {
		maxSize = MinPoolSize
	} else if maxSize > MaxPoolSize {
		maxSize = MaxPoolSize
	}
	return &JobPool{id,
		make(map[int64]*Job),
		maxSize,
		0,
		0,
		logger.New(os.Stdout, fmt.Sprintf("JobPool[pool-%s]", id), verbose),
		new(sync.RWMutex),
	}
}

func (p *JobPool) transitJobStatus(uuid int64, status int) {
	fromStatus := p.GetStatus(uuid)
	if fromStatus == status {
		p.logger.Printf("Job[%d] Ignore invalid status transition(%s to %s)\n", uuid, statusStringMap[status], statusStringMap[status])
		return
	}
	transitHandler := jobPoolTransitStrategies[fromStatus*10+status]
	p.logger.Printf("Job[%d] Transiting job status from %s to %s.\n", uuid, statusStringMap[fromStatus], statusStringMap[status])
	if transitHandler == nil {
		p.logger.Printf("Job[%d] Invalid job status transit from %s to %s. Job will be canceled.\n", uuid, statusStringMap[fromStatus], statusStringMap[status])
		p.CancelJob(uuid)
		return
	}
	p.setStatus(uuid, status)
	transitHandler(p, uuid)
}

func (p *JobPool) Size() int {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	return len(p.jobMap)
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

func (p *JobPool) Verbose(use bool) {
	p.logger.Verbose(use)
}

func (p *JobPool) scheduleJob(Job func(), duration time.Duration, executorStrategy int, runAsync bool) int64 {
	if p.Size() >= p.maxSize {
		p.logger.Println("Error: max pool size has been reached, new job will be evicted!")
		return -1
	}
	job := jobExecutorBuildingStrategies[executorStrategy](p, Job, duration)
	uuid := job.uuid
	p.jobMap[uuid] = job
	p.logger.Printf("Job %d has been scheduled\n", uuid)
	if runAsync {
		async.Schedule(job.executor)
	} else {
		job.executor()
	}
	return uuid
}

func (p *JobPool) ScheduleTimeoutJob(Job func(), duration time.Duration) int64 {
	return p.scheduleJob(Job, duration, timeoutExecutorStrategy, false)
}

func (p *JobPool) ScheduleIntervalJob(Job func(), duration time.Duration) int64 {
	return p.scheduleJob(Job, duration, intervalExecutorStrategy, false)
}

func (p *JobPool) ScheduleAsyncTimeoutJob(Job func(), duration time.Duration) int64 {
	return p.scheduleJob(Job, duration, timeoutExecutorStrategy, true)
}

func (p *JobPool) ScheduleAsyncIntervalJob(Job func(), duration time.Duration) int64 {
	return p.scheduleJob(Job, duration, intervalExecutorStrategy, true)
}

func (p *JobPool) CancelJob(uuid int64) bool {
	if !p.HasJob(uuid) {
		p.logger.Printf("Can not find job %d\n", uuid)
		return false
	}
	p.transitJobStatus(uuid, JobStatusTerminating)
	return true
}

// --------------- Static Functions --------------- //
func RunTimeout(job func(), duration time.Duration) int64 {
	return globalPool.ScheduleTimeoutJob(job, duration)
}

func RunAsyncTimeout(job func(), duration time.Duration) int64 {
	return globalPool.ScheduleAsyncTimeoutJob(job, duration)
}

func RunInterval(job func(), duration time.Duration) int64 {
	return globalPool.ScheduleIntervalJob(job, duration)
}

func RunAsyncInterval(job func(), duration time.Duration) int64 {
	return globalPool.ScheduleAsyncIntervalJob(job, duration)
}

func Cancel(uuid int64) bool {
	return globalPool.CancelJob(uuid)
}
