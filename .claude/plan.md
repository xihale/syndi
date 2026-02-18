# Plan: Port RSSHub to Go

## Project Overview
Port the RSSHub TypeScript/Node.js application to Go, maintaining 100% feature parity while leveraging Go's strengths (concurrency, performance, static typing).

## Target Analysis (from RSSHub codebase)

### Core Components Identified:
1. **HTTP Client** (`lib/utils/got.ts`): Wrapper around ofetch with retry, proxy, cookie jar support
2. **Cache** (`lib/utils/cache/`): Memory (LRU) and Redis support with `tryGet` pattern
3. **Date Parsing** (`lib/utils/parse-date.ts`): Complex natural language parser for multiple languages
4. **Middleware Stack** (in order):
   - TrimTrailingSlash
   - Compress
   - JSX Renderer (for RSS XML)
   - Logger
   - Trace/Sentry
   - Access Control (CORS)
   - Debug
   - Template
   - Header
   - Anti-hotlink
   - Parameter parsing
   - Cache
5. **Route Structure**: Each route exports `Route` object with path, handler, metadata (name, maintainers, categories, features, radar, etc.)
6. **Registry**: Dynamic route discovery in dev, pre-built bundle in production
7. **Route Patterns**: HTML scraping (cheerio), JSON APIs, XML feeds, with common patterns:
   - `cache.tryGet(link, async () => fetch_detail())` for detail page caching
   - `parseDate()` for all date handling
   - Return `Data` object with title, link, description, items array

### Data Structures (from `lib/types.ts`):
- **Feed/Data**: title, description, link, items, image, author, language, ttl, etc.
- **Item**: title, description, link, pubDate, guid, category, author, content (html/text), image, attachments, etc.
- **Route**: path, name, handler, maintainers, example, parameters, description, categories, features, radar, view
- **Features**: requireConfig, requirePuppeteer, antiCrawler, supportRadar, supportBT, supportPodcast, supportScihub, nsfw

## Go Architecture Design

### Package Structure:
```
rsshub-go/
├── cmd/
│   └── server/          # Main server entry point
├── internal/
│   ├── client/          # HTTP client with retry, proxy, cookie jar
│   ├── middleware/      # All middleware implementations
│   ├── parser/          # HTML (goquery), JSON, XML parsers
│   └── scraper/         # Common scraping patterns
├── pkg/
│   ├── models/          # Core data structures (Feed, Item, Route, etc.)
│   ├── rss/            # RSS/Atom XML generation (using encoding/xml or library)
│   ├── cache/          # Cache interface + memory + redis implementations
│   ├── config/         # Configuration management (env-based)
│   ├── logger/         # Logging (zap or logrus)
│   ├── utils/          # Utilities (date parsing, user-agent rotation, etc.)
│   └── registry/       # Route registration and discovery
├── routes/             # Route implementations (1500+ routes to port)
│   ├── 005/
│   │   ├── index.go
│   │   └── namespace.go
│   ├── github/
│   ├── twitter/
│   └── ...
├── api/                # API routes (if needed)
├── scripts/            # Build scripts to auto-generate route bundles
├── tests/              # Test files
├── go.mod
└── README.md
```

### Key Interfaces:

```go
// pkg/models/feed.go
type Feed struct {
    Title         string
    Description   string
    Link          string
    Items         []Item
    Image         string
    Author        string
    Language      string
    LastBuildDate time.Time
    TTL           int
    ID            string
    Icon          string
    Logo          string
    AtomLink      string
}

type Item struct {
    Title         string
    Description   string
    Link          string
    PubDate       time.Time
    GUID          string
    Category      []string
    Author        string
    Content       Content
    Image         string
    Banner        string
    Updated       time.Time
    Language      string
    EnclosureURL  string
    EnclosureType string
    EnclosureTitle string
    EnclosureLength int64
    ITunesDuration string
    ITunesItemImage string
    Media         map[string]map[string]string
    Attachments   []Attachment
    DOI           string
}

type Content struct {
    HTML string
    Text string
}

type Route struct {
    Path          string
    Name          string
    Handler       HandlerFunc
    Maintainers   []string
    Example       string
    Parameters    map[string]Parameter
    Description   string
    Categories    []string
    Features      Features
    Radar         []RadarRule
    View          ViewType
    NamespaceData *Namespace // added during registration
}

type HandlerFunc func(ctx *Context) (*Feed, error)

type Context struct {
    Req    *http.Request
    Writer http.ResponseWriter
    Params map[string]string
    Query  url.Values
    Client *client.HTTPClient
    Cache  cache.Cache
    // Additional fields:
    CacheKey        string
    CacheControlKey string
    Data            *Feed // set by middleware/handler
}

type Features struct {
    RequireConfig     []ConfigRequirement
    RequirePuppeteer  bool
    AntiCrawler       bool
    SupportRadar      bool
    SupportBT         bool
    SupportPodcast    bool
    SupportScihub     bool
    NSFW              bool
}

type Parameter struct {
    Description string
    Default     string
    Options     []Option
}

type Option struct {
    Value string
    Label string
}

type RadarRule struct {
    Title  string
    Docs   string
    Source []string
    Target  string // or function
}

type Namespace struct {
    Name        string
    URL         string
    Description string
    Lang        string
}

// ViewType enum (Articles, SocialMedia, Pictures, Videos, Audios, Notifications)
```

