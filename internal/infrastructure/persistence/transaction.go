package persistence

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// WithTransaction 在事务中执行函数，自动处理提交和回滚
func WithTransaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// DBFromContext 从 context 中获取事务连接，如果没有则返回默认连接
func DBFromContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db
}
