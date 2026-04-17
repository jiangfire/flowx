package crud

import (
	"context"
	"errors"
)

// TenantOwned 租户隔离实体接口
type TenantOwned interface {
	GetTenantID() string
}

// ErrNotFound 未找到错误
var ErrNotFound = errors.New("记录不存在")

// ErrTenantMismatch 租户不匹配错误
var ErrTenantMismatch = errors.New("租户不匹配")

// GetAndValidateTenant 获取实体并校验租户归属
// ctx: 上下文
// getFn: 仓储的 GetByID 方法
// tenantID: 当前租户 ID
// id: 实体 ID
// notFoundErr: 未找到时返回的业务错误
func GetAndValidateTenant[T any](ctx context.Context, getFn func(ctx context.Context, id string) (*T, error), tenantID, id string, notFoundErr error) (*T, error) {
	entity, err := getFn(ctx, id)
	if err != nil {
		return nil, notFoundErr
	}
	// Type assertion for TenantOwned
	if owned, ok := any(entity).(TenantOwned); ok {
		if owned.GetTenantID() != tenantID {
			return nil, ErrTenantMismatch
		}
	}
	return entity, nil
}
