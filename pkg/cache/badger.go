package cache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Tuning defaults sized for small deployments (<=512MB RAM VPS).
// Badger's out-of-the-box defaults assume a dedicated database server
// (64MB memtables x5, 256MB block cache) which is wasteful here.
const (
	defaultMemtableSize       int64  = 16 << 20 // 16MB
	defaultNumMemtables              = 4
	defaultBlockCacheSize     int64  = 32 << 20  // 32MB
	defaultValueLogFileSize   int64  = 128 << 20 // 128MB
	defaultValueLogMaxEntries uint32 = 50000
	defaultGCDiscardRatio            = 0.5
	maxGCCyclesPerTick               = 8 // bound GC work per interval
)

// Options configures the two-tier cache.
// Zero values select the tuned defaults above.
type Options struct {
	Path               string        // Badger data directory (required)
	MemoryEntries      int           // LRU capacity of the hot layer
	DefaultTTL         time.Duration // TTL used by Set when callers pass one
	CleanupInterval    time.Duration // Periodic scan that physically deletes expired keys
	GCInterval         time.Duration // Value-log GC cadence; 0 disables (not recommended for long runs)
	GCDiscardRatio     float64       // Fraction of garbage in a vlog file required before rewrite; 0 -> 0.5
	MemTableSizeBytes  int64         // Per-memtable memory budget
	NumMemtables       int           // Max in-memory memtables before flush/stall
	BlockCacheSize     int64         // Block (data) cache budget
	IndexCacheSize     int64         // Index cache budget; 0 lets Badger auto-size
	ValueLogFileSize   int64         // Max size of a single value-log file
	ValueLogMaxEntries uint32        // Max entries per vlog file before rotation
}

// BadgerCache implements a persistent cache using BadgerDB
// It works as a two-tier cache: Memory (hot) + Badger (cold/persistent)
//
// Expiration is delegated to Badger's native entry TTL: expired items are
// invisible to reads and physically reclaimed by compaction. The periodic
// cleanup scan only accelerates reclamation by deleting expired keys eagerly,
// and the value-log GC loop reclaims the disk space deletions leave behind.
type BadgerCache struct {
	mu              sync.RWMutex
	memory          *MemoryCache  // Hot cache (LRU)
	badger          *badger.DB    // Cold cache (persistent)
	badgerPath      string        // Path for Badger data
	defaultTTL      time.Duration // Default TTL for Set operations
	cleanupInterval time.Duration // Interval for periodic cleanup
	stopBackground  chan struct{} // Channel to stop background workers
	wg              sync.WaitGroup
	opts            Options
}

// NewBadgerCache creates a new two-tier cache (Memory + Badger) with tuned defaults.
// Kept for backward compatibility; prefer NewBadgerCacheWithOptions.
func NewBadgerCache(memorySize int, badgerPath string, defaultTTL time.Duration, cleanupInterval time.Duration) (*BadgerCache, error) {
	return NewBadgerCacheWithOptions(Options{
		Path:            badgerPath,
		MemoryEntries:   memorySize,
		DefaultTTL:      defaultTTL,
		CleanupInterval: cleanupInterval,
	})
}

// NewBadgerCacheWithOptions creates a new two-tier cache from explicit options.
func NewBadgerCacheWithOptions(opts Options) (*BadgerCache, error) {
	if opts.Path == "" {
		return nil, errors.New("cache: badger path must not be empty")
	}
	if opts.GCDiscardRatio <= 0 || opts.GCDiscardRatio >= 1 {
		opts.GCDiscardRatio = defaultGCDiscardRatio
	}

	badgerOpts := badger.DefaultOptions(opts.Path).
		WithLoggingLevel(badger.ERROR). // silence per-op chatter; errors still surface
		WithMemTableSize(orDefaultI64(opts.MemTableSizeBytes, defaultMemtableSize)).
		WithNumMemtables(orDefaultInt(opts.NumMemtables, defaultNumMemtables)).
		WithBlockCacheSize(orDefaultI64(opts.BlockCacheSize, defaultBlockCacheSize)).
		WithValueLogFileSize(orDefaultI64(opts.ValueLogFileSize, defaultValueLogFileSize)).
		WithValueLogMaxEntries(orDefaultU32(opts.ValueLogMaxEntries, defaultValueLogMaxEntries))
	if opts.IndexCacheSize > 0 {
		badgerOpts = badgerOpts.WithIndexCacheSize(opts.IndexCacheSize)
	}

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, err
	}

	c := &BadgerCache{
		memory:          NewMemoryCache(opts.MemoryEntries),
		badger:          db,
		badgerPath:      opts.Path,
		defaultTTL:      opts.DefaultTTL,
		cleanupInterval: opts.CleanupInterval,
		stopBackground:  make(chan struct{}),
		opts:            opts,
	}

	c.wg.Add(1)
	go c.cleanupLoop()
	if opts.GCInterval > 0 {
		c.wg.Add(1)
		go c.valueLogGCLoop()
	}

	return c, nil
}

