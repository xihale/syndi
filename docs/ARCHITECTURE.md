# 架构设计（Architecture）

本文描述 Syndi 的整体架构：设计目标、请求生命周期、包布局与各子系统的职责边界。
面向想理解系统全貌或改动核心基础设施的贡献者；只想新增路由的话读
[PORTING_GUIDE.md](./PORTING_GUIDE.md) 即可。

## 设计目标

Syndi 是 [RSSHub](https://github.com/DIYgod/RSSHub) 的 Go 重写，围绕四个目标取舍：

1. **单二进制、低内存**：目标是 ≤512MB 内存的 VPS 也能稳定跑全量路由（Badger
   缓存的默认参数就是按这个规格调的，见 [CACHING.md](./CACHING.md)）。
2. **元数据驱动**：路由的路径、参数、分类、TTL 全部声明在 `RouteSpec` 里，同一份
   元数据同时驱动 HTTP 路由、文档站、`/api/*` JSON 接口和路由目录验证，没有第二份
   需要手工同步的清单。
3. **传输基础设施与请求伪装分离**：重试、限速、代理、超时等全部收敛在一个共享
   HTTP client 里；反爬伪装（UA/Referer/Cookie）只构造 HTTP 头，不碰传输层。路由
   作者一行调用即可获得完整基础设施。
4. **优雅降级**：Badger 初始化失败自动回退内存缓存；`routeutils` 的缓存助手在
   cache 实例为 nil 时直接执行 fetch 函数；配置文件缺失时使用默认值启动。

## 总体架构

```
                Feed 阅读器 / FreshRSS / curl
                            │
                            ▼
        ┌───────────────────────────────────────────┐
        │  gin engine (cmd/server.go)               │
        │  Recovery → Logger → AccessKey → Header   │
        └───────────────────────────────────────────┘
                            │
        ┌───────────────────┼─────────────────────┐
        ▼                   ▼                     ▼
   文档站 & API        /status 健康检查      /rss/<route>  全部 feed 路由
   (pkg/docs)          （不走缓存）              │
                                                ▼
                              ┌──────────────────────────┐
                              │ internal/handlercache.Cached    │
                              │ 缓存键 = "feed:" + path  │
                              └──────────────────────────┘
                                 │ HIT              │ MISS
                                 ▼                  ▼
                    ProcessFeed(查询参数)   ctxpkg.Context{params,
                                 │           client, cache}
                                 │              │
                                 │              ▼
                                 │      路由 handler (routes/<ns>)
                                 │      经 routeutils / disguise 取数
                                 │              │
                                 │              ▼
                                 │      internal/client 共享 HTTP client
                                 │      （重试/限速/代理/重定向/Cookie）
                                 │              │
                                 │              ▼
                                 │         上游互联网
                                 │              │
                                 └──── *models.Feed ────┘
                                            │
                                            ▼
                          序列化：RSS (默认) / Atom / JSON
                          （?format= 查询参数选择）
                                            │
                                            ▼
                          ETag / 304、X-Cache: HIT|MISS
```

## 仓库布局

```
cmd/            入口。server.go 是 main；routes_gen.go 是生成物（勿手改）
internal/       不对外的实现细节
  handlercache/ handler 层缓存包装（internal/handlercache.Cached）
  client/       共享 HTTP client（重试、限速、代理、重定向、Cookie jar）
  disguise/     请求伪装：浏览器预设、UA 轮换、Referer/Cookie/Lang
  middleware/   gin 中间件：Recovery/Logger/AccessKey/Header + 查询参数处理
  parser/       HTML（goquery 封装）与 RSS/Atom/RDF 解析
  routeutils/   路由作者的工具集：RouteSpec、Get*、NewFeed/NewItem…
  testutil/     LIVE=1 端到端测试的 RunHandler 脚手架
pkg/            可复用的库
  cache/        两级缓存实现：内存 LRU + BadgerDB（gob 编码、原生 TTL）
  config/       config.yaml 加载与默认值
  context/      路由 handler 收到的 Context（params/query/client/cache）
  docs/         内置文档站与 /api/* JSON 接口
  logger/       zap + 自定义 KDL 编码器
  models/       核心数据模型：Feed/Item/Author/Route/…
  registry/     全局路由注册表（路径 → Route，按命名空间分组）
  rss/          RSS 2.0 / Atom 序列化
  utils/date/   宽容的日期解析
routes/         121 个命名空间包，每包一个站点；详见下文"路由系统"
scripts/        generate-routes.go、new-route.sh、verify-routes.sh、verify-all.sh
deploy/         systemd user service、env 模板与部署说明
docs/           本文档
```

`internal` 与 `pkg` 的分界是标准 Go 约定：`pkg` 下的包理论上可以被别的项目
import（比如两级缓存或 RSS 生成器），`internal` 强制不可。

## 请求生命周期

以 `GET /rss/github/trending/daily/go?limit=10` 为例：

1. **中间件链**（`cmd/server.go` 中按序装配）：
   - `Recovery`：panic 恢复（最外层）；
   - `Logger`：zap 结构化请求日志；
   - `AccessKey`：配置了 `middleware.access_key` 时，要求 `?key=` 或
     `X-Access-Key` 头（常数时间比较）；`/status` 豁免；未配置则为 no-op；
   - `Header`：CORS（`allow_origin`）、`X-Content-Type-Options: nosniff`、
     `Cache-Control: public, max-age=<全局默认 TTL>`、OPTIONS 预检直接 204。
2. **路由匹配**：所有 feed 挂在 `/rss` 前缀下（根路径留给文档站），Gin 按
   `:param` 模式匹配到具体路由。
3. **handler 层缓存**（`internal/handlercache.Cached` 包装）：
   - 缓存键只有路径（`feed:/rss/github/trending/daily/go`），**不含查询参数**；
   - 命中：取出缓存的原始 feed → `ProcessFeed` 按本次查询参数加工 → 序列化 →
     `X-Cache: HIT`，body 的 sha256 作为 ETag，命中 `If-None-Match` 回 304；
   - 未命中：构造 `ctxpkg.Context`（注入路径参数、共享 client、cache 实例），
     调用路由 handler。
4. **路由 handler**（`routes/github` 包内函数）：解析参数 → 经 `routeutils`
     或 `disguise` 取上游数据 → 组装 `*models.Feed` 返回。handler 内部也可用
     `routeutils.CacheFeed` 等做**细粒度缓存**（比如缓存 API 分页结果而不是
     整个 feed），两层缓存互不冲突。
5. **回写**：成功的原始 feed 按 TTL 写入缓存（错误响应与空 feed 不缓存）→
   `ProcessFeed` → 序列化为 RSS（默认）/Atom/JSON（`?format=`）→ `X-Cache: MISS`。

### 查询参数处理（ProcessFeed）

缓存的永远是**原始** feed，以下参数在每次请求时动态施加（`internal/middleware/parameter.go`）：

| 参数 | 作用 |
|---|---|
| `limit` | 截断条数 |
| `filter` / `filterout` | 按关键词正则包含/排除标题或描述 |
| `filter_time` | 按时间窗过滤（如 `7天`、`168h`） |
| `sorted` | 按发布时间排序，默认开启 |
| `brief` | 只保留标题等简要字段 |

这是"缓存键只有路径"的直接原因：同一份上游抓取可以同时服务
`?limit=5`、`?brief=true` 等任意变体，避免缓存被参数组合打爆。

## 路由系统：从 RouteSpec 到 HTTP 端点

```
routes/<ns>/routes.go          scripts/generate-routes.go      cmd/routes_gen.go
var Routes = []RouteSpec{…}  ──(扫描 routes/*/ 有 Routes ──►  生成 registerRoutePackages()
                               导出变量的包，无则跳过)         （make build 自动执行）
                                                                      │
                                                                      ▼
                                              registry.MustRegisterRoutesWithBase("<ns>", Routes)
                                                                      │
                                                                      ▼
                                              pkg/registry（全局单例：path → *models.Route，
                                              按首段路径分组成命名空间）
                                                                      │
                                                                      ▼
                                              cmd/server.go 遍历注册表，
                                              在 Gin 上挂载 /rss + route.Path
```

要点：

- **约定**：`routes/<ns>/` 包名必须是 `routes`，且必须有导出 `Routes` 切片的
  `routes.go`，否则被生成脚本静默跳过（这也是忘写 routes.go 时"路由不生效"的原因）。
- **命名空间 = 目录名**：`routes/github` → 基路径 `/github` → HTTP 路径
  `/rss/github/...`。`RouteSpec.Path` 写相对路径（`trending/:range/:language`）。
- **凭据声明**：依赖环境变量的命名空间在 `init()` 里调
  `registry.RegisterNamespaceEnv`，文档站的 CREDENTIALS 面板与 `/api/config`
  据此展示配置状态（只暴露布尔值，不回显内容）。
- **重复路径会在启动时 panic**（`MustRegister*`），故路径冲突在第一次启动即暴露。

## 核心数据模型（pkg/models）

```go
type HandlerFunc func(*ctxpkg.Context) (*Feed, error)   // 所有路由 handler 的签名

type Feed struct { Title, Link, Description …; Items []Item }
type Item  struct { Title, Link, GUID, Description, Author, Categories, PubDate … }
type Route struct { Path, Name, Example, Description, Maintainers,
                    Categories, Features, Parameters,
                    CacheTTL *time.Duration, Handler HandlerFunc }
```

`*models.Feed` 是贯穿全系统的通货：handler 产出它、缓存以 gob 存它、
`pkg/rss` 把它序列化成 RSS/Atom。`CacheTTL` 为 nil 时用全局默认 TTL。

## 子系统细节

### 共享 HTTP client（internal/client）

每个进程一个实例，所有路由复用。能力全部通过 `ClientOption` 配置：UA、超时、
代理/禁代理、Cookie jar、Bearer token、重试（幂等方法 + 408/425/429/500/502/
503/504 状态白名单，429/503 尊重 `Retry-After`）、最大重定向、**按 host 限速**。
配置项见 [CLIENT_CONFIG.md](./CLIENT_CONFIG.md)。重定向时保留自定义 UA，响应体
有大小上限。

### 请求伪装（internal/disguise）

`Chrome()`/`Firefox()` 等预设 + builder 覆盖（`.Referer` 默认同源、`.Lang`、
`.Cookie`、`.JSONAccept`、`.Delay`、UA 轮换三策略，默认按 host 粘滞）。伪装层
只产生 HTTP 头，传输行为完全由共享 client 决定——这是"零侵入"的关键。详见
[DISGUISE.md](./DISGUISE.md)。

### 两级缓存（pkg/cache）

内存 LRU（解码后的值，零序列化热路径）+ BadgerDB（gob 编码、原生 TTL、write-through、
命中异步回填内存）。附带过期清理与 value-log GC 两个后台机制。配置与内存调优见
[CACHING.md](./CACHING.md)。

### 解析器（internal/parser）

`parser.Document` 封装 goquery（`.Each/.Find/.Text/.AttrOr`），供 HTML 抓取路由
使用；`rssfeed` 子包解析原生 RSS2/Atom/RDF 为 `*models.Feed`（带真实字符集转换），
feed 包装类路由直接转发即可。

### RSS 生成（pkg/rss）

`GenerateRSS` / `GenerateAtom`。输出确定性：时间戳格式统一、`content:encoded`
按标准命名空间输出，相同 feed 生成相同 body → ETag 稳定。

### 文档站与 API（pkg/docs）

`/` 是瑞士极简风文档站（暗色自适应，按命名空间分页）；`/api/routes`、
`/api/routes/<path>`、`/api/namespaces`、`/api/categories`、`/api/config`
暴露注册表与凭据状态 JSON；`/rss`（无子路径）等价于 `/api/routes`；
`/robots.txt` 屏蔽爬虫。数据全部来自注册表，无独立数据源。

### 配置与日志

- 配置：`pkg/config`，YAML，查找顺序 `SYNDI_CONFIG` 环境变量 → `./config.yaml`
  → `/etc/syndi/config.yaml`，全量键参考 [CONFIGURATION.md](./CONFIGURATION.md)。
- 日志：`pkg/logger`，zap 核心配自定义 **KDL 编码器**；`env: development`
  时 Debug 级，其余（含 `production`）Info 级。

## 横切关注点

- **错误处理**：handler 返回 error 时，若 handler 通过 `c.Set("error_code", n)`
  标记了状态码则按其返回 JSON 错误，否则 500。错误与空 feed 不进缓存。
- **优雅关停**：SIGINT/SIGTERM → `server.Shutdown`（5s 排空在途请求）→ 之后再
  关闭缓存（Badger 必须显式 Close）。
- **多地址监听**：`server.host` 支持逗号分隔多地址（如 `127.0.0.1,172.17.0.1`），
  用于容器经 docker0 网关回连而不暴露公网。部署形态见
  [deploy/README.md](../deploy/README.md)。

## 主要设计决策记录

| 决策 | 理由 | 代价 / 权衡 |
|---|---|---|
| 缓存键只含路径 | 一份抓取服务所有 `limit/filter/brief` 变体，命中率最大化 | 原始 feed 必须存完整条目；加工必须在请求路径上做 |
| 缓存原始 feed 而非响应体 | `?format=rss/atom/json` 共享同一份缓存 | 序列化每次执行（代价小，且换来稳定 ETag） |
| 伪装只造 HTTP 头 | 传输层（重试/代理/限速）对路由作者永远可用，不可绕过 | TLS/JA3 层伪装暂不支持（可后续在单点接入） |
| 路由注册靠代码生成 | 121 个包的 import 不可能手工维护；`make build` 自动跑 | 改动命名空间后需重新生成（脚手架会代劳） |
| gob 序列化缓存值 | `*models.Feed` 结构化存取，无反射开销 | 类型需 `gob.Register`（server 启动时注册） |
| feed 挂 `/rss` 前缀 | 根路径留给文档站，二者不打架 | 与 RSSHub 原版路径不同，迁移要改 URL |

## 性能与资源画像

- 两级缓存稳态内存约 `block_cache_mb + memtable_mb × num_memtables`（默认
  ~96MB）+ LRU 条目本身，为 ≤512MB VPS 预算设计；
- 热路径（内存命中）无序列化，直接从 LRU 拿解码后的 feed；
- Badger 退化路径：value-log GC 每周期最多重写 8 个文件，避免刷新尖峰；
- `make build-packed` 产出 stripped + UPX 的最小体积二进制。
