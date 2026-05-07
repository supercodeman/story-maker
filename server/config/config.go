// server/config/config.go
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

//# 后端
//cd server && go mod tidy && go build -o bin/server ./cmd/main.go
//
//# 前端
//cd web && npm install && npm run dev
//
//# 或一键 Docker
//docker-compose up -d

// AppConfig 应用全局配置
type AppConfig struct {
	Server           ServerConfig      `mapstructure:"server"`
	Database         DatabaseConfig    `mapstructure:"database"`
	InterviewDatabase DatabaseConfig   `mapstructure:"interview_database"` // 面试系统数据库
	Redis            RedisConfig       `mapstructure:"redis"`
	JWT              JWTConfig         `mapstructure:"jwt"`
	Encrypt          EncryptConfig     `mapstructure:"encrypt"`
	Upload           UploadConfig      `mapstructure:"upload"`
	Kimi             KimiConfig        `mapstructure:"kimi"`
	Zhipu            ZhipuConfig       `mapstructure:"zhipu"`
	Qwen             QwenConfig        `mapstructure:"qwen"`
	Deepseek         DeepseekConfig    `mapstructure:"deepseek"`
	MiniMax          MiniMaxConfig     `mapstructure:"minimax"`
	CogVideo         CogVideoConfig    `mapstructure:"cogvideo"`
	AI               AIConfig          `mapstructure:"ai"`
	QWeather         QWeatherConfig    `mapstructure:"qweather"`
	NovelSearch      NovelSearchConfig `mapstructure:"novel_search"`
	Milvus           MilvusConfig      `mapstructure:"milvus"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

type DatabaseConfig struct {
	DSN             string `mapstructure:"dsn"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 秒
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	AccessTokenTTL  int    `mapstructure:"access_token_ttl"`  // 秒
	RefreshTokenTTL int    `mapstructure:"refresh_token_ttl"` // 秒
}

type EncryptConfig struct {
	Key string `mapstructure:"key"` // AES-256 密钥，用于加密 API Key
}

type UploadConfig struct {
	Path    string `mapstructure:"path"`
	MaxSize int64  `mapstructure:"max_size"` // 字节
}

type KimiConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type ZhipuConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type QwenConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type DeepseekConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// MiniMaxConfig MiniMax TTS 配置
type MiniMaxConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
	GroupID string `mapstructure:"group_id"` // MiniMax 分组ID
}

// CogVideoConfig 智谱视频生成配置
type CogVideoConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

type AIConfig struct {
	DefaultProvider string `mapstructure:"default_provider"` // mock, kimi, zhipu, qwen
}

// QWeatherConfig 和风天气配置
type QWeatherConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// NovelSearchConfig 小说搜索引擎配置
type NovelSearchConfig struct {
	WebSearchModel string `mapstructure:"web_search_model"` // AI 联网搜索使用的模型名称（zhipu/qwen/kimi）
	AIModel        string `mapstructure:"ai_model"`         // AI 离线兜底使用的模型名称
}

// MilvusConfig Milvus 向量数据库配置
type MilvusConfig struct {
	Address    string `mapstructure:"address"`    // Milvus 连接地址，如 localhost:19530
	DBName     string `mapstructure:"db_name"`    // 数据库名称
	Collection string `mapstructure:"collection"` // Collection 名称
	Dimension  int    `mapstructure:"dimension"`  // 向量维度
	Enabled    bool   `mapstructure:"enabled"`    // 是否启用 Milvus
}

// Global 全局配置实例
var Global *AppConfig

// Load 从指定路径加载配置文件
func Load(path string) (*AppConfig, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &AppConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	Global = cfg
	return cfg, nil
}
