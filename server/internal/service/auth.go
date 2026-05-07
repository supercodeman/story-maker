// server/internal/service/auth.go
package service

import (
	"errors"
	"fmt"
	"time"

	"ai-curton/server/config"
	"ai-curton/server/internal/dao"
	"ai-curton/server/internal/model"

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

	now := time.Now()
	user := &model.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		Role:          "admin",
		WriterLevel:   model.WriterLevelAdvanced,
		ViewMode:      model.ViewModeAdvanced,
		LevelSource:   model.LevelSourceAdmin,
		LevelUnlockAt: &now,
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

// ListUsers 返回所有用户列表（管理员用）
func (s *AuthService) ListUsers() ([]model.User, error) {
	return s.userDAO.ListUsers()
}

// UpdateUserRole 更新用户角色，校验 role 白名单
func (s *AuthService) UpdateUserRole(userID uint, role string) error {
	// 角色白名单校验
	validRoles := map[string]bool{"admin": true, "creator": true, "viewer": true}
	if !validRoles[role] {
		return errors.New("invalid role, must be one of: admin, creator, viewer")
	}

	// 确认用户存在
	_, err := s.userDAO.GetUserByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	return s.userDAO.UpdateRole(userID, role)
}

// generateTokenPair 生成 access_token（短期）和 refresh_token（长期）
func (s *AuthService) generateTokenPair(user *model.User) (*TokenResponse, error) {
	jwtCfg := config.Global.JWT

	// 生成 access_token
	accessClaims := jwt.MapClaims{
		"user_id":      user.ID,
		"username":     user.Username,
		"role":         user.Role,
		"writer_level": user.WriterLevel,
		"type":         "access",
		"exp":          time.Now().Add(time.Duration(jwtCfg.AccessTokenTTL) * time.Second).Unix(),
		"iat":          time.Now().Unix(),
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
	fmt.Println(accessTokenString)
	return &TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    jwtCfg.AccessTokenTTL,
	}, nil
}
