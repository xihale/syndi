# Documentation Index

| Document | Purpose |
|---|---|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 架构设计：设计目标、请求生命周期、包布局、子系统职责、设计决策记录 |
| [CONFIGURATION.md](./CONFIGURATION.md) | `config.yaml` 全量配置参考：查找顺序、各段键值、环境变量 |
| [PORTING_GUIDE.md](./PORTING_GUIDE.md) | How to add or port routes: RouteSpec, handlers, credentials, quality rules, verification |
| [TESTING.md](./TESTING.md) | Test commands, offline vs live tests, manual server checks, shipping checklist |
| [CACHING.md](./CACHING.md) | Handler-level caching, TTLs, ETag, cache headers, configuration |
| [DISGUISE.md](./DISGUISE.md) | Request disguise API: browser presets, UA rotation, Referer/Cookie/Language |
| [CLIENT_CONFIG.md](./CLIENT_CONFIG.md) | HTTP client settings in `config.yaml` (UA, timeout, proxy) |
| [ROUTES_CATALOG.md](./ROUTES_CATALOG.md) | All registered routes with examples and last live-verification results |
| [../deploy/README.md](../deploy/README.md) | Deployment: systemd user service, reverse proxy + auth, FreshRSS cutover |

Reading paths:

- **理解系统**：ARCHITECTURE.md → CONFIGURATION.md
- **新增/移植路由**：PORTING_GUIDE.md → DISGUISE.md → TESTING.md（发版前用
  `make verify-all` 刷新 ROUTES_CATALOG.md）
- **部署运维**：CONFIGURATION.md → CACHING.md → ../deploy/README.md
