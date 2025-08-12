// Package cache provides implementations of LRU and LFU caching algorithms
// with support for generics, TTL, and loading cache patterns.
//
// Example usage:
//
//	// Create an LRU cache
//	lruCache := cache.NewLRUCache[string, int](100)
//
//	// Set a value without TTL
//	lruCache.Set("key1", 42)
//
//	// Set a value with TTL
//	lruCache.SetWithTTL("key1", 42, time.Minute)
//
//	// Get a value
//	if val, ok := lruCache.Get("key1"); ok {
//		fmt.Println("Value:", val)
//	}
//
//	// Use GetWithLoader for automatic loading
//	val, err := lruCache.GetWithLoader("key2", func(key string) (int, error) {
//		// Load value from database or other source
//		return 100, nil
//	})
//
//	// Create an LFU cache
//	lfuCache := cache.NewLFUCache[string, int](100)
//	lfuCache.Set("key1", 42)
//	lfuCache.SetWithTTL("key1", 42, time.Minute)
package cache