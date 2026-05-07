// server/internal/service/character.go
package service

import (
	"encoding/json"
	"errors"

	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
)

// CharacterService 角色业务逻辑层
type CharacterService struct {
	characterDAO *dao.CharacterDAO
	portfolioDAO *dao.PortfolioDAO
	workspaceDAO *dao.WorkspaceDAO
}

// NewCharacterService 创建 CharacterService 实例
func NewCharacterService() *CharacterService {
	return &CharacterService{
		characterDAO: dao.NewCharacterDAO(),
		portfolioDAO: dao.NewPortfolioDAO(),
		workspaceDAO: dao.NewWorkspaceDAO(),
	}
}

// CreateCharacterRequest 创建角色请求参数
type CreateCharacterRequest struct {
	Name        string            `json:"name" binding:"required,max=100"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"` // 角色属性键值对
}

// UpdateCharacterRequest 更新角色请求参数
type UpdateCharacterRequest struct {
	Name        string            `json:"name" binding:"omitempty,max=100"`
	Description string            `json:"description"`
	Attributes  map[string]string `json:"attributes"`
}

// checkPortfolioEditorPermission 校验用户对作品集所属工作空间的编辑权限
// 遵循 DRY 原则：抽取公共权限校验逻辑
func (s *CharacterService) checkPortfolioEditorPermission(portfolioID, userID uint) (*model.Portfolio, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, errors.New("portfolio not found")
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleEditor)
	if err != nil || !ok {
		return nil, errors.New("access denied: editor permission required")
	}

	return p, nil
}

// Create 创建角色（需 editor 及以上权限）
func (s *CharacterService) Create(portfolioID, userID uint, req *CreateCharacterRequest) (*model.Character, error) {
	_, err := s.checkPortfolioEditorPermission(portfolioID, userID)
	if err != nil {
		return nil, err
	}

	// 序列化 attributes 为 JSON 字符串
	attrJSON := "{}"
	if req.Attributes != nil {
		b, err := json.Marshal(req.Attributes)
		if err != nil {
			return nil, errors.New("invalid attributes format")
		}
		attrJSON = string(b)
	}

	ch := &model.Character{
		PortfolioID:     portfolioID,
		Name:            req.Name,
		Description:     req.Description,
		ReferenceImages: "[]", // 初始化为空数组
		Attributes:      attrJSON,
	}

	if err := s.characterDAO.Create(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// GetByID 获取角色详情（需工作空间成员身份）
func (s *CharacterService) GetByID(characterID, userID uint) (*model.Character, error) {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return nil, err
	}

	p, err := s.portfolioDAO.GetByID(ch.PortfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return ch, nil
}

// List 获取作品集下的角色列表（需工作空间成员身份）
func (s *CharacterService) List(portfolioID, userID uint) ([]model.Character, error) {
	p, err := s.portfolioDAO.GetByID(portfolioID)
	if err != nil {
		return nil, err
	}

	_, err = s.workspaceDAO.GetMember(p.WorkspaceID, userID)
	if err != nil {
		return nil, errors.New("access denied: not a workspace member")
	}

	return s.characterDAO.ListByPortfolioID(portfolioID)
}

// Update 更新角色（需 editor 及以上权限）
func (s *CharacterService) Update(characterID, userID uint, req *UpdateCharacterRequest) (*model.Character, error) {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return nil, err
	}

	_, err = s.checkPortfolioEditorPermission(ch.PortfolioID, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		ch.Name = req.Name
	}
	ch.Description = req.Description
	if req.Attributes != nil {
		b, err := json.Marshal(req.Attributes)
		if err != nil {
			return nil, errors.New("invalid attributes format")
		}
		ch.Attributes = string(b)
	}

	if err := s.characterDAO.Update(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// Delete 删除角色（需 owner 权限）
func (s *CharacterService) Delete(characterID, userID uint) error {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return err
	}

	p, err := s.portfolioDAO.GetByID(ch.PortfolioID)
	if err != nil {
		return err
	}

	ok, err := s.workspaceDAO.CheckPermission(p.WorkspaceID, userID, model.WorkspaceRoleOwner)
	if err != nil || !ok {
		return errors.New("access denied: owner permission required")
	}

	return s.characterDAO.Delete(characterID)
}

// AddReferenceImage 添加参考图到角色的 ReferenceImages JSON 数组
func (s *CharacterService) AddReferenceImage(characterID, userID uint, imagePath string) (*model.Character, error) {
	ch, err := s.characterDAO.GetByID(characterID)
	if err != nil {
		return nil, err
	}

	_, err = s.checkPortfolioEditorPermission(ch.PortfolioID, userID)
	if err != nil {
		return nil, err
	}

	// 解析现有参考图列表
	var images []string
	if ch.ReferenceImages != "" && ch.ReferenceImages != "[]" {
		if err := json.Unmarshal([]byte(ch.ReferenceImages), &images); err != nil {
			images = []string{}
		}
	}

	// 限制最多 5 张参考图（MVP 阶段约束）
	if len(images) >= 5 {
		return nil, errors.New("maximum 5 reference images allowed")
	}

	images = append(images, imagePath)
	b, _ := json.Marshal(images)
	ch.ReferenceImages = string(b)

	if err := s.characterDAO.Update(ch); err != nil {
		return nil, err
	}
	return ch, nil
}
