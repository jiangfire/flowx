package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/internal/domain/tenant"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAuthHandlerTest 创建 Auth Handler 测试环境
func setupAuthHandlerTest(t *testing.T) (*AuthHandler, *gin.Engine) {
	t.Helper()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tenant.User{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建服务
	jwtService := auth.NewJWTService("test-secret-key-for-handler-test-12345", 24*time.Hour)
	userRepo := persistence.NewUserRepository(db)
	authService := auth.NewAuthService(userRepo, jwtService)
	authHandler := NewAuthHandler(authService)

	// 创建 Gin 引擎
	gin.SetMode(gin.TestMode)
	r := gin.New()

	return authHandler, r
}

// TestRegister_Success 验证 POST /api/v1/auth/register 正确注册
func TestRegister_Success(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	r.POST("/api/v1/auth/register", h.Register)

	body := map[string]string{
		"username":  "testuser",
		"email":     "test@example.com",
		"password":  "password123",
		"tenant_id": "tenant-001",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Errorf("期望 code 为 0，实际为 %v", resp["code"])
	}

	data := resp["data"].(map[string]any)
	userData := data["user"].(map[string]any)
	if userData["username"] != "testuser" {
		t.Errorf("期望 username 为 'testuser'，实际为 '%v'", userData["username"])
	}
	if data["token"] == nil || data["token"] == "" {
		t.Error("期望响应包含 token")
	}
}

// TestRegister_DuplicateUsername 验证 POST /api/v1/auth/register 重复用户名返回 409
func TestRegister_DuplicateUsername(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	r.POST("/api/v1/auth/register", h.Register)

	body := map[string]string{
		"username":  "testuser",
		"email":     "test@example.com",
		"password":  "password123",
		"tenant_id": "tenant-001",
	}
	jsonBody, _ := json.Marshal(body)

	// 第一次注册
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("第一次注册期望状态码 201，实际为 %d", w1.Code)
	}

	// 第二次注册（重复用户名）
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("期望状态码 409，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestLogin_Success 验证 POST /api/v1/auth/login 正确登录返回 token
func TestLogin_Success(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	// 先注册用户
	r.POST("/api/v1/auth/register", h.Register)
	regBody := map[string]string{
		"username":  "testuser",
		"email":     "test@example.com",
		"password":  "password123",
		"tenant_id": "tenant-001",
	}
	jsonRegBody, _ := json.Marshal(regBody)

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonRegBody))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("注册期望状态码 201，实际为 %d", w1.Code)
	}

	// 登录
	r.POST("/api/v1/auth/login", h.Login)
	loginBody := map[string]string{
		"username": "testuser",
		"password": "password123",
	}
	jsonLoginBody, _ := json.Marshal(loginBody)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonLoginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data := resp["data"].(map[string]any)
	if data["token"] == nil || data["token"] == "" {
		t.Error("期望响应包含 token")
	}
}

// TestLogin_WrongPassword 验证 POST /api/v1/auth/login 错误密码返回 401
func TestLogin_WrongPassword(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	// 先注册用户
	r.POST("/api/v1/auth/register", h.Register)
	regBody := map[string]string{
		"username":  "testuser",
		"email":     "test@example.com",
		"password":  "password123",
		"tenant_id": "tenant-001",
	}
	jsonRegBody, _ := json.Marshal(regBody)

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonRegBody))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	// 使用错误密码登录
	r.POST("/api/v1/auth/login", h.Login)
	loginBody := map[string]string{
		"username": "testuser",
		"password": "wrongpassword",
	}
	jsonLoginBody, _ := json.Marshal(loginBody)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonLoginBody))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestProfile_Success 验证 GET /api/v1/auth/profile 需要认证，返回用户信息
func TestProfile_Success(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	// 先注册用户
	r.POST("/api/v1/auth/register", h.Register)
	regBody := map[string]string{
		"username":  "testuser",
		"email":     "test@example.com",
		"password":  "password123",
		"tenant_id": "tenant-001",
	}
	jsonRegBody, _ := json.Marshal(regBody)

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonRegBody))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	// 从注册响应中获取 token
	var regResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &regResp)
	token := regResp["data"].(map[string]any)["token"].(string)

	// 创建 JWT 服务用于中间件
	jwtService := auth.NewJWTService("test-secret-key-for-handler-test-12345", 24*time.Hour)

	// 设置需要认证的路由
	r.GET("/api/v1/auth/profile", middleware.AuthMiddleware(jwtService), h.Profile)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var profileResp map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &profileResp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	data := profileResp["data"].(map[string]any)
	if data["username"] != "testuser" {
		t.Errorf("期望 username 为 'testuser'，实际为 '%v'", data["username"])
	}
	if data["email"] != "test@example.com" {
		t.Errorf("期望 email 为 'test@example.com'，实际为 '%v'", data["email"])
	}
	// 确保不返回密码
	if _, exists := data["password_hash"]; exists {
		t.Error("期望 profile 不包含 password_hash")
	}
}

// TestProfile_Unauthorized 验证 GET /api/v1/auth/profile 无认证返回 401
func TestProfile_Unauthorized(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	jwtService := auth.NewJWTService("test-secret-key-for-handler-test-12345", 24*time.Hour)
	r.GET("/api/v1/auth/profile", middleware.AuthMiddleware(jwtService), h.Profile)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}

// TestRegister_MissingFields 验证请求参数校验（缺少必填字段返回 422）
func TestRegister_MissingFields(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	r.POST("/api/v1/auth/register", h.Register)

	// 缺少 password
	body := map[string]string{
		"username":  "testuser",
		"email":     "test@example.com",
		"tenant_id": "tenant-001",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestLogin_MissingFields 验证登录请求参数校验（缺少必填字段返回 422）
func TestLogin_MissingFields(t *testing.T) {
	h, r := setupAuthHandlerTest(t)

	r.POST("/api/v1/auth/login", h.Login)

	// 缺少 password
	body := map[string]string{
		"username": "testuser",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}
