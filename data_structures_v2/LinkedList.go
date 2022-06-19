package data_structures

import (
	"github.com/dlshle/gommon/utils"
	"sync"
)

func defaultComparator[T comparable](a T, b T) int {
	if a == b {
		return 0
	}
	return 1
}

type listNode[T comparable] struct {
	prev *listNode[T]
	next *listNode[T]
	val  T
}

type LinkedList[T comparable] struct {
	head    *listNode[T]
	tail    *listNode[T]
	lock    *sync.RWMutex
	safe    bool
	size    int
	zeroVal T

	comparator func(T, T) int
}

func NewLinkedList[T comparable](safe bool) *LinkedList[T] {
	return &LinkedList[T]{
		lock: new(sync.RWMutex),
		safe: safe,
		size: 0,
	}
}

func (l *LinkedList[T]) withWrite(cb func()) {
	if l.safe {
		l.lock.Lock()
		defer l.lock.Unlock()
	}
	cb()
}

func (l *LinkedList[T]) withRead(cb func() interface{}) interface{} {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return cb()
}

func (l *LinkedList[T]) headNode() *listNode[T] {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return l.head
}

func (l *LinkedList[T]) tailNode() *listNode[T] {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return l.tail
}

func (l *LinkedList[T]) setHead(node *listNode[T]) {
	l.withWrite(func() {
		l.head = node
	})
}

func (l *LinkedList[T]) setTail(node *listNode[T]) {
	l.withWrite(func() {
		l.tail = node
	})
}

func (l *LinkedList[T]) Size() int {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return l.size
}

func (l *LinkedList[T]) Head() T {
	if l.head == nil {
		return l.zeroVal
	}
	return l.head.val
}

func (l *LinkedList[T]) Tail() T {
	if l.tail == nil {
		return l.zeroVal
	}
	return l.tail.val
}

func (l *LinkedList[T]) isValidIndex(index int, validateForInsert bool) bool {
	upperBound := l.Size()
	if validateForInsert {
		upperBound++
	}
	return l.Size() != 0 && index >= 0 && (index < upperBound)
}

func (l *LinkedList[T]) getNode(index int) *listNode[T] {
	if !l.isValidIndex(index, false) {
		return nil
	}
	if index == 0 {
		return l.headNode()
	}
	if index == l.Size()-1 {
		return l.tailNode()
	}
	return l.withRead(func() interface{} {
		var curr *listNode[T]
		fromHead := index <= (l.size / 2)
		offset := utils.ConditionalPick(fromHead, index, l.size-index+1).(int)
		if fromHead {
			curr = l.head
		} else {
			curr = l.tail
		}
		for offset > 0 {
			curr = utils.ConditionalGet(fromHead,
				func() interface{} { return curr.next },
				func() interface{} { return curr.prev }).(*listNode[T])
			offset--
		}
		return curr
	}).(*listNode[T])
}

func (l *LinkedList[T]) initFirstNode(value T) {
	l.withWrite(func() {
		l.head = &listNode[T]{val: value}
		l.tail = l.head
		l.size++
	})
}

func (l *LinkedList[T]) insertBeforeNode(node *listNode[T], value T) *listNode[T] {
	if node == nil {
		return nil
	}
	var newNode *listNode[T]
	l.withWrite(func() {
		newNode = &listNode[T]{
			prev: node.prev,
			next: node,
			val:  value,
		}
		if node.prev != nil {
			node.prev.next = newNode
		}
		node.prev = newNode
		l.size++
	})
	return newNode
}

func (l *LinkedList[T]) insertAfterNode(node *listNode[T], value T) *listNode[T] {
	if node == nil {
		return nil
	}
	var newNode *listNode[T]
	l.withWrite(func() {
		newNode = &listNode[T]{
			prev: node,
			next: node.next,
			val:  value,
		}
		if node.next != nil {
			node.next.prev = newNode
		}
		node.next = newNode
		l.size++
	})
	return newNode
}

func (l *LinkedList[T]) insert(index int, value T) bool {
	if l.Size() == 0 && index == 0 {
		l.initFirstNode(value)
		return true
	}
	if !l.isValidIndex(index, true) {
		return false
	}
	if index < l.Size() {
		l.insertBeforeNode(l.getNode(index), value)
	} else {
		// index == size
		l.Append(value)
	}
	return true
}

