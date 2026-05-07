// server/internal/dao/world_building.go
package dao

import (
	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// WorldBuildingDAO 世界构建数据访问层
type WorldBuildingDAO struct {
	db *gorm.DB
}

// NewWorldBuildingDAO 创建 WorldBuildingDAO 实例
func NewWorldBuildingDAO() *WorldBuildingDAO {
	return &WorldBuildingDAO{db: model.DB}
}

// ========== NovelWorldSetting CRUD ==========

// CreateWorldSetting 创建世界观设定
func (d *WorldBuildingDAO) CreateWorldSetting(setting *model.NovelWorldSetting) error {
	return d.db.Create(setting).Error
}

// GetWorldSetting 根据 ID 获取世界观设定
func (d *WorldBuildingDAO) GetWorldSetting(id uint) (*model.NovelWorldSetting, error) {
	var setting model.NovelWorldSetting
	err := d.db.First(&setting, id).Error
	return &setting, err
}

// UpdateWorldSetting 更新世界观设定
func (d *WorldBuildingDAO) UpdateWorldSetting(setting *model.NovelWorldSetting) error {
	return d.db.Save(setting).Error
}

// DeleteWorldSetting 删除世界观设定
func (d *WorldBuildingDAO) DeleteWorldSetting(id uint) error {
	return d.db.Delete(&model.NovelWorldSetting{}, id).Error
}

// ListWorldSettingsByNovel 获取小说下的所有世界观设定
func (d *WorldBuildingDAO) ListWorldSettingsByNovel(novelID uint) ([]model.NovelWorldSetting, error) {
	var settings []model.NovelWorldSetting
	err := d.db.Where("novel_id = ?", novelID).Order("category ASC, id ASC").Find(&settings).Error
	return settings, err
}

// ListWorldSettingsByCategory 按分类获取世界观设定
func (d *WorldBuildingDAO) ListWorldSettingsByCategory(novelID uint, category string) ([]model.NovelWorldSetting, error) {
	var settings []model.NovelWorldSetting
	err := d.db.Where("novel_id = ? AND category = ?", novelID, category).Order("id ASC").Find(&settings).Error
	return settings, err
}

// DeleteWorldSettingsByNovel 删除小说下所有世界观设定（用于重新生成）
func (d *WorldBuildingDAO) DeleteWorldSettingsByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.NovelWorldSetting{}).Error
}

// BatchCreateWorldSettings 批量创建世界观设定
func (d *WorldBuildingDAO) BatchCreateWorldSettings(settings []model.NovelWorldSetting) error {
	if len(settings) == 0 {
		return nil
	}
	return d.db.Create(&settings).Error
}

// ========== NovelForeshadow CRUD ==========

// CreateForeshadow 创建伏笔设定
func (d *WorldBuildingDAO) CreateForeshadow(f *model.NovelForeshadow) error {
	return d.db.Create(f).Error
}

// GetForeshadow 根据 ID 获取伏笔
func (d *WorldBuildingDAO) GetForeshadow(id uint) (*model.NovelForeshadow, error) {
	var f model.NovelForeshadow
	err := d.db.First(&f, id).Error
	return &f, err
}

// UpdateForeshadow 更新伏笔
func (d *WorldBuildingDAO) UpdateForeshadow(f *model.NovelForeshadow) error {
	return d.db.Save(f).Error
}

// DeleteForeshadow 删除伏笔
func (d *WorldBuildingDAO) DeleteForeshadow(id uint) error {
	return d.db.Delete(&model.NovelForeshadow{}, id).Error
}

// ListForeshadowsByNovel 获取小说下的所有伏笔
func (d *WorldBuildingDAO) ListForeshadowsByNovel(novelID uint) ([]model.NovelForeshadow, error) {
	var list []model.NovelForeshadow
	err := d.db.Where("novel_id = ?", novelID).Order("id ASC").Find(&list).Error
	return list, err
}

// DeleteForeshadowsByNovel 删除小说下所有伏笔（用于重新生成）
func (d *WorldBuildingDAO) DeleteForeshadowsByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.NovelForeshadow{}).Error
}

// BatchCreateForeshadows 批量创建伏笔
func (d *WorldBuildingDAO) BatchCreateForeshadows(list []model.NovelForeshadow) error {
	if len(list) == 0 {
		return nil
	}
	return d.db.Create(&list).Error
}

// ========== NovelPlotOutline CRUD ==========

// CreatePlotOutline 创建剧情大纲
func (d *WorldBuildingDAO) CreatePlotOutline(p *model.NovelPlotOutline) error {
	return d.db.Create(p).Error
}

// GetPlotOutline 根据 ID 获取剧情大纲
func (d *WorldBuildingDAO) GetPlotOutline(id uint) (*model.NovelPlotOutline, error) {
	var p model.NovelPlotOutline
	err := d.db.First(&p, id).Error
	return &p, err
}

// UpdatePlotOutline 更新剧情大纲
func (d *WorldBuildingDAO) UpdatePlotOutline(p *model.NovelPlotOutline) error {
	return d.db.Save(p).Error
}

// DeletePlotOutline 删除剧情大纲
func (d *WorldBuildingDAO) DeletePlotOutline(id uint) error {
	return d.db.Delete(&model.NovelPlotOutline{}, id).Error
}

// ListPlotOutlinesByNovel 获取小说下的所有剧情大纲（按幕次和排序）
func (d *WorldBuildingDAO) ListPlotOutlinesByNovel(novelID uint) ([]model.NovelPlotOutline, error) {
	var list []model.NovelPlotOutline
	err := d.db.Where("novel_id = ?", novelID).Order("act ASC, sort_order ASC").Find(&list).Error
	return list, err
}

// DeletePlotOutlinesByNovel 删除小说下所有剧情大纲（用于重新生成）
func (d *WorldBuildingDAO) DeletePlotOutlinesByNovel(novelID uint) error {
	return d.db.Where("novel_id = ?", novelID).Delete(&model.NovelPlotOutline{}).Error
}

// BatchCreatePlotOutlines 批量创建剧情大纲
func (d *WorldBuildingDAO) BatchCreatePlotOutlines(list []model.NovelPlotOutline) error {
	if len(list) == 0 {
		return nil
	}
	return d.db.Create(&list).Error
}

// ========== ReflectionLog CRUD ==========

// CreateReflectionLog 创建反思记录
func (d *WorldBuildingDAO) CreateReflectionLog(log *model.ReflectionLog) error {
	return d.db.Create(log).Error
}

// ListReflectionLogs 获取小说某阶段的所有反思记录
func (d *WorldBuildingDAO) ListReflectionLogs(novelID uint, phase string) ([]model.ReflectionLog, error) {
	var logs []model.ReflectionLog
	err := d.db.Where("novel_id = ? AND phase = ?", novelID, phase).Order("round ASC").Find(&logs).Error
	return logs, err
}

// GetBestReflectionLog 获取某阶段最高分的反思记录
func (d *WorldBuildingDAO) GetBestReflectionLog(novelID uint, phase string) (*model.ReflectionLog, error) {
	var log model.ReflectionLog
	err := d.db.Where("novel_id = ? AND phase = ?", novelID, phase).Order("total_score DESC").First(&log).Error
	return &log, err
}

// DeleteReflectionLogsByPhase 删除某阶段的所有反思记录（用于重新生成）
func (d *WorldBuildingDAO) DeleteReflectionLogsByPhase(novelID uint, phase string) error {
	return d.db.Where("novel_id = ? AND phase = ?", novelID, phase).Delete(&model.ReflectionLog{}).Error
}
