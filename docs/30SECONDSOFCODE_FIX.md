# 30secondsofcode 路由修复说明

## 问题

之前实现返回的每个 item 的 title 和 description 都相同：
```xml
<item>
  <title>Code Snippet</title>
  <description><p>Code snippet from 30secondsofcode.org...</p></description>
  <link>https://www.30secondsofcode.org/js/s/deep-clone-structured-clone</link>
</item>
```

## 原因

`fetchDetailItem` 函数只是一个占位符（stub），没有真正实现详情页抓取逻辑。

## 解决方案

已实现完整的详情页抓取，与 TypeScript 版本一致：

### 实现逻辑

1. **从列表页获取**
   - 链接：`article > h3 > a` 的 href 属性
   - 日期：`article > small > time` 的 datetime 属性

2. **抓取详情页**（对每个链接）
   - 使用 `httpClient.Get()` 获取详情页 HTML
   - 解析 HTML 提取：
     - **标题**：`main > article > h1`
     - **分类标签**：`body > main > nav > ol > li:not(:first-child):not(:last-child) > a`
     - **描述**：`main > article` 的 HTML 内容

3. **缓存机制**
   - 每个 item 缓存 1 小时
   - 缓存键：`30secondsofcode:{relative_path}`

### 与 TypeScript 版本对比

| 功能 | TypeScript | Go | 状态 |
|------|-----------|-----|------|
| 解析列表页 | ✅ | ✅ | 完全一致 |
| 抓取详情页 | ✅ | ✅ | 完全一致 |
| 提取标题 | ✅ | ✅ | 完全一致 |
| 提取分类 | ✅ | ✅ | 完全一致 |
| 提取描述 | ✅ | ✅ | 完全一致 |
| 缓存详情 | ✅ | ✅ | 完全一致 |
| 图片 URL 处理 | ✅ | ⚠️ | 简化（Go 中无法修改 DOM） |

### 关键代码片段

```go
// 抓取详情页
func fetchDetailItem(ctx context.Context, httpClient *client.Client, rootURL, url string, pubDate time.Time) (*models.Item, error) {
    // 获取详情页 HTML
    body, err := httpClient.Get(ctx, url)

    // 解析 HTML
    doc, err := parser.LoadString(string(body))

    // 提取标题
    article := doc.FindSelector("main > article")
    title := article.Find("h1").Text()

    // 提取分类
    doc.Each("body > main > nav > ol > li:not(:first-child):not(:last-child)", func(i int, sel *parser.Selection) {
        tagText := sel.Find("a").Text()
        categories = append(categories, tagText)
    })

    // 提取描述（HTML）
    description, _ := article.Html()

    return &models.Item{
        Title:       title,
        Link:        url,
        Description: description,
        PubDate:     pubDate,
        Categories:  categories,
        Author:      &models.Author{Name: "30 Seconds of Code"},
    }
}
```

## 预期输出

修复后，每个 item 应该有独特的标题和内容：

```xml
<item>
  <title>Deep Clone / Structured Clone</title>
  <description>&lt;article&gt;
    &lt;h1&gt;Deep Clone / Structured Clone&lt;/h1&gt;
    &lt;p&gt;Code snippet content...&lt;/p&gt;
    &lt;pre&gt;&lt;code&gt;// JavaScript code&lt;/code&gt;&lt;/pre&gt;
  &lt;/article&gt;</description>
  <link>https://www.30secondsofcode.org/js/s/deep-clone-structured-clone</link>
  <category>JavaScript</category>
  <category>Object</category>
  <author>30 Seconds of Code</author>
</item>
```

## 测试验证

```bash
# 启动服务器
./build/syndi

# 测试路由
curl "http://localhost:1200/30secondsofcode/category/js" | head -100

# 检查不同的 title
curl "http://localhost:1200/30secondsofcode/category/js" | grep -o '<title>.*</title>' | head -10
```

## 注意事项

1. **性能影响**：现在会为每个 item 额外请求详情页，首次访问会较慢（但有缓存）
2. **缓存策略**：详情页缓存 1 小时，列表页缓存 6 小时
3. **错误处理**：如果详情页抓取失败，会降级使用列表页的基本信息（标题和链接）
4. **Go 限制**：Go 的 goquery 无法像 cheerio 一样直接修改 DOM 属性，所以图片 URL 处理被简化了

## 改进空间

- [ ] 实现 HTML 清理（移除 h1 和 script 标签）
- [ ] 处理图片 URL（转换为绝对路径）
- [ ] 添加更多错误处理和日志
- [ ] 优化并发抓取（使用 goroutine 并发请求详情页）
