# 测试指南快速参考

## 快速测试命令

### 1. 单元测试（短模式，不访问网络）
```bash
# 运行所有测试（不访问网络）
go test -short ./...

# 只测试新增的路由
go test -short ./routes/academia ./routes/onex ./routes/thirtysec

# 测试特定包
go test -short ./internal/parser
```

### 2. 完整集成测试（访问网络）
```bash
# 运行所有测试（包括网络测试）
go test ./...

# 测试特定路由（完整测试）
go test -v ./routes/academia
```

### 3. 编译和启动测试
```bash
# 编译
make build

# 启动服务器
./build/syndi

# 在另一个终端测试路由
curl "http://localhost:1200/academia/topic/Urban_History" | head -50
curl "http://localhost:1200/1x/latest/awarded" | head -50
curl "http://localhost:1200/30secondsofcode/category/js" | head -50
```

### 4. API 测试
```bash
# 查看所有路由
curl "http://localhost:1200/api/routes" | python3 -m json.tool

# 查看路由详情
curl "http://localhost:1200/api/routes/academia/topic/:interest" | python3 -m json.tool

# 查看分类
curl "http://localhost:1200/api/categories" | python3 -m json.tool
```

### 5. 使用测试脚本
```bash
# 运行完整测试脚本
./scripts/test-routes.sh

# 只测试编译和单元测试
./scripts/test-routes.sh 2>&1 | grep -E "^(✓|✗|测试)"
```

### 6. 基准测试（性能测试）
```bash
# 运行基准测试
go test -bench=. -benchmem ./routes/academia

# 运行所有基准测试
go test -bench=. -benchmem ./...
```

## 测试文件位置

- **Academia 测试**: `routes/academia/topics_test.go`
- **1x 测试**: `routes/onex/gallery_test.go`
- **Parser 测试**: `internal/parser/html_test.go`
- **完整测试指南**: `docs/TESTING.md`

## 测试覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 查看特定包的覆盖率
go test -cover ./routes/academia
```

## 常见测试场景

### 测试单个选择器
创建 `test_selector.go`:
```go
package main

import (
    "fmt"
    "log"

    "github.com/xihale/rsshub-go/internal/client"
    "github.com/xihale/rsshub-go/internal/parser"
)

func main() {
    c := client.New()
    ctx := context.Background()

    body, err := c.Get(ctx, "https://www.academia.edu/Documents/in/Urban_History")
    if err != nil {
        log.Fatal(err)
    }

    doc, err := parser.LoadString(string(body))
    if err != nil {
        log.Fatal(err)
    }

    doc.Each(".works > .div", func(i int, sel *parser.Selection) {
        title := sel.Find(".title").Text()
        fmt.Printf("Item %d: %s\n", i, title)
    })
}
```

运行:
```bash
go run test_selector.go
```

### 调试网络请求
```bash
# 查看原始 HTML
curl "https://www.academia.edu/Documents/in/Urban_History" | head -100

# 使用 verbose 模式
curl -v "http://localhost:1200/academia/topic/Urban_History"
```

## 测试结果示例

成功的测试输出:
```
=== RUN   TestAcademiaTopicHandler_Basic
--- PASS: TestAcademiaTopicHandler_Basic (1.23s)
    topics_test.go:72: Feed has 15 items
=== RUN   TestAcademiaTopicHandler_EmptyInterest
--- PASS: TestAcademiaTopicHandler_EmptyInterest (0.00s)
PASS
ok      github.com/xihale/rsshub-go/routes/academia    0.002s
```

基准测试输出:
```
BenchmarkAcademiaTopicHandler-8   	     100	   12345678 ns/op
PASS
```
