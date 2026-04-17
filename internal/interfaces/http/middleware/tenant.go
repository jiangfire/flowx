package middleware

import (
	"net/http"

	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// TenantMiddleware 多租户中间件
// 防御性校验：确保 tenant_id 已由 AuthMiddleware 正确注入。
// 如果 AuthMiddleware 被绕过或配置错误，这里会拦截请求。
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		if tenantID == "" {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "缺少租户标识")
			c.Abort()
			return
		}

		// 将 tenant_id 设置到 GIN context（已在认证中间件中设置，这里做二次校验）
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}
