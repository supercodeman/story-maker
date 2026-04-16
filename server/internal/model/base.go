// server/internal/model/base.go
package model

import (
	"fmt"
	"time"

	"ai-curton/server/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库实例
var DB *gorm.DB

// InitDB 初始化 MySQL 连接并执行自动迁移
func InitDB(cfg *config.DatabaseConfig) error {
	// 根据运行模式设置日志级别
	logLevel := logger.Info
	if config.Global.Server.Mode == "release" {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 连接池配置
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	DB = db

	// 自动迁移所有模型
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	return nil
}

// autoMigrate 执行数据库表自动迁移
func autoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&Workspace{},
		&WorkspaceMember{},
		&Portfolio{},
		&Character{},
		&Asset{},
		&AITask{},
		&Conversation{},
		&Message{},
		&Novel{},
		&Chapter{},
		&ChapterVersion{},
		&PromptTemplate{},
		&AIWorkflow{},
		&AIWorkflowNode{},
		&NovelKnowledge{},
		&NovelCharacterRelation{},
		&WritingStyle{},
		&ScenePreset{},
		&UserStyle{},
		// 记忆系统
		&WritingMemory{},
		&WritingMemoryVersion{},
		&MemoryEmbedding{},
		&NovelMemoryBinding{},
		// 钱包与交易
		&UserWallet{},
		&WalletTransaction{},
		&MemoryOrder{},
		&MemoryLicense{},
		&MemoryReview{},
		// 剧情结构模板 & 爆款拆解
		&PlotStructureTemplate{},
		&HitAnalysis{},
		// 赛道分类
		&Genre{},
		&MemoryGenre{},
		// 用户行为与偏好
		&UserBehaviorEvent{},
		&UserPreference{},
		// 动态记忆事实
		&NovelMemoryFact{},
		// 章节审核评分
		&ChapterReview{},
		// 递归摘要树
		&ChapterSummaryNode{},
		// 知识图谱关系边
		&KnowledgeRelation{},
		// 世界构建与规划
		&NovelWorldSetting{},
		&NovelForeshadow{},
		&NovelPlotOutline{},
		&ReflectionLog{},
		// 模型可用性状态
		&AIModelStatus{},
	)
}
