// server/internal/service/api_key.go
package service

import (
	"context"
	"errors"

	"story-maker/server/config"
	"story-maker/server/internal/dao"
	"story-maker/server/internal/model"
	"story-maker/server/internal/util"
)

// APIKeyService API Key 服务层，负责加密存储和解密读取
type APIKeyService struct {
	keyDAO     *dao.APIKeyDAO
	encryptKey []byte // AES-256 加密密钥（32 字节）
}

// NewAPIKeyService 创建 APIKeyService 实例
func NewAPIKeyService(keyDAO *dao.APIKeyDAO) *APIKeyService {
	// 从配置获取加密密钥
	encryptKey := []byte(config.Global.Encrypt.Key)
	if len(encryptKey) < 32 {
		// 补齐到 32 字节
		padded := make([]byte, 32)
		copy(padded, encryptKey)
		encryptKey = padded
	}

	return &APIKeyService{
		keyDAO:     keyDAO,
		encryptKey: encryptKey[:32],
	}
}

// APIKeyResponse API Key 响应（脱敏）
type APIKeyResponse struct {
	ID        uint   `json:"id"`
	Provider  string `json:"provider"`
	KeyMask   string `json:"key_mask"` // 脱敏后的 Key（仅显示前4后4位）
	IsDefault bool   `json:"is_default"`
}

// CreateKey 创建 API Key（加密存储）
func (s *APIKeyService) CreateKey(ctx context.Context, userID uint, provider, keyValue string) (*APIKeyResponse, error) {
	if provider == "" || keyValue == "" {
		return nil, errors.New("provider and key_value are required")
	}

	// 校验 Provider 白名单
	if !isValidProvider(provider) {
		return nil, errors.New("invalid provider, supported: kimi, claude, copilot")
	}

	// 加密 Key
	encrypted, err := util.EncryptAES(keyValue, s.encryptKey)
	if err != nil {
		return nil, errors.New("failed to encrypt API key")
	}

	key := &model.APIKey{
		UserID:    userID,
		Provider:  provider,
		KeyValue:  encrypted,
		IsDefault: true,
	}

	if err := s.keyDAO.CreateKey(ctx, key); err != nil {
		return nil, err
	}

	return &APIKeyResponse{
		ID:        key.ID,
		Provider:  key.Provider,
		KeyMask:   maskKey(keyValue),
		IsDefault: key.IsDefault,
	}, nil
}

// GetKeys 获取用户的 API Key 列表（脱敏）
func (s *APIKeyService) GetKeys(ctx context.Context, userID uint) ([]*APIKeyResponse, error) {
	keys, err := s.keyDAO.GetKeys(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		// 解密后脱敏
		decrypted, err := util.DecryptAES(key.KeyValue, s.encryptKey)
		mask := "****"
		if err == nil {
			mask = maskKey(decrypted)
		}

		result = append(result, &APIKeyResponse{
			ID:        key.ID,
			Provider:  key.Provider,
			KeyMask:   mask,
			IsDefault: key.IsDefault,
		})
	}

	return result, nil
}

// UpdateKey 更新 API Key
func (s *APIKeyService) UpdateKey(ctx context.Context, keyID, userID uint, keyValue string, isDefault *bool) error {
	key, err := s.keyDAO.GetKeyByID(ctx, keyID)
	if err != nil {
		return err
	}

	// 权限校验
	if key.UserID != userID {
		return errors.New("permission denied")
	}

	// 更新 Key 值
	if keyValue != "" {
		encrypted, err := util.EncryptAES(keyValue, s.encryptKey)
		if err != nil {
			return errors.New("failed to encrypt API key")
		}
		key.KeyValue = encrypted
	}

	// 更新默认标记
	if isDefault != nil {
		key.IsDefault = *isDefault
	}

	return s.keyDAO.UpdateKey(ctx, key)
}

// DeleteKey 删除 API Key
func (s *APIKeyService) DeleteKey(ctx context.Context, keyID, userID uint) error {
	key, err := s.keyDAO.GetKeyByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key.UserID != userID {
		return errors.New("permission denied")
	}

	return s.keyDAO.DeleteKey(ctx, keyID)
}

// GetUserKey 获取用户指定 Provider 的解密后 Key（供 Dispatcher 调用）
func (s *APIKeyService) GetUserKey(ctx context.Context, userID uint, provider string) (string, error) {
	key, err := s.keyDAO.GetUserKey(ctx, userID, provider)
	if err != nil {
		return "", err
	}

	decrypted, err := util.DecryptAES(key.KeyValue, s.encryptKey)
	if err != nil {
		return "", errors.New("failed to decrypt API key")
	}

	return decrypted, nil
}

// GetDefaultKey 获取平台默认 Key（从配置文件读取）
func (s *APIKeyService) GetDefaultKey(ctx context.Context, provider string) (string, error) {
	var key string
	switch provider {
	case model.ProviderKimi:
		key = config.Global.Kimi.APIKey
	case model.ProviderZhipu:
		key = config.Global.Zhipu.APIKey
	case model.ProviderQwen:
		key = config.Global.Qwen.APIKey
	case model.ProviderDeepseek:
		key = config.Global.Deepseek.APIKey
	case model.ProviderMinimax:
		key = config.Global.MiniMax.APIKey
	default:
		return "", errors.New("no default API key configured for " + provider)
	}
	if key == "" {
		return "", errors.New("no default API key configured for " + provider)
	}
	return key, nil
}

// isValidProvider 校验 Provider 白名单
func isValidProvider(provider string) bool {
	switch provider {
	case model.ProviderKimi, model.ProviderClaude, model.ProviderCopilot, model.ProviderZhipu, model.ProviderQwen, model.ProviderDeepseek, model.ProviderMinimax:
		return true
	default:
		return false
	}
}

// maskKey 脱敏 Key：显示前4后4位，中间用 **** 替代
func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
