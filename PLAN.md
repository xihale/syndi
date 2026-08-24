# RSSHub Go - Implementation Plan

## Current Status (updated 2026-08-24)

**Framework complete. Route coverage: 102 registered paths across ~70 namespaces.**

- Core infrastructure, middleware stack, caching, HTTP client: DONE (phases 1-6 below)
- RSS2/Atom/RDF feed parser (`internal/parser/rssfeed`) + `routeutils.GetFeed` for native-feed wrappers
- Request disguise API (`internal/disguise`): browser presets, UA rotation, Referer/Cookie/Language — see `docs/DISGUISE.md`
- 100+ routes ported from RSSHub TypeScript and live-verified (`make verify-all`, results in `docs/ROUTES_CATALOG.md`: 98/102 OK)
- Porting workflow for contributors: `docs/PORTING_GUIDE.md` + `internal/testutil.RunHandler`

Known limitations:
- `steam/news` blocked from some networks (upstream 403); `techne98.com` domain dead
- Reddit heavily rate-limits unauthenticated .json APIs; route uses native .rss via disguise profile

---


## Current Status

### ✅ Completed (Middleware & Caching Implementation)

#### Phase 1: Core Infrastructure
- [x] Package structure setup
  - `pkg/models/` - Feed, Item, Route, Features data structures
  - `pkg/config/` - Environment-based configuration
  - `pkg/logger/` - Zap-based structured logging
  - `pkg/context/` - Request context wrapper
  - `pkg/registry/` - Route registration system

#### Phase 2: Middleware Stack (COMPLETE)
- [x] **Recovery** (`internal/middleware/recovery.go`)
  - Panic recovery with stack traces
  - Returns 500 errors on panics
  - Prevents server crashes

- [x] **Logger** (`internal/middleware/logger.go`)
  - Request timing and logging
  - Status-based log levels
  - Color-coded output (dev) / JSON (prod)

- [x] **Header** (`internal/middleware/header.go`)
  - CORS headers
  - Cache-Control headers
  - Security headers (X-Content-Type-Options)
  - Custom route headers

- [x] **Cache** (`internal/middleware/cache.go`)
  - Response caching with LRU
  - Request deduplication (cache stampede prevention)
  - Cache status headers
  - ETag support for 304 responses
  - **Note**: Kept as backup due to Gin writer wrapping limitations

- [x] **Parameter** (`internal/middleware/parameter.go`)
  - Query parameter processing (limit, filter, filterout, filter_time)
  - Feed modification based on params
  - Regex-based filtering

- [x] **Handler-Level Caching** (`internal/cache/handler.go`) ⭐
  - **Recommended approach** (avoids Gin writer wrapping issues)
  - Per-route customization
  - Custom TTL, key generation, conditional caching
  - ETag support with 304 responses
  - X-Cache headers (HIT/MISS)
  - Prevents caching of error responses (404, 500)

#### Phase 3: Server Integration (COMPLETE)
- [x] `cmd/server.go` - Main server with Gin framework
- [x] Middleware stack configured
- [x] Handler-level caching applied to main routes
- [x] Custom ShouldCache functions for error handling
- [x] Graceful shutdown support

#### Phase 4: Documentation (COMPLETE)
- [x] `docs/CACHING.md` - Comprehensive caching documentation
- [x] `docs/MIGRATION.md` - Middleware → Handler migration guide
- [x] `examples/handler_caching_example.go` - Usage examples

#### Phase 5: Testing (COMPLETE)
- [x] Unit tests for all middleware
- [x] Unit tests for handler-level caching
- [x] Cache behavior tests (HIT/MISS, ETag, 304)

#### Phase 6: Git Repository (COMPLETE)
- [x] Repository initialized
- [x] All changes committed (2 commits)
  - Commit 1: Handler-level caching implementation (18 files, 3928 lines)
  - Commit 2: Complete implementation (27 files, 5144 lines)

---

## Next Steps

### Immediate Verification (Recommended)

1. **Run All Tests**
   ```bash
   # Test middleware
   go test ./internal/middleware/...

   # Test cache handlers
   go test ./internal/cache/...

   # Test all packages
   go test ./...
   ```

2. **Start Server**
   ```bash
   go run cmd/server.go
   ```

3. **Test Endpoints**
   ```bash
   # Health check
   curl http://localhost:1200/status

   # Test feed (should return X-Cache: MISS on first request)
   curl -I http://localhost:1200/github/torvalds

   # Test cache hit (should return X-Cache: HIT)
   curl -I http://localhost:1200/github/torvalds

   # Test ETag (should return 304)
   curl -I -H "If-None-Match: <etag>" http://localhost:1200/github/torvalds

   # Test limit parameter
   curl "http://localhost:1200/github/torvalds?limit=5"
   ```

