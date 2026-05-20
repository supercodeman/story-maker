# Plan 1: 后端基础设施 + Auth 模块 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 Go 后端项目骨架，实现配置加载、数据库连接、Redis 连接、JWT 认证、用户注册/登录/个人信息 CRUD。

**Architecture:** 模块化单体架构，handler → service → dao 三层分离，Gin 作为 HTTP 框架，GORM 作为 ORM，JWT 做认证。

**Tech Stack:** Go 1.21+, Gin, GORM, MySQL 8.0, Redis 7, jwt-go, bcrypt, viper

---

### Task 1: Go 项目初始化

**Files:**
- Create: `server/go.mod`
- Create: `server/cmd/main.go`

- [ ] **Step 1: 初始化 Go Module**

在 `server/` 目录下执行：

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go mod init story-maker/server
```

- [ ] **Step 2: 安装核心依赖**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go get github.com/gin-gonic/gin@v1.9.1
go get gorm.io/gorm@v1.25.7
go get gorm.io/driver/mysql@v1.5.4
go get github.com/redis/go-redis/v9@v9.4.0
go get github.com/golang-jwt/jwt/v5@v5.2.1
go get golang.org/x/crypto@latest
go get github.com/spf13/viper@v1.18.2
go get github.com/gin-contrib/cors@v1.5.0
```

- [ ] **Step 3: 创建入口文件 `server/cmd/main.go`（占位，后续 Task 9 完善）**

```go
// server/cmd/main.go
package main

import "fmt"

func main() {
	fmt.Println("Ai-Curton server starting...")
}
```

- [ ] **Step 4: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/...
```

- [ ] **Step 5: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/go.mod server/go.sum server/cmd/main.go
git commit -m "feat: init Go module with core dependencies and entry point"
```

---

### Task 2: 配置模块

**Files:**
- Create: `server/config/config.go`
- Create: `server/config.yaml`

- [ ] **Step 1: 创建配置结构体与加载函数 `server/config/config.go`**

```go
// server/config/config.go
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// AppConfig 应用全局配置
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Encrypt  EncryptConfig  `mapstructure:"encrypt"`
	Upload   UploadConfig   `mapstructure:"upload"`
	Kimi     KimiConfig     `mapstructure:"kimi"`
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
	Secret           string `mapstructure:"secret"`
	AccessTokenTTL   int    `mapstructure:"access_token_ttl"`  // 秒
	RefreshTokenTTL  int    `mapstructure:"refresh_token_ttl"` // 秒
}

type EncryptConfig struct {
	Key string `mapstructure:"key"` // AES-256 密钥，用于加密 API Key
}

type UploadConfig struct {
	Path       string `mapstructure:"path"`
	MaxSize    int64  `mapstructure:"max_size"` // 字节
}

type KimiConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
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
```

- [ ] **Step 2: 创建默认配置文件 `server/config.yaml`**

```yaml
# server/config.yaml
server:
  port: 8080
  mode: debug

database:
  dsn: "root:password@tcp(127.0.0.1:3306)/ai_curton?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

jwt:
  secret: "change-me-to-a-random-string"
  access_token_ttl: 7200      # 2 小时
  refresh_token_ttl: 604800   # 7 天

encrypt:
  key: "change-me-32-byte-aes256-key!!"  # 必须 32 字节

upload:
  path: "./uploads"
  max_size: 20971520  # 20MB

kimi:
  api_key: ""
  base_url: "https://api.moonshot.cn/v1"
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/config/ server/config.yaml
git commit -m "feat: add config module with viper-based YAML loading"
```

---

### Task 3: 数据库连接与 Model 定义

**Files:**
- Create: `server/internal/model/base.go`
- Create: `server/internal/model/user.go`

- [ ] **Step 1: 创建数据库初始化函数 `server/internal/model/base.go`**

```go
// server/internal/model/base.go
package model

import (
	"fmt"
	"time"

	"story-maker/server/config"

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
	)
}
```

- [ ] **Step 2: 创建 User 模型 `server/internal/model/user.go`**

```go
// server/internal/model/user.go
package model

import "time"

// User 用户模型
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50" json:"username"`
	Email        string    `gorm:"uniqueIndex;size:100" json:"email"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	Role         string    `gorm:"size:20;default:creator" json:"role"` // admin, creator, viewer
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/model/
git commit -m "feat: add database initialization and User model with auto-migrate"
```

