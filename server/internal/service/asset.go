// server/internal/service/asset.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
	"story-maker/server/internal/storage"
)

// AssetService 资源业务逻辑层
type AssetService struct {
	assetDAO     *dao.AssetDAO
	portfolioDAO *dao.PortfolioDAO
	workspaceDAO *dao.WorkspaceDAO
	storage      storage.Storage
}

// NewAssetService 创建 AssetService 实例
func NewAssetService(store storage.Storage) *AssetService {
	return &AssetService{
		assetDAO:     dao.NewAssetDAO(),
		portfolioDAO: dao.NewPortfolioDAO(),
		workspaceDAO: dao.NewWorkspaceDAO(),
		storage:      store,
	}
}

// Upload 上传资源文件并创建记录
func (s *AssetService) Upload(ctx context.Context, userID uint, portfolioID uint, assetType string, filename string, file io.Reader, metadata map[string]string) (*model.Asset, error) {
	// 校验资源类型白名单
	if !model.ValidAssetTypes[assetType] {
		return nil, errors.New("invalid asset type")
	}

	// 校验用户对作品集所属工作空间的编辑权限
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, errors.New("portfolio not found")
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return nil, errors.New("access denied: editor permission required")
	}

	// 构造存储路径：assets/{portfolio_id}/{timestamp}_{filename}
	storagePath := fmt.Sprintf("assets/%d/%s_%s", portfolioID, timePrefix(), filename)
	url, err := s.storage.Upload(ctx, file, storagePath)
	if err != nil {
		return nil, errors.New("failed to upload file")
	}

	// 序列化 metadata
	metaJSON := "{}"
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		metaJSON = string(b)
	}

	asset := &model.Asset{
		PortfolioID: portfolioID,
		Type:        assetType,
		FilePath:    url,
		Metadata:    metaJSON,
		CreatedBy:   userID,
	}

	if err := s.assetDAO.Create(asset); err != nil {
		return nil, err
	}
	return asset, nil
}

// GetByID 获取资源详情（需工作空间成员身份）
func (s *AssetService) GetByID(assetID, userID uint) (*model.Asset, error) {
	a, err := s.assetDAO.GetByID(assetID)
	if err != nil {
		return nil, err
	}

	p, err := s.portfolioDAO.GetByID(a.PortfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return a, nil
}

// List 获取作品集下的资源列表（需工作空间成员身份）
func (s *AssetService) List(portfolioID, userID uint) ([]model.Asset, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return s.assetDAO.ListByPortfolioID(portfolioID)
}

// Delete 删除资源（需 editor 及以上权限，同时删除存储文件）
func (s *AssetService) Delete(ctx context.Context, assetID, userID uint) error {
	a, err := s.assetDAO.GetByID(assetID)
	if err != nil {
		return err
	}

	p, err := s.portfolioDAO.GetByID(a.PortfolioID)
	if err != nil {
		return err
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return errors.New("access denied: editor permission required")
	}

	// 删除存储文件（忽略文件不存在的错误）
	_ = s.storage.Delete(ctx, a.FilePath)

	return s.assetDAO.Delete(assetID)
}

// timePrefix 辅助函数：生成时间戳前缀
func timePrefix() string {
	return time.Now().Format("20060102150405")
}

// SetCharacterRef 设为角色参考图（同一 portfolio 只保留一张）
func (s *AssetService) SetCharacterRef(ctx context.Context, assetID, userID uint) error {
	a, err := s.assetDAO.GetByID(assetID)
	if err != nil {
		return err
	}
	if a.Type != model.AssetTypeImage {
		return errors.New("only image assets can be set as character reference")
	}

	p, err := s.portfolioDAO.GetByID(a.PortfolioID)
	if err != nil {
		return err
	}
	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return errors.New("access denied: editor permission required")
	}

	return s.assetDAO.SetCharacterRef(assetID, a.PortfolioID)
}

// UnsetCharacterRef 取消角色参考图
func (s *AssetService) UnsetCharacterRef(ctx context.Context, assetID, userID uint) error {
	a, err := s.assetDAO.GetByID(assetID)
	if err != nil {
		return err
	}

	p, err := s.portfolioDAO.GetByID(a.PortfolioID)
	if err != nil {
		return err
	}
	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return errors.New("access denied: editor permission required")
	}

	return s.assetDAO.UnsetCharacterRef(assetID)
}
