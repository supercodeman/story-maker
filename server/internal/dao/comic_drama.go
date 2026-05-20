// server/internal/dao/comic_drama.go
package dao

import (
	"context"

	"story-maker/server/internal/model"

	"gorm.io/gorm"
)

// ComicDramaDAO 漫剧数据访问层
type ComicDramaDAO struct {
	db *gorm.DB
}

func NewComicDramaDAO(db *gorm.DB) *ComicDramaDAO {
	return &ComicDramaDAO{db: db}
}

func (d *ComicDramaDAO) CreateComicDrama(ctx context.Context, drama *model.ComicDrama) error {
	return d.db.WithContext(ctx).Create(drama).Error
}

func (d *ComicDramaDAO) GetComicDrama(ctx context.Context, id uint) (*model.ComicDrama, error) {
	var drama model.ComicDrama
	err := d.db.WithContext(ctx).First(&drama, id).Error
	if err != nil {
		return nil, err
	}
	return &drama, nil
}

func (d *ComicDramaDAO) UpdateComicDrama(ctx context.Context, drama *model.ComicDrama) error {
	return d.db.WithContext(ctx).Save(drama).Error
}

func (d *ComicDramaDAO) DeleteComicDrama(ctx context.Context, id uint) error {
	return d.db.WithContext(ctx).Delete(&model.ComicDrama{}, id).Error
}

func (d *ComicDramaDAO) ListComicDramasByUser(ctx context.Context, userID uint, limit, offset int) ([]*model.ComicDrama, int64, error) {
	var dramas []*model.ComicDrama
	var total int64
	query := d.db.WithContext(ctx).Where("user_id = ?", userID)
	if err := query.Model(&model.ComicDrama{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&dramas).Error
	return dramas, total, err
}

func (d *ComicDramaDAO) UpdateComicDramaStage(ctx context.Context, id uint, stage string, stageIndex int, status string) error {
	return d.db.WithContext(ctx).Model(&model.ComicDrama{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"stage":       stage,
			"stage_index": stageIndex,
			"status":      status,
		}).Error
}

func (d *ComicDramaDAO) BatchCreateScripts(ctx context.Context, scripts []*model.ComicScript) error {
	return d.db.WithContext(ctx).Create(&scripts).Error
}

func (d *ComicDramaDAO) ListScriptsByDrama(ctx context.Context, dramaID uint) ([]*model.ComicScript, error) {
	var scripts []*model.ComicScript
	err := d.db.WithContext(ctx).Where("comic_drama_id = ?", dramaID).Order("seq_no ASC").Find(&scripts).Error
	return scripts, err
}

func (d *ComicDramaDAO) UpdateScript(ctx context.Context, script *model.ComicScript) error {
	return d.db.WithContext(ctx).Save(script).Error
}

func (d *ComicDramaDAO) DeleteScriptsByDrama(ctx context.Context, dramaID uint) error {
	return d.db.WithContext(ctx).Where("comic_drama_id = ?", dramaID).Delete(&model.ComicScript{}).Error
}

func (d *ComicDramaDAO) BatchCreateStoryboards(ctx context.Context, boards []*model.Storyboard) error {
	return d.db.WithContext(ctx).Create(&boards).Error
}

func (d *ComicDramaDAO) ListStoryboardsByDrama(ctx context.Context, dramaID uint) ([]*model.Storyboard, error) {
	var boards []*model.Storyboard
	err := d.db.WithContext(ctx).Where("comic_drama_id = ?", dramaID).Order("seq_no ASC").Find(&boards).Error
	return boards, err
}

func (d *ComicDramaDAO) GetStoryboard(ctx context.Context, id uint) (*model.Storyboard, error) {
	var board model.Storyboard
	err := d.db.WithContext(ctx).First(&board, id).Error
	if err != nil {
		return nil, err
	}
	return &board, nil
}

func (d *ComicDramaDAO) UpdateStoryboard(ctx context.Context, board *model.Storyboard) error {
	return d.db.WithContext(ctx).Save(board).Error
}

func (d *ComicDramaDAO) UpdateStoryboardStatus(ctx context.Context, ids []uint, status string) error {
	return d.db.WithContext(ctx).Model(&model.Storyboard{}).Where("id IN ?", ids).Update("status", status).Error
}

func (d *ComicDramaDAO) CountStoryboardsByStatus(ctx context.Context, dramaID uint, status string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.Storyboard{}).
		Where("comic_drama_id = ? AND status = ?", dramaID, status).
		Count(&count).Error
	return count, err
}

func (d *ComicDramaDAO) BatchCreateCharacterRefs(ctx context.Context, refs []*model.CharacterRef) error {
	return d.db.WithContext(ctx).Create(&refs).Error
}

func (d *ComicDramaDAO) ListCharacterRefsByDrama(ctx context.Context, dramaID uint) ([]*model.CharacterRef, error) {
	var refs []*model.CharacterRef
	err := d.db.WithContext(ctx).Where("comic_drama_id = ?", dramaID).Find(&refs).Error
	return refs, err
}

func (d *ComicDramaDAO) GetCharacterRef(ctx context.Context, id uint) (*model.CharacterRef, error) {
	var ref model.CharacterRef
	err := d.db.WithContext(ctx).First(&ref, id).Error
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (d *ComicDramaDAO) UpdateCharacterRef(ctx context.Context, ref *model.CharacterRef) error {
	return d.db.WithContext(ctx).Save(ref).Error
}
