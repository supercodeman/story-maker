---
name: Ai-curton 编程规范
description: Ai-curton 项目的编程规范与约定，涵盖 Go 后端和 Vue 前端的分层架构、命名约定、错误处理、安全规范等。开发新功能或修改代码时自动触发。
version: 1.0.0
---

## 一、项目架构概览

```
server/                          # Go 后端（Gin + GORM + MySQL + Redis）
├── cmd/main.go                  # 入口
├── config/                      # 配置加载（Viper + config.yaml）
├── internal/
│   ├── handler/                 # HTTP 请求处理层
│   ├── service/                 # 业务逻辑层
│   ├── dao/                     # 数据访问层（GORM）
│   ├── model/                   # 数据模型 + DB 初始化
│   ├── middleware/              # 中间件（JWT、CORS、日志、Recovery）
│   ├── router/router.go         # 路由注册 + 依赖注入
│   ├── agent/                   # AI 任务分发与执行
│   │   ├── orchestrator/        # DAG 工作流引擎
│   │   └── tools/               # AI 工具注册
│   ├── storage/                 # 文件存储
│   └── util/                    # 工具函数
web/                             # Vue 3 + TypeScript + Pinia + Tailwind
├── src/
│   ├── api/                     # API 调用层（axios 封装）
│   ├── store/                   # Pinia 状态管理
│   ├── views/                   # 页面组件
│   ├── components/              # 可复用组件
│   ├── router/                  # Vue Router
│   └── utils/                   # 工具函数
```

## 二、Go 后端规范

### 2.1 分层职责（严格遵守，禁止跨层调用）

| 层级 | 职责 | 禁止 |
|------|------|------|
| **Handler** | 参数绑定（`ShouldBindJSON`/`Query`/`Param`）、调用 Service、统一响应 | 禁止直接操作 DB、禁止包含业务逻辑 |
| **Service** | 业务逻辑、权限校验、参数验证、事务管理、调用 DAO | 禁止直接操作 `*gin.Context` |
| **DAO** | 数据库 CRUD、查询构建 | 禁止包含业务逻辑、禁止直接返回 HTTP 响应 |
| **Model** | 结构体定义、GORM 标签、常量/枚举 | 禁止包含方法逻辑（纯数据结构） |

### 2.2 命名约定

```go
// 文件名：snake_case
// novel.go, prompt_template.go, executor_text.go

// 结构体：PascalCase，带中文注释说明职责
// NovelDAO 小说数据访问层
type NovelDAO struct { ... }

// 构造函数：New + 结构体名
func NewNovelDAO() *NovelDAO { ... }

// 方法名：动词开头，PascalCase
func (d *NovelDAO) CreateNovel(novel *model.Novel) error { ... }
func (d *NovelDAO) GetNovel(id uint) (*model.Novel, error) { ... }
func (d *NovelDAO) ListNovelsByPortfolio(portfolioID uint) ([]model.Novel, error) { ... }

// 常量：PascalCase，按语义分组
const (
    NovelStatusDraft     = "draft"
    NovelStatusWriting   = "writing"
    NovelStatusCompleted = "completed"
)

// 白名单 map：Valid + 名词 + 复数
var ValidNovelStatuses = map[string]bool{ ... }
```

### 2.3 Handler 层模式

```go
// 每个 Handler 文件头部声明结构体和构造函数
type NovelHandler struct {
    svc *service.NovelService
}

func NewNovelHandler(svc *service.NovelService) *NovelHandler {
    return &NovelHandler{svc: svc}
}

// 方法注释格式：功能说明 + HTTP 方法 + 路径
// CreateNovel 创建小说
// POST /api/v1/novels
func (h *NovelHandler) CreateNovel(c *gin.Context) {
    userID := c.GetUint("user_id")  // 从 JWT 中间件获取

    var req service.CreateNovelRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        BadRequest(c, err.Error())
        return
    }

    result, err := h.svc.CreateNovel(userID, &req)
    if err != nil {
        InternalError(c, err.Error())
        return
    }

    Success(c, result)
}
```

### 2.4 统一响应格式

```go
// 使用 handler/response.go 中的工具函数，禁止直接 c.JSON
Success(c, data)                    // { "code": 0, "message": "success", "data": ... }
SuccessWithMessage(c, "msg", data)  // { "code": 0, "message": "msg", "data": ... }
BadRequest(c, "error msg")          // 400
Unauthorized(c, "error msg")        // 401
InternalError(c, "error msg")       // 500
```

### 2.5 DAO 层模式

```go
// DAO 通过全局 model.DB 初始化
type NovelDAO struct {
    db *gorm.DB
}

func NewNovelDAO() *NovelDAO {
    return &NovelDAO{db: model.DB}
}

// 查询必须使用参数化（GORM 自动处理），禁止字符串拼接 SQL
// 正确 ✓
db.Where("portfolio_id = ?", portfolioID).Find(&novels)

// 错误 ✗ — SQL 注入风险
db.Where(fmt.Sprintf("portfolio_id = %d", portfolioID)).Find(&novels)

// ORDER BY 等不可参数化的字段必须白名单校验
allowedFields := map[string]bool{"created_at": true, "updated_at": true, "title": true}
if !allowedFields[orderField] {
    orderField = "created_at" // 回退默认值
}
```