---

### Task 4: Redis 连接

**Files:**
- Create: `server/internal/model/redis.go`

- [ ] **Step 1: 创建 Redis 初始化函数 `server/internal/model/redis.go`**

```go
// server/internal/model/redis.go
package model

import (
	"context"
	"fmt"
	"time"

	"story-maker/server/config"

	"github.com/redis/go-redis/v9"
)

// RDB 全局 Redis 客户端实例
var RDB *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.RedisConfig) error {
	RDB = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	return nil
}

// CloseRedis 关闭 Redis 连接
func CloseRedis() error {
	if RDB != nil {
		return RDB.Close()
	}
	return nil
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/model/redis.go
git commit -m "feat: add Redis client initialization"
```

---

### Task 5: 中间件

**Files:**
- Create: `server/internal/middleware/cors.go`
- Create: `server/internal/middleware/auth.go`

- [ ] **Step 1: 创建 CORS 中间件 `server/internal/middleware/cors.go`**

```go
// server/internal/middleware/cors.go
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 返回 CORS 中间件，允许前端跨域访问
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
```

- [ ] **Step 2: 创建 JWT 认证中间件 `server/internal/middleware/auth.go`**

```go
// server/internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"story-maker/server/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims 自定义 JWT 声明
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthRequired JWT 认证中间件，解析 token 并将 user_id 注入 context
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Authorization header 中提取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
			})
			return
		}

		// 验证 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header format must be Bearer {token}",
			})
			return
		}

		tokenString := parts[1]

		// 解析并验证 token
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// 验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.Global.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid or expired token",
			})
			return
		}

		// 将用户信息注入 context，后续 handler 可通过 c.Get("user_id") 获取
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// GetUserID 从 gin.Context 中获取当前用户 ID 的辅助函数
func GetUserID(c *gin.Context) uint {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(uint)
}

// GetUsername 从 gin.Context 中获取当前用户名的辅助函数
func GetUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	return username.(string)
}
```

- [ ] **Step 3: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/middleware/
git commit -m "feat: add CORS and JWT authentication middleware"
```

---

### Task 6: Auth DAO 层

**Files:**
- Create: `server/internal/dao/user.go`

- [ ] **Step 1: 创建 User DAO `server/internal/dao/user.go`**

```go
// server/internal/dao/user.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// UserDAO 用户数据访问对象
type UserDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建 UserDAO 实例
func NewUserDAO() *UserDAO {
	return &UserDAO{db: model.DB}
}

// CreateUser 创建用户记录
func (d *UserDAO) CreateUser(user *model.User) error {
	return d.db.Create(user).Error
}

// GetUserByEmail 根据邮箱查询用户
func (d *UserDAO) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := d.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 根据 ID 查询用户
func (d *UserDAO) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	err := d.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息（仅更新非零值字段）
func (d *UserDAO) UpdateUser(user *model.User) error {
	return d.db.Model(user).Updates(map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
	}).Error
}

// GetUserByUsername 根据用户名查询用户
func (d *UserDAO) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := d.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/dao/
git commit -m "feat: add User DAO with CRUD operations"
```

---

### Task 7: Auth Service 层

**Files:**
- Create: `server/internal/service/auth.go`

- [ ] **Step 1: 创建 Auth Service `server/internal/service/auth.go`**

```go
// server/internal/service/auth.go
package service

import (
	"errors"
	"time"

	"story-maker/server/config"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	userDAO *dao.UserDAO
}

// NewAuthService 创建 AuthService 实例
func NewAuthService() *AuthService {
	return &AuthService{
		userDAO: dao.NewUserDAO(),
	}
}

// RegisterRequest 注册请求参数
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

// LoginRequest 登录请求参数
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse 令牌响应
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // access_token 有效期（秒）
}

// UpdateProfileRequest 更新个人信息请求
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=2,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// Register 用户注册：密码 bcrypt 加密后存储
func (s *AuthService) Register(req *RegisterRequest) (*model.User, error) {
	// 检查邮箱是否已注册
	existing, err := s.userDAO.GetUserByEmail(req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("internal server error")
	}
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// 检查用户名是否已存在
	existing, err = s.userDAO.GetUserByUsername(req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("internal server error")
	}
	if existing != nil {
		return nil, errors.New("username already taken")
	}

	// bcrypt 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "creator",
	}

	if err := s.userDAO.CreateUser(user); err != nil {
		return nil, errors.New("failed to create user")
	}

	return user, nil
}

