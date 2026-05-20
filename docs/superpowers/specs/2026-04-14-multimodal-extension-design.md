# story-maker 多模态扩展设计文档

> 日期：2026-04-14
> 方案：最小扩展方案（方案 A）— 沿用现有 dispatcher/executor/storage 体系

## 一、需求概述

1. 章节编辑页扩展文生音频（TTS）和文生视频能力
2. 一键导出全本小说（Word 文档，图文混排）
3. 一键导出全本音频（分章节 MP3 + 合并全本音频，ZIP 打包）

## 二、技术选型

| 能力 | 服务商 | 说明 |
|------|--------|------|
| 文生音频（TTS） | MiniMax TTS | 国产服务，中文效果优秀，支持情感语音 |
| 文生视频 | 智谱 CogVideoX / 可灵 Kling | 国产领先视频生成模型 |
| Word 导出 | unioffice (Go 库) | 纯 Go 实现，无外部依赖 |
| 音频合并 | ffmpeg | 成熟的音视频处理工具 |

## 三、数据模型扩展

### 3.1 新增 TaskType 常量

```go
// ai_task.go 新增
const (
    TaskTypeAudioGen = "audio_gen"   // 文生音频
    TaskTypeVideoGen = "video_gen"   // 文生视频
    TaskTypeExportWord  = "export_word"  // 导出 Word
    TaskTypeExportAudio = "export_audio" // 导出全本音频
)
```

### 3.2 Asset 模型扩展

现有 Asset 模型已支持 Type 字段区分资产类型。

新增类型常量：
```go
const (
    AssetTypeAudio = "audio"  // 音频资产（MP3）
    AssetTypeVideo = "video"  // 视频资产（MP4）
)
```

Asset 表新增字段：
```go
type Asset struct {
    // ... 现有字段 ...
    Duration  float64 `json:"duration"`             // 音频/视频时长（秒）
    ChapterID *uint   `json:"chapter_id,omitempty"` // 关联章节ID
}
```

### 3.3 Chapter 模型不改动

通过 Asset.ChapterID 反向关联，保持章节表简洁。查询章节多媒体资产通过 `WHERE chapter_id = ?` 实现。

## 四、后端模块设计

### 4.1 MiniMax TTS Provider

**文件：** `internal/agent/provider_minimax_tts.go`

- 实现 `AIProvider` 接口，注册到 dispatcher
- 配置项：voice_id（音色ID）、speed（语速 0.5-2.0）、emotion（情感标签）
- 调用 MiniMax T2A API，返回音频流
- 将音频流保存到 Storage，返回文件 URL

```go
type MiniMaxTTSProvider struct {
    apiKey  string
    baseURL string
    storage storage.Storage
}

func (p *MiniMaxTTSProvider) Name() string { return "minimax_tts" }

// Execute 调用 MiniMax TTS API 生成音频
// 输入：text（章节文本）、voice_id、speed、emotion
// 输出：音频文件 URL + 时长
```

### 4.2 智谱/可灵 Video Provider

**文件：** `internal/agent/provider_video.go`

- 实现 `AIProvider` 接口
- 视频生成为异步任务（通常需要 2-5 分钟）
- 流程：提交生成请求 → 获取 task_id → 轮询状态 → 下载视频 → 保存到 Storage

```go
type VideoProvider struct {
    apiKey  string
    baseURL string
    storage storage.Storage
}

func (p *VideoProvider) Name() string { return "cogvideo" }

// Execute 提交视频生成任务并轮询结果
// 输入：prompt（场景描述文本）
// 输出：视频文件 URL + 时长
```

### 4.3 AudioGenExecutor

**文件：** `internal/agent/executor_audio.go`

```go
type AudioGenExecutor struct{}

func (e *AudioGenExecutor) TaskType() string { return TaskTypeAudioGen }

// Execute 流程：
// 1. 从 AITask 中获取章节文本和 TTS 配置
// 2. 调用 MiniMax TTS Provider 生成音频
// 3. 保存音频文件到 Storage
// 4. 创建 Asset 记录（type=audio, chapter_id=xxx）
// 5. 通过 WebSocket 推送生成完成通知
```

### 4.4 VideoGenExecutor

**文件：** `internal/agent/executor_video.go`

```go
type VideoGenExecutor struct{}

func (e *VideoGenExecutor) TaskType() string { return TaskTypeVideoGen }

// Execute 流程：
// 1. 从 AITask 中获取场景描述文本
// 2. 调用 Video Provider 生成视频（异步轮询）
// 3. 保存视频文件到 Storage
// 4. 创建 Asset 记录（type=video, chapter_id=xxx）
// 5. 通过 WebSocket 推送生成完成通知
```

