// server/internal/dao/genre.go
package dao

import (
	"ai-curton/server/internal/model"

	"gorm.io/gorm"
)

// GenreDAO 赛道数据访问层
type GenreDAO struct {
	db *gorm.DB
}

// NewGenreDAO 创建 GenreDAO 实例
func NewGenreDAO() *GenreDAO {
	return &GenreDAO{db: model.DB}
}

// Create 创建赛道
func (d *GenreDAO) Create(genre *model.Genre) error {
	return d.db.Create(genre).Error
}

// Get 根据 ID 获取赛道
func (d *GenreDAO) Get(id uint) (*model.Genre, error) {
	var genre model.Genre
	err := d.db.First(&genre, id).Error
	if err != nil {
		return nil, err
	}
	return &genre, nil
}

// Update 更新赛道
func (d *GenreDAO) Update(genre *model.Genre) error {
	return d.db.Save(genre).Error
}

// Delete 删除赛道
func (d *GenreDAO) Delete(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 删除关联关系
		if err := tx.Where("genre_id = ?", id).Delete(&model.MemoryGenre{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Genre{}, id).Error
	})
}

// ListAll 获取所有赛道（按 sort_order 排序）
func (d *GenreDAO) ListAll() ([]model.Genre, error) {
	var genres []model.Genre
	err := d.db.Order("sort_order ASC, id ASC").Find(&genres).Error
	return genres, err
}

// ListByParent 获取指定父赛道下的子赛道
func (d *GenreDAO) ListByParent(parentID uint) ([]model.Genre, error) {
	var genres []model.Genre
	err := d.db.Where("parent_id = ?", parentID).Order("sort_order ASC, id ASC").Find(&genres).Error
	return genres, err
}

// ========== MemoryGenre 关联操作 ==========

// SetMemoryGenres 设置记忆的赛道关联（先删后插）
func (d *GenreDAO) SetMemoryGenres(memoryID uint, genreIDs []uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("memory_id = ?", memoryID).Delete(&model.MemoryGenre{}).Error; err != nil {
			return err
		}
		for _, gid := range genreIDs {
			mg := model.MemoryGenre{MemoryID: memoryID, GenreID: gid}
			if err := tx.Create(&mg).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListByMemory 获取记忆关联的赛道 ID 列表
func (d *GenreDAO) ListByMemory(memoryID uint) ([]uint, error) {
	var mgs []model.MemoryGenre
	err := d.db.Where("memory_id = ?", memoryID).Find(&mgs).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(mgs))
	for i, mg := range mgs {
		ids[i] = mg.GenreID
	}
	return ids, nil
}

// ListMemoryIDsByGenre 获取某赛道下的所有记忆 ID（含子赛道）
func (d *GenreDAO) ListMemoryIDsByGenre(genreID uint) ([]uint, error) {
	// 查询该赛道及其子赛道 ID
	genreIDs := []uint{genreID}
	var childIDs []uint
	if err := d.db.Model(&model.Genre{}).Where("parent_id = ?", genreID).Pluck("id", &childIDs).Error; err == nil {
		genreIDs = append(genreIDs, childIDs...)
		// 查第三级
		for _, cid := range childIDs {
			var grandChildIDs []uint
			if err := d.db.Model(&model.Genre{}).Where("parent_id = ?", cid).Pluck("id", &grandChildIDs).Error; err == nil {
				genreIDs = append(genreIDs, grandChildIDs...)
			}
		}
	}

	var memoryIDs []uint
	err := d.db.Model(&model.MemoryGenre{}).Where("genre_id IN ?", genreIDs).Pluck("memory_id", &memoryIDs).Error
	return memoryIDs, err
}

// DeleteByMemory 删除记忆的所有赛道关联
func (d *GenreDAO) DeleteByMemory(memoryID uint) error {
	return d.db.Where("memory_id = ?", memoryID).Delete(&model.MemoryGenre{}).Error
}

// Count 返回赛道总数
func (d *GenreDAO) Count() (int64, error) {
	var count int64
	err := d.db.Model(&model.Genre{}).Count(&count).Error
	return count, err
}

// SeedGenres 幂等写入默认赛道（slug 不存在则插入，已存在则跳过）
func (d *GenreDAO) SeedGenres(genres []model.Genre) error {
	for i := range genres {
		var count int64
		d.db.Model(&model.Genre{}).Where("slug = ?", genres[i].Slug).Count(&count)
		if count == 0 {
			if err := d.db.Create(&genres[i]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// FindBySlug 根据 slug 查找赛道
func (d *GenreDAO) FindBySlug(slug string) (*model.Genre, error) {
	var g model.Genre
	err := d.db.Where("slug = ?", slug).First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}
