# Syndi

**Syndi** 是 [RSSHub](https://docs.rsshub.app) 的 Go 重写版：把各站点内容统一转写成 RSS/Atom feed。单二进制、低内存、内置缓存与请求伪装。

## 特性

- 单二进制部署
- RSS / Atom 输出，原生 feed 直接转发
- 路由注册表机制，移植自 RSSHub 的同名路由保持路径一致
- 路由级 TTL 缓存 + ETag
- 请求伪装：UA 轮换、Referer/Cookie/Language 一行调用（见 [`docs/DISGUISE.md`](docs/DISGUISE.md)）
- 凭据声明机制：命名空间声明所需环境变量，文档站自动展示配置状态
- 内置文档站，feed 挂载在 `/rss/<route>`

## 项目状态

已移植 **288 条路由 / 122 个命名空间**，完整清单见 [`docs/ROUTES_CATALOG.md`](docs/ROUTES_CATALOG.md)。

已知限制：

- `steam/news` 部分网络下被上游 403 拦截
- Reddit 对未认证 `.json` API 限流严重，路由改用伪装请求走原生 `.rss`

## 快速开始

```bash
cp config.example.yaml config.yaml   # 可选：按需修改配置
make build
./build/syndi          # 或 go run ./cmd
```

配置文件按以下顺序查找（先命中先用，全量键见 [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md)）：
1. `$SYNDI_CONFIG` 环境变量指定的路径
2. 当前目录 `./config.yaml`
3. `/etc/syndi/config.yaml`

环境变量凭据示例：

```bash
ZHIHU_COOKIES='z_c0=xxx; d_c0=yyy' ./build/syndi
```

## 文档

| 文档 | 内容 |
|---|---|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | 架构设计：请求生命周期、包布局、子系统与设计决策 |
| [`docs/CONFIGURATION.md`](docs/CONFIGURATION.md) | `config.yaml` 全量配置参考 |
| [`docs/PORTING_GUIDE.md`](docs/PORTING_GUIDE.md) | 新增/移植路由完整指南 |
| [`docs/TESTING.md`](docs/TESTING.md) | 测试命令与上线检查清单 |
| [`docs/CACHING.md`](docs/CACHING.md) | 缓存架构与配置 |
| [`docs/DISGUISE.md`](docs/DISGUISE.md) | 请求伪装 API |
| [`docs/CLIENT_CONFIG.md`](docs/CLIENT_CONFIG.md) | HTTP 客户端配置项 |
| [`docs/ROUTES_CATALOG.md`](docs/ROUTES_CATALOG.md) | 全部路由目录 |
| [`deploy/README.md`](deploy/README.md) | 部署：systemd 服务、反向代理与认证 |

## 与 RSSHub 的关系

Syndi 是 [RSSHub](https://github.com/DIYgod/RSSHub)（AGPL-3.0）的衍生作品：不仅大量路由直接移植自 RSSHub 的同名实现，缓存策略、请求伪装等关键逻辑也参照并翻译自 RSSHub 的设计。因此 Syndi 同样以 AGPL-3.0 发行。感谢 RSSHub 及其所有贡献者。

## 协议

[GNU Affero General Public License v3.0](LICENSE) © 2026 xihale

任何网络服务形式分发本项目的修改版本时，必须开放其完整源代码。