// Login 用户登录：校验密码，生成 JWT access_token + refresh_token
func (s *AuthService) Login(req *LoginRequest) (*TokenResponse, error) {
	// 查找用户
	user, err := s.userDAO.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, errors.New("internal server error")
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// 生成 token 对
	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return tokens, nil
}

// GetProfile 获取用户个人信息
func (s *AuthService) GetProfile(userID uint) (*model.User, error) {
	user, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("internal server error")
	}
	return user, nil
}

// UpdateProfile 更新用户个人信息
func (s *AuthService) UpdateProfile(userID uint, req *UpdateProfileRequest) (*model.User, error) {
	user, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("internal server error")
	}

	// 如果要更新用户名，检查唯一性
	if req.Username != "" && req.Username != user.Username {
		existing, err := s.userDAO.GetUserByUsername(req.Username)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("internal server error")
		}
		if existing != nil {
			return nil, errors.New("username already taken")
		}
		user.Username = req.Username
	}

	// 如果要更新邮箱，检查唯一性
	if req.Email != "" && req.Email != user.Email {
		existing, err := s.userDAO.GetUserByEmail(req.Email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("internal server error")
		}
		if existing != nil {
			return nil, errors.New("email already registered")
		}
		user.Email = req.Email
	}

	if err := s.userDAO.UpdateUser(user); err != nil {
		return nil, errors.New("failed to update profile")
	}

	return user, nil
}

// generateTokenPair 生成 access_token（短期）和 refresh_token（长期）
func (s *AuthService) generateTokenPair(user *model.User) (*TokenResponse, error) {
	jwtCfg := config.Global.JWT

	// 生成 access_token
	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"type":     "access",
		"exp":      time.Now().Add(time.Duration(jwtCfg.AccessTokenTTL) * time.Second).Unix(),
		"iat":      time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		return nil, err
	}

	// 生成 refresh_token
	refreshClaims := jwt.MapClaims{
		"user_id": user.ID,
		"type":    "refresh",
		"exp":     time.Now().Add(time.Duration(jwtCfg.RefreshTokenTTL) * time.Second).Unix(),
		"iat":     time.Now().Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    jwtCfg.AccessTokenTTL,
	}, nil
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/service/
git commit -m "feat: add Auth service with register, login, profile CRUD and JWT generation"
```

---

### Task 8: Auth Handler 层

**Files:**
- Create: `server/internal/handler/auth.go`
- Create: `server/internal/handler/user.go`
- Create: `server/internal/handler/response.go`

- [ ] **Step 1: 创建统一响应辅助函数 `server/internal/handler/response.go`**

```go
// server/internal/handler/response.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一 API 响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 返回带自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 返回错误响应
func Error(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, Response{
		Code:    httpStatus,
		Message: message,
	})
}

// BadRequest 返回 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 返回 401 错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// InternalError 返回 500 错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
```

- [ ] **Step 2: 创建 Auth Handler `server/internal/handler/auth.go`**

```go
// server/internal/handler/auth.go
package handler

import (
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关请求处理
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建 AuthHandler 实例
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(),
	}
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, "Registration successful", gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	tokens, err := h.authService.Login(&req)
	if err != nil {
		Unauthorized(c, err.Error())
		return
	}

	Success(c, tokens)
}

// Logout 用户登出（客户端清除 token 即可，服务端预留 token 黑名单扩展点）
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// MVP 阶段：登出由客户端清除本地 token 实现
	// 后续可在此处将 token 加入 Redis 黑名单
	SuccessWithMessage(c, "Logout successful", nil)
}
```

- [ ] **Step 3: 创建 User Handler `server/internal/handler/user.go`**

```go
// server/internal/handler/user.go
package handler

import (
	"story-maker/server/internal/middleware"
	"story-maker/server/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户信息相关请求处理
type UserHandler struct {
	authService *service.AuthService
}

// NewUserHandler 创建 UserHandler 实例
func NewUserHandler() *UserHandler {
	return &UserHandler{
		authService: service.NewAuthService(),
	}
}

// GetProfile 获取当前用户个人信息
// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, user)
}

