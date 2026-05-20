// server/internal/router/router.go
package router

import (
	"context"
	"log"
	"time"

	"story-maker/server/config"
	"story-maker/server/internal/agent"
	"story-maker/server/internal/agent/tools"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/handler"
	"story-maker/server/internal/middleware"
	"story-maker/server/internal/model"
	"story-maker/server/internal/service"
	"story-maker/server/internal/storage"
	"story-maker/server/internal/vectordb"

	"github.com/gin-gonic/gin"
)

// Setup 初始化并返回配置好的 Gin 引擎
func Setup() *gin.Engine {
	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// 静态文件服务（用于访问上传的文件）
	r.Static("/uploads", "./uploads")

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 初始化文件存储（本地存储）
	store := storage.NewLocalStorage("./uploads", "http://localhost:8080/uploads")

	// 初始化 DAO
	aiTaskDAO := dao.NewAITaskDAO(model.DB)
	apiKeyDAO := dao.NewAPIKeyDAO(model.DB)
	convDAO := dao.NewConversationDAO(model.DB)
	msgDAO := dao.NewMessageDAO(model.DB)

	// 初始化 WebSocket Hub
	hub := handler.NewHub()
	go hub.Run()

	// 初始化 AI Dispatcher
	apiKeyService := service.NewAPIKeyService(apiKeyDAO)
	dispatcher := agent.NewDispatcher(apiKeyService, aiTaskDAO, hub)

	// 根据配置注册 AI Provider（统一注册所有 Provider，不再按 DefaultProvider 分支）
	providers := map[string]agent.AIProvider{
		"minimax":  agent.NewMinimaxProvider(""),
		"zhipu":    agent.NewZhipuProvider(""),
		"qwen":     agent.NewQwenProvider(""),
		"deepseek": agent.NewDeepSeekProvider(""),
		"kimi":     agent.NewKimiProvider(""),
	}
	if config.Global.AI.DefaultProvider == "mock" {
		mockProvider := agent.NewMockProvider()
		for name := range providers {
			providers[name] = mockProvider
		}
		providers["mock"] = mockProvider
	}
	for name, p := range providers {
		dispatcher.RegisterProvider(name, p)
	}
	// mock 模式外也注册 mock provider 供开发调试
	if config.Global.AI.DefaultProvider != "mock" {
		dispatcher.RegisterProvider("mock", agent.NewMockProvider())
	}

	// 注册 Tools（Function Calling 工具）
	toolRegistry := agent.NewToolRegistry()
	if config.Global.QWeather.APIKey != "" {
		toolRegistry.Register(tools.NewWeatherTool(config.Global.QWeather.APIKey))
	}
	dispatcher.SetToolRegistry(toolRegistry)

	// 注册 Qwen TTS Provider（优先主力），Qwen 文本 Key 复用
	// 未配置 Qwen Key 时，降级注册 MiniMax TTS（如果其 Key 存在）
	if config.Global.Qwen.APIKey != "" {
		ttsProvider := agent.NewQwenTTSProvider(
			config.Global.Qwen.APIKey,
			"", // DashScope 用默认 base_url
			config.Global.Qwen.TTSModel,
			store,
		)
		dispatcher.RegisterTTSProvider(ttsProvider, dao.NewAssetDAO())
	} else if config.Global.MiniMax.APIKey != "" {
		ttsProvider := agent.NewMiniMaxTTSProvider(
			config.Global.MiniMax.APIKey,
			config.Global.MiniMax.GroupID,
			config.Global.MiniMax.BaseURL,
			store,
		)
		dispatcher.RegisterTTSProvider(ttsProvider, dao.NewAssetDAO())
	}

	// 注册 CogVideo Provider（如果配置了 API Key）
	if config.Global.CogVideo.APIKey != "" {
		videoProvider := agent.NewCogVideoProvider(
			config.Global.CogVideo.APIKey,
			config.Global.CogVideo.BaseURL,
			store,
		)
		dispatcher.RegisterVideoProvider(videoProvider)
	}

	// 注册万相文生图 Provider（复用 Qwen DashScope API Key）
	if config.Global.Qwen.APIKey != "" {
		imageGenProvider := agent.NewQwenImageProvider(
			config.Global.Qwen.APIKey,
			"",
			"wanx-v1",
			store,
		)
		dispatcher.RegisterImageGenProvider(imageGenProvider, dao.NewAssetDAO())
	}

	// 注册漫剧多模态 executor（依赖 TTS/Video/ImageGen Provider 已注册）
	dispatcher.RegisterComicProviders()

	// 初始化 Workflow DAO
	workflowDAO := dao.NewAIWorkflowDAO(model.DB)

	// 初始化 Service
	promptTplService := service.NewPromptTemplateService()
	// Seed 默认模板
	_ = promptTplService.SeedDefaults()

	// 确保所有用户都是 admin + 大神写手
	_ = dao.NewUserDAO().EnsureAllUsersAdmin()

	aiService := service.NewAIService(aiTaskDAO, dispatcher)
	convService := service.NewConversationService(convDAO, msgDAO, dispatcher)
	knowledgeService := service.NewKnowledgeService(aiTaskDAO, dispatcher)
	writingStyleService := service.NewWritingStyleService()
	userStyleService := service.NewUserStyleService(dispatcher)

	// 初始化 ModelRegistry（中心化模型管理）
	modelStatusDAO := dao.NewAIModelStatusDAO(model.DB)
	modelRegistry := service.NewModelRegistryService(modelStatusDAO, apiKeyService)
	modelRegistry.SeedFromMeta(context.Background())
	modelRegistry.LoadCache(context.Background())
	modelRegistry.StartPeriodicCheck(context.Background(), 10*time.Minute, providers)
	dispatcher.SetModelChecker(modelRegistry)

	// 剧情结构模板服务
	plotStructureDAO := dao.NewPlotStructureDAO()
	plotStructureSvc := service.NewPlotStructureService(plotStructureDAO, dispatcher)
	plotStructureSvc.SeedSystemTemplates()

	// 工作流服务（需要先初始化，爆款拆解依赖它）
	workflowService := service.NewWorkflowService(workflowDAO, dispatcher, hub, writingStyleService, knowledgeService)

	// 爆款拆解服务
	hitAnalysisDAO := dao.NewHitAnalysisDAO()
	hitAnalysisSvc := service.NewHitAnalysisService(hitAnalysisDAO, workflowService)

	// 用户偏好与行为服务
	userPrefSvc := service.NewUserPreferenceService(dispatcher)
	userBehaviorSvc := service.NewUserBehaviorService(userPrefSvc)
	intentSvc := service.NewIntentService()

	memorySvc := service.NewMemoryService(dispatcher, workflowService, hub)

	// 初始化 Milvus 向量数据库（可选，未配置时降级运行）
	var milvusClient *vectordb.MilvusClient
	if config.Global.Milvus.Enabled {
		mc, err := vectordb.NewMilvusClient(&config.Global.Milvus)
		if err != nil {
			log.Printf("[milvus] 连接失败，动态记忆功能降级: %v", err)
		} else {
			milvusClient = mc
		}
	}

	// 动态记忆事实服务
	novelFactSvc := service.NewNovelFactService(milvusClient, dispatcher)

	novelService := service.NewNovelService(aiTaskDAO, dispatcher, promptTplService, knowledgeService, writingStyleService, plotStructureSvc, hitAnalysisSvc, userBehaviorSvc, intentSvc, memorySvc, novelFactSvc)

	// 联想服务
	suggestionSvc := service.NewSuggestionService(dispatcher, writingStyleService, intentSvc)

	// 注入 ModelRegistry 到各 Service（延迟注入，避免改构造函数签名）
	knowledgeService.SetModelRegistry(modelRegistry)
	userStyleService.SetModelRegistry(modelRegistry)
	plotStructureSvc.SetModelRegistry(modelRegistry)
	workflowService.SetModelRegistry(modelRegistry)
	hitAnalysisSvc.SetModelRegistry(modelRegistry)
	novelService.SetModelRegistry(modelRegistry)
	suggestionSvc.SetModelRegistry(modelRegistry)
	novelFactSvc.SetModelRegistry(modelRegistry)
	userPrefSvc.SetModelRegistry(modelRegistry)

	// 漫剧服务（需在 OnTaskCompleted 之前初始化，闭包引用 orchestrator）
	comicDramaDAO := dao.NewComicDramaDAO(model.DB)
	novelDAO := dao.NewNovelDAO()
	pipelineOrchestrator := service.NewPipelineOrchestrator(comicDramaDAO, aiTaskDAO, novelDAO, dispatcher, hub)
	comicDramaSvc := service.NewComicDramaService(comicDramaDAO)
	comicDramaSvc.SetOrchestrator(pipelineOrchestrator)
	comicDramaHandler := handler.NewComicDramaHandler(comicDramaSvc)

	// 注入任务完成回调：刷新小说 token 缓存并推送更新 + Token 计费 + 漫剧 Pipeline 推进
	tokenBillingSvc := service.NewTokenBillingService()
	dispatcher.OnTaskCompleted = func(task *model.AITask) {
		ctx := context.Background()
		if task.NovelID > 0 {
			novelService.RefreshTokenUsedForUser(ctx, task.NovelID, task.UserID, hub)
		}
		// Token 计费：高消耗任务自动扣费
		tokenBillingSvc.OnTaskCompleted(task)
		// 漫剧流水线推进
		if task.PipelineID > 0 {
			if err := pipelineOrchestrator.OnTaskDone(ctx, task); err != nil {
				log.Printf("[ComicDrama] pipeline advance failed for task %d: %v", task.ID, err)
			}
		}
	}

	// 初始化 handler
	authHandler := handler.NewAuthHandler()
	userHandler := handler.NewUserHandler()
	workspaceHandler := handler.NewWorkspaceHandler()
	portfolioHandler := handler.NewPortfolioHandler()
	characterHandler := handler.NewCharacterHandler(store)
	assetHandler := handler.NewAssetHandler(store)
	aiHandler := handler.NewAIHandler(aiService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	convHandler := handler.NewConversationHandler(convService)
	// 初始化小说搜索 Provider 链（按优先级：AI 联网搜索 → 番茄 → AI 离线兜底）
	var searchProviders []service.NovelSearchProvider
	webSearchModel := config.Global.NovelSearch.WebSearchModel
	if webSearchModel == "" {
		webSearchModel = "qwen"
	}
	aiWebSearch := service.NewAIWebSearchProvider(dispatcher, webSearchModel)
	aiWebSearch.SetModelRegistry(modelRegistry)
	searchProviders = append(searchProviders, aiWebSearch)
	searchProviders = append(searchProviders, service.NewFanqieSearchProvider())
	aiModel := config.Global.NovelSearch.AIModel
	if aiModel == "" {
		aiModel = config.Global.AI.DefaultProvider
	}
	aiSearch := service.NewAISearchProvider(dispatcher, aiModel)
	aiSearch.SetModelRegistry(modelRegistry)
	searchProviders = append(searchProviders, aiSearch)
	novelSearchService := service.NewNovelSearchService(searchProviders...)

	novelHandler := handler.NewNovelHandler(novelService, novelSearchService)
	plotStructureHandler := handler.NewPlotStructureHandler(plotStructureSvc)
	hitAnalysisHandler := handler.NewHitAnalysisHandler(hitAnalysisSvc)
	promptTplHandler := handler.NewPromptTemplateHandler(promptTplService)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeService)
	writingStyleHandler := handler.NewWritingStyleHandler(writingStyleService)
	userStyleHandler := handler.NewUserStyleHandler(userStyleService)
	workflowHandler := handler.NewWorkflowHandler(workflowService)
	overviewService := service.NewOverviewService(aiTaskDAO, dispatcher, workflowService)
	overviewHandler := handler.NewOverviewHandler(overviewService)
	behaviorHandler := handler.NewUserBehaviorHandler(userBehaviorSvc)
	suggestionHandler := handler.NewSuggestionHandler(suggestionSvc)
	writerLevelHandler := handler.NewWriterLevelHandler()
	wsHandler := handler.NewWSHandler(hub)
	factHandler := handler.NewNovelFactHandler(novelFactSvc)
	modelHandler := handler.NewModelHandler(modelRegistry, providers)

	// 世界构建服务
	worldBuildingSvc := service.NewWorldBuildingService(aiTaskDAO, dispatcher, promptTplService)
	worldBuildingHandler := handler.NewWorldBuildingHandler(worldBuildingSvc)

	// 管家多轮迭代服务
	butlerIterSvc := service.NewButlerIterativeService(aiTaskDAO, dispatcher, hub)
	butlerIterSvc.SetModelRegistry(modelRegistry)
	convService.SetModelRegistry(modelRegistry)
	butlerIterHandler := handler.NewButlerIterativeHandler(butlerIterSvc)

	// 导出服务
	exportSvc := service.NewExportService(store, hub, dispatcher.GetTTSProvider())
	exportHandler := handler.NewExportHandler(exportSvc)

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证路由（无需登录）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// 需要 JWT 认证的路由
		authorized := v1.Group("")
		authorized.Use(middleware.AuthRequired())
		{
			// 用户路由
			user := authorized.Group("/user")
			{
				user.GET("/profile", userHandler.GetProfile)
				user.PUT("/profile", userHandler.UpdateProfile)
				user.GET("/level", writerLevelHandler.GetLevelInfo)
				user.POST("/level/purchase", writerLevelHandler.PurchaseUpgrade)
				user.PUT("/view-mode", writerLevelHandler.UpdateViewMode)
			}

			// Workspace 路由
			workspaces := authorized.Group("/workspaces")
			{
				workspaces.GET("", workspaceHandler.List)
				workspaces.POST("", workspaceHandler.Create)
				workspaces.GET("/:id", workspaceHandler.Get)
				workspaces.PUT("/:id", workspaceHandler.Update)
				workspaces.DELETE("/:id", workspaceHandler.Delete)
				workspaces.GET("/:id/members", workspaceHandler.GetMembers)
				workspaces.POST("/:id/members", workspaceHandler.AddMember)
				workspaces.DELETE("/:id/members/:user_id", workspaceHandler.RemoveMember)
			}

			// Portfolio 路由
			portfolios := authorized.Group("/portfolios")
			{
				portfolios.GET("", portfolioHandler.List)
				portfolios.POST("", portfolioHandler.Create)
				portfolios.GET("/:id", portfolioHandler.Get)
				portfolios.PUT("/:id", portfolioHandler.Update)
				portfolios.DELETE("/:id", portfolioHandler.Delete)

				// Portfolio 下的 Character 路由
				portfolios.GET("/:id/characters", characterHandler.List)
				portfolios.POST("/:id/characters", characterHandler.Create)

				// Portfolio 下的 Asset 路由
				portfolios.GET("/:id/assets", assetHandler.List)
			}

			// Character 路由
			characters := authorized.Group("/characters")
			{
				characters.GET("/:id", characterHandler.Get)
				characters.PUT("/:id", characterHandler.Update)
				characters.DELETE("/:id", characterHandler.Delete)
				characters.POST("/:id/reference", characterHandler.UploadReference)
			}

			// Asset 路由
			assets := authorized.Group("/assets")
			{
				assets.POST("/upload", assetHandler.Upload)
				assets.GET("/:id", assetHandler.Get)
				assets.DELETE("/:id", assetHandler.Delete)
				assets.PUT("/:id/set-character-ref", assetHandler.SetCharacterRef)
				assets.PUT("/:id/unset-character-ref", assetHandler.UnsetCharacterRef)
			}

			// AI 路由
			ai := authorized.Group("/ai")
			{
				ai.POST("/text/generate", aiHandler.GenerateText)
				ai.POST("/image/generate", aiHandler.GenerateImage)
				ai.POST("/character/adjust", aiHandler.AdjustCharacter)
				ai.POST("/audio/generate", aiHandler.GenerateAudio)
				ai.POST("/video/generate", aiHandler.GenerateVideo)
				ai.GET("/tasks", aiHandler.ListTasks)
				ai.GET("/tasks/:id", aiHandler.GetTask)
				ai.DELETE("/tasks/:id", aiHandler.CancelTask)
			}

			// API Key 路由
			apikeys := authorized.Group("/apikeys")
			{
				apikeys.GET("", apiKeyHandler.ListKeys)
				apikeys.POST("", apiKeyHandler.CreateKey)
				apikeys.PUT("/:id", apiKeyHandler.UpdateKey)
				apikeys.DELETE("/:id", apiKeyHandler.DeleteKey)
			}

			// 模型管理路由
			models := authorized.Group("/models")
			{
				models.GET("/available", modelHandler.GetAvailableModels)
				models.GET("/status", modelHandler.GetModelStatus)
				models.POST("/check", modelHandler.TriggerHealthCheck)
				models.POST("", modelHandler.AddModel)
				models.DELETE("/:id", modelHandler.DeleteModel)
				models.POST("/test", modelHandler.TestModel)
				models.PUT("/:id/priority", modelHandler.UpdatePriority)
			}

			// Conversation 会话路由
			conversations := authorized.Group("/conversations")
			{
				conversations.POST("", convHandler.CreateConversation)
				conversations.GET("", convHandler.ListConversations)
				conversations.GET("/:id", convHandler.GetConversation)
				conversations.POST("/:id/messages", convHandler.SendMessage)
				conversations.DELETE("/:id", convHandler.ArchiveConversation)
			}

			// UserStyle 用户风格库路由
			userStyles := authorized.Group("/user-styles")
			{
				userStyles.GET("", userStyleHandler.List)
				userStyles.POST("", userStyleHandler.Create)
				userStyles.POST("/ai-generate", userStyleHandler.AIGenerate)
				userStyles.PUT("/:id", userStyleHandler.Update)
				userStyles.DELETE("/:id", userStyleHandler.Delete)
			}

			// Novel 小说工坊路由
			novels := authorized.Group("/novels")
			{
				novels.POST("", novelHandler.CreateNovel)
				novels.GET("", novelHandler.ListNovels)
				novels.POST("/repair-butler", novelHandler.RepairButlerLinks)
				novels.GET("/:id", novelHandler.GetNovel)
				novels.PUT("/:id", novelHandler.UpdateNovel)
				novels.DELETE("/:id", novelHandler.DeleteNovel)
				novels.POST("/:id/chapters", novelHandler.CreateChapter)
				novels.GET("/:id/chapters", novelHandler.ListChapters)
				novels.PUT("/:id/chapters/reorder", novelHandler.ReorderChapters)
				novels.POST("/:id/expand-chapters", novelHandler.ExpandChapters)
				novels.GET("/:id/token-usage", novelHandler.GetTokenUsage)
				novels.PUT("/:id/token-budget", novelHandler.UpdateTokenBudget)
				novels.POST("/:id/suggest", suggestionHandler.Suggest)
				novels.POST("/:id/facts/cold-start", novelHandler.TriggerFactColdStart)

				// 导出路由
				novels.POST("/:id/export/word", exportHandler.ExportWord)
				novels.POST("/:id/export/audio", exportHandler.ExportAudio)

				// Facts 记忆事实路由（挂在 novels/:id 下）
				novels.GET("/:id/facts", factHandler.List)
				novels.POST("/:id/facts", factHandler.Create)

				// Prompt 模板路由
				novels.GET("/:id/prompt-templates", promptTplHandler.ListTemplates)
				novels.PUT("/:id/prompt-templates", promptTplHandler.UpsertTemplate)
				novels.DELETE("/:id/prompt-templates/:tid", promptTplHandler.DeleteTemplate)
				novels.POST("/:id/prompt-templates/preview", promptTplHandler.PreviewTemplate)

				// Knowledge 知识库路由（挂在 novels/:id 下）
				novels.GET("/:id/knowledge", knowledgeHandler.List)
				novels.POST("/:id/knowledge", knowledgeHandler.Create)
				novels.GET("/:id/knowledge/search", knowledgeHandler.Search)
				novels.POST("/:id/knowledge/batch-confirm", knowledgeHandler.BatchConfirm)
				novels.POST("/:id/knowledge/extract", knowledgeHandler.Extract)
				novels.POST("/:id/knowledge/parse-extract", knowledgeHandler.ParseExtract)

				// WritingStyle 写作风格路由（挂在 novels/:id 下）
				novels.GET("/:id/writing-style", writingStyleHandler.GetStyle)
				novels.PUT("/:id/writing-style", writingStyleHandler.UpsertStyle)
				novels.DELETE("/:id/writing-style", writingStyleHandler.DeleteStyle)
				novels.GET("/:id/scene-presets", writingStyleHandler.ListPresets)
				novels.POST("/:id/scene-presets", writingStyleHandler.CreatePreset)
				novels.PUT("/:id/scene-presets/:pid", writingStyleHandler.UpdatePreset)
				novels.DELETE("/:id/scene-presets/:pid", writingStyleHandler.DeletePreset)

				// 小说绑定/解绑用户风格
				novels.PUT("/:id/bind-style", userStyleHandler.BindStyle)
				novels.DELETE("/:id/bind-style", userStyleHandler.UnbindStyle)

				// Overview 总览路由（挂在 novels/:id/overview 下）
				novelOverview := novels.Group("/:id/overview")
				{
					novelOverview.GET("", overviewHandler.GetOverview)
					novelOverview.POST("/relations", overviewHandler.CreateRelation)
					novelOverview.PUT("/relations/:rid", overviewHandler.UpdateRelation)
					novelOverview.DELETE("/relations/:rid", overviewHandler.DeleteRelation)
					novelOverview.POST("/extract", overviewHandler.Extract)
					novelOverview.POST("/extract/parse", overviewHandler.ParseExtract)
					novelOverview.POST("/revision", overviewHandler.SubmitRevision)
					novelOverview.POST("/revision/execute", overviewHandler.ExecuteRevision)
				}
			}

			// Facts 记忆事实独立路由（不依赖 novel_id 前缀）
			facts := authorized.Group("/facts")
			{
				facts.GET("/:fid", factHandler.Get)
				facts.PUT("/:fid", factHandler.Update)
				facts.DELETE("/:fid", factHandler.Delete)
			}

			// 世界构建与规划路由
			wb := authorized.Group("/world-building")
			{
				wb.POST("/start", worldBuildingHandler.StartPhase)
				wb.POST("/review", worldBuildingHandler.ReviewResult)
				wb.POST("/process-review", worldBuildingHandler.ProcessReview)
				wb.POST("/optimize", worldBuildingHandler.Optimize)
				wb.POST("/accept", worldBuildingHandler.AcceptResult)
				wb.GET("/status", worldBuildingHandler.GetStatus)
				wb.GET("/summary", worldBuildingHandler.GetSummary)
			}

			// Chapter 章节路由
			chapters := authorized.Group("/chapters")
			{
				chapters.PUT("/:id", novelHandler.UpdateChapter)
				chapters.DELETE("/:id", novelHandler.DeleteChapter)
				chapters.POST("/:id/ai", novelHandler.ChapterAIAction)
				chapters.POST("/:id/accept", novelHandler.AcceptAIResult)
				chapters.POST("/:id/reject", novelHandler.RejectAIResult)
				chapters.GET("/:id/versions", novelHandler.ListVersions)
				chapters.POST("/:id/revert", novelHandler.RevertToVersion)
				chapters.GET("/:id/assets", exportHandler.GetChapterAssets)
			}

			// 用户行为上报路由
			behavior := authorized.Group("/behavior")
			{
				behavior.POST("/events", behaviorHandler.RecordEvent)
			}

			// Outline 大纲生成路由
			outline := authorized.Group("/outline")
			{
				outline.GET("/search-novels", novelHandler.SearchNovels)
				outline.POST("/generate", novelHandler.GenerateOutline)
				outline.POST("/adopt", novelHandler.AdoptOutline)
				outline.POST("/chapter-ai", novelHandler.OutlineChapterAI)
				outline.POST("/butler-iterate", butlerIterHandler.StartIteration)
				outline.GET("/butler-iterate/:iteration_id", butlerIterHandler.GetIterationStatus)
			}

			// PlotTemplate 剧情结构模板路由
			plotTemplates := authorized.Group("/plot-templates")
			{
				plotTemplates.GET("", plotStructureHandler.List)
				plotTemplates.GET("/:id", plotStructureHandler.Get)
				plotTemplates.POST("", plotStructureHandler.Create)
				plotTemplates.POST("/ai-generate", plotStructureHandler.AIGenerate)
				plotTemplates.PUT("/:id", plotStructureHandler.Update)
				plotTemplates.DELETE("/:id", plotStructureHandler.Delete)
			}

			// HitAnalysis 爆款拆解路由
			hitAnalysis := authorized.Group("/hit-analysis")
			{
				hitAnalysis.POST("", hitAnalysisHandler.Submit)
				hitAnalysis.GET("", hitAnalysisHandler.List)
				hitAnalysis.GET("/:id", hitAnalysisHandler.Get)
				hitAnalysis.DELETE("/:id", hitAnalysisHandler.Delete)
			}

			// 漫剧路由
			comicDrama := authorized.Group("/comic-drama")
			{
				comicDrama.POST("", comicDramaHandler.Create)
				comicDrama.GET("", comicDramaHandler.List)
				comicDrama.GET("/:id", comicDramaHandler.Get)
				comicDrama.DELETE("/:id", comicDramaHandler.Delete)
				comicDrama.PUT("/:id/config", comicDramaHandler.UpdateConfig)
				comicDrama.POST("/:id/start", comicDramaHandler.StartPipeline)
				comicDrama.POST("/:id/advance", comicDramaHandler.AdvanceStage)
				comicDrama.POST("/:id/retry", comicDramaHandler.RetryFailed)
				comicDrama.GET("/:id/script", comicDramaHandler.GetScript)
				comicDrama.PUT("/:id/script", comicDramaHandler.UpdateScript)
				comicDrama.POST("/:id/script/approve", comicDramaHandler.ApproveScript)
				comicDrama.GET("/:id/storyboard", comicDramaHandler.GetStoryboard)
				comicDrama.PUT("/storyboard/:shot_id", comicDramaHandler.UpdateStoryboardShot)
				comicDrama.POST("/:id/storyboard/approve", comicDramaHandler.ApproveStoryboard)
				comicDrama.GET("/:id/characters", comicDramaHandler.GetCharacters)
				comicDrama.POST("/characters/:char_id/regenerate", comicDramaHandler.RegenerateCharacter)
				comicDrama.POST("/:id/characters/approve", comicDramaHandler.ApproveCharacters)
				comicDrama.GET("/:id/segments", comicDramaHandler.GetSegments)
				comicDrama.POST("/:id/compose", comicDramaHandler.TriggerCompose)
				comicDrama.GET("/:id/download", comicDramaHandler.GetDownloadURL)
			}

			// Workflow 工作流路由
			workflows := authorized.Group("/ai/workflows")
			{
				workflows.GET("/active", workflowHandler.ListActive) // 放在 /:id 之前避免路由冲突
				workflows.POST("/submit", workflowHandler.Submit)
				workflows.GET("/:id", workflowHandler.Get)
				workflows.DELETE("/:id", workflowHandler.Cancel)
			}

			// 记忆管理路由
			marketSvc := service.NewMarketService()
			walletSvc := service.NewWalletService()
			genreSvc := service.NewGenreService()
			genreSvc.SeedDefaults()
			memoryHandler := handler.NewMemoryHandler(memorySvc, marketSvc, genreSvc)
			marketHandler := handler.NewMarketHandler(marketSvc)
			walletHandler := handler.NewWalletHandler(walletSvc)
			genreHandler := handler.NewGenreHandler(genreSvc)

			// 赛道路由（公开）
			genres := authorized.Group("/genres")
			{
				genres.GET("", genreHandler.ListTree)
				genres.GET("/:id", genreHandler.Get)
			}

			memories := authorized.Group("/memories")
			{
				memories.POST("", memoryHandler.Create)
				memories.GET("", memoryHandler.List)
				memories.GET("/accessible", memoryHandler.ListAccessible)
				memories.GET("/:mid", memoryHandler.Get)
				memories.PUT("/:mid", memoryHandler.Update)
				memories.DELETE("/:mid", memoryHandler.Delete)
				memories.POST("/:mid/refine", memoryHandler.Refine)
				memories.POST("/:mid/publish", memoryHandler.Publish)
				memories.POST("/:mid/archive", memoryHandler.Archive)
				memories.POST("/:mid/preview", memoryHandler.Preview)
			}

			// 小说-记忆绑定路由（挂在 novels 下）
			authorized.GET("/novels/:id/memory-bindings", memoryHandler.ListBindings)
			authorized.PUT("/novels/:id/memory-bindings", memoryHandler.SetBindings)

			// 记忆市场路由
			market := authorized.Group("/market/memories")
			{
				market.GET("", marketHandler.ListMemories)
				market.GET("/:mid", marketHandler.GetMemory)
				market.POST("/:mid/buy", marketHandler.Buy)
				market.GET("/:mid/reviews", marketHandler.ListReviews)
				market.POST("/:mid/reviews", marketHandler.SubmitReview)
			}

			// 钱包路由
			wallet := authorized.Group("/wallet")
			{
				wallet.GET("", walletHandler.GetWallet)
				wallet.GET("/transactions", walletHandler.ListTransactions)
				wallet.POST("/recharge", walletHandler.Recharge)
			}

			// 管理员路由
			admin := authorized.Group("/admin")
			admin.Use(middleware.RequireAdmin())
			{
				admin.GET("/memories/reviewing", memoryHandler.AdminListReviewing)
				admin.POST("/memories/:mid/approve", memoryHandler.AdminApprove)
				admin.POST("/memories/:mid/reject", memoryHandler.AdminReject)

				// 赛道管理
				admin.POST("/genres", genreHandler.AdminCreate)
				admin.PUT("/genres/:id", genreHandler.AdminUpdate)
				admin.DELETE("/genres/:id", genreHandler.AdminDelete)

				// 用户管理
				admin.GET("/users", userHandler.AdminListUsers)
				admin.PUT("/users/:uid/role", userHandler.AdminUpdateRole)
				admin.PUT("/users/:uid/writer-level", writerLevelHandler.AdminSetWriterLevel)
			}
		}
	}

	// WebSocket 路由
	r.GET("/ws", wsHandler.HandleWebSocket)

	// 启动时恢复卡住的工作流和记忆
	go func() {
		log.Println("[recovery] recovering stale workflows...")
		workflowService.RecoverStaleWorkflows()
		log.Println("[recovery] recovering stale memories...")
		memorySvc.RecoverStaleMemories()
		log.Println("[recovery] done")
	}()

	return r
}
