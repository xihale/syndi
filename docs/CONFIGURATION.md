# 配置参考（Configuration）

`config.yaml` 全量键说明。仓库根目录的 [config.yaml](../config.yaml) 是带注释的
完整示例。各子系统的深入文档：缓存见 [CACHING.md](./CACHING.md)，HTTP 客户端见
[CLIENT_CONFIG.md](./CLIENT_CONFIG.md)。

## 配置文件查找顺序

`pkg/config.Load` 依次查找，先命中先用：

1. `$SYNDI_CONFIG` 环境变量指定的路径；
2. 当前目录 `./config.yaml`；
3. `/etc/syndi/config.yaml`。

都没有时直接用内置默认值启动（日志会提示）。`make install-config` 可把示例
配置安装到系统路径（已存在时自动备份为 `config.yaml.bak`）。

## server — 服务器

| 键 | 默认值 | 说明 |
|---|---|---|
| `host` | `127.0.0.1` | 监听地址。逗号分隔可绑多个网卡（如 `127.0.0.1,172.17.0.1`，后者供容器经 docker0 网关回连）；空字符串绑所有接口，**公网暴露请改用反向代理** |
| `port` | `1200` | 监听端口 |
| `env` | `production` | 仅 `development` 时日志为 Debug 级，其余值（含 `production`）为 Info 级 |
| `read_timeout` | `30s` | 读请求超时 |
| `write_timeout` | `30s` | 写响应超时 |
| `idle_timeout` | `120s` | Keep-alive 空闲超时 |

## cache — 缓存

| 键 | 默认值 | 说明 |
|---|---|---|
| `type` | `memory` | `memory` 纯内存；`badger` 两级缓存（内存 + 持久化），重启不丢 |
| `badger.path` | `./data/cache` | Badger 数据目录 |
| `ttl` | `15m` | 全局默认 TTL；路由可在 `RouteSpec.CacheTTL` 覆盖 |
| `memory_size` | `10000` | 内存 LRU 容量（条数） |
| `cleanup_interval` | `5m` | 过期键清理周期 |
| `memtable_mb` / `num_memtables` | `16` / `4` | Badger memtable 预算 |
| `block_cache_mb` | `32` | Badger 数据块缓存 |
| `index_cache_mb` | `0` | 索引缓存，`0` = Badger 自动 |
| `vlog_file_mb` | `128` | 单个 value-log 文件上限 |
| `gc_interval` | `10m` | value-log GC 周期，`0s` 关闭（长期运行不建议关） |
| `gc_discard_ratio` | `0.5` | value-log 文件垃圾占比达到该阈值才重写 |

内存语义与磁盘回收细节见 [CACHING.md](./CACHING.md)。

## client — 出站 HTTP

| 键 | 默认值 | 说明 |
|---|---|---|
| `user_agent` | `Syndi/0.0.1 (+…)` | 出站请求默认 UA（路由可用伪装 API 覆盖） |
| `timeout` | `30s` | 单请求总超时 |
| `max_redirects` | `10` | 最大重定向次数（重定向保留 UA） |
| `proxy` | `""` | 显式代理 URL，如 `http://127.0.0.1:7890` |
| `no_proxy` | `false` | `true` 时彻底禁用代理（含环境变量代理） |

`proxy` 为空时回退到 `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` 环境变量。重试策略
（幂等方法 + 状态白名单 + `Retry-After`）见 [CLIENT_CONFIG.md](./CLIENT_CONFIG.md)。

## routes — 路由行为

| 键 | 默认值 | 说明 |
|---|---|---|
| `disable_nsfw` | `false` | 保留字段，当前未在运行时生效 |

## middleware — 中间件

| 键 | 默认值 | 说明 |
|---|---|---|
| `enable_cache` | `true` | `false` 时所有 feed 路由绕过缓存直接回源（响应带 `X-Cache: BYPASS`） |
| `access_key` | `""` | 访问密钥。**非空时**所有请求须携带 `?key=<值>` 或 `X-Access-Key` 头（常数时间比较），仅 `/status` 健康检查豁免；为空则实例完全开放 |
| `allow_origin` | `*` | CORS `Access-Control-Allow-Origin` 值 |

注意：设置 `access_key` 后文档站与 `/api/*` 也一并受保护。

## 环境变量

| 变量 | 用途 |
|---|---|
| `SYNDI_CONFIG` | 指定配置文件路径 |
| `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` | `client.proxy` 为空且未设 `no_proxy` 时的代理来源 |
| 各路由凭据（如 `ZHIHU_COOKIES`） | 命名空间在 `init()` 里经 `registry.RegisterNamespaceEnv` 声明后，文档站 CREDENTIALS 面板展示其已配置/未配置状态；部署模板见 [`deploy/syndi.env.example`](../deploy/syndi.env.example) |
