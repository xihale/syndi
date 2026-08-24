package cache

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestBadgerCache_GetExpiredDeletesEntry(t *testing.T) {
	cache, closeFn := newTestBadgerCache(t)
	defer closeFn()

	cache.Set("expired-get", "value", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	if _, ok := cache.Get("expired-get"); ok {
		t.Fatal("expected expired key to be missing")
	}
	if badgerKeyExists(t, cache, "expired-get") {
		t.Fatal("expected expired key to be deleted from badger")
	}
}

func TestBadgerCache_ExistsExpiredDeletesEntry(t *testing.T) {
	cache, closeFn := newTestBadgerCache(t)
	defer closeFn()

	cache.Set("expired-exists", "value", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	if cache.Exists("expired-exists") {
		t.Fatal("expected expired key to not exist")
	}
	if badgerKeyExists(t, cache, "expired-exists") {
		t.Fatal("expected expired key to be deleted from badger")
	}
}

func TestBadgerCache_NativeTTLPersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()

	cache, err := NewBadgerCacheWithOptions(Options{
		Path:            dir,
		MemoryEntries:   128,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create badger cache: %v", err)
	}

	cache.Set("persist-ttl", "value", 10*time.Millisecond)
	if _, ok := cache.Get("persist-ttl"); !ok {
		t.Fatal("expected fresh key to be readable")
	}
	if err := cache.Close(); err != nil {
		t.Fatalf("failed to close badger cache: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	reopened, err := NewBadgerCacheWithOptions(Options{
		Path:            dir,
		MemoryEntries:   128,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Hour,
		GCInterval:      time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to reopen badger cache: %v", err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Fatalf("failed to close badger cache: %v", err)
		}
	}()

	// Expiration must survive restart (native Badger TTL), not just process lifetime.
	if _, ok := reopened.Get("persist-ttl"); ok {
		t.Fatal("expected expired key to be missing after reopen")
	}
}

func TestBadgerCache_ValueLogGCRunsClean(t *testing.T) {
	cache, closeFn := newTestBadgerCache(t)
	defer closeFn()

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("gc-churn-%d", i%20) // rewrite same keys to create stale versions
		cache.Set(key, strings.Repeat("x", 4096), time.Minute)
	}
	for i := 0; i < 20; i++ {
		cache.Delete(fmt.Sprintf("gc-churn-%d", i))
	}

	// Must not panic or hang; ErrNoRewrite and rewrites are both fine.
	cache.runValueLogGC()
}

func newTestBadgerCache(t *testing.T) (*BadgerCache, func()) {
	t.Helper()

	cache, err := NewBadgerCache(128, t.TempDir(), time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to create badger cache: %v", err)
	}

	return cache, func() {
		if err := cache.Close(); err != nil {
			t.Fatalf("failed to close badger cache: %v", err)
		}
	}
}

func badgerKeyExists(t *testing.T, cache *BadgerCache, key string) bool {
	t.Helper()

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	found := false
	err := cache.badger.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		if err == nil {
			found = true
			return nil
		}
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	})
	if err != nil {
		t.Fatalf("badger lookup failed: %v", err)
	}

	return found
}
