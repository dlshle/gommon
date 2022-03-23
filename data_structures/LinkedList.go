package data_structures

import (
	"sync"
	"github.com/dlshle/gommon/utils"
)

func defaultComparator(a interface{}, b interface{}) int {
	if a == b {
		return 0
	}
	return 1
}

type listNode struct {
	prev *listNode
	next *listNode
	val  interface{}
}

type LinkedList struct {
	head *listNode
	tail *listNode
	lock *sync.RWMutex
	safe bool
	size int

	comparator func(interface{}, interface{}) int
}

func NewLinkedList(safe bool) *LinkedList {
	return &LinkedList{
		lock: new(sync.RWMutex),
		safe: safe,
		size: 0,
	}
}

func (l *LinkedList) withWrite(cb func()) {
	if l.safe {
		l.lock.Lock()
		defer l.lock.Unlock()
	}
	cb()
}

func (l *LinkedList) withRead(cb func() interface{}) interface{} {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return cb()
}

func (l *LinkedList) headNode() *listNode {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return l.head
}

func (l *LinkedList) tailNode() *listNode {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return l.tail
}

func (l *LinkedList) setHead(node *listNode) {
	l.withWrite(func() {
		l.head = node
	})
}

func (l *LinkedList) setTail(node *listNode) {
	l.withWrite(func() {
		l.tail = node
	})
}

func (l *LinkedList) Size() int {
	if l.safe {
		l.lock.RLock()
		defer l.lock.RUnlock()
	}
	return l.size
}

func (l *LinkedList) Head() interface{} {
	if l.head == nil {
		return nil
	}
	return l.head.val
}

func (l *LinkedList) Tail() interface{} {
	if l.tail == nil {
		return nil
	}
	return l.tail.val
}

func (l *LinkedList) isValidIndex(index int, validateForInsert bool) bool {
	upperBound := l.Size()
	if validateForInsert {
		upperBound++
	}
	return l.Size() != 0 && index >= 0 && (index < upperBound)
}

func (l *LinkedList) getNode(index int) *listNode {
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
		var curr *listNode
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
				func() interface{} { return curr.prev }).(*listNode)
			offset--
		}
		return curr
	}).(*listNode)
}

func (l *LinkedList) initFirstNode(value interface{}) {
	l.withWrite(func() {
		l.head = &listNode{val: value}
		l.tail = l.head
		l.size++
	})
}

func (l *LinkedList) insertBeforeNode(node *listNode, value interface{}) *listNode {
	if node == nil {
		return nil
	}
	var newNode *listNode
	l.withWrite(func() {
		newNode = &listNode{
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

func (l *LinkedList) insertAfterNode(node *listNode, value interface{}) *listNode {
	if node == nil {
		return nil
	}
	var newNode *listNode
	l.withWrite(func() {
		newNode = &listNode{
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

func (l *LinkedList) insert(index int, value interface{}) bool {
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

func (l *LinkedList) Get(index int) interface{} {
	node := l.getNode(index)
	if node == nil {
		return nil
	}
	return node.val
}

func (l *LinkedList) removeOnNode(node *listNode) *listNode {
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

func (l *LinkedList) remove(index int) *listNode {
	node := l.getNode(index)
	if node == nil {
		return nil
	}
	return l.removeOnNode(node)
}

func (l *LinkedList) Remove(index int) interface{} {
	node := l.remove(index)
	if node != nil {
		return node.val
	}
	return nil
}

func (l *LinkedList) Insert(index int, value interface{}) bool {
	return l.insert(index, value)
}

func (l *LinkedList) Append(value interface{}) {
	if l.tailNode() == nil {
		l.initFirstNode(value)
	} else if l.Size() == 1 {
		l.withWrite(func() {
			l.tail = &listNode{
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

func (l *LinkedList) Prepend(value interface{}) {
	if l.headNode() == nil {
		l.initFirstNode(value)
	} else if l.Size() == 1 {
		l.withWrite(func() {
			l.head = &listNode{
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
func (l *LinkedList) Poll() interface{} {
	node := l.removeOnNode(l.headNode())
	if node != nil {
		l.setHead(node.next)
		return node.val
	}
	return nil
}

// get and remove last
func (l *LinkedList) Pop() interface{} {
	node := l.removeOnNode(l.tailNode())
	if node != nil {
		l.setTail(node.prev)
		return node.val
	}
	return nil
}

func (l *LinkedList) ForEach(cb func(item interface{}, index int)) {
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

func (l *LinkedList) Map(cb func(item interface{}, index int) interface{}) *LinkedList {
	list := NewLinkedList(true)
	l.ForEach(func(item interface{}, index int) {
		list.Append(cb(item, index))
	})
	return list
}

func (l *LinkedList) ToSlice() []interface{} {
	slice := make([]interface{}, l.size, l.size)
	l.ForEach(func(val interface{}, index int) {
		slice[index] = val
	})
	return slice
}

func (l *LinkedList) Search(val interface{}, comparator func(a interface{}, b interface{}) int) int {
	index := -1
	l.ForEach(func(value interface{}, i int) {
		if comparator(value, val) == 0 {
			index = i
		}
	})
	return index
}

func (l *LinkedList) IndexOf(val interface{}) int {
	if l.comparator != nil {
		return l.Search(val, l.comparator)
	}
	return l.Search(val, defaultComparator)
}

func (l *LinkedList) Has(val interface{}) bool {
	return l.IndexOf(val) != -1
}

func (l *LinkedList) SetSafe() {
	l.withWrite(func() {
		l.safe = true
	})
}

func (l *LinkedList) SetUnsafe() {
	l.withWrite(func() {
		l.safe = false
	})
}

func (l *LinkedList) IsSafe() bool {
	return l.withRead(func() interface{} {
		return l.safe
	}).(bool)
}