### Core Components to Implement:

#### 1. HTTP Client (`internal/client/`)
- Wrapper around http.Client
- Retry logic with exponential backoff
- Proxy support (HTTP/HTTPS/SOCKS5)
- Cookie jar
- User-Agent rotation (random from list)
- Timeout configuration
- Hook system (beforeRequest, afterResponse)
- Response types: JSON, XML, Buffer

#### 2. Cache System (`pkg/cache/`)
```go
type Cache interface {
    Get(ctx context.Context, key string) (string, bool, error)
    Set(ctx context.Context, key string, value string, ttl time.Duration) error
    TryGet(ctx context.Context, key string, getValue func() (interface{}, error), ttl time.Duration) (interface{}, error)
    Delete(ctx context.Context, key string) error
    Close() error
}

// Memory: sync.Map or LRU (github.com/hashicorp/golang-lru)
// Redis: github.com/redis/go-redis/v9
// Also need global cache for locking (request-in-progress protection)
```

#### 3. Date Parsing (`pkg/utils/date/`)
- Port parse-date.ts logic
- Support natural language: "just now", "3 hours ago", "yesterday", "last Monday", Chinese dates (昨天, 前天, 周一, etc.)
- Use regex patterns from original
- Return time.Time

#### 4. Logger (`pkg/logger/`)
- Structured logging (zap or logrus)
- Levels: debug, info, warn, error
- Optional file output
- JSON format

#### 5. Configuration (`pkg/config/`)
- Environment variable based
- Struct with defaults
- Support for:
  - Server config (PORT, etc.)
  - Cache config (CACHE_TYPE, CACHE_EXPIRE, REDIS_URL, MEMORY_MAX)
  - Network (REQUEST_RETRY, REQUEST_TIMEOUT, UA list, PROXY_*)
  - Access control (ACCESS_KEY)
  - Logging (DEBUG_INFO, LOGGER_LEVEL)
  - Feed config (TITLE_LENGTH_LIMIT, DISABLE_NSFW, SUFFIX)
  - Route-specific: dynamic prefixes (BILIBILI_COOKIE_*, etc.)
  - OpenAI, Follow, etc.

#### 6. Middleware (`internal/middleware/`)
Implement as `func(next HandlerFunc) HandlerFunc`

Order (from app-bootstrap.tsx):
1. TrimTrailingSlash - normalize path
2. Compress - gzip middleware (standard library accepts gzip)
3. RSS Renderer - set Content-Type, render XML (replaces JSX renderer)
4. Logger - request logging
5. Trace - OpenTelemetry metrics if DEBUG_INFO
6. Access Control - CORS headers
7. Debug - add debug info to response if DEBUG_INFO
8. Template - not needed for RSS, maybe for HTML views (skip)
9. Header - custom headers
10. AntiHotlink - check Referer, serve placeholder
11. Parameter - parse route parameters and query params
12. Cache - implement full cache middleware with request-in-progress locking

#### 7. Router & Registry (`pkg/registry/`)
- Allow routes to register themselves via `init()` or explicit registration
- Support for:
  - Pattern matching with `:param` syntax
  - Sorting routes (literal before params)
  - Namespace grouping
  - Two modes:
    - Development: Scan `routes/` directory and load dynamically (using fs and plugin? or build tags)
    - Production: Load from pre-built `assets/build/routes.json` that contains all route metadata
