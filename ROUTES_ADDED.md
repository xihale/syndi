# 新增 HTML 解析路由总结

本文档总结了从 RSSHub TypeScript 版本移植到 Go 版本的 3 个基于 HTML 解析的路由。

## 新增路由

### 1. Academia.edu Topics
- **路径**: `/academia/topic/:interest`
- **文件**: `routes/academia/topics.go`
- **功能**: 从 Academia.edu 抓取指定主题的学术论文列表
- **特点**:
  - 纯 HTML 列表页解析，无需详情页
  - 提取标题、链接、作者、描述
  - 2 小时缓存时间

### 2. 1x.com Gallery
- **路径**: `/1x/:category`
- **文件**: `routes/onex/gallery.go`
- **功能**: 从 1x.com 抓取摄影作品集
- **特点**:
  - 两步获取：先获取页面模式参数，再调用 API
  - 提取作品标题、图片、作者信息
  - 图片嵌入在 RSS 描述中
  - 2 小时缓存时间

### 3. 30 Seconds of Code
- **路径**: `/30secondsofcode/category/:category/:subCategory?`
- **文件**: `routes/thirtysec/category.go`
- **功能**: 从 30secondsofcode.org 抓取代码片段
- **特点**:
  - 支持分类和子分类
  - 从列表页提取链接，可选择抓取详情页
  - 6 小时缓存时间

## 使用的技术

### HTML 解析工具
项目已有完善的 HTML 解析工具：
- `internal/parser/html.go` - goquery 包装器
- `internal/parser/extract.go` - 提取工具（链接、图片、元数据等）
- `internal/routeutils/jsonapi.go` - HTTP/JSON/XML/HTML 获取

### 常用工具函数

#### 路由工具 (routeutils)
```go
// 创建 Feed
feed := routeutils.NewFeed(title, link, description)

// 创建 Item
item := routeutils.NewItem(title, link, description, pubDate)

// 添加 Item 到 Feed
routeutils.AddItem(feed, item)

// 设置作者
routeutils.SetItemAuthor(item, name, email, uri)

// 设置分类
routeutils.SetCategories(item, categories...)

// 获取 HTML
doc, err := routeutils.GetHTML(ctx, client, url)
```

#### 解析器工具 (parser)
```go
// 选择元素并迭代
doc.Each("selector", func(i int, sel *parser.Selection) {
    title := sel.Text()
    href, _ := sel.Attr("href")
    child := sel.Find("childSelector")
})

// 查找单个元素
if elem := doc.First("selector"); elem != nil {
    text := elem.Text()
}

// 提取属性
value, exists := elem.Attr("attribute")
valueOr := elem.AttrOr("attribute", defaultValue)

// 获取 HTML
html, _ := elem.Html()
```

#### 日期解析 (date)
```go
import "github.com/xihale/rsshub-go/pkg/utils/date"

// 解析多种日期格式
t, err := date.ParseDate("2024-01-15")
t, err := date.ParseDate("3 hours ago")
t, err := date.ParseDate("2024年01月15日")
```

## 路由实现模式

### 标准 Route 结构
```go
func init() {
    cacheTTL := 2 * time.Hour

    route := &models.Route{
        Path:         "/path/:param",
        Name:         "Route Name",
        Example:      "path/example",
        Maintainers:  []string{"yourname"},
        Description:  "Route description",
        Categories:   []models.Category{{Name: "category"}},
        Features:     models.Features{},
        Handler:      RouteHandler,
        Parameters: []models.Parameter{
            {Name: "param", Required: true, Description: "..."},
        },
        CacheTTL: &cacheTTL,
    }
    if err := registry.GetRegistry().Register(route); err != nil {
        panic(err)
    }
}

func RouteHandler(c *ctxpkg.Context) (*models.Feed, error) {
    // 1. 获取参数
    param := c.Param("param")

    // 2. 获取上下文
    ctx := c.Parent()

    // 3. 获取 HTML
    doc, err := routeutils.GetHTML(ctx, c.Client(), url)
    if err != nil {
        return nil, err
    }

    // 4. 创建 Feed
    feed := routeutils.NewFeed(title, link, description)

    // 5. 解析并添加 Items
    doc.Each("selector", func(i int, sel *parser.Selection) {
        item := routeutils.NewItem(title, link, desc, pubDate)
        routeutils.AddItem(feed, item)
    })

    return feed, nil
}
```

## 缓存策略

### Handler 级别缓存
所有路由使用 `internal/cache/handler.go` 中的缓存包装器：
- 默认缓存时间：15 分钟
- 可通过 `CacheTTL` 字段自定义
- 支持 ETag 和 304 响应
- 错误响应（4xx/5xx）不缓存

### Item 级别缓存
对于需要抓取详情页的路由（如 30secondsofcode）：
```go
cacheKey := fmt.Sprintf("namespace:%s", itemLink)
if cached, ok := cache.Get(cacheKey); ok {
    item = cached.(*models.Item)
} else {
    item = fetchDetail(...)
    cache.Set(cacheKey, item, ttl)
}
```

## 测试

所有路由已编译通过并注册成功：
```bash
$ make build
$ go run ./cmd/server.go
```

服务器会输出注册的路由：
```
Registered route: /academia/topic/:interest - Academia.edu Topics
Registered route: /1x/:category - 1x.com Gallery
Registered route: /30secondsofcode/category/:category/:subCategory? - 30 Seconds of Code
```

## 下一步建议

1. **完善 30secondsofcode 详情页抓取**: 当前使用简化版本，需要实现完整的详情页解析
2. **添加更多路由**: 参考 RSSHub 的其他简单 HTML 解析路由继续移植
3. **增强错误处理**: 添加更详细的错误日志和重试机制
4. **添加测试**: 为每个路由编写单元测试
5. **性能优化**: 对于有大量项目的列表，考虑分页或限制数量

## 参考资源

- 原始 RSSHub 路由: `/home/xihale/go/rsshub/RSSHub/lib/routes/`
- Go 路由实现: `routes/academia/`, `routes/onex/`, `routes/thirtysec/`
- 解析器文档: `internal/parser/`
- 路由工具: `internal/routeutils/`
