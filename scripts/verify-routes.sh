#!/bin/bash
# 验证新增的 HTML 解析路由

echo "======================================"
echo "验证 RSSHub Go 新增路由"
echo "======================================"
echo ""

# 启动服务器
./build/rsshub-go > /tmp/rsshub-test.log 2>&1 &
SERVER_PID=$!

# 等待服务器启动
sleep 3

echo "服务器已启动 (PID: $SERVER_PID)"
echo ""

# 测试 API 端点
echo "1. 测试 /api/routes 端点:"
curl -s http://localhost:1200/api/routes | python3 -m json.tool | grep -E "(path|name)" | head -30
echo ""

# 测试文档端点
echo "2. 测试 /docs 端点可访问性:"
curl -s -o /dev/null -w "状态码: %{http_code}\n" http://localhost:1200/docs
echo ""

# 测试新增路由
echo "3. 新增的路由已注册:"
grep "Registered route:" /tmp/rsshub-test.log | grep -E "(academia|1x|30secondsofcode)"
echo ""

# 停止服务器
echo "停止服务器..."
kill $SERVER_PID 2>/dev/null
sleep 1

echo "======================================"
echo "验证完成"
echo "======================================"
