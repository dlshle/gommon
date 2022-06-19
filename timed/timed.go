package timed

import (
	"github.com/dlshle/gommon/async"
	"github.com/dlshle/gommon/logger"
	"github.com/dlshle/gommon/utils"
	"os"
	"runtime"
	"sync"
	"time"
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

var jobPoolEvictStrategies = make(map[int]func(p *jobPool, uuid int64))
var jobPoolTransitStrategies = make(map[int]func(p *jobPool, uuid int64))
var jobExecutorBuildingStrategies = make(map[int]func(p *jobPool, Job func(), duration time.Duration) *Job)
var statusStringMap = make(map[int]string)

func init() {
	initPoolEvictStrategy()
	initPoolExecutorBuilder()
	initStatusStringMap()
	initTransitStrategy()
}

func initPoolEvictStrategy() {
	jobPoolEvictStrategies[EvictPolicyCancelLast] = func(p *jobPool, uuid int64) {
		job := p.jobMap[uuid]
		if job != nil {
			delete(p.jobMap, uuid)
		} else {
			p.logger.Infof("WARN job[%d] does not exist!\n", uuid)
		}
	}
}

func initTransitStrategy() {
	jobPoolTransitStrategies[transitWaitingRunning] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d started running\n", uuid)
	}
	jobPoolTransitStrategies[transitWaitingDone] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d is canceled\n", uuid)
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transitWaitingTerminating] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d is terminating after waiting...\n", uuid)
	}
	jobPoolTransitStrategies[transitRunningDone] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d finished\n", uuid)
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
	jobPoolTransitStrategies[transitRunningTerminating] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d is terminating after running...\n", uuid)
	}
	jobPoolTransitStrategies[transitRunningWaiting] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d interval done, onto the next interval...\n", uuid)
	}
	jobPoolTransitStrategies[transitTerminatingDone] = func(p *jobPool, uuid int64) {
		p.logger.Infof("Job %d final interval done, Job has been terminated\n", uuid)
		jobPoolEvictStrategies[p.evictPolicy](p, uuid)
	}
}

func initPoolExecutorBuilder() {
	transitWithTerminateCheck := func(p *jobPool, uuid int64, status int) bool {
		if p.GetStatus(uuid) == JobStatusTerminating {
			p.logger.Infof("Job %d received terminating signal, will terminate the job.\n", uuid)
			p.transitJobStatus(uuid, JobStatusDone)
			return false
		}
		p.transitJobStatus(uuid, status)
		return true
	}
	jobExecutorBuildingStrategies[timeoutExecutorStrategy] = func(p *jobPool, Job func(), duration time.Duration) *Job {
		uuid := time.Now().Unix() + utils.Rando.Int63()
		return NewJob(uuid, func() {
			time.Sleep(duration)
			if !transitWithTerminateCheck(p, uuid, JobStatusRunning) {
				return
			}
			Job()
			p.transitJobStatus(uuid, JobStatusDone)
		})
	}
	jobExecutorBuildingStrategies[intervalExecutorStrategy] = func(p *jobPool, Job func(), duration time.Duration) *Job {
		uuid := time.Now().Unix() + utils.Rando.Int63()
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
	id       int64
	status   int
}

func NewJob(uuid int64, executor func()) *Job {
	return &Job{executor, uuid, JobStatusWaiting}
}

type jobPool struct {
	id           string
	jobMap       map[int64]*Job
	maxSize      int
	evictPolicy  int
	finishPolicy int
	logger       logger.Logger
	executor     async.Executor
	*sync.RWMutex
}

type JobPool interface {
	Timeout(Job func(), duration time.Duration) int64
	Interval(Job func(), duration time.Duration) int64
	Cancel(uuid int64) bool
	HasJob(uuid int64) bool
	GetStatus(uuid int64) int
	Size() int
	Verbose(use bool)
}

func NewJobPool(id string, maxSize int, verbose bool) JobPool {
	if maxSize < MinPoolSize {
		maxSize = MinPoolSize
	} else if maxSize > MaxPoolSize {
		maxSize = MaxPoolSize
	}
	return &jobPool{id,
		make(map[int64]*Job),
		maxSize,
		0,
		0,
		logger.StdOutLevelLogger("JobPool[pool-" + id + "]"),
		async.NewAsyncPool("[JobPoolExecutor"+id+"]", runtime.NumCPU()*4, runtime.NumCPU()*32),
		new(sync.RWMutex),
	}
}

func (p *jobPool) transitJobStatus(uuid int64, status int) {
	fromStatus := p.GetStatus(uuid)
	if fromStatus == status {
		p.logger.Infof("Job[%d] Ignore invalid status transition(%s to %s)\n", uuid, statusStringMap[status], statusStringMap[status])
		return
	}
	transitHandler := jobPoolTransitStrategies[fromStatus*10+status]
	p.logger.Infof("Job[%d] Transiting job status from %s to %s.\n", uuid, statusStringMap[fromStatus], statusStringMap[status])
	if transitHandler == nil {
		p.logger.Infof("Job[%d] Invalid job status transit from %s to %s. Job will be canceled.\n", uuid, statusStringMap[fromStatus], statusStringMap[status])
		p.Cancel(uuid)
		return
	}
	p.setStatus(uuid, status)
	transitHandler(p, uuid)
}

func (p *jobPool) Size() int {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	return len(p.jobMap)
}

func (p *jobPool) setStatus(uuid int64, status int) {
	p.RWMutex.Lock()
	defer p.RWMutex.Unlock()
	p.jobMap[uuid].status = status
}

func (p *jobPool) HasJob(id int64) bool {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	return p.jobMap[id] != nil
}

func (p *jobPool) GetStatus(id int64) int {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	job := p.jobMap[id]
	if job == nil {
		return JobStatusDone
	} else {
		return job.status
	}
}

func (p *jobPool) Verbose(use bool) {
	if use {
		p.logger.Writer(logger.NewConsoleLogWriter(os.Stdout))
	} else {
		p.logger.Writer(logger.NewNoopWriter())
	}
}

func (p *jobPool) scheduleJob(Job func(), duration time.Duration, executorStrategy int) int64 {
	if p.Size() >= p.maxSize {
		p.logger.Info("Error: max pool size has been reached, new job will be evicted!")
		return -1
	}
	job := jobExecutorBuildingStrategies[executorStrategy](p, Job, duration)
	uuid := job.id
	p.jobMap[uuid] = job
	p.logger.Infof("Job %d has been scheduled\n", uuid)
	p.executor.Execute(job.executor)
	return uuid
}

func (p *jobPool) Timeout(Job func(), duration time.Duration) int64 {
	return p.scheduleJob(Job, duration, timeoutExecutorStrategy)
}

func (p *jobPool) Interval(Job func(), duration time.Duration) int64 {
	return p.scheduleJob(Job, duration, intervalExecutorStrategy)
}

func (p *jobPool) Cancel(uuid int64) bool {
	if !p.HasJob(uuid) {
		p.logger.Infof("Can not find job %d\n", uuid)
		return false
	}
	p.logger.Infof("cancel job %s", uuid)
	p.transitJobStatus(uuid, JobStatusTerminating)
	return true
}
