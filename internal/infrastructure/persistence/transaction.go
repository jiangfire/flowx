package persistence

import (
	"context"

	"git.neolidy.top/neo/flowx/pkg/transaction"
	"gorm.io/gorm"
)

// DBFromContext delegates to pkg/transaction.DBFromContext to ensure
// transaction context propagation works across package boundaries.
func DBFromContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	return transaction.DBFromContext(ctx, db)
}
