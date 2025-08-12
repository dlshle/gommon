package cache

import (
	"errors"
	"testing"
	"time"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache[string, int](3)

	// Test Set and Get
	cache.Set("key1", 1)
	cache.Set("key2", 2)
	cache.Set("key3", 3)

	if val, ok := cache.Get("key1"); !ok || val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}

	if val, ok := cache.Get("key2"); !ok || val != 2 {
		t.Errorf("Expected 2, got %d", val)
	}

	// Test eviction
	cache.Set("key4", 4) // This should evict key3 (LRU)

	if _, ok := cache.Get("key3"); ok {
		t.Error("Expected key3 to be evicted")
	}

	// Test updating existing key
	cache.Set("key2", 20)
	if val, ok := cache.Get("key2"); !ok || val != 20 {
		t.Errorf("Expected 20, got %d", val)
	}

	// Test Delete
	cache.Delete("key2")
	if _, ok := cache.Get("key2"); ok {
		t.Error("Expected key2 to be deleted")
	}

	// Test Has
	if !cache.Has("key4") {
		t.Error("Expected key4 to exist")
	}

	// Test Len
	if cache.Len() != 2 {
		t.Errorf("Expected length 2, got %d", cache.Len())
	}

	// Test Keys
	keys := cache.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Test Clear
	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", cache.Len())
	}

	// Test GetWithLoader
	loader := func(key string) (int, error) {
		if key == "loadedKey" {
			return 42, nil
		}
		return 0, errors.New("key not found")
	}

	val, err := cache.GetWithLoader("loadedKey", loader)
	if err != nil || val != 42 {
		t.Errorf("Expected 42, got %d with error %v", val, err)
	}

	// Test TTL
	cache.SetWithTTL("expiringKey", 100, time.Millisecond*100)
	if val, ok := cache.Get("expiringKey"); !ok || val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	time.Sleep(time.Millisecond * 150)
	if _, ok := cache.Get("expiringKey"); ok {
		t.Error("Expected expiringKey to be expired")
	}
}

func TestLFUCache(t *testing.T) {
	cache := NewLFUCache[string, int](3)

	// Test Set and Get
	cache.Set("key1", 1)
	cache.Set("key2", 2)
	cache.Set("key3", 3)

	// Access key1 and key2 to increase their frequency
	cache.Get("key1")
	cache.Get("key2")
	cache.Get("key2")
	cache.Get("key1")

	// Test eviction - key3 should be evicted as it has the lowest frequency
	cache.Set("key4", 4)

	if _, ok := cache.Get("key3"); ok {
		t.Error("Expected key3 to be evicted")
	}

	// Test updating existing key
	cache.Set("key1", 10)
	if val, ok := cache.Get("key1"); !ok || val != 10 {
		t.Errorf("Expected 10, got %d", val)
	}

	// Test Delete
	cache.Delete("key1")
	if _, ok := cache.Get("key1"); ok {
		t.Error("Expected key1 to be deleted")
	}

	// Test Has
	if !cache.Has("key2") {
		t.Error("Expected key2 to exist")
	}

	// Test Len
	if cache.Len() != 2 {
		t.Errorf("Expected length 2, got %d", cache.Len())
	}

	// Test Keys
	keys := cache.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Test Clear
	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", cache.Len())
	}

	// Test GetWithLoader
	loader := func(key string) (int, error) {
		if key == "loadedKey" {
			return 42, nil
		}
		return 0, errors.New("key not found")
	}

	val, err := cache.GetWithLoader("loadedKey", loader)
	if err != nil || val != 42 {
		t.Errorf("Expected 42, got %d with error %v", val, err)
	}

	// Test TTL
	cache.SetWithTTL("expiringKey", 100, time.Millisecond*100)
	if val, ok := cache.Get("expiringKey"); !ok || val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	time.Sleep(time.Millisecond * 150)
	if _, ok := cache.Get("expiringKey"); ok {
		t.Error("Expected expiringKey to be expired")
	}
}