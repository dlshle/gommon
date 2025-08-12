# Cache

A generic cache library for Go with support for LRU and LFU eviction policies, TTL, and loading cache patterns.

## Features

- **Generic Support**: Works with any key/value types that satisfy Go's comparable constraint
- **LRU Cache**: Least Recently Used eviction policy
- **LFU Cache**: Least Frequently Used eviction policy
- **TTL Support**: Time-to-live for cached entries
- **Loading Cache**: Automatic loading of values when not present (similar to Caffeine/Guava)
- **Thread-Safe**: All operations are safe for concurrent use

## Installation

```bash
go get github.com/dlshle/gommon/cache
```

## Usage

### LRU Cache

```go
import "github.com/dlshle/gommon/cache"

// Create an LRU cache with capacity of 1000 entries
lruCache := cache.NewLRUCache[string, int](1000)

// Set a value without TTL
lruCache.Set("key1", 42)

// Set a value with TTL
lruCache.SetWithTTL("key1", 42, time.Minute)

// Get a value
if val, ok := lruCache.Get("key1"); ok {
    fmt.Println("Value:", val)
}

// Use GetWithLoader for automatic loading
val, err := lruCache.GetWithLoader("key2", func(key string) (int, error) {
    // Load value from database or other source
    return 100, nil
})
```

### LFU Cache

```go
// Create an LFU cache with capacity of 1000 entries
lfuCache := cache.NewLFUCache[string, int](1000)

// Set a value without TTL
lfuCache.Set("key1", 42)

// Set a value with TTL
lfuCache.SetWithTTL("key1", 42, time.Minute)

// Get a value
if val, ok := lfuCache.Get("key1"); ok {
    fmt.Println("Value:", val)
}
```

## Interface

All cache implementations implement the `Cache` interface:

```go
type Cache[K comparable, V any] interface {
    Get(key K) (V, bool)
    Set(key K, value V) error
    SetWithTTL(key K, value V, ttl time.Duration) error
    Delete(key K) error
    Has(key K) bool
    Len() int
    Clear() error
    Keys() []K
    GetWithLoader(key K, loader func(K) (V, error)) (V, error)
}
```

## License

MIT