package cache

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBadgerCachePersistence(t *testing.T) {
	tmpDir := "./test_badger_cache"
	defer os.RemoveAll(tmpDir)

	// First instance - write data
	cache1, err := NewBadgerCache(tmpDir)
	assert.NoError(t, err)

	cache1.Set("key1", "value1", 1*time.Hour)
	cache1.Set("key2", 123, 1*time.Hour)
	cache1.Close()

	// Second instance - read persisted data
	cache2, err := NewBadgerCache(tmpDir)
	assert.NoError(t, err)
	defer cache2.Close()

	val1, ok1 := cache2.Get("key1")
	assert.True(t, ok1)
	assert.Equal(t, "value1", val1)

	val2, ok2 := cache2.Get("key2")
	assert.True(t, ok2)
	assert.Equal(t, float64(123), val2)
}

func TestBadgerCacheExpiration(t *testing.T) {
	tmpDir := "./test_badger_exp"
	defer os.RemoveAll(tmpDir)

	cache, err := NewBadgerCache(tmpDir)
	assert.NoError(t, err)
	defer cache.Close()

	cache.Set("key", "value", 1*time.Second)

	val, ok := cache.Get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", val)

	time.Sleep(1100 * time.Millisecond)

	_, ok = cache.Get("key")
	assert.False(t, ok)
}
