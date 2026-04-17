package auth

import (
	"context"
	"errors"
	"fmt"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tenant"
	"golang.org/x/crypto/bcrypt"
)

// 定义认证相关错误
var (
	ErrUserAlreadyExists  = errors.New("用户名已存在")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrUserNotFound       = errors.New("用户不存在")
)

// AuthService 认证服务
type AuthService struct {
	userRepo   UserRepository
	jwtService JWTService
}

// NewAuthService 创建认证服务实例
func NewAuthService(userRepo UserRepository, jwtService JWTService) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

// Register 注册新用户
// 返回创建的用户信息和 JWT token
func (s *AuthService) Register(username, email, password, tenantID string) (*tenant.User, string, error) {
	// 检查用户名是否已存在
	_, err := s.userRepo.GetByUsername(context.Background(), username)
	if err == nil {
		return nil, "", ErrUserAlreadyExists
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("密码哈希失败: %w", err)
	}

	// 创建用户
	user := &tenant.User{
		BaseModel:    base.BaseModel{TenantID: tenantID},
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		Status:       "active",
	}

	if err := s.userRepo.Create(context.Background(), user); err != nil {
		return nil, "", fmt.Errorf("创建用户失败: %w", err)
	}

	// 生成 JWT token
	token, err := s.jwtService.GenerateToken(&TokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Roles:    []string{user.Role},
	})
	if err != nil {
		return nil, "", fmt.Errorf("生成令牌失败: %w", err)
	}

	return user, token, nil
}

// Login 用户登录
// 返回 JWT token
func (s *AuthService) Login(username, password string) (string, error) {
	// 查找用户
	user, err := s.userRepo.GetByUsername(context.Background(), username)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// 检查用户状态
	if user.Status != "active" {
		return "", ErrInvalidCredentials
	}

	// 生成 JWT token
	token, err := s.jwtService.GenerateToken(&TokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Roles:    []string{user.Role},
	})
	if err != nil {
		return "", fmt.Errorf("生成令牌失败: %w", err)
	}

	return token, nil
}

// GetProfile 获取用户信息（不含密码）
func (s *AuthService) GetProfile(userID string) (*tenant.User, error) {
	return s.userRepo.GetByID(context.Background(), userID)
}
