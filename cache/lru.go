package cache

import (
	"container/list"
	"sync"
	"time"
)

// LRUCache implements a Least Recently Used cache
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	list     *list.List
	mutex    sync.RWMutex
}

type entry[K comparable, V any] struct {
	key   K
	value V
	ttl   time.Time
}

// NewLRUCache creates a new LRU cache with the specified capacity
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		list:     list.New(),
	}
}

// Get retrieves a value from the cache by key
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.items[key]; exists {
		// Check TTL
		e := element.Value.(*entry[K, V])
		if !e.ttl.IsZero() && time.Now().After(e.ttl) {
			// Expired, remove it
			c.list.Remove(element)
			delete(c.items, key)
			var zero V
			return zero, false
		}

		// Move to front (most recently used)
		c.list.MoveToFront(element)
		return e.value, true
	}

	var zero V
	return zero, false
}

// Set adds or updates a value in the cache without TTL
func (c *LRUCache[K, V]) Set(key K, value V) error {
	return c.SetWithTTL(key, value, 0)
}

// SetWithTTL adds or updates a value in the cache with optional TTL
func (c *LRUCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if key already exists
	if element, exists := c.items[key]; exists {
		// Update existing entry
		e := element.Value.(*entry[K, V])
		e.value = value
		if ttl > 0 {
			e.ttl = time.Now().Add(ttl)
		} else {
			e.ttl = time.Time{}
		}
		c.list.MoveToFront(element)
		return nil
	}

	// Create new entry
	e := &entry[K, V]{
		key:   key,
		value: value,
	}
	if ttl > 0 {
		e.ttl = time.Now().Add(ttl)
	}

	// Add to front of list
	element := c.list.PushFront(e)
	c.items[key] = element

	// Evict oldest if over capacity
	if c.list.Len() > c.capacity {
		c.evict()
	}

	return nil
}

// Delete removes a value from the cache by key
func (c *LRUCache[K, V]) Delete(key K) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, exists := c.items[key]; exists {
		c.list.Remove(element)
		delete(c.items, key)
	}

	return nil
}

// Has checks if a key exists in the cache
func (c *LRUCache[K, V]) Has(key K) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.items[key]
	return exists
}

// Len returns the number of items in the cache
func (c *LRUCache[K, V]) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.list.Len()
}

// Clear removes all items from the cache
func (c *LRUCache[K, V]) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[K]*list.Element)
	c.list.Init()

	return nil
}

// Keys returns all keys in the cache
func (c *LRUCache[K, V]) Keys() []K {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]K, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	return keys
}

// GetWithLoader retrieves a value from the cache, using the loader function if not present
func (c *LRUCache[K, V]) GetWithLoader(key K, loader func(K) (V, error)) (V, error) {
	return c.GetWithLoaderAndTTL(key, loader, 0)
}

// GetWithLoader retrieves a value from the cache, using the loader function if not present
func (c *LRUCache[K, V]) GetWithLoaderAndTTL(key K, loader func(K) (V, error), ttl time.Duration) (V, error) {
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	value, err := loader(key)
	if err != nil {
		var zero V
		return zero, err
	}

	c.SetWithTTL(key, value, ttl)
	return value, nil
}

// evict removes the least recently used item
func (c *LRUCache[K, V]) evict() {
	// Remove from back of list (least recently used)
	element := c.list.Back()
	if element != nil {
		e := element.Value.(*entry[K, V])
		delete(c.items, e.key)
		c.list.Remove(element)
	}
}