// UpdateProfile 更新当前用户个人信息
// PUT /api/v1/user/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		Unauthorized(c, "User not authenticated")
		return
	}

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request parameters: "+err.Error())
		return
	}

	user, err := h.authService.UpdateProfile(userID, &req)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, user)
}
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/handler/
git commit -m "feat: add Auth and User handlers with unified response format"
```

---

### Task 9: 路由注册与启动

**Files:**
- Create: `server/internal/router/router.go`
- Modify: `server/cmd/main.go`

- [ ] **Step 1: 创建路由注册模块 `server/internal/router/router.go`**

```go
// server/internal/router/router.go
package router

import (
	"story-maker/server/internal/handler"
	"story-maker/server/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化并返回配置好的 Gin 引擎
func Setup() *gin.Engine {
	r := gin.Default()

	// 全局中间件
	r.Use(middleware.CORS())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 初始化 handler
	authHandler := handler.NewAuthHandler()
	userHandler := handler.NewUserHandler()

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

		// 用户路由（需要登录）
		user := v1.Group("/user")
		user.Use(middleware.AuthRequired())
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
		}
	}

	return r
}
```

- [ ] **Step 2: 完善入口文件 `server/cmd/main.go`**

将 `server/cmd/main.go` 替换为以下完整内容：

```go
// server/cmd/main.go
package main

import (
	"fmt"
	"log"

	"story-maker/server/config"
	"story-maker/server/internal/model"
	"story-maker/server/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Config loaded successfully")

	// 2. 设置 Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 3. 初始化数据库
	if err := model.InitDB(&cfg.Database); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	log.Println("Database connected and migrated")

	// 4. 初始化 Redis
	if err := model.InitRedis(&cfg.Redis); err != nil {
		log.Fatalf("Failed to init redis: %v", err)
	}
	defer model.CloseRedis()
	log.Println("Redis connected")

	// 5. 初始化路由
	r := router.Setup()

	// 6. 启动 HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /Users/sangchenglong/go/src/story-maker/server
go build ./cmd/...
```

- [ ] **Step 4: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add server/internal/router/ server/cmd/main.go
git commit -m "feat: add router setup and wire all modules in main.go entry point"
```

---

### Task 10: Docker Compose

**Files:**
- Create: `docker-compose.yml`
- Create: `server/Dockerfile`
- Create: `server/.dockerignore`

- [ ] **Step 1: 创建 Server Dockerfile `server/Dockerfile`**

```dockerfile
# server/Dockerfile

# ---- 构建阶段 ----
FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache gcc musl-dev

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码
COPY . .

# 编译二进制（静态链接，适合 alpine 运行）
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/server ./cmd/main.go

# ---- 运行阶段 ----
FROM alpine:3.19

WORKDIR /app

# 安装 CA 证书（HTTPS 外部请求需要）和时区数据
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制二进制文件
COPY --from=builder /build/server .

# 复制配置文件
COPY config.yaml .

# 创建上传目录
RUN mkdir -p /app/uploads

EXPOSE 8080

CMD ["./server"]
```

- [ ] **Step 2: 创建 .dockerignore `server/.dockerignore`**

```
# server/.dockerignore
.git
.gitignore
README.md
*.md
tmp/
uploads/
```

- [ ] **Step 3: 创建 Docker Compose 配置 `docker-compose.yml`（项目根目录）**

```yaml
# docker-compose.yml
version: "3.8"

services:
  mysql:
    image: mysql:8.0
    container_name: story-maker-mysql
    restart: unless-stopped
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: ai_curton
      MYSQL_CHARSET: utf8mb4
      MYSQL_COLLATION: utf8mb4_unicode_ci
    volumes:
      - ./data/mysql:/var/lib/mysql
    command: --default-authentication-plugin=mysql_native_password --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: story-maker-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - ./data/redis:/data
    command: redis-server --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  server:
    build:
      context: ./server
      dockerfile: Dockerfile
    container_name: story-maker-server
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data/uploads:/app/uploads
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      GIN_MODE: release
```

- [ ] **Step 4: 创建 Docker 专用配置文件 `server/config.docker.yaml`**

由于 Docker 环境中服务名不同于本地 127.0.0.1，需要额外提供一份面向容器网络的配置。实际部署时将此文件复制为 `config.yaml` 或通过环境变量覆盖。

