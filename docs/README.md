# Documentation Index

| Document | Purpose |
|---|---|
| [PORTING_GUIDE.md](./PORTING_GUIDE.md) | How to add or port routes: architecture, RouteSpec, handlers, credentials, quality rules, verification |
| [TESTING.md](./TESTING.md) | Test commands, offline vs live tests, manual server checks, shipping checklist |
| [CACHING.md](./CACHING.md) | Handler-level caching, TTLs, ETag, cache headers, configuration |
| [DISGUISE.md](./DISGUISE.md) | Request disguise API: browser presets, UA rotation, Referer/Cookie/Language |
| [CLIENT_CONFIG.md](./CLIENT_CONFIG.md) | HTTP client settings in `config.yaml` (UA, timeout, proxy) |
| [ROUTES_CATALOG.md](./ROUTES_CATALOG.md) | All registered routes with examples and last live-verification results |

Start with [PORTING_GUIDE.md](./PORTING_GUIDE.md) before writing your first route; keep
[ROUTES_CATALOG.md](./ROUTES_CATALOG.md) updated when routes change (regenerate via `make verify-all`).
