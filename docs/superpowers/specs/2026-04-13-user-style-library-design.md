# 用户风格库设计文档

## 概述

将章节编辑页面的"我的风格"模块提升为全局功能，在侧边栏新增"我的风格"入口，进入独立的风格库管理页面。支持用户自定义风格模板（命名 + 配置 + prompt），支持通过 AI 根据描述自动生成风格配置。章节编辑页面支持将小说绑定用户风格库中的风格。

## 一、数据模型

### 1.1 新增 UserStyle 表（用户级风格模板库）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_id | uint | 所属用户，索引 |
| name | string(100) | 风格名称，如"我的武侠风" |
| description | string(500) | 风格描述（也用于 AI 生成时的输入） |
| narrative_voice | string(30) | 叙事视角（枚举：first/third_limited/third_omniscient/multi_pov） |
| tone | string(30) | 文风调性（枚举：serious/humorous/lyrical/sharp/warm/neutral） |
| language_level | string(30) | 语言风格（枚举：literary/standard/colloquial/web_novel） |
| reference_authors | string(500) | 参考作家 |
| forbidden_patterns | text | 禁用句式 |
| custom_rules | text | 自定义规范 |
| custom_prompt | text | 用户自定义 prompt（核心新增字段） |
| is_ai_generated | bool | 是否由 AI 生成 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### 1.2 扩展现有 WritingStyle 表

新增字段：
- `bound_user_style_id` (uint, nullable)：引用 UserStyle。绑定后 AI 生成优先使用 UserStyle 配置；未绑定时使用小说自身 WritingStyle 配置。

### 1.3 与现有数据的关系

- 现有 WritingStyle（小说级一对一）和 ScenePreset（小说级一对多）逻辑完全保留
- UserStyle 是独立的用户级实体，与 Novel 无直接关联
- 小说通过 WritingStyle.bound_user_style_id 引用用户风格
- 绑定用户风格后，小说自身的 WritingStyle 字段仍保留（解绑后可恢复）

## 二、后端 API

### 2.1 UserStyle CRUD

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/user-styles` | 获取当前用户的风格列表 |
| POST | `/user-styles` | 创建风格 |
| PUT | `/user-styles/:id` | 更新风格 |
| DELETE | `/user-styles/:id` | 删除风格（需校验归属） |

### 2.2 AI 生成风格

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/user-styles/ai-generate` | 传入描述，AI 返回完整风格配置 |

请求体：
```json
{ "description": "金庸风格的武侠小说，古风韵味浓厚" }
```

响应：返回 AI 生成的完整风格字段（narrative_voice、tone、language_level、reference_authors、forbidden_patterns、custom_rules、custom_prompt），用户预览确认后保存。

### 2.3 小说绑定/解绑用户风格

| 方法 | 路径 | 说明 |
|------|------|------|
| PUT | `/novels/:novelId/bind-style` | 绑定，body: `{ "user_style_id": 123 }` |
| DELETE | `/novels/:novelId/bind-style` | 解绑，回退到小说自身风格 |

### 2.4 后端分层

遵循现有 Model → DAO → Service → Handler 模式：
- `model/user_style.go`：UserStyle 结构体 + 枚举复用
- `dao/user_style.go`：CRUD 操作
- `service/user_style.go`：业务逻辑 + AI 生成调用
- `handler/user_style.go`：HTTP 处理 + 参数校验
- `router/router.go`：注册路由

## 三、前端架构

### 3.1 新增文件

| 文件 | 职责 |
|------|------|
| `web/src/api/user_style.ts` | UserStyle API 接口 + 类型定义 |
| `web/src/store/user_style.ts` | Pinia store，管理用户风格列表状态 |
| `web/src/views/style/UserStyleList.vue` | 风格库管理页面 |

### 3.2 修改文件

| 文件 | 改动 |
|------|------|
| `AppSidebar.vue` | 新增"我的风格"导航项 |
| `router/index.ts` | 新增 `/styles` 路由 |
| `NovelWorkshop.vue` | 右栏"写作风格"区域顶部新增绑定用户风格的下拉选择器 |
| `WritingStylePanel.vue` | 新增"从风格库导入"按钮 |

### 3.3 UserStyleList.vue 页面结构

- 顶部：标题 + "新建风格"按钮
- 风格卡片列表：每张卡片显示名称、描述、关键标签（调性/视角等）、操作按钮（编辑/删除）
- 新建/编辑对话框：
  - 上半部分：名称、描述输入 + "AI 生成"按钮
  - 下半部分：与现有 WritingStylePanel 相同的表单字段 + custom_prompt 文本域
  - 预设模板快速选择（复用现有 6 个模板）

### 3.4 章节编辑页绑定交互

- 在 NovelWorkshop 右栏"写作风格"区域顶部，新增下拉选择器：
  - 选项来源：用户风格库列表
  - 选中后调用 bind-style API 绑定到当前小说
  - 显示"已绑定：xxx风格"状态，支持解绑
- 绑定用户风格后，下方 WritingStylePanel 显示为只读预览（数据来自 UserStyle）
- 解绑后恢复为小说自身可编辑的风格配置

## 四、AI 生成 Prompt 流程

### 4.1 交互流程

1. 用户在风格编辑对话框中填写描述
2. 点击"AI 生成"按钮
3. 后端将描述发送给 AI 模型，使用固定系统 prompt 引导输出结构化配置
4. 返回结果自动填充到表单各字段，用户预览、微调后保存

### 4.2 后端实现

- Service 层构造 prompt，要求 AI 以 JSON 格式返回所有风格字段
- 复用现有 AI 模型分发能力（智谱/通义/Kimi），默认使用低成本模型
- 对 AI 返回的枚举字段做白名单校验，不合法则回退默认值

### 4.3 系统 Prompt

```
你是一个写作风格分析专家。根据用户的描述，生成一套完整的写作风格配置。

输出严格 JSON 格式，字段如下：
- narrative_voice: 叙事视角，枚举值 first/third_limited/third_omniscient/multi_pov
- tone: 文风调性，枚举值 serious/humorous/lyrical/sharp/warm/neutral
- language_level: 语言风格，枚举值 literary/standard/colloquial/web_novel
- reference_authors: 参考作家（字符串）
- forbidden_patterns: 禁用句式（字符串）
- custom_rules: 自定义规范（字符串）
- custom_prompt: 完整的写作风格指令 prompt（字符串，供 AI 写作时直接使用）

仅输出 JSON，不要其他内容。
```

## 五、安全性

- UserStyle 的所有操作校验 user_id 归属
- 绑定/解绑操作校验小说归属（novel.user_id == 当前用户）
- 枚举字段使用白名单校验（复用现有 ValidNarrativeVoices、ValidTones、ValidLanguageLevels）
- AI 生成返回的枚举值做白名单过滤
- description、custom_prompt 等文本字段做长度限制