- Register routes at `/:namespace/:path` and `/api/:namespace/:path`
- Middleware chain execution

#### 8. RSS Generation (`pkg/rss/`)
- Encode Feed to RSS 2.0 XML (and optionally Atom)
- Use `encoding/xml` or a library like `github.com/jteeuwen/go-pkg-xmlx`
- Proper escaping
- CDATA for description content
- Optional HTML content

### Implementation Phases

#### Phase 1: Refactor Foundation (Week 1-2)
**Existing work**: `main.go` has basic implementation (Engine, Feed, Item, Cache, Route registration, XML generation). Need to restructure and expand.

- [ ] **Restructure into packages**: Move code from monolithic `main.go` to:
  - `pkg/models/` - Feed, Item, Route, Features, etc. (expand to match full RSSHub types)
  - `pkg/rss/` - RSS/Atom XML generation (extract from main.go itemsXML)
  - `internal/client/` - HTTP client with retry/proxy (new)
  - `pkg/cache/` - Cache interface + LRU memory implementation (enhance existing simple cache)
  - `pkg/utils/date/` - Date parser (new, port from TS)
  - `pkg/logger/` - Structured logging (new, use zap)
  - `pkg/config/` - Configuration (new)
- [ ] **Choose web framework**: Add Gin to go.mod, create basic server
- [ ] **Implement Context (`pkg/context/`)**: Wrap Gin context with our fields (Params, Client, Cache, etc.)
- [ ] **Create route registry (`pkg/registry/`)**: Allow routes to register via `init()`, support namespaces
- [ ] **Implement namespace grouping**: Routes registered at `/:namespace/:path`
- [ ] **Test framework**: Write 2-3 simple routes (status, echo, health) to validate