4. **Verify Logs**
   - Check that request timing is logged
   - Verify cache status (HIT/MISS) is logged
   - Confirm errors are logged with context

5. **Simulate Panic**
   - Trigger a panic in a handler
   - Verify server returns 500 and continues running

---

### Phase 7: Route Implementation (NEXT MAJOR PHASE)

#### Target: Port First 10-50 Routes

**Priority Routes** (diverse patterns, high popularity):

1. **GitHub** (`routes/github/`)
   - [ ] User repositories
   - [ ] User profile
   - [ ] Repository issues
   - [ ] Trending repositories
   - Pattern: JSON API

2. **Hacker News** (`routes/hackernews/`)
   - [ ] Front page
   - [ ] Item comments
   - Pattern: Simple JSON API

3. **V2EX** (`routes/v2ex/`)
   - [ ] Topics
   - [ ] Node listings
   - Pattern: HTML scraping with goquery

4. **Reddit** (`routes/reddit/`)
   - [ ] Subreddit feed
   - [ ] User posts
   - [ ] Multi-reddit
   - Pattern: JSON API

5. **Twitter/X** (`routes/twitter/`)
   - [ ] User timeline
   - [ ] Search
   - Pattern: API scraping

6. **YouTube** (`routes/youtube/`)
   - [ ] Channel videos
   - [ ] Playlist
   - Pattern: HTML/API hybrid

7. **Bilibili** (`routes/bilibili/`)
   - [ ] User uploads
   - [ ] Video comments
   - Pattern: Complex HTML scraping

8. **Docker Hub** (`routes/dockerhub/`)
   - [ ] Repository tags
   - Pattern: Simple API

9. **NPM** (`routes/npm/`)
   - [ ] Package info
   - [ ] Search results
   - Pattern: JSON API

10. **Arxiv** (`routes/arxiv/`)
    - [ ] Category feed
    - [ ] Search results
    - Pattern: XML/JSON API

#### Implementation Pattern for Each Route

```go
// routes/github/repos.go
package github

import (
    "github.com/rsshub/go/pkg/context"
    "github.com/rsshub/go/pkg/models"
)

func init() {
    registry.Register("github", &models.Route{
        Path:        "/user/:id/repos",
        Name:        "GitHub User Repositories",
        Description: "Fetches public repositories for a GitHub user",
        Handler:     handleUserRepos,
        Parameters: map[string]models.Parameter{
            "id": {
                Description: "GitHub username",
                Required:    true,
            },
        },
        Categories:  []string{"programming"},
        Maintainers: []string{"your-handle"},
    })
}

func handleUserRepos(ctx *context.Context) (*models.Feed, error) {
    userID := ctx.Params["id"]

    // Use HTTP client with auto-retry, proxy, etc.
    resp, err := ctx.Client.Get("https://api.github.com/users/" + userID + "/repos")
    if err != nil {
        return nil, err
    }

    // Parse response
    var repos []GitHubRepo
    if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
        return nil, err
    }

    // Build feed
    items := make([]models.Item, len(repos))
    for i, repo := range repos {
        items[i] = models.Item{
            Title:       repo.Name,
            Description: repo.Description,
            Link:        repo.HTMLURL,
            PubDate:     repo.CreatedAt,
            GUID:        repo.ID,
        }
    }

    return &models.Feed{
        Title:       fmt.Sprintf("%s's Repositories", userID),
        Link:        fmt.Sprintf("https://github.com/%s?tab=repos", userID),
        Description: fmt.Sprintf("Public repositories for %s", userID),
        Items:       items,
    }, nil
}
```

#### Required Additional Components

- [ ] **HTTP Client Enhancement** (`internal/client/`)
  - [ ] Response type helpers (JSON, XML, HTML)
  - [ ] Request builder pattern
  - [ ] Cookie persistence

- [ ] **HTML Parser Utilities** (`internal/parser/`)
  - [ ] goquery wrapper functions
  - [ ] Common selectors (title, content, date)
  - [ ] URL resolution

- [ ] **Date Parser** (`pkg/utils/date/`)
  - [ ] Port from TypeScript `parse-date.ts`
  - [ ] Support natural language ("3 hours ago", "yesterday")
  - [ ] Chinese date support ("昨天", "前天")
  - [ ] Comprehensive test cases

