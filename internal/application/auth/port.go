package auth

import (
	"context"

	"git.neolidy.top/neo/flowx/internal/domain/tenant"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*tenant.User, error)
	GetByID(ctx context.Context, id string) (*tenant.User, error)
	Create(ctx context.Context, user *tenant.User) error
}
