package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// BadgerCache implements a persistent cache using BadgerDB
// It works as a two-tier cache: Memory (hot) + Badger (cold/persistent)
type BadgerCache struct {
	mu              sync.RWMutex
	memory          *MemoryCache  // Hot cache (LRU)
	badger          *badger.DB    // Cold cache (persistent)
	badgerPath      string        // Path for Badger data
	defaultTTL      time.Duration // Default TTL for Set operations
	cleanupInterval time.Duration // Interval for periodic cleanup
	stopCleanup     chan struct{} // Channel to stop background cleanup
	wg              sync.WaitGroup
}

// badgerCacheEntry is stored in Badger with metadata
type badgerCacheEntry struct {
	Value     []byte // gob-encoded value
	ExpiresAt int64  // Unix timestamp (nanoseconds)
}

// NewBadgerCache creates a new two-tier cache (Memory + Badger)
func NewBadgerCache(memorySize int, badgerPath string, defaultTTL time.Duration, cleanupInterval time.Duration) (*BadgerCache, error) {
	// Initialize Badger
	opts := badger.DefaultOptions(badgerPath)
	opts.ValueLogFileSize = 256 << 20 // 256MB
	opts.ValueLogMaxEntries = 50000

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	// Initialize memory cache
	memory := NewMemoryCache(memorySize)

	c := &BadgerCache{
		memory:          memory,
		badger:          db,
		badgerPath:      badgerPath,
		defaultTTL:      defaultTTL,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start background cleanup goroutine
	c.wg.Add(1)
	go c.cleanupExpired()

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
	var expiresAt int64
	var expired bool
	err := c.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		valCopy, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		var entry badgerCacheEntry
		if err := json.Unmarshal(valCopy, &entry); err != nil {
			return err
		}

		// Check expiration
		if time.Now().UnixNano() > entry.ExpiresAt {
			expired = true
			return nil
		}

		// Decode gob data
		var cacheEntry cacheEntry
		if err := gob.NewDecoder(bytes.NewReader(entry.Value)).Decode(&cacheEntry); err != nil {
			return err
		}

		result = cacheEntry.Value
		expiresAt = entry.ExpiresAt
		return nil
	})
	c.mu.RUnlock()

	if err != nil || result == nil {
		if expired {
			c.deleteFromBadger(key)
		}
		return nil, false
	}

	// Promote to memory cache (async, don't block)
	if expiresAt > 0 {
		remainingTTL := time.Duration(expiresAt - time.Now().UnixNano())
		go c.promoteToMemory(key, result, remainingTTL)
	}

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

// setInBadger stores a value in Badger
func (c *BadgerCache) setInBadger(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Encode value using gob
	var buf bytes.Buffer
	entry := cacheEntry{
		Value: value,
		Exp:   time.Now().Add(ttl),
	}
	if err := gob.NewEncoder(&buf).Encode(entry); err != nil {
		return
	}

	badgerEntry := badgerCacheEntry{
		Value:     buf.Bytes(),
		ExpiresAt: time.Now().Add(ttl).UnixNano(),
	}

	data, err := json.Marshal(badgerEntry)
	if err != nil {
		return
	}

	_ = c.badger.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// deleteFromBadger removes a key from Badger
func (c *BadgerCache) deleteFromBadger(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.badger.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// Exists checks if a key exists in either Memory or Badger
func (c *BadgerCache) Exists(key string) bool {
	// Check memory first
	if c.memory.Exists(key) {
		return true
	}

	// Check badger
	c.mu.RLock()

	var exists bool
	var expired bool
	_ = c.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return nil
		}

		valCopy, err := item.ValueCopy(nil)
		if err != nil {
			return nil
		}

		var entry badgerCacheEntry
		if err := json.Unmarshal(valCopy, &entry); err != nil {
			return nil
		}

		// Check expiration
		if time.Now().UnixNano() <= entry.ExpiresAt {
			exists = true
		} else {
			expired = true
		}
		return nil
	})
	c.mu.RUnlock()

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

// cleanupExpired periodically removes expired entries from Badger
func (c *BadgerCache) cleanupExpired() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cleanupInterval) // Clean at configured interval
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupBadger()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanupBadger removes all expired entries from Badger
func (c *BadgerCache) cleanupBadger() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	var keysToDelete [][]byte

	_ = c.badger.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Seek(nil); iter.Valid(); iter.Next() {
			item := iter.Item()
			valCopy, err := item.ValueCopy(nil)
			if err != nil {
				continue
			}

			var entry badgerCacheEntry
			if err := json.Unmarshal(valCopy, &entry); err != nil {
				continue
			}

			if now > entry.ExpiresAt {
				keysToDelete = append(keysToDelete, item.Key())
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

// Close closes the Badger database and stops cleanup goroutine
func (c *BadgerCache) Close() error {
	close(c.stopCleanup)
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
