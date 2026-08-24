#!/bin/bash
# 快速测试新增的 HTML 解析路由

set -e

echo "======================================"
echo "测试新增的 HTML 解析路由"
echo "======================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 编译测试
echo -e "${YELLOW}1. 编译测试...${NC}"
make build > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 编译成功${NC}"
else
    echo -e "${RED}✗ 编译失败${NC}"
    exit 1
fi
echo ""

# 2. 单元测试
echo -e "${YELLOW}2. 运行单元测试（短模式）...${NC}"
go test -short ./routes/academia ./routes/onex ./routes/thirtysec -v 2>&1 | grep -E "^(PASS|FAIL|===|---)" || true
echo ""

# 3. 启动服务器
echo -e "${YELLOW}3. 启动服务器...${NC}"
./build/syndi > /tmp/rsshub-test.log 2>&1 &
SERVER_PID=$!
sleep 3

# 检查服务器是否启动
if ! ps -p $SERVER_PID > /dev/null; then
    echo -e "${RED}✗ 服务器启动失败${NC}"
    cat /tmp/rsshub-test.log
    exit 1
fi
echo -e "${GREEN}✓ 服务器已启动 (PID: $SERVER_PID)${NC}"
echo ""

# 4. 测试路由 API
echo -e "${YELLOW}4. 测试路由 API...${NC}"
echo "新增的路由:"
curl -s http://localhost:1200/api/routes | python3 -m json.tool 2>/dev/null | grep -E "\"path\":|\"name\":" | grep -A 1 "academia\|1x\|30secondsofcode" | head -20
echo ""

# 5. 测试实际路由
echo -e "${YELLOW}5. 测试实际路由（可能需要网络）...${NC}"

test_route() {
    local route_name="$1"
    local url="$2"

    echo -n "测试 $route_name ... "
    response=$(curl -s -w "%{http_code}" "$url" -o /tmp/response.xml 2>/dev/null)

    if [ "$response" = "200" ]; then
        # 检查是否为有效的 RSS
        if grep -q "<?xml" /tmp/response.xml && grep -q "<rss" /tmp/response.xml; then
            item_count=$(grep -c "<item>" /tmp/response.xml || echo "0")
            echo -e "${GREEN}✓ 成功 (${item_count} 个项目)${NC}"
        else
            echo -e "${YELLOW}⚠ 响应不是有效的 RSS${NC}"
        fi
    else
        echo -e "${RED}✗ HTTP $response${NC}"
    fi
}

# 测试新增路由（可能需要网络和较长时间）
test_route "Academia Topics" "http://localhost:1200/academia/topic/Urban_History"
test_route "1x Gallery" "http://localhost:1200/1x/latest/awarded"
test_route "30Seconds of Code" "http://localhost:1200/30secondsofcode/category/js"
echo ""

# 6. 清理
echo -e "${YELLOW}6. 清理...${NC}"
kill $SERVER_PID 2>/dev/null
rm -f /tmp/response.xml
echo -e "${GREEN}✓ 清理完成${NC}"
echo ""

# 7. 测试总结
echo "======================================"
echo -e "${GREEN}测试完成！${NC}"
echo "======================================"
echo ""
echo "如需手动测试，可以使用："
echo "  - 启动服务器: ./build/syndi"
echo "  - 测试路由: curl 'http://localhost:1200/academia/topic/Urban_History'"
echo "  - 查看文档: curl 'http://localhost:1200/docs'"
echo ""