// Get retrieves a value from the cache
// Priority: Memory (hot) -> Badger (cold) -> Not Found
func (c *BadgerCache) Get(key string) (interface{}, bool) {
	// Try memory first (fast path)
	if val, ok := c.memory.Get(key); ok {
		return val, true
	}

	// Try badger (slow path)
	c.mu.RLock()

	var result interface{}
	var remainingTTL time.Duration
	var expired bool
	err := c.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		if item.IsDeletedOrExpired() {
			expired = true
			return badger.ErrKeyNotFound
		}

		valCopy, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		// Decode gob payload (single decode; no outer wrapper since v2 format)
		var entry cacheEntry
		if err := gob.NewDecoder(bytes.NewReader(valCopy)).Decode(&entry); err != nil {
			return err
		}

		result = entry.Value
		remainingTTL = time.Until(entry.Exp)
		return nil
	})
	c.mu.RUnlock()

	if err != nil || result == nil {
		if expired {
			c.deleteExpiredKey(key)
		}
		return nil, false
	}

	// Promote to memory cache (async, don't block)
	go c.promoteToMemory(key, result, remainingTTL)

	return result, true
}

// promoteToMemory promotes a value from Badger to Memory cache
func (c *BadgerCache) promoteToMemory(key string, value interface{}, remainingTTL time.Duration) {
	if remainingTTL <= 0 {
		return
	}
	c.memory.Set(key, value, remainingTTL)
}

// Set stores a value in both Memory and Badger (write-through)
func (c *BadgerCache) Set(key string, value interface{}, ttl time.Duration) {
	// Always set in memory (hot cache)
	c.memory.Set(key, value, ttl)

	// Also set in badger (cold cache)
	c.setInBadger(key, value, ttl)
}

// setInBadger stores a gob-encoded value with Badger-native TTL
func (c *BadgerCache) setInBadger(key string, value interface{}, ttl time.Duration) {
	// Encode value using gob - wrap in struct to preserve type information
	var buf bytes.Buffer
	entry := cacheEntry{
		Value: value,
		Exp:   time.Now().Add(ttl),
	}
	if err := gob.NewEncoder(&buf).Encode(entry); err != nil {
		return
	}

	e := badger.NewEntry([]byte(key), buf.Bytes()).WithTTL(ttl)

	c.mu.Lock()
	defer c.mu.Unlock()
	// Best-effort: a failed cold write only costs a future cache miss.
	_ = c.badger.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(e)
	})
}

// deleteExpiredKey physically removes an expired key so its space can be
// reclaimed by compaction/value-log GC ahead of the next cleanup pass.
func (c *BadgerCache) deleteExpiredKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Best-effort; the periodic cleanup will retry if this fails.
	_ = c.badger.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// deleteFromBadger removes a key from Badger
func (c *BadgerCache) deleteFromBadger(key string) {
	c.deleteExpiredKey(key)
}

