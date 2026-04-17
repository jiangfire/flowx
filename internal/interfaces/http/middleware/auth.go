package middleware

import (
	"net/http"
	"strings"

	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 认证中间件
// 从 Authorization header 中提取 Bearer token 并验证
func AuthMiddleware(jwtService auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少认证令牌")
			c.Abort()
			return
		}

		// 验证 Bearer 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "认证令牌格式错误")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析 token
		claims, err := jwtService.ParseToken(tokenString)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "认证令牌无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息设置到 context
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}
