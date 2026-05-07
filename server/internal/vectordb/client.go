// server/internal/vectordb/client.go
package vectordb

import (
	"context"
	"fmt"
	"log"

	"story-maker/server/config"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
)

// MilvusClient Milvus 向量数据库客户端封装
type MilvusClient struct {
	client     client.Client
	collection string
	dimension  int
}

// NewMilvusClient 创建 Milvus 客户端连接
func NewMilvusClient(cfg *config.MilvusConfig) (*MilvusClient, error) {
	ctx := context.Background()

	// 先用 default database 连接，确保目标 database 存在
	if cfg.DBName != "" && cfg.DBName != "default" {
		defaultC, err := client.NewClient(ctx, client.Config{
			Address: cfg.Address,
		})
		if err != nil {
			return nil, fmt.Errorf("连接 Milvus 失败: %w", err)
		}
		dbs, err := defaultC.ListDatabases(ctx)
		if err == nil {
			found := false
			for _, db := range dbs {
				if db.Name == cfg.DBName {
					found = true
					break
				}
			}
			if !found {
				if err := defaultC.CreateDatabase(ctx, cfg.DBName); err != nil {
					defaultC.Close()
					return nil, fmt.Errorf("创建 Milvus database %s 失败: %w", cfg.DBName, err)
				}
				log.Printf("[milvus] 已创建 database: %s", cfg.DBName)
			}
		}
		defaultC.Close()
	}

	c, err := client.NewClient(ctx, client.Config{
		Address: cfg.Address,
		DBName:  cfg.DBName,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 Milvus 失败: %w", err)
	}

	mc := &MilvusClient{
		client:     c,
		collection: cfg.Collection,
		dimension:  cfg.Dimension,
	}

	// 确保 collection 存在
	if err := mc.EnsureCollection(ctx); err != nil {
		c.Close()
		return nil, fmt.Errorf("初始化 Milvus collection 失败: %w", err)
	}

	log.Printf("[milvus] 连接成功，collection=%s, dimension=%d", cfg.Collection, cfg.Dimension)
	return mc, nil
}

// Close 关闭 Milvus 连接
func (m *MilvusClient) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// Client 返回底层 Milvus 客户端（供 novel_facts.go 使用）
func (m *MilvusClient) Client() client.Client {
	return m.client
}

// Collection 返回 collection 名称
func (m *MilvusClient) Collection() string {
	return m.collection
}

// Dimension 返回向量维度
func (m *MilvusClient) Dimension() int {
	return m.dimension
}
