package async

import (
	"sync"
	"sync/atomic"
)

type taskNode struct {
	t    AsyncTask
	next *taskNode
}

type taskQueue struct {
	head  *taskNode
	tail  *taskNode
	size  int32
	mutex *sync.Mutex
}

func newTaskQueue() *taskQueue {
	return &taskQueue{
		nil,
		nil,
		0,
		new(sync.Mutex),
	}
}

func (q *taskQueue) addTask(e AsyncTask) {
	q.mutex.Lock()
	if q.size == 0 {
		q.head = &taskNode{e, nil}
		q.tail = q.head
		q.mutex.Unlock()
		q.incrSizeBy(1)
		return
	}
	q.tail.next = &taskNode{e, nil}
	q.tail = q.tail.next
	q.mutex.Unlock()
	q.incrSizeBy(1)
}

func (q *taskQueue) getTask() AsyncTask {
	var val AsyncTask = nil
	q.mutex.Lock()
	if q.size == 0 {
		q.mutex.Unlock()
		return val
	}
	lastHead := q.head
	val = lastHead.t
	q.head = q.head.next
	lastHead = nil
	q.mutex.Unlock()
	q.incrSizeBy(-1)
	return val
}

func (q *taskQueue) numTasks() int {
	return int(atomic.LoadInt32(&q.size))
}

func (q *taskQueue) isEmpty() bool {
	return q.numTasks() == 0
}

func (q *taskQueue) incrSizeBy(d int32) {
	atomic.AddInt32(&q.size, d)
}
