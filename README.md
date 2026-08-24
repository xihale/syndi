# Syndi

**Syndi** 是 [RSSHub](https://docs.rsshub.app) 的轻量级 Go 重写版（当前版本 **0.0.1**）。

它保留了 RSSHub 的核心思路——把互联网上五花八门的内容统一转写成 RSS/Atom feed，但用 Go 从零实现：单二进制、低内存、内置缓存与请求伪装，部署只要一个文件。

## 特性

- 🚀 单二进制部署，Go 并发模型
- 📡 RSS / Atom 输出，原生 RSS/Atom 直接转发
- 🧩 路由注册表机制，移植自 RSSHub 的同名路由保持路径一致
- ⚡ 路由级 TTL 缓存 + ETag
- 🥸 请求伪装：UA 轮换、Referer/Cookie/Language 一行调用（见 `docs/DISGUISE.md`）
- 🔐 凭据声明机制：命名空间声明所需环境变量，文档站自动展示配置状态
- 📄 内置瑞士极简风文档站（暗色自适应），feed 挂载在 `/rss/<route>`

## 快速开始

```bash
make build
./build/syndi          # 或 go run ./cmd
```

配置文件按以下顺序查找 `config.yaml`：
1. 当前目录
2. `$XDG_CONFIG_HOME/syndi/config.yaml`
3. `/etc/syndi/config.yaml`

环境变量凭据示例：

```bash
ZHIHU_COOKIES='z_c0=xxx; d_c0=yyy' ./build/syndi
```

## 与 RSSHub 的关系

Syndi 是 [RSSHub](https://github.com/DIYgod/RSSHub)（AGPL-3.0）的衍生作品：不仅大量路由直接移植自 RSSHub 的同名实现，缓存策略、请求伪装等关键逻辑也参照并翻译自 RSSHub 的设计。因此 Syndi 同样以 AGPL-3.0 发行。感谢 RSSHub 及其所有贡献者。

## 协议

[GNU Affero General Public License v3.0](LICENSE) © 2026 xihale

任何网络服务形式分发本项目的修改版本时，必须开放其完整源代码。
