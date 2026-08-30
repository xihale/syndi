# Caching in Syndi

Syndi uses a **two-tier cache**: a hot in-memory LRU layer plus a persistent
BadgerDB cold layer (`pkg/cache`). Route handlers consume it through small
helpers in `internal/routeutils`. All keys live in the `cache:` block of
[`CONFIGURATION.md`](./CONFIGURATION.md).

---

## Architecture

```
Get(key)
  │
  ├─ hit? ────────────────► MemoryCache (hashicorp/golang-lru, decoded values)
  │                          no serialization on the hot path
  │
  └─ miss ▼
     BadgerDB (persistent) ── gob-encoded values with native TTL
       │
       └─ found? promote to memory asynchronously, return value
```

- **Memory layer** stores decoded `interface{}` values. Retrieved values are
  shared across Gets until eviction/expiry, so treat them as read-only.
- **Badger layer** persists every `Set` (write-through). Values are
  gob-encoded once; expiration uses Badger's native entry TTL.
- On upgrade from the pre-TTL on-disk format, old entries decode as misses
  and are reclaimed by compaction/GC. No manual migration needed.

## How handlers use the cache

`internal/routeutils/cache.go` exposes typed wrappers:

```go
// Cache a whole feed or just items
feed, err := routeutils.CacheFeed(cacheInstance, key, ttl, fetchFeed)
items, err := routeutils.CacheItems(cacheInstance, key, ttl, fetchItems)

// Cache any JSON-marshalable value
value, err := routeutils.CacheJSON(cacheInstance, key, ttl, compute)

// Read/write/invalidate directly
routeutils.GetFeedFromCache(cacheInstance, key)
routeutils.SetFeedInCache(cacheInstance, key, feed, ttl)
routeutils.InvalidateCacheEntry(cacheInstance, key)
```

All of them degrade gracefully to calling `fn()` when `cacheInstance == nil`.

## Memory tuning

Badger's out-of-the-box defaults assume a dedicated database server
(256MB block cache, five 64MB memtables ≈ up to ~576MB RAM). Syndi ships
smaller defaults suited to a ≤512MB VPS: steady-state memory for the cold
layer is roughly `block_cache_mb + memtable_mb × num_memtables` (~96MB at
defaults), plus the LRU entries themselves.

## Disk reclamation

Deleting or expiring keys does **not** return disk space by itself: Badger's
value log only shrinks when rewritten. Two mechanisms handle this:

1. `cleanup_interval`: scans item metadata (no value copies) and physically
   deletes expired keys so compaction can drop them.
2. `gc_interval`: runs `RunValueLogGC(gc_discard_ratio)` up to 8 times per
   tick, rewriting one value-log file per call while files qualify.

Keep `gc_interval` enabled for long-running instances; without it the value
log grows monotonically under feed-refresh churn even though logical content
stays small.

## HTTP cache semantics

- `X-Cache: HIT|MISS` response header reflects whether the feed was served
  from cache (set by `middleware.Header`).
- Responses carry an `ETag`; requests with `If-None-Match` get `304 Not
  Modified`.
- `Cache-Control: max-age` mirrors the configured TTL.
