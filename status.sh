#!/bin/bash

# Story-Maker 服务状态检查脚本

echo "========================================"
echo "  Story-Maker 服务状态检查"
echo "========================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 检查函数
check_service() {
    local name=$1
    local port=$2
    local check_cmd=$3

    printf "${BLUE}%-15s${NC}" "$name: "

    if eval "$check_cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 运行中${NC} (端口: $port)"
        return 0
    else
        echo -e "${RED}✗ 未运行${NC} (端口: $port)"
        return 1
    fi
}

# 检查各项服务
echo "📊 核心服务:"
echo "----------------------------------------"
check_service "后端服务" "8080" "curl -s http://localhost:8080/health | grep -q ok"
check_service "前端服务" "3000" "lsof -i :3000 -t"
echo ""

echo "💾 数据库服务:"
echo "----------------------------------------"
check_service "MySQL" "3306" "mysql -u root -pSuper!123 -e 'SELECT 1' 2>/dev/null"
check_service "Redis" "6379" "redis-cli ping | grep -q PONG"
echo ""

echo "🧠 向量数据库:"
echo "----------------------------------------"
if docker ps | grep -q milvus; then
    echo -e "${GREEN}✓ Milvus 运行中${NC} (Docker)"
else
    echo -e "${YELLOW}⚠ Milvus 未运行${NC} (可选，用于动态记忆)"
fi
echo ""

echo "📋 详细信息:"
echo "----------------------------------------"

# 后端服务详细信息
if curl -s http://localhost:8080/health | grep -q ok; then
    echo -e "${GREEN}后端服务${NC}: PID $(lsof -ti :8080) | 内存 $(ps aux | grep '[g]o run' | awk '{print $6}' | head -1) KB"
fi

# 前端服务详细信息
if lsof -i :3000 -t > /dev/null 2>&1; then
    echo -e "${GREEN}前端服务${NC}: PID $(lsof -ti :3000) | Node $(node --version 2>/dev/null || echo 'N/A')"
fi

echo ""
echo "========================================"
echo "  检查完成"
echo "========================================"

# 如果有任何服务未运行，显示帮助信息
if ! curl -s http://localhost:8080/health | grep -q ok || ! lsof -i :3000 -t > /dev/null 2>&1; then
    echo ""
    echo -e "${YELLOW}💡 提示：${NC} 使用 ./start.sh 启动所有服务"
fi