---

### Phase 8: Build System & Registry

- [ ] **Route Discovery**
  - [ ] Auto-scan `routes/` directory
  - [ ] Extract namespace metadata
  - [ ] Generate `assets/build/routes.json`
  - [ ] Generate `assets/build/maintainers.json`

- [ ] **Registry Enhancement**
  - [ ] Route grouping by namespace
  - [ ] Pattern matching (`:param`, `*wildcard`)
  - [ ] Priority sorting (literal before params)

---

### Phase 9: Testing & Documentation

- [ ] **Integration Tests**
  - [ ] Test each route with mocked responses
  - [ ] Validate RSS XML output
  - [ ] Test cache behavior per route

- [ ] **Documentation**
  - [ ] Route authoring guide
  - [ ] API documentation
  - [ ] Contribution guidelines
  - [ ] Deployment guide

---

### Phase 10: Production Readiness

- [ ] **Performance**
  - [ ] Benchmark requests/sec
  - [ ] Profile with pprof
  - [ ] Optimize hot paths
  - [ ] Connection pooling

- [ ] **Deployment**
  - [ ] Dockerfile (multi-stage)
  - [ ] docker-compose.yml
  - [ ] Kubernetes manifests (optional)
  - [ ] CI/CD pipeline

- [ ] **Monitoring**
  - [ ] Prometheus metrics
  - [ ] Health check endpoint
  - [ ] Log aggregation
  - [ ] Error tracking (Sentry)

---

## Long-term Roadmap

### Beyond v1.0

- [ ] Port remaining 1000+ routes
- [ ] Puppeteer/JS rendering support (chromedp)
- [ ] Advanced features:
  - [ ] OpenAI summarization
  - [ ] Radar rules
  - [ ] Follow integration
- [ ] Distributed caching (Redis cluster)
- [ ] Hot reload (SIGHUP)
- [ ] WebSocket support for real-time updates

---

## Architecture Decisions

### Web Framework: Gin ✅
- Chosen for: Performance, ecosystem, middleware support
- Status: Implemented and working

### Caching: Handler-Level ✅
- **Chosen over middleware** due to Gin writer wrapping limitations
- Status: Implemented with production-ready features
- Benefits: Per-route customization, reliable headers, ETag support

### Route Registration: init() ✅
- All routes compiled into binary
- Simple and reliable
- Alternative: Plugin system (complex, platform-dependent)

### RSS Formats: RSS 2.0 + Atom
- Primary: RSS 2.0
- Secondary: Atom (via `?format=atom`)
- Implementation: `pkg/rss/`

---

## Progress Tracking

### Milestones

| Milestone | Target | Status | Notes |
|-----------|--------|--------|-------|
| M1: Core Framework | Week 2 | ✅ Complete | Server, middleware, caching done |
| M2: First 10 Routes | Week 4 | 🔲 Next | Ready to implement |
| M3: 50 Routes | Week 7 | 🔲 Pending | Depends on M2 |
| M4: Build System | Week 8 | 🔲 Pending | Route discovery, registry |
| M5: Production Ready | Week 10 | 🔲 Pending | Testing, docs, deployment |

---

## Quick Reference

### Start Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run server
go run cmd/server.go

# Build binary
go build -o rsshub-go cmd/server.go
```

### Environment Variables

```bash
# Server
PORT=1200

# Cache
CACHE_TYPE=memory
CACHE_TTL=900

# Logging
LOGGER_LEVEL=info
NODE_ENV=development
```

### Project Structure

```
rsshub-go/
├── cmd/
│   └── server.go          # Main entry point
├── internal/
│   ├── cache/            # Handler-level caching ⭐
│   ├── client/           # HTTP client
│   ├── middleware/       # Gin middleware ✅
│   └── parser/           # HTML/JSON parsing
├── pkg/
│   ├── cache/            # Cache interface + LRU
│   ├── config/           # Configuration
│   ├── context/          # Request context
│   ├── logger/           # Zap logging
│   ├── models/           # Data structures
│   ├── registry/         # Route registry
│   └── rss/              # RSS/Atom generation
├── routes/               # Route implementations 🔲
│   ├── github/
│   ├── hackernews/
│   └── ...
├── docs/
│   ├── CACHING.md        # Caching docs ✅
│   └── MIGRATION.md      # Migration guide ✅
└── examples/             # Code examples ✅
```

---

## Last Updated

- **Date**: 2026-02-18
- **Status**: Middleware & caching complete, ready for route implementation
- **Next Action**: Run verification tests, then implement first 10 routes
