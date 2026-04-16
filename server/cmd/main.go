// server/cmd/main.go
package main

import (
	"fmt"
	"log"

	"ai-curton/server/config"
	"ai-curton/server/internal/model"
	"ai-curton/server/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置（支持通过环境变量 CONFIG_PATH 指定路径，默认查找 ../config.yaml）
	cfg, err := config.Load("./config/config.yaml")
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