// Exists checks if a key exists in either Memory or Badger
func (c *BadgerCache) Exists(key string) bool {
	// Check memory first
	if c.memory.Exists(key) {
		return true
	}

	// Check badger: txn.Get already filters deleted/expired versions;
	// detect the expired case separately to physically delete it.
	c.mu.RLock()

	exists := false
	expired := false
	_ = c.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		switch {
		case err != nil:
			return nil
		case item.IsDeletedOrExpired():
			expired = true
		default:
			exists = true
		}
		return nil
	})
	c.mu.RUnlock() // release before deleteFromBadger takes the write lock

	if expired {
		c.deleteFromBadger(key)
	}

	return exists
}

// Delete removes a key from both Memory and Badger
func (c *BadgerCache) Delete(key string) {
	// Delete from memory
	c.memory.Delete(key)

	// Delete from badger
	c.deleteFromBadger(key)
}

// Clear removes all entries from both Memory and Badger
func (c *BadgerCache) Clear() {
	// Clear memory
	c.memory.Clear()

	// Clear badger
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.badger.Update(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		batch := &badger.WriteBatch{}
		for iter.Seek(nil); iter.Valid(); iter.Next() {
			// Best-effort bulk delete; Flush below reports the real error.
			_ = batch.Delete(iter.Item().Key())
		}
		return batch.Flush()
	})
}

// cleanupLoop periodically removes expired entries from Badger
func (c *BadgerCache) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cleanupInterval) // Clean at configured interval
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupBadger()
		case <-c.stopBackground:
			return
		}
	}
}

// valueLogGCLoop rewrites value-log files whose contents are mostly stale.
// Without this, vlog files grow monotonically under a churn of Set/Delete:
// deletions only mark keys gone, the space comes back only via RunValueLogGC.
func (c *BadgerCache) valueLogGCLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.opts.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.runValueLogGC()
		case <-c.stopBackground:
			return
		}
	}
}

// runValueLogGC performs up to maxGCCyclesPerTick rewrites, stopping as soon
// as no file qualifies. Each call rewrites at most one vlog file.
func (c *BadgerCache) runValueLogGC() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := 0; i < maxGCCyclesPerTick; i++ {
		err := c.badger.RunValueLogGC(c.opts.GCDiscardRatio)
		if err != nil {
			// ErrNoRewrite: nothing worth reclaiming right now. Any other
			// error is transient (concurrent rewrite, DB busy); retry next tick.
			return
		}
	}
}

// cleanupBadger removes all expired entries from Badger.
// Reads only item metadata (no value copies) to keep the scan cheap.
func (c *BadgerCache) cleanupBadger() {
	c.mu.Lock()
	defer c.mu.Unlock()

	iterOpts := badger.DefaultIteratorOptions
	iterOpts.PrefetchValues = false

	var keysToDelete [][]byte

	_ = c.badger.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(iterOpts)
		defer iter.Close()

		for iter.Seek(nil); iter.Valid(); iter.Next() {
			item := iter.Item()
			if item.IsDeletedOrExpired() {
				keysToDelete = append(keysToDelete, item.KeyCopy(nil))
			}
		}
		return nil
	})

	if len(keysToDelete) > 0 {
		_ = c.badger.Update(func(txn *badger.Txn) error {
			for _, key := range keysToDelete {
				// Best-effort cleanup of expired entries; skipped keys are
				// simply retried on the next cleanup cycle.
				_ = txn.Delete(key)
			}
			return nil
		})
	}
}

// Close stops background workers and closes the Badger database
func (c *BadgerCache) Close() error {
	close(c.stopBackground)
	c.wg.Wait()
	return c.badger.Close()
}

// GetStats returns cache statistics
func (c *BadgerCache) GetStats() (memorySize, badgerSize int64) {
	// Memory stats
	memorySize = int64(c.memory.Len())

	// Badger stats
	c.mu.RLock()
	defer c.mu.RUnlock()

	_ = c.badger.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek(nil); iter.Valid(); iter.Next() {
			badgerSize++
		}
		return nil
	})

	return memorySize, badgerSize
}

func orDefaultI64(v, def int64) int64 {
	if v > 0 {
		return v
	}
	return def
}

func orDefaultU32(v, def uint32) uint32 {
	if v > 0 {
		return v
	}
	return def
}

func orDefaultInt(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}
