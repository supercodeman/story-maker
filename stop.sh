#!/bin/bash

# Story-Maker 停止脚本

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/logs"

echo "========================================"
echo "  Story-Maker 停止脚本"
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
        return
    fi

    # PID 文件不存在时，通过端口查找但只杀工作目录在本项目下的进程
    local port_pids
    port_pids=$(lsof -ti :"$port" 2>/dev/null)
    if [ -n "$port_pids" ]; then
        for pid in $port_pids; do
            local cwd
            cwd=$(lsof -p "$pid" -Fn 2>/dev/null | grep "^ncwd" | sed 's/^n//' || true)
            if [ -z "$cwd" ]; then
                cwd=$(ps -o command= -p "$pid" 2>/dev/null || true)
            fi
            if echo "$cwd" | grep -q "$SCRIPT_DIR"; then
                kill "$pid" 2>/dev/null
                echo -e "${GREEN}✓ ${name}已停止 (PID: $pid, 端口 $port)${NC}"
            else
                echo -e "${YELLOW}⚠ 端口 $port 被 PID $pid 占用，但不属于本项目，跳过${NC}"
            fi
        done
    fi
}

stop_service "后端服务" "$LOG_DIR/server.pid" 8080
stop_service "前端服务" "$LOG_DIR/web.pid" 3000

rm -f "$LOG_DIR/story-maker-server"

echo ""
echo -e "${GREEN}所有服务已停止${NC}"
echo "========================================"
