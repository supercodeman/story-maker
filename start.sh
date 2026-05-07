#!/bin/bash

# Story-Maker 一键启动脚本

set -euo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/logs"
GO_BIN="${GO_BIN:-go1.22.12}"
BACKEND_PORT=8080
FRONTEND_PORT=3000
HEALTH_TIMEOUT=60

mkdir -p "$LOG_DIR"

echo "========================================"
echo "  Story-Maker 一键启动脚本"
echo "========================================"

# 检查依赖服务
echo -e "${BLUE}检查服务状态...${NC}"

if ! nc -z 127.0.0.1 3306 2>/dev/null; then
    echo -e "${RED}MySQL 未运行，请先启动: brew services start mysql${NC}"
    exit 1
fi
echo -e "${GREEN}✓ MySQL 运行正常${NC}"

if ! nc -z 127.0.0.1 6379 2>/dev/null; then
    echo -e "${RED}Redis 未运行，请先启动: brew services start redis${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Redis 运行正常${NC}"

if ! docker ps 2>/dev/null | grep -q milvus; then
    echo -e "${YELLOW}⚠ Milvus 未运行，动态记忆功能将不可用${NC}"
    echo "  可选: docker-compose -f $SCRIPT_DIR/docker-compose-milvus.yml up -d"
fi

# 先编译后端，避免 go run 的 PID 问题
echo ""
echo -e "${BLUE}编译后端服务...${NC}"
cd "$SCRIPT_DIR/server"
$GO_BIN build -o "$SCRIPT_DIR/logs/story-maker-server" ./cmd/main.go
echo -e "${GREEN}✓ 编译完成${NC}"

# 启动后端
echo -e "${BLUE}启动后端服务...${NC}"
"$LOG_DIR/story-maker-server" > "$LOG_DIR/server.log" 2>&1 &
SERVER_PID=$!
echo $SERVER_PID > "$LOG_DIR/server.pid"

# 等待后端健康检查通过
elapsed=0
while [ $elapsed -lt $HEALTH_TIMEOUT ]; do
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo -e "${RED}后端服务进程已退出，最后 30 行日志：${NC}"
        tail -30 "$LOG_DIR/server.log"
        rm -f "$LOG_DIR/server.pid"
        exit 1
    fi
    if curl -s "http://localhost:$BACKEND_PORT/health" | grep -q "ok"; then
        echo -e "${GREEN}✓ 后端服务启动成功 (PID: $SERVER_PID)${NC}"
        break
    fi
    sleep 1
    elapsed=$((elapsed + 1))
done

if [ $elapsed -ge $HEALTH_TIMEOUT ]; then
    if kill -0 $SERVER_PID 2>/dev/null; then
        echo -e "${YELLOW}后端服务仍在初始化中（进程存活），请稍后手动检查: curl http://localhost:$BACKEND_PORT/health${NC}"
        echo -e "${YELLOW}日志: $LOG_DIR/server.log${NC}"
    else
        echo -e "${RED}后端服务启动失败（进程已退出），请检查日志: $LOG_DIR/server.log${NC}"
        rm -f "$LOG_DIR/server.pid"
        exit 1
    fi
fi

# 启动前端
echo ""
echo -e "${BLUE}启动前端服务...${NC}"
cd "$SCRIPT_DIR/web"
npx vite > "$LOG_DIR/web.log" 2>&1 &
WEB_PID=$!
echo $WEB_PID > "$LOG_DIR/web.pid"

# 等待前端端口可用
elapsed=0
while [ $elapsed -lt $HEALTH_TIMEOUT ]; do
    if curl -s "http://localhost:$FRONTEND_PORT" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 前端服务启动成功 (PID: $WEB_PID)${NC}"
        break
    fi
    sleep 1
    elapsed=$((elapsed + 1))
done

if [ $elapsed -ge $HEALTH_TIMEOUT ]; then
    echo -e "${RED}前端服务启动失败，请检查日志: $LOG_DIR/web.log${NC}"
    kill $WEB_PID 2>/dev/null
    rm -f "$LOG_DIR/web.pid"
    exit 1
fi

echo ""
echo "========================================"
echo -e "${GREEN}所有服务启动完成${NC}"
echo "========================================"
echo ""
echo "  前端界面: http://localhost:$FRONTEND_PORT"
echo "  后端 API: http://localhost:$BACKEND_PORT"
echo "  后端日志: $LOG_DIR/server.log"
echo "  前端日志: $LOG_DIR/web.log"
echo "  停止服务: ./stop.sh"
echo "========================================"