### 4.5 ExportService（新增）

**文件：** `internal/service/export_service.go`

职责：全本小说导出（Word + 音频）

#### ExportNovelToWord

```
流程：
1. 查询小说基本信息（标题、描述）
2. 查询所有章节（按 sort_order 排序）
3. 查询各章节关联的图片资产（scene 图片）
4. 使用 unioffice 生成 docx：
   - 封面页：小说标题 + 描述
   - 目录页：自动生成
   - 各章节：标题 + 正文 + 场景图片（图文混排）
5. 保存到 Storage，返回下载 URL
6. 通过 WebSocket 推送导出完成通知
```

#### ExportNovelAudio

```
流程：
1. 查询所有章节（按 sort_order 排序）
2. 检查各章节是否已有音频资产
   - 已有：直接使用
   - 未有：调用 TTS 逐章生成（WebSocket 推送进度："正在生成第 3/20 章音频"）
3. 收集所有章节 MP3 文件
4. 使用 ffmpeg 合并为全本音频（带章节标记）
5. 打包为 ZIP（分章节 MP3 + 全本合并音频）
6. 保存到 Storage，返回下载 URL
7. 通过 WebSocket 推送导出完成通知
```

### 4.6 ExportHandler

**文件：** `internal/handler/export_handler.go`

```go
type ExportHandler struct {
    exportService *service.ExportService
}

// ExportWord 处理 POST /api/novels/:id/export/word
// ExportAudio 处理 POST /api/novels/:id/export/audio
// Download 处理 GET /api/exports/:taskId/download
```

## 五、API 设计

### 5.1 多媒体生成

```
POST /api/ai/audio/generate
Body: {
    "chapter_id": 123,
    "voice_id": "female-yujie",
    "speed": 1.0,
    "emotion": "neutral"
}
Response: { "task_id": 456 }  // 异步任务，通过 WebSocket 推送结果

POST /api/ai/video/generate
Body: {
    "chapter_id": 123,
    "prompt": "场景描述文本（可选，默认使用章节摘要）"
}
Response: { "task_id": 789 }
```

### 5.2 资产管理

```
GET /api/chapters/:id/assets?type=audio|video
Response: {
    "assets": [
        {
            "id": 1,
            "type": "audio",
            "url": "/storage/audio/xxx.mp3",
            "duration": 180.5,
            "created_at": "2026-04-14T10:00:00Z"
        }
    ]
}

DELETE /api/assets/:id
Response: { "message": "ok" }
```

### 5.3 导出

```
POST /api/novels/:id/export/word
Response: { "task_id": 101 }  // 异步任务

POST /api/novels/:id/export/audio
Body: {
    "voice_id": "female-yujie",
    "speed": 1.0,
    "emotion": "neutral"
}
Response: { "task_id": 102 }  // 异步任务

GET /api/exports/:taskId/download
Response: 文件流（docx 或 zip）
```

## 六、前端设计

### 6.1 章节编辑页 — 多媒体面板

在 `NovelWorkshop.vue` 章节编辑区域新增"多媒体"Tab：

```
┌─────────────────────────────────────────┐
│ [正文] [版本历史] [多媒体]              │
├─────────────────────────────────────────┤
│ 🔊 音频                                │
│ ┌─────────────────────────────────────┐ │
│ │ 音色: [御姐 ▼]  语速: [1.0]  情感: [中性 ▼] │
│ │ [生成本章音频]                       │
│ └─────────────────────────────────────┘ │
│                                         │
│ 已生成音频：                            │
│ ┌─ audio_ch01.mp3 (3:20) ──── [▶] [🗑] │
│                                         │
│ 🎬 视频                                │
│ ┌─────────────────────────────────────┐ │
│ │ 场景描述: [自动使用章节摘要]         │
│ │ [生成本章视频]                       │
│ └─────────────────────────────────────┘ │
│                                         │
│ 已生成视频：                            │
│ ┌─ video_ch01.mp4 (0:15) ──── [▶] [🗑] │
└─────────────────────────────────────────┘
```

**组件拆分：**
- `ChapterMediaPanel.vue` — 多媒体面板容器
- `AudioGenerator.vue` — 音频生成配置 + 操作
- `VideoGenerator.vue` — 视频生成配置 + 操作
- `MediaAssetList.vue` — 资产列表（播放、删除）

