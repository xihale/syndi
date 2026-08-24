# Request Disguise API (internal/disguise)

统一请求伪装方案。从已移植路由中提炼的共性需求：

| 共性需求 | 典型场景 | API |
|---|---|---|
| 浏览器级 UA + 头部集 | 默认 UA 被 403（crates.io、部分 CDN） | `disguise.Chrome()` 等预设 |
| Referer / 防盗链 | 图片站、API 校验来源 | `.Referer(url)`（默认自动同源） |
| Accept-Language | 区域内容、中文站 | `.Lang("zh-CN,zh;q=0.9")` |
| Cookie 注入 | 半公开接口、登录墙 | `.Cookie("k=v; k2=v2")` |
| XHR 伪装 | JSON API 需要 CORS 头 | `.JSONAccept()` |
| UA 轮换 | 反指纹采集 | `Custom(ua1, ua2).Rotate(...)` |
| 礼貌延迟 | 高频抓取易封禁 | `.Delay(min, max)` |

## 用法（路由内一行接入）

```go
import "github.com/xihale/syndi/internal/disguise"

// HTML 抓取
doc, err := disguise.Chrome().Lang("zh-CN").
    GetHTML(ctx, c.Client(), pageURL)

// JSON API
var v myResp
err := disguise.Chrome().JSONAccept().Fetch(apiURL).GetJSON(ctx, c.Client(), &v)

// 原生 RSS 包装（反爬站点）
feed, err := disguise.Firefox().Referer(siteURL).GetFeed(ctx, c.Client(), feedURL)

// GraphQL / JSON POST
err := disguise.Chrome().PostJSON(graphqlURL, map[string]any{"query": q}).
    GetJSON(ctx, c.Client(), &resp)
```

返回方法与 `routeutils.Get*` 一一对应：`GetBytes / GetString / GetJSON / GetHTML / GetXML / GetFeed`。

## 设计要点

1. **Profile 只产生 HTTP 头**，传输仍走共享 client：重试、代理、限速、超时等基础设施全部保留，伪装层零侵入。
2. **预设即默认值，可逐项覆盖**：`Chrome()/Firefox()/Safari()/ChromeMobile()/Custom(...)`；`.WithHeader(k, v)` 优先级最高，传空串删除该头。
3. **UA 轮换策略**：
   - `RotateRoundRobin` 全局轮转
   - `RotateRandom` 每次随机
   - `RotateStickyPerHost`（默认）按目标 host 固定一个 UA，保证同一站点指纹稳定，避免同会话内 UA 跳变触发风控。
4. **Referer 默认同源**：未显式设置时自动补 `scheme://host/`，显著降低图片/API 防盗链拒绝率。
5. **Cookie 显式优先**：设置的 Cookie 头覆盖 client cookie jar 对该请求的影响，行为可预期。
6. 可拓展方向：TLS/JA3 指纹（需 utls transport）、HTTP/2 帧指纹、代理池轮换 —— 均可在 `Request.GetBytes` 单点接入，不改变调用方 API。

## 测试

```bash
go test ./internal/disguise/
```

覆盖预设头部完整性、builder 覆盖优先级、三种轮换策略语义、httptest 端到端 GET/POST。
