#!/bin/bash

BASE_URL="http://localhost:8080/api/v1"
TOKEN="your-jwt-token-here"

echo "=== Testing AI Text Generation ==="
curl -X POST "$BASE_URL/ai/text/generate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "portfolio_id": 1,
    "model_name": "kimi",
    "prompt": "写一个科幻漫画的开场白"
  }'

echo -e "\n\n=== Testing AI Image Generation ==="
curl -X POST "$BASE_URL/ai/image/generate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "portfolio_id": 1,
    "model_name": "kimi",
    "prompt": "一个未来城市的夜景，赛博朋克风格"
  }'

echo -e "\n\n=== Testing Task List ==="
curl -X GET "$BASE_URL/ai/tasks?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"

echo -e "\n\n=== Testing API Key Creation ==="
curl -X POST "$BASE_URL/apikeys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "kimi",
    "key_value": "sk-test-key-123456"
  }'

echo -e "\n\n=== Testing API Key List ==="
curl -X GET "$BASE_URL/apikeys" \
  -H "Authorization: Bearer $TOKEN"
