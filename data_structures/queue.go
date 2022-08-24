package data_structures

import (
	"sync"
	"sync/atomic"
)

type Queue[T any] interface {
	Enqueue(T)
	Dequeue() T
	Size() int
	IsEmpty() bool
}

type queueNode[T any] struct {
	e    T
	next *queueNode[T]
}

type safeQueue[T any] struct {
	head  *queueNode[T]
	tail  *queueNode[T]
	size  int32
	mutex *sync.Mutex
}

func NewSafeQueue[T any]() Queue[T] {
	return &safeQueue[T]{
		nil,
		nil,
		0,
		new(sync.Mutex),
	}
}

func (q *safeQueue[T]) Enqueue(e T) {
	q.mutex.Lock()
	if q.size == 0 {
		q.head = &queueNode[T]{e, nil}
		q.tail = q.head
		q.mutex.Unlock()
		q.incrSizeBy(1)
		return
	}
	q.tail.next = &queueNode[T]{e, nil}
	q.tail = q.tail.next
	q.mutex.Unlock()
	q.incrSizeBy(1)
}

func (q *safeQueue[T]) Dequeue() T {
	var val T
	q.mutex.Lock()
	if q.size == 0 {
		q.mutex.Unlock()
		return val
	}
	lastHead := q.head
	val = lastHead.e
	q.head = q.head.next
	lastHead = nil
	q.mutex.Unlock()
	q.incrSizeBy(-1)
	return val
}

func (q *safeQueue[T]) Size() int {
	return int(atomic.LoadInt32(&q.size))
}

func (q *safeQueue[T]) IsEmpty() bool {
	return q.Size() == 0
}

func (q *safeQueue[T]) incrSizeBy(d int32) {
	atomic.AddInt32(&q.size, d)
}
