package tenant

import "context"

// contextKey 自定义类型，避免 context key 冲突
type contextKey struct{}

// WithTenantID 将 tenantID 注入 context
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextKey{}, tenantID)
}

// TenantIDFromContext 从 context 提取 tenantID
func TenantIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKey{}).(string); ok {
		return v
	}
	return ""
}
