package cache

import (
	"time"
)

// Cache defines the common interface for all cache implementations
type Cache[K comparable, V any] interface {
	// Get retrieves a value from the cache by key
	Get(key K) (V, bool)

	// Set adds or updates a value in the cache without TTL
	Set(key K, value V) error

	// SetWithTTL adds or updates a value in the cache with optional TTL
	SetWithTTL(key K, value V, ttl time.Duration) error

	// Delete removes a value from the cache by key
	Delete(key K) error

	// Has checks if a key exists in the cache
	Has(key K) bool

	// Len returns the number of items in the cache
	Len() int

	// Clear removes all items from the cache
	Clear() error

	// Keys returns all keys in the cache
	Keys() []K

	// GetWithLoader retrieves a value from the cache, using the loader function if not present
	GetWithLoader(key K, loader func(K) (V, error)) (V, error)

	// GetWithLoaderWithTTL retrieves a value from the cache, using the loader function if not present and set ttl to the loaded value
	GetWithLoaderAndTTL(key K, loader func(K) (V, error), ttl time.Duration) (V, error)
}
