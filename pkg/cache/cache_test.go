package cache

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache(100)

	cache.Set("key1", "value1", time.Minute)
	val, ok := cache.Get("key1")

	if !ok {
		t.Fatal("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestMemoryCache_GetNotFound(t *testing.T) {
	cache := NewMemoryCache(100)

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent key")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache(100)

	// Set a key with very short TTL
	cache.Set("expire", "value", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	_, ok := cache.Get("expire")
	if ok {
		t.Error("expected expired key to not exist")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(100)

	cache.Set("key", "value", time.Minute)
	cache.Delete("key")

	_, ok := cache.Get("key")
	if ok {
		t.Error("expected deleted key to not exist")
	}
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache(100)

	cache.Set("key", "value", time.Minute)

	if !cache.Exists("key") {
		t.Error("expected key to exist")
	}

	if cache.Exists("nonexistent") {
		t.Error("expected nonexistent key to not exist")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(100)

	cache.Set("key1", "value1", time.Minute)
	cache.Set("key2", "value2", time.Minute)
	cache.Clear()

	if cache.Exists("key1") || cache.Exists("key2") {
		t.Error("expected all keys to be cleared")
	}
}

func TestMemoryCache_TryGet(t *testing.T) {
	cache := NewMemoryCache(100)
	callCount := 0

	fn := func() (interface{}, error) {
		callCount++
		return "computed", nil
	}

	// First call should compute
	val, err := cache.TryGet("key", time.Minute, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "computed" {
		t.Errorf("expected computed, got %v", val)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Second call should use cache
	val, err = cache.TryGet("key", time.Minute, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "computed" {
		t.Errorf("expected computed, got %v", val)
	}
	if callCount != 1 {
		t.Errorf("expected function to not be called again, got %d calls", callCount)
	}
}

func TestMemoryCache_LRU(t *testing.T) {
	cache := NewMemoryCache(3) // Small cache size

	// Fill cache
	cache.Set("key1", "value1", time.Minute)
	cache.Set("key2", "value2", time.Minute)
	cache.Set("key3", "value3", time.Minute)

	// Add one more, should evict key1
	cache.Set("key4", "value4", time.Minute)

	// key1 should be evicted
	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected key1 to be evicted from LRU cache")
	}

	// Others should still exist
	for _, key := range []string{"key2", "key3", "key4"} {
		if !cache.Exists(key) {
			t.Errorf("expected %s to exist", key)
		}
	}
}

func TestMemoryCache_Concurrency(t *testing.T) {
	cache := NewMemoryCache(100)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Set(string(rune('a'+n%26)), n, time.Minute)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cache.Get(string(rune('a' + n%26)))
		}(i)
	}

	wg.Wait()
	// If we got here without deadlock or race, test passes
}

func TestMemoryCache_SetOverwrite(t *testing.T) {
	cache := NewMemoryCache(100)

	cache.Set("key", "value1", time.Minute)
	cache.Set("key", "value2", time.Minute)

	val, ok := cache.Get("key")
	if !ok {
		t.Fatal("expected to find key")
	}
	if val != "value2" {
		t.Errorf("expected value2, got %v", val)
	}
}

func TestMemoryCache_TryGetError(t *testing.T) {
	cache := NewMemoryCache(100)

	expectedErr := testError("compute failed")
	fn := func() (interface{}, error) {
		return nil, expectedErr
	}

	_, err := cache.TryGet("key", time.Minute, fn)
	if err != expectedErr {
		t.Errorf("expected error to be propagated, got %v", err)
	}

	// Key should not be cached on error
	_, ok := cache.Get("key")
	if ok {
		t.Error("expected key to not be cached on error")
	}
}

type testError string

func (e testError) Error() string {
	return string(e)
}
