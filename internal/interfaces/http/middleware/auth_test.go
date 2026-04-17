package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAuthMiddleware_ValidToken 验证有效 token 通过中间件，设置 user_id 和 tenant_id 到 context
func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	// 生成有效 token
	token, err := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "user-001",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}

	// 创建测试路由
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(AuthMiddleware(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		userID := c.GetString("user_id")
		tenantID := c.GetString("tenant_id")
		roles := c.GetStringSlice("roles")

		response.Success(c, gin.H{
			"user_id":   userID,
			"tenant_id": tenantID,
			"roles":     roles,
		})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("期望响应包含 data 对象，实际响应: %s", w.Body.String())
	}
	if data["user_id"] != "user-001" {
		t.Errorf("期望 user_id 为 'user-001'，实际为 '%v'", data["user_id"])
	}
	if data["tenant_id"] != "tenant-001" {
		t.Errorf("期望 tenant_id 为 'tenant-001'，实际为 '%v'", data["tenant_id"])
	}
}

// TestAuthMiddleware_NoToken 验证无 token 返回 401
func TestAuthMiddleware_NoToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(AuthMiddleware(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}

// TestAuthMiddleware_InvalidToken 验证无效 token 返回 401
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(AuthMiddleware(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid.token.here")
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}

// TestAuthMiddleware_ExpiredToken 验证过期 token 返回 401
func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	// 使用已过期的 JWT 服务生成 token
	jwtService := auth.NewJWTService("test-secret-key-1234567890123456", -1*time.Hour)

	token, err := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "user-001",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}

	// 使用正常过期时间的 JWT 服务验证
	validJwtService := auth.NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(AuthMiddleware(validJwtService))
	r.GET("/protected", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}

// TestAuthMiddleware_MalformedHeader 验证格式错误的 Authorization header 返回 401
func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(AuthMiddleware(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "/protected", nil)
	c.Request.Header.Set("Authorization", "NotBearer sometoken")
	r.ServeHTTP(w, c.Request)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}
