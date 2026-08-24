# RSSHub Go 测试指南

本文档说明如何测试新增的 HTML 解析路由。

## 目录

1. [快速测试](#快速测试)
2. [单元测试](#单元测试)
3. [集成测试](#集成测试)
4. [手动测试](#手动测试)
5. [API 测试](#api-测试)
6. [性能测试](#性能测试)

---

## 快速测试

### 1. 编译测试

```bash
# 编译项目
make build

# 或直接使用 go build
go build -o build/syndi ./cmd
```

### 2. 单元测试（全部）

```bash
# 运行所有测试
make test

# 或
go test -v ./...

# 运行特定包的测试
go test -v ./internal/parser
go test -v ./routes/academia
```

### 3. 快速验证路由注册

```bash
# 启动服务器
./build/syndi

# 查看日志输出中的路由注册信息
# 应该看到：
# Registered route: /academia/topic/:interest - Academia.edu Topics
# Registered route: /1x/:category - 1x.com Gallery
# Registered route: /30secondsofcode/category/:category/:subCategory? - 30 Seconds of Code
```

---

## 单元测试

### 测试 HTML 解析功能

创建 `routes/academia/topics_test.go`:

```go
package routes

import (
	"context"
	"testing"
	"time"

	"github.com/xihale/rsshub-go/internal/client"
	"github.com/xihale/rsshub-go/pkg/cache"
	"github.com/xihale/rsshub-go/pkg/context"
)

// TestAcademiaTopicHandler_Basic 测试基本的学术主题路由
func TestAcademiaTopicHandler_Basic(t *testing.T) {
	// 创建测试上下文
	ctx := context.Background()
	httpClient := client.New()
	cacheInstance := cache.NewMemoryCache(100)

	c := context.NewContext(nil, nil)
	c.SetParent(ctx)
	c.SetClient(httpClient)
	c.SetCache(cacheInstance)
	c.SetParams(map[string]string{"interest": "Urban_History"})

	// 调用处理器
	feed, err := AcademiaTopicHandler(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证 Feed
	if feed == nil {
		t.Fatal("expected non-nil feed")
	}

	if feed.Title == "" {
		t.Error("expected feed title to be set")
	}

	if feed.Link == "" {
		t.Error("expected feed link to be set")
	}

	if len(feed.Items) == 0 {
		t.Error("expected at least one item in feed")
	}

	// 验证第一个 Item
	if len(feed.Items) > 0 {
		item := feed.Items[0]
		if item.Title == "" {
			t.Error("expected item title to be set")
		}
		if item.Link == "" {
			t.Error("expected item link to be set")
		}
	}

	t.Logf("Feed has %d items", len(feed.Items))
}

// TestAcademiaTopicHandler_EmptyInterest 测试空兴趣参数
func TestAcademiaTopicHandler_EmptyInterest(t *testing.T) {
	ctx := context.Background()
	httpClient := client.New()
	cacheInstance := cache.NewMemoryCache(100)

	c := context.NewContext(nil, nil)
	c.SetParent(ctx)
	c.SetClient(httpClient)
	c.SetCache(cacheInstance)
	c.SetParams(map[string]string{"interest": ""})

	_, err := AcademiaTopicHandler(c)
	if err == nil {
		t.Error("expected error for empty interest parameter")
	}
}

// BenchmarkAcademiaTopicHandler 基准测试
func BenchmarkAcademiaTopicHandler(b *testing.B) {
	ctx := context.Background()
	httpClient := client.New()
	cacheInstance := cache.NewMemoryCache(100)

	c := context.NewContext(nil, nil)
	c.SetParent(ctx)
	c.SetClient(httpClient)
	c.SetCache(cacheInstance)
	c.SetParams(map[string]string{"interest": "Urban_History"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := AcademiaTopicHandler(c)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
```

### 测试 HTML 解析器

```go
package parser

import (
	"strings"
	"testing"
)

func TestExtractLinks(t *testing.T) {
	html := `
		<html>
			<body>
				<a href="/link1">Link 1</a>
				<a href="https://example.com/link2">Link 2</a>
				<a href="javascript:void(0)">Ignore</a>
			</body>
		</html>
	`

	doc, _ := LoadString(html)
	links := doc.ExtractLinks("a")

	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}

	if links[0].Href != "/link1" {
		t.Errorf("expected '/link1', got '%s'", links[0].Href)
	}

	if links[1].Href != "https://example.com/link2" {
		t.Errorf("expected 'https://example.com/link2', got '%s'", links[1].Href)
	}
}

func TestCleanText(t *testing.T) {
	input := "  This is   some  text  with   extra  spaces  "
	expected := "This is some text with extra spaces"

	result := CleanText(input)
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestAbsoluteURL(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		href     string
		expected string
	}{
		{
			name:     "absolute URL",
			base:     "https://example.com",
			href:     "https://other.com/page",
			expected: "https://other.com/page",
		},
		{
			name:     "relative URL",
			base:     "https://example.com/path/",
			href:     "page.html",
			expected: "https://example.com/path/page.html",
		},
		{
			name:     "root relative",
			base:     "https://example.com/path/",
			href:     "/root",
			expected: "https://example.com/root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AbsoluteURL(tt.base, tt.href)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
```

---

## 集成测试

### 使用 httptest 模拟 HTTP 服务器

创建 `routes/academia/integration_test.go`:

```go
package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xihale/rsshub-go/internal/client"
	"github.com/xihale/rsshub-go/pkg/cache"
	"github.com/xihale/rsshub-go/pkg/context"
)

// TestAcademiaTopicHandler_Integration 集成测试（模拟 HTTP 响应）
func TestAcademiaTopicHandler_Integration(t *testing.T) {
	// 模拟 Academia.edu HTML 响应
	mockHTML := `
		<!DOCTYPE html>
		<html>
		<body>
			<div class="works">
				<div class="div">
					<div class="title">
						<a href="/document/12345">Test Paper Title</a>
					</div>
					<div class="authors">by John Doe</div>
					<div class="summarized">This is a test abstract.</div>
				</div>
				<div class="div">
					<div class="title">
						<a href="/document/67890">Another Paper</a>
					</div>
					<div class="authors">by Jane Smith</div>
					<div class="summarized">Another test abstract.</div>
				</div>
			</div>
		</body>
		</html>
	`

	// 创建模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// 创建测试上下文，使用模拟服务器的 URL
	ctx := context.Background()
	httpClient := client.New()
	cacheInstance := cache.NewMemoryCache(100)

	c := context.NewContext(nil, nil)
	c.SetParent(ctx)
	c.SetClient(httpClient)
	c.SetCache(cacheInstance)

	// 注意：需要修改 handler 或创建测试辅助函数来注入自定义 URL
	// 这里仅展示思路

	t.Log("Integration test setup complete")
}
```

---

## 手动测试

### 1. 启动服务器

```bash
# 方式 1: 使用编译好的二进制
./build/syndi

# 方式 2: 直接运行
go run ./cmd
```

### 2. 测试新增路由

```bash
# 测试 Academia 路由
curl "http://localhost:1200/academia/topic/Urban_History" | head -50

# 测试 1x 路由
curl "http://localhost:1200/1x/latest/awarded" | head -50

# 测试 30secondsofcode 路由
curl "http://localhost:1200/30secondsofcode/category/js" | head -50
```

### 3. 验证 RSS 输出

```bash
# 使用 xmllint 验证 RSS 格式
curl -s "http://localhost:1200/academia/topic/Urban_History" | xmllint --format -

# 或使用 python 检查
curl -s "http://localhost:1200/academia/topic/Urban_History" | python3 -c "import sys; print(sys.stdin.read()[:500])"
```

### 4. 查看原始 HTML（调试用）

```bash
# 查看解析前的原始 HTML
curl "https://www.academia.edu/Documents/in/Urban_History" | head -100
```

---

## API 测试

### 1. 测试路由列表 API

```bash
# 获取所有路由（JSON 格式）
curl "http://localhost:1200/api/routes" | python3 -m json.tool

# 过滤新增的路由
curl "http://localhost:1200/api/routes" | python3 -m json.tool | grep -A 5 "academia\|1x\|30secondsofcode"
```

### 2. 测试单个路由详情

```bash
# 获取 academia 路由详情
curl "http://localhost:1200/api/routes/academia/topic/:interest" | python3 -m json.tool
```

### 3. 测试分类 API

```bash
# 获取所有分类
curl "http://localhost:1200/api/categories" | python3 -m json.tool
```

### 4. 使用 Postman 或类似工具

导入以下集合（Postman Collection）:

```json
{
  "info": {
    "name": "RSSHub Go Test",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Academia Topics",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:1200/academia/topic/Urban_History",
          "protocol": "http",
          "host": ["localhost"],
          "port": "1200",
          "path": ["academia", "topic", "Urban_History"]
        }
      }
    },
    {
      "name": "1x Gallery",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:1200/1x/latest/awarded",
          "protocol": "http",
          "host": ["localhost"],
          "port": "1200",
          "path": ["1x", "latest", "awarded"]
        }
      }
    },
    {
      "name": "30Seconds of Code",
      "request": {
        "method": "GET",
        "header": [],
        "url": {
          "raw": "http://localhost:1200/30secondsofcode/category/js",
          "protocol": "http",
          "host": ["localhost"],
          "port": "1200",
          "path": ["30secondsofcode", "category", "js"]
        }
      }
    }
  ]
}
```

---

## 性能测试

### 1. 基准测试

```bash
# 运行所有基准测试
go test -bench=. -benchmem ./...

# 运行特定包的基准测试
go test -bench=. -benchmem ./routes/academia
```

### 2. 压力测试（使用 ab）

```bash
# 安装 Apache Bench
# Ubuntu/Debian: sudo apt-get install apache2-utils

# 测试并发请求
ab -n 1000 -c 10 "http://localhost:1200/academia/topic/Urban_History"
```

### 3. 使用 wrk 压力测试

```bash
# 安装 wrk
# Ubuntu/Debian: sudo apt-get install wrk

# 压力测试
wrk -t4 -c100 -d30s "http://localhost:1200/academia/topic/Urban_History"
```

---

## 调试技巧

### 1. 启用调试日志

```bash
# 设置环境变量
export GIN_MODE=debug

# 或在代码中设置
gin.SetMode(gin.DebugMode)
```

### 2. 查看 HTTP 请求详情

```bash
# 使用 -v 参数查看详细请求
curl -v "http://localhost:1200/academia/topic/Urban_History"
```

### 3. 测试单个选择器

创建测试文件 `test_selector.go`:

```go
package main

import (
	"fmt"
	"log"

	"github.com/xihale/rsshub-go/internal/client"
	"github.com/xihale/rsshub-go/internal/parser"
)

func main() {
	// 创建客户端
	c := client.New()

	// 获取 HTML
	ctx := context.Background()
	body, err := c.Get(ctx, "https://www.academia.edu/Documents/in/Urban_History")
	if err != nil {
		log.Fatal(err)
	}

	// 解析 HTML
	doc, err := parser.LoadString(string(body))
	if err != nil {
		log.Fatal(err)
	}

	// 测试选择器
	doc.Each(".works > .div", func(i int, sel *parser.Selection) {
		title := sel.Find(".title").Text()
		fmt.Printf("Item %d: %s\n", i, title)
	})
}
```

运行测试：
```bash
go run test_selector.go
```

---

## 测试检查清单

### 编译测试
- [ ] `make build` 成功
- [ ] 无编译错误和警告
- [ ] 二进制文件大小合理

### 单元测试
- [ ] `go test ./...` 全部通过
- [ ] 测试覆盖率 > 80%
- [ ] 无内存泄漏

### 集成测试
- [ ] 服务器成功启动
- [ ] 所有路由正确注册
- [ ] RSS 输出格式正确

### 功能测试
- [ ] Academia 路由返回数据
- [ ] 1x 路由返回数据
- [ ] 30secondsofcode 路由返回数据

### 性能测试
- [ ] 响应时间 < 2s
- [ ] 内存使用合理
- [ ] 无 CPU 100%

---

## 常见问题

### Q: 测试时遇到网络错误
**A**: 确保可以访问目标网站，或使用集成测试模拟 HTTP 响应。

### Q: RSS 输出为空
**A**: 检查：
1. HTML 选择器是否正确
2. 网站是否更改了结构
3. 是否有反爬限制

### Q: 如何模拟不同的 HTTP 响应？
**A**: 使用 `httptest.NewServer` 创建模拟服务器（参考集成测试章节）。

---

## 下一步

1. 添加更多单元测试覆盖边界情况
2. 实现端到端测试
3. 添加 CI/CD 自动化测试
4. 监控生产环境性能指标
