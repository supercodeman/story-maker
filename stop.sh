#!/bin/bash

# AI-Curton 停止脚本

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/logs"

echo "========================================"
echo "  AI-Curton 停止脚本"
echo "========================================"

stop_service() {
    local name="$1"
    local pid_file="$2"
    local port="$3"

    if [ -f "$pid_file" ]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid"
            echo -e "${GREEN}✓ ${name}已停止 (PID: $pid)${NC}"
        fi
        rm -f "$pid_file"
    fi

    local port_pid
    port_pid=$(lsof -ti :"$port" 2>/dev/null)
    if [ -n "$port_pid" ]; then
        kill $port_pid 2>/dev/null
        echo -e "${GREEN}✓ ${name}已停止 (端口 $port)${NC}"
    fi
}

stop_service "后端服务" "$LOG_DIR/server.pid" 8080
stop_service "前端服务" "$LOG_DIR/web.pid" 3000

rm -f "$LOG_DIR/ai-curton-server"

echo ""
echo -e "${GREEN}所有服务已停止${NC}"
echo "========================================"