```yaml
# server/config.docker.yaml
# Docker Compose 环境专用配置，容器内使用服务名访问 MySQL/Redis
server:
  port: 8080
  mode: release

database:
  dsn: "root:password@tcp(mysql:3306)/ai_curton?charset=utf8mb4&parseTime=True&loc=Local"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

redis:
  addr: "redis:6379"
  password: ""
  db: 0

jwt:
  secret: "change-me-to-a-random-string"
  access_token_ttl: 7200
  refresh_token_ttl: 604800

encrypt:
  key: "change-me-32-byte-aes256-key!!"

upload:
  path: "/app/uploads"
  max_size: 20971520

kimi:
  api_key: ""
  base_url: "https://api.moonshot.cn/v1"
```

> **注意：** Dockerfile 默认复制的是 `config.yaml`（本地开发配置）。若要在 Docker Compose 中运行，需在构建前将 `config.docker.yaml` 复制为 `config.yaml`，或修改 Dockerfile 的 COPY 指令指向 `config.docker.yaml`。推荐做法是在 `docker-compose.yml` 中通过 `volumes` 挂载：
>
> ```yaml
> volumes:
>   - ./server/config.docker.yaml:/app/config.yaml
>   - ./data/uploads:/app/uploads
> ```

- [ ] **Step 5: 验证 Docker 构建**

```bash
cd /Users/sangchenglong/go/src/story-maker
docker-compose config  # 验证 compose 文件语法
```

- [ ] **Step 6: Commit**

```bash
cd /Users/sangchenglong/go/src/story-maker
git add docker-compose.yml server/Dockerfile server/.dockerignore server/config.docker.yaml
git commit -m "feat: add Docker Compose (mysql + redis + server) and server Dockerfile"
```

---

## 完成标准

所有 Task 完成后，项目应满足以下条件：

1. **编译通过**：`cd server && go build ./cmd/...` 无报错
2. **服务可启动**：本地启动 MySQL + Redis 后，`cd server && go run ./cmd/main.go` 能正常监听 8080 端口
3. **API 可用**：
   - `POST /api/v1/auth/register` — 注册新用户，返回用户信息
   - `POST /api/v1/auth/login` — 登录返回 access_token + refresh_token
   - `POST /api/v1/auth/logout` — 登出成功
   - `GET /api/v1/user/profile`（带 Bearer token）— 返回当前用户信息
   - `PUT /api/v1/user/profile`（带 Bearer token）— 更新并返回用户信息
   - `GET /health` — 健康检查返回 `{"status":"ok"}`
4. **Docker 部署**：`docker-compose up -d` 能启动完整的 mysql + redis + server 环境
5. **代码规范**：每个文件均有包注释，遵循 handler → service → dao 三层分离

## 目录结构预览

完成本 Plan 后的项目目录结构：

```
story-maker/
├── docker-compose.yml
├── data/                          # Docker 数据卷（gitignore）
│   ├── mysql/
│   ├── redis/
│   └── uploads/
├── docs/
│   └── superpowers/
│       ├── specs/
│       │   └── 2026-03-31-ai-curton-design.md
│       └── plans/
│           └── 2026-03-31-plan1-backend-infra-auth.md
├── server/
│   ├── cmd/
│   │   └── main.go               # 入口：加载配置→初始化DB→初始化Redis→注册路由→启动
│   ├── config/
│   │   └── config.go             # Viper 配置加载 + 结构体定义
│   ├── config.yaml               # 本地开发配置
│   ├── config.docker.yaml        # Docker 环境配置
│   ├── internal/
│   │   ├── dao/
│   │   │   └── user.go           # UserDAO：CreateUser/GetUserByEmail/GetUserByID/UpdateUser
│   │   ├── handler/
│   │   │   ├── auth.go           # AuthHandler：Register/Login/Logout
│   │   │   ├── response.go       # 统一响应辅助函数
│   │   │   └── user.go           # UserHandler：GetProfile/UpdateProfile
│   │   ├── middleware/
│   │   │   ├── auth.go           # JWT 认证中间件
│   │   │   └── cors.go           # CORS 中间件
│   │   ├── model/
│   │   │   ├── base.go           # DB 初始化 + AutoMigrate
│   │   │   ├── redis.go          # Redis 初始化
│   │   │   └── user.go           # User 模型定义
│   │   ├── router/
│   │   │   └── router.go         # 路由注册
│   │   └── service/
│   │       └── auth.go           # AuthService：Register/Login/GetProfile/UpdateProfile/JWT生成
│   ├── Dockerfile
│   ├── .dockerignore
│   ├── go.mod
│   └── go.sum
└── web/                           # 前端（后续 Plan 实现）
```