### 2.6 错误处理

```go
// Service 层：使用 fmt.Errorf 包装错误，保留上下文
if err := s.dao.CreateNovel(novel); err != nil {
    return nil, fmt.Errorf("create novel failed: %w", err)
}

// 非关键路径可用 _ 忽略（如日志、通知），但需注释说明
_ = s.workflowDAO.Update(ctx, workflow) // 状态更新失败不阻塞主流程

// 禁止在 Service/DAO 层使用 panic，仅在 main 初始化阶段允许
```

### 2.7 注释规范

```go
// 每个文件首行：文件路径注释
// server/internal/dao/novel.go

// 导出的结构体/函数必须有中文注释
// NovelDAO 小说数据访问层
type NovelDAO struct { ... }

// 功能分区使用分隔注释
// ========== Novel CRUD ==========
// ========== Chapter CRUD ==========

// 关键逻辑处添加行内注释
models := append([]string{node.ModelName}, node.FallbackModels...) // 构建尝试列表：主模型 + 降级模型
```

### 2.8 依赖注入

```go
// 在 router/router.go 中统一初始化，手动注入依赖
// DAO → Service → Handler 的顺序
func Setup() *gin.Engine {
    // 初始化 DAO
    novelDAO := dao.NewNovelDAO()
    // 初始化 Service
    novelSvc := service.NewNovelService(novelDAO)
    // 初始化 Handler
    novelHandler := handler.NewNovelHandler(novelSvc)
    // 注册路由
    v1.POST("/novels", novelHandler.CreateNovel)
}
```

### 2.9 AI Agent 模块规范

```go
// Executor 实现 Execute(ctx, *ExecContext) (interface{}, error) 接口
// 返回值统一为 map[string]interface{}{"content": "..."}

// DAG 节点通过 Node.MaxTokens 控制输出长度，0 表示使用 executor 默认值
// 通过 task.History JSON 传递节点级配置给 executor

// 模型降级：主模型 + FallbackModels 按顺序尝试
// 仅 isRetryableError 返回 true 时才降级，其他错误直接返回
```

## 三、Vue 前端规范

### 3.1 API 层

```typescript
// api/*.ts：类型定义 + API 函数对象
// 类型用 interface，字段名与后端 JSON tag 一致（snake_case）
export interface Novel {
  id: number
  portfolio_id: number
  title: string
  // ...
}

// API 函数用对象字面量组织
export const novelApi = {
  list(portfolioId: number) { return request.get('/novels', { params: { portfolio_id: portfolioId } }) },
  create(data: Partial<Novel>) { return request.post('/novels', data) },
}
```

### 3.2 Store 层（Pinia）

```typescript
// store/*.ts：Composition API 风格（setup store）
export const useNovelStore = defineStore('novel', () => {
  // 状态用 ref
  const novels = ref<Novel[]>([])
  const loading = ref(false)

  // 异步操作包裹 loading 状态
  async function fetchNovels(portfolioId: number) {
    loading.value = true
    try {
      const data = await novelApi.list(portfolioId)
      novels.value = Array.isArray(data) ? data : []
    } finally {
      loading.value = false
    }
  }

  return { novels, loading, fetchNovels }
})
```

### 3.3 组件规范

```
views/         → 页面级组件（路由对应），如 views/novel/NovelWorkshop.vue
components/    → 可复用组件
```

- 使用 `<script setup lang="ts">` + Composition API
- 样式使用 Tailwind CSS 工具类为主
- 类型从 `api/*.ts` 导入，store 层 re-export

## 四、安全规范

### 4.1 SQL 注入防护

- [ ] DAO 层所有查询使用 GORM 参数化查询（`Where("col = ?", val)`）
- [ ] ORDER BY、GROUP BY 等不可参数化的字段必须白名单校验
- [ ] 排序方向限定为 `asc` / `desc` 枚举

### 4.2 认证与授权

- [ ] 所有需认证的路由使用 `middleware.JWTAuth()` 中间件
- [ ] Service 层校验资源归属（`userID` 匹配）
- [ ] API Key 使用 AES-256 加密存储

### 4.3 输入校验

- [ ] Handler 层使用 `ShouldBindJSON` 绑定并校验请求体
- [ ] Service 层对业务参数做二次校验（白名单、范围检查）
- [ ] 禁止将用户输入直接拼接到 SQL、Shell 命令、模板中

## 五、代码质量红线

- [ ] 单个函数不超过 80 行，单个文件不超过 500 行（超出则拆分）
- [ ] 遵循 DRY：相同逻辑出现 3 次以上必须抽取
- [ ] 遵循 KISS：优先选择简单直接的实现
- [ ] 遵循 YAGNI：不为假设的未来需求编码
- [ ] 新增代码必须通过 `go build ./...` 编译
- [ ] 错误信息使用英文，注释和文档使用中文