func (l *LinkedList[T]) Get(index int) T {
	node := l.getNode(index)
	if node == nil {
		return l.zeroVal
	}
	return node.val
}

func (l *LinkedList[T]) removeOnNode(node *listNode[T]) *listNode[T] {
	if node == nil {
		return nil
	}
	l.withWrite(func() {
		if node.prev != nil {
			node.prev.next = node.next
		}
		if node.next != nil {
			node.next.prev = node.prev
		}
		l.size--
	})
	return node
}

func (l *LinkedList[T]) remove(index int) *listNode[T] {
	node := l.getNode(index)
	if node == nil {
		return nil
	}
	return l.removeOnNode(node)
}

func (l *LinkedList[T]) Remove(index int) T {
	node := l.remove(index)
	if node != nil {
		return node.val
	}
	return l.zeroVal
}

func (l *LinkedList[T]) Insert(index int, value T) bool {
	return l.insert(index, value)
}

func (l *LinkedList[T]) Append(value T) {
	if l.tailNode() == nil {
		l.initFirstNode(value)
	} else if l.Size() == 1 {
		l.withWrite(func() {
			l.tail = &listNode[T]{
				val:  value,
				prev: l.head,
			}
			l.head.next = l.tail
			l.size++
		})
	} else {
		newTail := l.insertAfterNode(l.tailNode(), value)
		l.withWrite(func() {
			l.tail = newTail
		})
	}
}

func (l *LinkedList[T]) Prepend(value T) {
	if l.headNode() == nil {
		l.initFirstNode(value)
	} else if l.Size() == 1 {
		l.withWrite(func() {
			l.head = &listNode[T]{
				val:  value,
				next: l.tail,
			}
			l.tail.prev = l.head
			l.size++
		})
	} else {
		newHead := l.insertBeforeNode(l.headNode(), value)
		l.withWrite(func() {
			l.head = newHead
		})
	}
}

// get and remove first
func (l *LinkedList[T]) Poll() T {
	node := l.removeOnNode(l.headNode())
	if node != nil {
		l.setHead(node.next)
		return node.val
	}
	return l.zeroVal
}

// get and remove last
func (l *LinkedList[T]) Pop() T {
	node := l.removeOnNode(l.tailNode())
	if node != nil {
		l.setTail(node.prev)
		return node.val
	}
	return l.zeroVal
}

func (l *LinkedList[T]) ForEach(cb func(item T, index int)) {
	l.withRead(func() interface{} {
		counter := 0
		curr := l.head
		for curr != nil {
			cb(curr.val, counter)
			curr = curr.next
			counter++
		}
		return nil
	})
}

func (l *LinkedList[T]) Map(cb func(item T, index int) T) *LinkedList[T] {
	list := NewLinkedList[T](true)
	l.ForEach(func(item T, index int) {
		list.Append(cb(item, index))
	})
	return list
}

func (l *LinkedList[T]) ToSlice() []T {
	slice := make([]T, l.size, l.size)
	l.ForEach(func(val T, index int) {
		slice[index] = val
	})
	return slice
}

func (l *LinkedList[T]) Search(val T, comparator func(a T, b T) int) int {
	index := -1
	l.ForEach(func(value T, i int) {
		if comparator(value, val) == 0 {
			index = i
		}
	})
	return index
}

func (l *LinkedList[T]) IndexOf(val T) int {
	if l.comparator != nil {
		return l.Search(val, l.comparator)
	}
	return l.Search(val, defaultComparator[T])
}

func (l *LinkedList[T]) Has(val T) bool {
	return l.IndexOf(val) != -1
}

func (l *LinkedList[T]) SetSafe() {
	l.withWrite(func() {
		l.safe = true
	})
}

func (l *LinkedList[T]) SetUnsafe() {
	l.withWrite(func() {
		l.safe = false
	})
}

func (l *LinkedList[T]) IsSafe() bool {
	return l.withRead(func() interface{} {
		return l.safe
	}).(bool)
}
