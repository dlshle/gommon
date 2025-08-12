package cache

import (
	"container/heap"
	"sync"
	"time"
)

// LFUCache implements a Least Frequently Used cache
type LFUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*lfuItem[K, V]
	freqHeap *lfuHeap[K, V]
	mutex    sync.RWMutex
}

type lfuItem[K comparable, V any] struct {
	key       K
	value     V
	freq      int
	ttl       time.Time
	heapIndex int
}

// lfuHeap implements heap.Interface and holds lfuItems
type lfuHeap[K comparable, V any] []*lfuItem[K, V]

func (h lfuHeap[K, V]) Len() int           { return len(h) }
func (h lfuHeap[K, V]) Less(i, j int) bool { return h[i].freq < h[j].freq }
func (h lfuHeap[K, V]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}

func (h *lfuHeap[K, V]) Push(x interface{}) {
	n := len(*h)
	item := x.(*lfuItem[K, V])
	item.heapIndex = n
	*h = append(*h, item)
}

func (h *lfuHeap[K, V]) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil      // avoid memory leak
	item.heapIndex = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// NewLFUCache creates a new LFU cache with the specified capacity
func NewLFUCache[K comparable, V any](capacity int) *LFUCache[K, V] {
	h := &lfuHeap[K, V]{}
	heap.Init(h)

	return &LFUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*lfuItem[K, V]),
		freqHeap: h,
	}
}

// Get retrieves a value from the cache by key
func (c *LFUCache[K, V]) Get(key K) (V, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, exists := c.items[key]; exists {
		// Check TTL
		if !item.ttl.IsZero() && time.Now().After(item.ttl) {
			// Expired, remove it
			heap.Remove(c.freqHeap, item.heapIndex)
			delete(c.items, key)
			var zero V
			return zero, false
		}

		// Update frequency
		item.freq++
		heap.Fix(c.freqHeap, item.heapIndex)
		return item.value, true
	}

	var zero V
	return zero, false
}

// Set adds or updates a value in the cache without TTL
func (c *LFUCache[K, V]) Set(key K, value V) error {
	return c.SetWithTTL(key, value, 0)
}

// SetWithTTL adds or updates a value in the cache with optional TTL
func (c *LFUCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if key already exists
	if item, exists := c.items[key]; exists {
		// Update existing entry
		item.value = value
		item.freq++
		if ttl > 0 {
			item.ttl = time.Now().Add(ttl)
		} else {
			item.ttl = time.Time{}
		}
		heap.Fix(c.freqHeap, item.heapIndex)
		return nil
	}

	// Create new item
	item := &lfuItem[K, V]{
		key:   key,
		value: value,
		freq:  1,
	}
	if ttl > 0 {
		item.ttl = time.Now().Add(ttl)
	}

	// Add to heap
	heap.Push(c.freqHeap, item)
	c.items[key] = item

	// Evict if over capacity
	if len(c.items) > c.capacity {
		c.evict()
	}

	return nil
}

// Delete removes a value from the cache by key
func (c *LFUCache[K, V]) Delete(key K) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if item, exists := c.items[key]; exists {
		heap.Remove(c.freqHeap, item.heapIndex)
		delete(c.items, key)
	}

	return nil
}

// Has checks if a key exists in the cache
func (c *LFUCache[K, V]) Has(key K) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.items[key]
	return exists
}

// Len returns the number of items in the cache
func (c *LFUCache[K, V]) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.items)
}

// Clear removes all items from the cache
func (c *LFUCache[K, V]) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[K]*lfuItem[K, V])
	c.freqHeap = &lfuHeap[K, V]{}
	heap.Init(c.freqHeap)

	return nil
}

// Keys returns all keys in the cache
func (c *LFUCache[K, V]) Keys() []K {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]K, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	return keys
}

// GetWithLoader retrieves a value from the cache, using the loader function if not present
func (c *LFUCache[K, V]) GetWithLoader(key K, loader func(K) (V, error)) (V, error) {
	return c.GetWithLoaderAndTTL(key, loader, 0)
}

func (c *LFUCache[K, V]) GetWithLoaderAndTTL(key K, loader func(K) (V, error), ttl time.Duration) (V, error) {
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	value, err := loader(key)
	if err != nil {
		var zero V
		return zero, err
	}

	// Default TTL of 0 (no expiration)
	c.SetWithTTL(key, value, ttl)
	return value, nil
}

// evict removes the least frequently used item
func (c *LFUCache[K, V]) evict() {
	// Remove the item with the lowest frequency
	if c.freqHeap.Len() > 0 {
		item := heap.Pop(c.freqHeap).(*lfuItem[K, V])
		delete(c.items, item.key)
	}
}
