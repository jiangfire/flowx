package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// TestTenantMiddleware_ValidTenantID 验证从 context 获取 tenant_id 并设置到 GIN context
func TestTenantMiddleware_ValidTenantID(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	// 使用前置中间件模拟认证中间件设置 tenant_id
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-001")
		c.Next()
	})
	r.Use(TenantMiddleware())
	r.GET("/test", func(c *gin.Context) {
		tenantID := c.GetString("tenant_id")
		response.Success(c, gin.H{
			"tenant_id": tenantID,
		})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

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
	if data["tenant_id"] != "tenant-001" {
		t.Errorf("期望 tenant_id 为 'tenant-001'，实际为 '%v'", data["tenant_id"])
	}
}

// TestTenantMiddleware_MissingTenantID 验证缺少 tenant_id 返回错误
func TestTenantMiddleware_MissingTenantID(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(TenantMiddleware())
	r.GET("/test", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400，实际为 %d", w.Code)
	}
}
