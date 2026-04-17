package persistence

import (
	"context"
	"errors"
	"fmt"

	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tenant"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) auth.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*tenant.User, error) {
	var user tenant.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*tenant.User, error) {
	var user tenant.User
	if err := r.db.WithContext(ctx).Select("id", "tenant_id", "username", "email", "role", "status", "created_at", "updated_at").
		Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *tenant.User) error {
	if user.ID == "" {
		user.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(user).Error
}