#### Phase 2: Middleware & Utilities (Week 3)
- [ ] **Implement all middleware** in `internal/middleware/`:
  - TrimTrailingSlash
  - Compress (gzip - use gin's built-in or custom)
  - Logger (structured, with request ID)
  - AccessControl (CORS)
  - AntiHotlink (Referer check)
  - Parameter (query param parsing)
  - Cache (full middleware with request-in-progress lock, cache-key generation)
  - Debug (optional debug info)
- [ ] **Implement HTTP client** (`internal/client/client.go`):
  - Retry with exponential backoff (3 retries)
  - Proxy rotation (if PROXY_* configured)
  - User-Agent rotation (from config list)
  - Cookie jar
  - Timeout (30s default)
- [ ] **Implement date parser** (`pkg/utils/date/parser.go`):
  - Port parse-date.ts logic exactly
  - Comprehensive test coverage from TS tests
- [ ] **Implement config** (`pkg/config/config.go`):
  - Load from env with defaults
  - Support all core config options (PORT, CACHE_TYPE, REDIS_URL, REQUEST_RETRY, REQUEST_TIMEOUT, UA, PROXY_*, DISABLE_NSFW, etc.)
  - Support dynamic route-specific config (prefix-based like BILIBILI_COOKIE_*)

#### Phase 3: Port First 10 Routes (Week 4)
Build confidence by porting a diverse selection:

- [ ] **005** (`routes/005/index.go`): HTML scraping with goquery, detail page caching - 1 route
- [ ] **GitHub** (`routes/github/`): JSON API, multiple endpoints:
  - repos (user/repos), user, events, trending - 4 routes
- [ ] **Hacker News** (`routes/hackernews/`): Simple JSON API - 2 routes (frontpage, item)
- [ ] **jike** (`routes/jike/`): API with custom client - 2 routes (user, topic)
- [ ] **v2ex** (`routes/v2ex/`): HTML scraping - 2 routes (topics, node)

Total: ~11 routes across 5 namespaces.

**For each route**:
- Convert TSX to Go
- Implement namespace metadata file (`namespace.go`)
- Implement route handler(s)
- Match original behavior functionally
- Write tests using mocked HTTP responses (httptest + goquery)
- Verify RSS output is valid

#### Phase 4: Build System & Registry (Week 5)
- [ ] **Build script** (`scripts/build-routes/main.go` or shell):
  - Scans `routes/` directory recursively
  - For each `namespace.go` extracts name, URL, lang, description
  - For each route finds Go files, extracts route struct (or requires explicit registration file)
  - Generates `assets/build/routes.json` with complete registry in format used by RSSHub
  - Generates `assets/build/maintainers.json`
- [ ] **Registry loader**:
  - Dev mode: Use Go plugin package or build tags to load routes dynamically (simpler: just use Go's init() - all routes register at startup)
  - Prod mode: Load `routes.json` to get metadata only, lazy-load actual route handlers via reflection or module
  - Since Go doesn't have dynamic import like JS, we'll need:
    - Option A: Compile ALL routes into binary (simple, but large binary)
    - Option B: Use plugin system (`plugin` package - not all platforms support)
    - Option C: Load from config file, instantiate handlers via factory registry (handlers must be registered in a global map)
  - **Recommended**: Compile all routes into binary (like RSSHub). No dynamic loading needed. Simplicity first.
- [ ] **Route registration** in main:
  - Scan `routes/` directory (or use a generated list)
  - For each namespace, call `namespace.Register()` to register all its routes into global registry
  - Routes auto-register in `init()` like: `registry.Register(namespace, route)`
- [ ] **Wire up**: All routes compiled, registry populated, Gin groups created (`/:namespace` and `/api/:namespace`)

#### Phase 5: Expand to 50 Routes (Week 6-7)
Port additional namespaces to reach ~50 total routes (~20 more namespaces):

- [ ] **Popular APIs**: Reddit, YouTube, Spotify, SoundCloud, Arxiv, PyPI, NPM, Docker Hub
- [ ] **Social**: Weibo, Bilibili, Telegram, Medium
- [ ] **News/Search**: Bing, Google (if feasible), 500px, Unsplash
- [ ] **Development**: Crates (crates.io), etc.
- [ ] **Entertainment**: IMDb, Steam
- [ ] **Chinese**: Douban

Total ~40 additional routes across ~20 namespaces.

- [ ] **Write integration tests** for all 50 routes (at least smoke tests)
- [ ] **Test cache behavior** extensively
- [ ] **Verify date parsing** on real data from all routes
- [ ] **Profile memory and CPU** under load

#### Phase 6: Polish & Release Candidate (Week 8-10)
- [ ] **Performance optimization**:
  - Use connection pooling in HTTP client
  - Tune cache sizes (memory, TTLs)
  - Profile with pprof, optimize hot paths
  - Consider pooling goquery documents
- [ ] **Error handling & logging**:
  - Ensure all errors logged with context
  - Add request IDs for tracing
  - Recover from panics in handlers
- [ ] **Configuration**:
  - Add all remaining config options from RSSHub (access control, OpenAI, Follow, etc.)
  - Document all env vars
- [ ] **Testing**:
  - Unit tests for all utilities (client, cache, date, config)
  - Integration tests for ~20 routes using recorded HTTP fixtures
  - End-to-end test: start server, make real requests to all 50 routes, validate RSS output
- [ ] **Documentation**:
  - Comprehensive README with quick start, config, development guide
  - Route-specific docs generation (like RSSHub)
  - API documentation
- [ ] **Containerization**:
  - Dockerfile (multi-stage build, scratch/alpine)
  - docker-compose.yml with Redis + optional browserless (for Puppeteer routes - but we may skip Puppeteer initially)
- [ ] **Benchmarking**:
  - Compare performance with Node.js RSSHub on same hardware
  - Document results
- [ ] **Release**:
  - Semantic versioning (v0.1.0 initial)
  - GitHub releases
  - License (AGPL-3.0 to match RSSHub)

### Long-term Roadmap (Beyond v1.0)
- Port remaining 1000+ routes (can be community effort)
- Puppeteer support via chrome-remote-interface or chromedp (for JS-heavy sites)
- Advanced features: OpenAI summarization, radar rules, Follow integration
- Distributed cache (Redis cluster)
- Metrics (Prometheus/OpenTelemetry)
- Hot reload of routes (without restart) - difficult in Go, maybe via SIGHUP config reload

### API Design for Route Authors

Inspired by RSSHub's simplicity:

```go
// routes/example/example.go
package example

import (
    "github.com/rsshub/go/ctx"
    "github.com/rsshub/go/models"
    "github.com/rsshub/go/pkg/utils"
)

func init() {
    routes.Register("example", &models.Route{
        Path: "/example/:id",
        Name: "Example Feed",
        Handler: Handler,
        Maintainers: []string{"yourname"},
        Categories: []string{"other"},
        Description: "Example description",
        Parameters: map[string]models.Parameter{
            "id": {Description: "Item ID"},
        },
        Features: models.Features{
            RequireConfig: false,
            SupportRadar: true,
        },
    })
}

func Handler(c *ctx.Context) (*models.Feed, error) {
    id := c.Params["id"]
    // Use c.Client for HTTP requests (auto-retry, proxy, UA)
    // Use c.Cache for caching with TryGet pattern
    // Return feed
    return &models.Feed{
        Title: "Example",
        Items: []models.Item{
            {
                Title: "Item 1",
                Link: "https://example.com/1",
                PubDate: time.Now(),
                GUID: "1",
            },
        },
    }, nil
}
```

### Web Framework Choice
**Decision: Use Gin (github.com/gin-gonic/gin)**

Reasons:
- Popular, mature, well-maintained
- Built-in middleware: compression, recovery, logging
- Excellent performance (comparable to stdlib)
- Context-based API is clean
- Easy to add custom middleware
- Supports grouping (for `/namespace` and `/api/namespace` prefixes)

```go
r := gin.Default() // already includes logger, recovery
r.Use(customMiddleware...)
api := r.Group("/api")
v1 := api.Group("/v1")
```

Alternative: Echo (also good, slightly simpler). Both are fine. Gin chosen for ecosystem.

### RSS Output Formats
- **Primary**: RSS 2.0 (`application/rss+xml`)
- **Secondary**: Atom (`application/atom+xml`)
- Support via query param `?format=atom` or Accept header negotiation
- Use a unified renderer that can output both formats
- Atom: more structured, better for some readers
- Implementation: generate common internal representation, then marshal to RSS or Atom

### Porting Fidelity
**Functional Equivalence** - priority on working routes that produce similar feeds, not pixel-perfect output matching.
- Main fields preserved: title, link, description, pubDate, author, image
- Metadata: categories, language, ttl, lastBuildDate
- Items: GUID, content (HTML + text), attachments
- Accept small differences in dates, ordering, formatting as long as content is equivalent
- Focus on getting 20-50 routes working end-to-end

### Route Scope (Phase 1)
**Minimal Viable Set: 20-50 routes**
- Goal: Prove architecture, demonstrate capability, have useful feed types
- Selection criteria:
  1. **High popularity**: GitHub, Twitter/X, YouTube, Reddit, Hacker News
  2. **Diverse patterns**: HTML scraping, JSON API, XML feeds, detail page loading
  3. **Different features**: Cache usage, pagination, multiple endpoints, authentication needs
  4. **Good test coverage**: Routes with clear examples that can be tested

**Proposed initial route set (25 namespaces, ~50 routes total):**
- `github` (repos, user, events, trending) - 5 routes - JSON API
- `twitter` (user, list, likes, media) - 4 routes - API scraping
- `youtube` (channel, playlist, search) - 3 routes - HTML/API hybrid
- `reddit` (subreddit, user, popular) - 3 routes - JSON API
- `hackernews` (frontpage, item, user) - 3 routes - simple API
- `zhihu` (zhuanlan, people) - 2 routes - HTML scraping
- `weibo` (user, supertopic) - 2 routes - complex scraping
- `bilibili` (user, video, bangumi) - 3 routes - mixed
- `v2ex` (topics, nodes) - 2 routes - HTML
- `jike` (user, topic) - 2 routes - API
- `telegram` (channel) - 1 route - HTML
- `dockerhub` (repository) - 1 route - simple
- `npm` (package, search) - 2 routes - API
- `pypi` (package, releases) - 2 routes - API
- `crates` (crate) - 1 route - API
- `medium` (user, publication) - 2 routes - HTML/API
- `spotify` (playlist, artist) - 2 routes - API
- `soundcloud` (user, track) - 2 routes - API
- `arxiv` (category, search) - 2 routes - API
- `arxiv-sanity` (papers) - 1 route - API
- `steam` (workshop, profile) - 2 routes - mixed
- `douban` (movie, book, music) - 3 routes - HTML
- `imdb` (movie, actor) - 2 routes - HTML
- `bing` (images, video) - 2 routes - API/scraping
- `google` (search, news) - 2 routes - scraping
- `500px` (user, popular) - 1 route - API
- `unsplash` (user, collection) - 2 routes - API

That's ~45-50 routes across 25 namespaces. This gives excellent coverage of patterns and use cases.

**Route count can scale up later** - once framework is proven, porting more routes is straightforward.

### Error Handling
- Return errors from handlers
- Middleware catches and logs with stack trace
- Return 500 with minimal error message (don't leak internals in prod)
- Cache cleared for failed requests

### Testing Strategy
- Unit tests for: client, cache, date parser, utilities
- Integration tests for: sample routes using httptest
- End-to-end: Start server and test actual HTTP responses

### Porting Strategy for 1500+ Routes
1. **Automated porting**: Write a script to convert TSX to Go boilerplate (copy structure, convert JS to Go)
2. **Manual polishing**: Fix logic, adjust patterns
3. **Categorize by complexity**: Simple JSON → Simple; Cheerio HTML → Medium; Puppeteer → Hard
4. **Prioritize**: Most popular routes first (GitHub, Twitter, YouTube, Reddit, etc.)
5. **Test each route individually**

### Performance Considerations
- Use connection pooling (http.Client default)
- LRU cache for memory (size limit)
- Redis for distributed cache (optional)
- Parallel fetching for multi-API calls (Go routines)
- Reduce allocations: reuse buffers, pre-allocate slices
- Use `strings.Builder` for XML generation

### Request "Request Uniformization" API (from requirement #2)
Provide a unified, easy-to-use client wrapper that handles:
- Common headers (User-Agent, Accept, etc.) with rotation
- Proxy selection (round-robin or random)
- Retry with backoff (3 retries default)
- Cookie persistence (per domain)
- Rate limiting (optional, per domain)
- Cache layer (automatic, via tryGet)
- Error handling with context cancellation
- Request/response logging

Example API:
```go
client := client.NewClient()
client.SetUserAgent("random")
client.SetProxy("http://proxy:8080")
client.SetRetries(3)
client.SetTimeout(30 * time.Second)

// Auto-cache
feed, err := client.TryGet("key", func() (interface{}, error) {
    return fetchData()
}, cache.WithTTL(10*time.Minute))

// Simple request
resp, err := client.Get(url).Do()
```

## Detailed Implementation Tasks (Next Steps)

### Immediate (Today):
1. Fix main.go - remove broken imports, consolidate into proper package structure
2. Set up proper module: `module github.com/rsshub/go`
3. Create directory structure as above
4. Implement core types in `pkg/models`
5. Implement basic `Engine` that can register routes and serve XML

### Next:
6. Implement HTTP client with retry
7. Implement memory cache with tryGet
8. Implement date parser
9. Implement basic middleware: logging, compression, cache
10. Write 2-3 sample routes to test the framework

Then proceed with porting more routes and building the full system.

## Risks & Mitigation

| Risk | Mitigation |
|------|------------|
| 1500 routes too many to port manually | Write automated converter scripts, prioritize popular routes, accept partial port initially |
| Performance may not match Node.js | Benchmark early, optimize hotspots, leverage Go's concurrency |
| Date parsing complexity | Direct port from TS, comprehensive test cases |
| HTML scraping with goquery slower than cheerio? | Benchmark, use pool for documents, consider alternatives |
| Large binary size | Use build tags, modular builds, upx compression |
| Memory usage | LRU cache limits, careful with large responses |

## Verification Plan

### Unit Tests
- Test HTTP client: retry logic, proxy, headers
- Test cache: Get/Set/TryGet, expiration, concurrency
- Test date parser: comprehensive suite from TS tests
- Test middleware: each individually

### Integration Tests
- Test Engine: route registration and matching
- Test full request cycle: handler → feed → XML output
- Test 10 sample routes with mocked HTTP responses

### End-to-End Test
- Start server on test port
- Make real requests to sample routes
- Validate RSS XML output is valid (use rss validator)
- Test cache behavior
- Test concurrent requests

### Performance Tests
- Benchmark: requests/sec, memory usage
- Compare with Node.js version on same hardware
- Profile with pprof

## Milestones

1. **M1**: Core framework complete with 5 working routes (End of Week 2)
2. **M2**: 50 routes ported, fully working middleware stack (End of Week 4)
3. **M3**: Build system complete, 200+ routes, ready for testing (End of Week 6)
4. **M4**: 500+ routes, performance optimized, Docker ready (End of Week 8)
5. **M5**: 1000+ routes, documentation complete, first release candidate (End of Week 10)
