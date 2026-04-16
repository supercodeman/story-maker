// server/internal/vectordb/novel_facts.go
package vectordb

import (
	"context"
	"fmt"
	"log"

	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// FactVector 待插入 Milvus 的事实向量数据
type FactVector struct {
	FactID   int64     // 对应 MySQL NovelMemoryFact.ID
	NovelID  int64     // 小说 ID（分区键）
	FactType string    // 事实类型
	Vector   []float32 // embedding 向量
}

// SearchResult Milvus 检索结果
type SearchResult struct {
	FactID   int64   // MySQL 事实 ID
	NovelID  int64   // 小说 ID
	FactType string  // 事实类型
	Score    float32 // 相似度分数
}

// EnsureCollection 创建 collection + index（幂等操作）
func (m *MilvusClient) EnsureCollection(ctx context.Context) error {
	exists, err := m.client.HasCollection(ctx, m.collection)
	if err != nil {
		return fmt.Errorf("检查 collection 是否存在失败: %w", err)
	}
	if exists {
		// 确保 collection 已加载
		err = m.client.LoadCollection(ctx, m.collection, false)
		if err != nil {
			log.Printf("[milvus] 加载 collection 警告: %v", err)
		}
		return nil
	}

	// 定义 schema
	schema := &entity.Schema{
		CollectionName: m.collection,
		Description:    "小说动态记忆事实向量存储",
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "fact_id",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "novel_id",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:       "fact_type",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "30"},
			},
			{
				Name:       "embedding",
				DataType:   entity.FieldTypeFloatVector,
				TypeParams: map[string]string{"dim": fmt.Sprintf("%d", m.dimension)},
			},
		},
	}

	if err := m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		return fmt.Errorf("创建 collection 失败: %w", err)
	}

	// 创建 IVF_FLAT 索引
	idx, err := entity.NewIndexIvfFlat(entity.COSINE, 128)
	if err != nil {
		return fmt.Errorf("创建索引参数失败: %w", err)
	}
	if err := m.client.CreateIndex(ctx, m.collection, "embedding", idx, false); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 加载 collection 到内存
	if err := m.client.LoadCollection(ctx, m.collection, false); err != nil {
		return fmt.Errorf("加载 collection 失败: %w", err)
	}

	log.Printf("[milvus] collection %s 创建并加载成功", m.collection)
	return nil
}

// InsertFacts 批量插入事实向量
func (m *MilvusClient) InsertFacts(ctx context.Context, facts []FactVector) ([]int64, error) {
	if len(facts) == 0 {
		return nil, nil
	}

	log.Printf("[milvus-write] 准备插入 %d 条向量, collection=%s", len(facts), m.collection)

	factIDs := make([]int64, len(facts))
	novelIDs := make([]int64, len(facts))
	factTypes := make([]string, len(facts))
	vectors := make([][]float32, len(facts))

	for i, f := range facts {
		factIDs[i] = f.FactID
		novelIDs[i] = f.NovelID
		factTypes[i] = f.FactType
		vectors[i] = f.Vector
		log.Printf("[milvus-write]   #%d fact_id=%d novel_id=%d type=%s vec_dim=%d", i, f.FactID, f.NovelID, f.FactType, len(f.Vector))
	}

	factIDCol := entity.NewColumnInt64("fact_id", factIDs)
	novelIDCol := entity.NewColumnInt64("novel_id", novelIDs)
	factTypeCol := entity.NewColumnVarChar("fact_type", factTypes)
	vectorCol := entity.NewColumnFloatVector("embedding", m.dimension, vectors)

	result, err := m.client.Insert(ctx, m.collection, "",
		factIDCol, novelIDCol, factTypeCol, vectorCol)
	if err != nil {
		log.Printf("[milvus-write] 插入失败: %v", err)
		return nil, fmt.Errorf("插入向量失败: %w", err)
	}

	// 提取自动生成的 Milvus ID
	ids := result.(*entity.ColumnInt64).Data()
	log.Printf("[milvus-write] 插入成功, 返回 %d 个 Milvus ID", len(ids))
	return ids, nil
}

// SearchByNovel 按小说维度检索相似事实
func (m *MilvusClient) SearchByNovel(ctx context.Context, novelID int64, queryVec []float32, topK int, factTypes []string) ([]SearchResult, error) {
	// 构建过滤表达式
	filter := fmt.Sprintf("novel_id == %d", novelID)
	if len(factTypes) > 0 {
		filter += " && fact_type in ["
		for i, ft := range factTypes {
			if i > 0 {
				filter += ","
			}
			filter += fmt.Sprintf(`"%s"`, ft)
		}
		filter += "]"
	}

	log.Printf("[milvus-read] 开始检索 novel_id=%d topK=%d filter=%q vec_dim=%d", novelID, topK, filter, len(queryVec))

	// 构建搜索向量
	vectors := []entity.Vector{entity.FloatVector(queryVec)}

	sp, err := entity.NewIndexIvfFlatSearchParam(16)
	if err != nil {
		return nil, fmt.Errorf("创建搜索参数失败: %w", err)
	}

	results, err := m.client.Search(ctx, m.collection, nil, filter, []string{"fact_id", "novel_id", "fact_type"},
		vectors, "embedding", entity.COSINE, topK, sp)
	if err != nil {
		log.Printf("[milvus-read] 检索失败: %v", err)
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	var searchResults []SearchResult
	for _, result := range results {
		log.Printf("[milvus-read] 结果集: %d 条命中", result.ResultCount)
		for i := 0; i < result.ResultCount; i++ {
			var factID, nID int64
			var factType string

			// 提取字段值
			for _, field := range result.Fields {
				switch field.Name() {
				case "fact_id":
					col, ok := field.(*entity.ColumnInt64)
					if ok {
						factID, _ = col.ValueByIdx(i)
					}
				case "novel_id":
					col, ok := field.(*entity.ColumnInt64)
					if ok {
						nID, _ = col.ValueByIdx(i)
					}
				case "fact_type":
					col, ok := field.(*entity.ColumnVarChar)
					if ok {
						factType, _ = col.ValueByIdx(i)
					}
				}
			}

			searchResults = append(searchResults, SearchResult{
				FactID:   factID,
				NovelID:  nID,
				FactType: factType,
				Score:    result.Scores[i],
			})
			log.Printf("[milvus-read]   #%d fact_id=%d type=%s score=%.4f", i, factID, factType, result.Scores[i])
		}
	}

	log.Printf("[milvus-read] 检索完成, 共返回 %d 条结果", len(searchResults))
	return searchResults, nil
}

// DeleteByFactIDs 根据 MySQL 事实 ID 删除 Milvus 中的向量
func (m *MilvusClient) DeleteByFactIDs(ctx context.Context, factIDs []int64) error {
	if len(factIDs) == 0 {
		return nil
	}

	// 构建删除表达式
	expr := "fact_id in ["
	for i, id := range factIDs {
		if i > 0 {
			expr += ","
		}
		expr += fmt.Sprintf("%d", id)
	}
	expr += "]"

	log.Printf("[milvus-delete] 删除向量: expr=%q (%d 条)", expr, len(factIDs))
	err := m.client.Delete(ctx, m.collection, "", expr)
	if err != nil {
		log.Printf("[milvus-delete] 删除失败: %v", err)
	} else {
		log.Printf("[milvus-delete] 删除成功")
	}
	return err
}