### 6.2 小说导出入口

在小说列表页或小说详情页新增"导出"下拉菜单：

```
[导出 ▼]
├── 导出 Word 文档（图文混排）
└── 导出全本音频（ZIP）
```

点击后弹出配置对话框（音频导出需选择音色等参数），确认后提交异步任务。

**组件：**
- `ExportDialog.vue` — 导出配置对话框
- 导出进度通过 WebSocket 实时展示

### 6.3 API 调用层

**文件：** `web/src/api/media.ts`

```typescript
// 音频生成
export function generateAudio(params: AudioGenParams): Promise<TaskResponse>
// 视频生成
export function generateVideo(params: VideoGenParams): Promise<TaskResponse>
// 获取章节资产
export function getChapterAssets(chapterId: number, type?: string): Promise<AssetListResponse>
// 删除资产
export function deleteAsset(assetId: number): Promise<void>
// 导出 Word
export function exportWord(novelId: number): Promise<TaskResponse>
// 导出音频
export function exportAudio(novelId: number, params: AudioExportParams): Promise<TaskResponse>
// 下载导出文件
export function downloadExport(taskId: number): string  // 返回下载 URL
```

## 七、WebSocket 消息扩展

新增消息类型：

```json
// 音频生成完成
{ "type": "audio_gen_complete", "task_id": 456, "asset": { "id": 1, "url": "...", "duration": 180.5 } }

// 视频生成进度
{ "type": "video_gen_progress", "task_id": 789, "status": "generating", "progress": 60 }

// 视频生成完成
{ "type": "video_gen_complete", "task_id": 789, "asset": { "id": 2, "url": "...", "duration": 15.0 } }

// 导出进度
{ "type": "export_progress", "task_id": 101, "message": "正在生成第 3/20 章音频", "progress": 15 }

// 导出完成
{ "type": "export_complete", "task_id": 101, "download_url": "/api/exports/101/download" }

// 任务失败（通用）
{ "type": "task_failed", "task_id": 456, "error": "TTS 服务暂时不可用" }
```

## 八、依赖新增

### 后端 Go 依赖
- `github.com/unidoc/unioffice` — Word 文档生成
- `ffmpeg` — 系统级依赖，音频合并（通过 os/exec 调用）

### 前端无新增依赖
- 音频播放：原生 `<audio>` 标签
- 视频播放：原生 `<video>` 标签

## 九、文件清单

### 后端新增文件
1. `internal/agent/provider_minimax_tts.go` — MiniMax TTS Provider
2. `internal/agent/provider_video.go` — 智谱/可灵 Video Provider
3. `internal/agent/executor_audio.go` — 音频生成 Executor
4. `internal/agent/executor_video.go` — 视频生成 Executor
5. `internal/service/export_service.go` — 导出服务
6. `internal/handler/export_handler.go` — 导出 Handler

### 后端修改文件
7. `internal/model/ai_task.go` — 新增 TaskType 常量
8. `internal/model/asset.go` — 新增 Asset 类型常量 + Duration/ChapterID 字段
9. `internal/model/base.go` — AutoMigrate 更新
10. `internal/agent/dispatcher.go` — 注册新 Executor 和 Provider
11. `internal/router/router.go` — 注册新路由
12. `internal/handler/ai_handler.go` — 新增音频/视频生成接口

### 前端新增文件
13. `web/src/api/media.ts` — 多媒体 API 调用
14. `web/src/views/novel/ChapterMediaPanel.vue` — 多媒体面板
15. `web/src/views/novel/AudioGenerator.vue` — 音频生成组件
16. `web/src/views/novel/VideoGenerator.vue` — 视频生成组件
17. `web/src/views/novel/MediaAssetList.vue` — 资产列表组件
18. `web/src/views/novel/ExportDialog.vue` — 导出对话框

### 前端修改文件
19. `web/src/views/novel/NovelWorkshop.vue` — 集成多媒体 Tab
20. `web/src/views/novel/NovelList.vue`（或详情页） — 集成导出入口

## 十、错误处理

- TTS 生成失败：重试 1 次，仍失败则通过 WebSocket 推送错误信息
- 视频生成超时：设置 10 分钟超时，超时后标记任务失败
- 导出过程中某章节音频生成失败：跳过该章节，在 ZIP 中附带错误报告
- Word 导出图片缺失：跳过图片，仅保留文本
- ffmpeg 不可用：降级为仅提供分章节 MP3，不合并全本
