package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	notificationapp "git.neolidy.top/neo/flowx/internal/application/notification"
	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/internal/domain/notification"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupNotificationHandlerTest 创建通知 Handler 测试环境
func setupNotificationHandlerTest(t *testing.T) (*NotificationHandler, *gin.Engine, auth.JWTService) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&notification.Notification{},
		&notification.NotificationTemplate{},
		&notification.NotificationPreference{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	notifRepo := persistence.NewNotificationRepository(db)
	templateRepo := persistence.NewNotificationTemplateRepository(db)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)
	notifService := notificationapp.NewNotificationService(notifRepo, templateRepo, preferenceRepo)
	notifHandler := NewNotificationHandler(notifService)

	jwtService := auth.NewJWTService("test-secret-key-for-notif-handler", 24*time.Hour)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	return notifHandler, r, jwtService
}

// setupNotificationHandlerWithAuth 创建带认证的通知 Handler 测试环境
func setupNotificationHandlerWithAuth(t *testing.T) (*NotificationHandler, *gin.Engine, string) {
	t.Helper()
	h, r, jwtService := setupNotificationHandlerTest(t)
	token, err := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "test-user-id",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("生成测试 token 失败: %v", err)
	}

	authMiddleware := middleware.AuthMiddleware(jwtService)
	tenantMiddleware := middleware.TenantMiddleware()

	// 通知路由
	notifRoutes := r.Group("/api/v1/notifications")
	notifRoutes.Use(authMiddleware, tenantMiddleware)
	{
		notifRoutes.POST("", h.CreateNotification)
		notifRoutes.GET("", h.ListNotifications)
		notifRoutes.GET("/unread-count", h.GetUnreadCount)
		notifRoutes.PUT("/read-all", h.MarkAllAsRead)
		notifRoutes.GET("/:id", h.GetNotification)
		notifRoutes.PUT("/:id", h.UpdateNotification)
		notifRoutes.DELETE("/:id", h.DeleteNotification)
		notifRoutes.PUT("/:id/read", h.MarkAsRead)
		notifRoutes.POST("/send", h.SendNotification)
	}

	// 模板路由
	tplRoutes := r.Group("/api/v1/notification-templates")
	tplRoutes.Use(authMiddleware, tenantMiddleware)
	{
		tplRoutes.POST("", h.CreateTemplate)
		tplRoutes.GET("", h.ListTemplates)
		tplRoutes.GET("/:id", h.GetTemplate)
		tplRoutes.PUT("/:id", h.UpdateTemplate)
		tplRoutes.DELETE("/:id", h.DeleteTemplate)
	}

	// 偏好路由
	prefRoutes := r.Group("/api/v1/notification-preferences")
	prefRoutes.Use(authMiddleware, tenantMiddleware)
	{
		prefRoutes.POST("", h.CreatePreference)
		prefRoutes.GET("", h.ListPreferences)
		prefRoutes.PUT("/:id", h.UpdatePreference)
		prefRoutes.DELETE("/:id", h.DeletePreference)
	}

	return h, r, token
}

// ==================== Notification Handler 测试 ====================

// TestCreateNotification_Success POST /notifications 创建成功返回 201
func TestCreateNotification_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	body := map[string]string{
		"title":       "测试通知",
		"type":        "system",
		"receiver_id": "user-001",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("期望 code 为 0，实际为 %v", resp["code"])
	}

	data := resp["data"].(map[string]any)
	if data["title"] != "测试通知" {
		t.Errorf("期望 title 为 '测试通知'，实际为 '%v'", data["title"])
	}
}

