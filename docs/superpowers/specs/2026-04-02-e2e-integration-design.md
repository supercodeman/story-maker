# Ai-Curton 端到端完整链路打通 - 设计文档

## 日期：2026-04-02

## 一、目标

在现有代码基础上，打通从注册登录到 AI 生成的完整链路：
**注册/登录 → 工作空间 → 作品集 → 角色管理 → AI 任务（Mock）→ WebSocket 推送 → 结果展示**

使用 Mock Provider 模拟 AI 生成，无需真实 API Key。

## 二、后端补全

### 2.1 修复 Go 版本问题
- 更新 `go.mod` 中 `toolchain` 指令为 `go1.23.10`
- 清理编译缓存

### 2.2 Mock Provider
- 新增 `internal/agent/mock.go`，实现 `AIProvider` 接口
- 文本生成：延迟 2-3 秒，返回模拟漫画脚本
- 图像生成：延迟 3-5 秒，返回占位图 URL
- 角色调整：延迟 2 秒，返回模拟调整结果
- 通过 `config.yaml` 中 `ai.default_provider: mock` 切换

### 2.3 Dispatcher 集成 Mock
- `dispatcher.go` 中注册 Mock Provider
- 配置驱动：debug 模式自动使用 Mock

### 2.4 中间件完善
- 全局 Recovery 中间件（panic 恢复）
- 请求日志中间件（记录请求路径、耗时、状态码）

### 2.5 输入验证
- 文件上传大小限制（10MB）
- MIME 类型白名单校验

## 三、前端补全

### 3.1 认证流程
- Login.vue：表单验证、错误提示、加载状态
- Register.vue：完善注册表单、密码确认

### 3.2 工作空间页面
- WorkspaceList.vue：卡片列表展示、创建弹窗
- WorkspaceDetail.vue：详情页、成员管理面板

### 3.3 作品集页面
- PortfolioList.vue：作品集网格展示、创建/编辑
- PortfolioDetail.vue：作品集详情、资源列表

### 3.4 角色管理
- CharacterList.vue：角色卡片列表
- CharacterDetail.vue：角色详情、参考图上传

### 3.5 AI 创作工坊
- StudioView.vue：任务提交表单（文本/图像/角色调整）
- 任务列表：实时状态展示
- 结果展示：文本/图像结果渲染

### 3.6 WebSocket 集成
- 完善 websocket.ts：自动重连、心跳检测
- AI Store 集成 WebSocket，实时更新任务状态

### 3.7 补充 Store
- Portfolio Store：作品集状态管理
- Character Store：角色状态管理

### 3.8 API Key 管理
- SettingsView.vue：API Key 的增删改查

### 3.9 其他
- 404 页面
- 全局加载状态

## 四、不在本次范围

- 真实 AI API 集成（等有 Key 再接入）
- LoRA 微调
- 分镜编辑器
- 作品发布/分享
- 单元测试（后续迭代）
- Swagger 文档（后续迭代）

## 五、验收标准

1. `go run ./cmd/main.go -debug` 可正常启动
2. 前端 `npm run dev` 可正常启动
3. 完整链路可走通：注册 → 登录 → 创建工作空间 → 创建作品集 → 创建角色 → 发起 AI 任务 → 查看 Mock 结果
4. WebSocket 实时推送任务状态变更
5. 所有页面交互流畅，表单验证正常
