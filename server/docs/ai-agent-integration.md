# AI Agent 层集成文档

## 架构概览

AI Agent 层采用 Provider 适配器模式，统一多模型接口，通过 Dispatcher 实现任务路由和异步执行，WebSocket Hub 负责实时推送。

### 核心组件

1. **AIProvider 接口**：统一 AI 模型调用接口
2. **KimiProvider**：Kimi 模型适配器
3. **Dispatcher**：任务分发器，负责路由、异步执行、状态管理
4. **WebSocket Hub**：连接管理和消息推送
5. **AIService**：业务逻辑层，任务提交和查询
6. **APIKeyService**：API Key 加密存储和解密读取

### 数据流

```
用户请求 → AI Handler → AI Service → Dispatcher → Provider 适配层
                                                      ├── Kimi
                                                      ├── Claude（预留）
                                                      └── Copilot（预留）
                                                           ↓
                                            更新 AITask 状态 → WebSocket 推送
```

## API 接口

### 文本生成

```bash
POST /api/v1/ai/text/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "portfolio_id": 1,
  "model_name": "kimi",
  "prompt": "写一个科幻漫画的开场白"
}
```

### 图像生成

```bash
POST /api/v1/ai/image/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "portfolio_id": 1,
  "model_name": "kimi",
  "prompt": "一个未来城市的夜景，赛博朋克风格"
}
```

### 角色调整

```bash
POST /api/v1/ai/character/adjust
Authorization: Bearer <token>
Content-Type: application/json

{
  "portfolio_id": 1,
  "model_name": "kimi",
  "prompt": "调整角色发型为短发，服装为科技战甲"
}
```

### 任务查询

```bash
GET /api/v1/ai/tasks?page=1&page_size=20&portfolio_id=1
Authorization: Bearer <token>
```

### API Key 管理

创建 API Key：
```bash
POST /api/v1/apikeys
Authorization: Bearer <token>
Content-Type: application/json

{
  "provider": "kimi",
  "key_value": "sk-your-api-key"
}
```

## 配置说明

### config.yaml

```yaml
# AES-256 加密密钥（32 字节）
encrypt_key: "your-32-byte-encryption-key-here"

# AI Provider 默认 API Key
ai:
  kimi:
    api_key: "sk-your-kimi-api-key"
```

## 安全注意事项

1. **API Key 加密**：所有用户 API Key 使用 AES-256-GCM 加密存储
2. **权限校验**：任务查询和取消操作需校验 user_id
3. **Provider 白名单**：仅允许 kimi、claude、copilot
4. **WebSocket 认证**：连接时需携带有效 JWT Token

## 测试

运行测试脚本：

```bash
cd /Users/sangchenglong/tmp/Ai-curton/server
./scripts/test_api.sh
```