// TestCreateNotification_MissingTitle POST /notifications 缺少 title 返回 422
func TestCreateNotification_MissingTitle(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	body := map[string]string{
		"type":        "system",
		"receiver_id": "user-001",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestListNotifications_Success GET /notifications 返回列表
func TestListNotifications_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["total"].(float64) != 0 {
		t.Errorf("期望 total 为 0，实际为 %v", data["total"])
	}
}

// TestGetNotification_NotFound GET /notifications/:id 不存在返回 404
func TestGetNotification_NotFound(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/non-existent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestMarkAsRead_Success PUT /notifications/:id/read 标记已读成功
func TestMarkAsRead_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	// 先创建通知
	body := map[string]string{
		"title":       "测试通知",
		"type":        "system",
		"receiver_id": "test-user-id",
	}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	notifID := createResp["data"].(map[string]any)["id"].(string)

	// 标记已读
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/"+notifID+"/read", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestMarkAllAsRead_Success PUT /notifications/read-all 全部标记已读成功
func TestMarkAllAsRead_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/notifications/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestGetUnreadCount_Success GET /notifications/unread-count 获取未读数量成功
func TestGetUnreadCount_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["count"].(float64) != 0 {
		t.Errorf("期望 count 为 0，实际为 %v", data["count"])
	}
}

// ==================== Template Handler 测试 ====================

// TestCreateTemplate_Success POST /notification-templates 创建成功
func TestCreateTemplate_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	body := map[string]string{
		"name":        "审批通知模板",
		"code":        "approval_notify",
		"type":        "system",
		"channel":     "in_app",
		"title_tpl":   "审批通知 - {{.title}}",
		"content_tpl": "您有一条新的审批任务：{{.content}}",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notification-templates", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["code"] != "approval_notify" {
		t.Errorf("期望 code 为 'approval_notify'，实际为 '%v'", data["code"])
	}
}

// TestListTemplates_Success GET /notification-templates 返回列表
func TestListTemplates_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/notification-templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// ==================== Preference Handler 测试 ====================

// TestCreatePreference_Success POST /notification-preferences 创建成功
func TestCreatePreference_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	body := map[string]string{
		"user_id": "user-001",
		"type":    "system",
		"channel": "in_app",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/notification-preferences", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["user_id"] != "user-001" {
		t.Errorf("期望 user_id 为 'user-001'，实际为 '%v'", data["user_id"])
	}
}

// ==================== SendNotification Handler 测试 ====================

// TestSendNotification_Success POST /notifications/send 发送通知成功
func TestSendNotification_Success(t *testing.T) {
	_, r, token := setupNotificationHandlerWithAuth(t)

	// 先创建模板
	tplBody := map[string]string{
		"name":        "测试模板",
		"code":        "send_test",
		"type":        "system",
		"channel":     "in_app",
		"title_tpl":   "通知 - {{.title}}",
		"content_tpl": "内容：{{.content}}",
	}
	tplJSON, _ := json.Marshal(tplBody)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/notification-templates", bytes.NewBuffer(tplJSON))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	// 发送通知
	sendBody := map[string]any{
		"template_code": "send_test",
		"receiver_id":   "user-001",
		"variables": map[string]string{
			"title":   "测试标题",
			"content": "测试内容",
		},
	}
	sendJSON, _ := json.Marshal(sendBody)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/notifications/send", bytes.NewBuffer(sendJSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["title"] != "通知 - 测试标题" {
		t.Errorf("期望 title 为 '通知 - 测试标题'，实际为 '%v'", data["title"])
	}
}

// ==================== Auth 测试 ====================

// TestNotificationEndpoints_RequireAuth 通知端点需要认证
func TestNotificationEndpoints_RequireAuth(t *testing.T) {
	_, r, _ := setupNotificationHandlerWithAuth(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/notifications"},
		{http.MethodGet, "/api/v1/notifications/unread-count"},
		{http.MethodPost, "/api/v1/notification-templates"},
		{http.MethodGet, "/api/v1/notification-templates"},
		{http.MethodPost, "/api/v1/notification-preferences"},
		{http.MethodGet, "/api/v1/notification-preferences"},
	}

	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(ep.method, ep.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("期望 %s %s 未认证返回 401，实际为 %d", ep.method, ep.path, w.Code)
		}
	}
}
