package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupToolHandlerTest 创建工具 Handler 测试环境
func setupToolHandlerTest(t *testing.T) (*ToolHandler, *gin.Engine, auth.JWTService) {
	t.Helper()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tool.Tool{}, &tool.Connector{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建服务
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	toolService := toolapp.NewToolService(toolRepo, connectorRepo, nil, nil, nil, nil)
	excelService := toolapp.NewExcelService(toolRepo)
	toolHandler := NewToolHandler(toolService, excelService)

	// 创建 JWT 服务
	jwtService := auth.NewJWTService("test-secret-key-for-tool-handler", 24*time.Hour)

	// 创建 Gin 引擎
	gin.SetMode(gin.TestMode)
	r := gin.New()

	return toolHandler, r, jwtService
}

// generateTestToken 生成测试用 JWT token
func generateTestToken(t *testing.T, jwtService auth.JWTService, tenantID string) string {
	t.Helper()
	token, err := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "test-user-id",
		TenantID: tenantID,
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("生成测试 token 失败: %v", err)
	}
	return token
}

// setupAuthedRouter 设置带认证中间件的路由
func setupAuthedRouter(r *gin.Engine, jwtService auth.JWTService, token string, registerRoutes func(*gin.Engine, *ToolHandler)) {
	authMiddleware := middleware.AuthMiddleware(jwtService)
	tenantMiddleware := middleware.TenantMiddleware()

	r.Use(func(c *gin.Context) {
		c.Request.Header.Set("Authorization", "Bearer "+token)
		c.Next()
	})
	r.Use(authMiddleware, tenantMiddleware)
	registerRoutes(r, nil) // handler 在路由注册时设置
}

// registerToolRoutes 注册工具路由
func registerToolRoutes(r *gin.Engine, h *ToolHandler) {
	tools := r.Group("/api/v1/tools")
	{
		tools.POST("", h.CreateTool)
		tools.GET("", h.ListTools)
		tools.GET("/:id", h.GetTool)
		tools.PUT("/:id", h.UpdateTool)
		tools.DELETE("/:id", h.DeleteTool)
		tools.POST("/export", h.ExportTools)
		tools.POST("/import", h.ImportTools)
		tools.GET("/export/:task_id", h.GetExportStatus)
	}

	connectors := r.Group("/api/v1/connectors")
	{
		connectors.POST("", h.CreateConnector)
		connectors.GET("", h.ListConnectors)
		connectors.GET("/:id", h.GetConnector)
		connectors.PUT("/:id", h.UpdateConnector)
		connectors.DELETE("/:id", h.DeleteConnector)
	}
}

// setupToolHandlerWithAuth 创建带认证的工具 Handler 测试环境
func setupToolHandlerWithAuth(t *testing.T) (*ToolHandler, *gin.Engine, string) {
	t.Helper()
	h, r, jwtService := setupToolHandlerTest(t)
	token := generateTestToken(t, jwtService, "tenant-001")

	// 注册路由（带认证中间件）
	authMiddleware := middleware.AuthMiddleware(jwtService)
	tenantMiddleware := middleware.TenantMiddleware()

	tools := r.Group("/api/v1/tools")
	tools.Use(authMiddleware, tenantMiddleware)
	{
		tools.POST("", h.CreateTool)
		tools.GET("", h.ListTools)
		tools.GET("/:id", h.GetTool)
		tools.PUT("/:id", h.UpdateTool)
		tools.DELETE("/:id", h.DeleteTool)
		tools.POST("/export", h.ExportTools)
		tools.POST("/import", h.ImportTools)
		tools.GET("/export/:task_id", h.GetExportStatus)
	}

	connectors := r.Group("/api/v1/connectors")
	connectors.Use(authMiddleware, tenantMiddleware)
	{
		connectors.POST("", h.CreateConnector)
		connectors.GET("", h.ListConnectors)
		connectors.GET("/:id", h.GetConnector)
		connectors.PUT("/:id", h.UpdateConnector)
		connectors.DELETE("/:id", h.DeleteConnector)
	}

	return h, r, token
}

// ==================== Tool Handler 测试 ====================

// TestCreateTool_Success POST /tools 创建成功返回 201
func TestCreateTool_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	body := map[string]string{
		"name": "Altium Designer",
		"type": "eda",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
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
	if data["name"] != "Altium Designer" {
		t.Errorf("期望 name 为 'Altium Designer'，实际为 '%v'", data["name"])
	}
}

// TestCreateTool_MissingName POST /tools 缺少 name 返回 422
func TestCreateTool_MissingName(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	body := map[string]string{
		"type": "eda",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestListTools GET /tools 返回分页列表
func TestListTools(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tools", nil)
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

// TestListTools_FilterByType GET /tools?type=eda 按类型过滤
func TestListTools_FilterByType(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建两个不同类型的工具
	for _, name := range []string{"Tool1", "Tool2"} {
		typ := "eda"
		if name == "Tool2" {
			typ = "cae"
		}
		body := map[string]string{"name": name, "type": typ}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)
	}

	// 按类型过滤
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tools?type=eda", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["total"].(float64) != 1 {
		t.Errorf("期望过滤后 total 为 1，实际为 %v", data["total"])
	}
}

// TestGetTool_Success GET /tools/:id 返回详情
func TestGetTool_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建工具
	body := map[string]string{"name": "TestTool", "type": "eda"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	toolID := createResp["data"].(map[string]any)["id"].(string)

	// 查询详情
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/tools/"+toolID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var getResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &getResp)
	data := getResp["data"].(map[string]any)
	if data["name"] != "TestTool" {
		t.Errorf("期望 name 为 'TestTool'，实际为 '%v'", data["name"])
	}
}

// TestGetTool_NotFound GET /tools/:id 不存在返回 404
func TestGetTool_NotFound(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tools/non-existent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestUpdateTool_Success PUT /tools/:id 更新成功
func TestUpdateTool_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建工具
	body := map[string]string{"name": "OldName", "type": "eda"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	toolID := createResp["data"].(map[string]any)["id"].(string)

	// 更新
	updateBody := map[string]string{"name": "NewName"}
	updateJSON, _ := json.Marshal(updateBody)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPut, "/api/v1/tools/"+toolID, bytes.NewBuffer(updateJSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var updateResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &updateResp)
	data := updateResp["data"].(map[string]any)
	if data["name"] != "NewName" {
		t.Errorf("期望 name 为 'NewName'，实际为 '%v'", data["name"])
	}
}

// TestDeleteTool_Success DELETE /tools/:id 删除成功
func TestDeleteTool_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建工具
	body := map[string]string{"name": "ToDelete", "type": "eda"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	toolID := createResp["data"].(map[string]any)["id"].(string)

	// 删除
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodDelete, "/api/v1/tools/"+toolID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestExportTools POST /tools/export 创建导出任务返回 task_id
func TestExportTools(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建一个工具
	body := map[string]string{"name": "TestTool", "type": "eda"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/tools", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	// 导出
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tools/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	// 验证返回的是 xlsx 文件
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("期望 Content-Type 为 xlsx，实际为 '%s'", contentType)
	}
}

// TestGetExportStatus GET /tools/export/:task_id 返回任务状态
func TestGetExportStatus(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 查询任务状态（使用任意 task_id）
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tools/export/test-task-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var statusResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &statusResp)
	data := statusResp["data"].(map[string]any)
	if data["status"] == nil {
		t.Error("期望响应包含 status")
	}
}

// TestImportTools POST /tools/import 导入 Excel
func TestImportTools(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 创建测试 xlsx 文件
	f := excelize.NewFile()
	f.SetSheetRow("Sheet1", "A1", &[]string{"name", "type", "status"})
	f.SetSheetRow("Sheet1", "A2", &[]string{"ImportedTool", "eda", "active"})
	buf, _ := f.WriteToBuffer()

	// 创建 multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, _ := writer.CreateFormFile("file", "tools.xlsx")
	part.Write(buf.Bytes())
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/tools/import", &requestBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// ==================== Connector Handler 测试 ====================

// TestCreateConnector_Success POST /connectors 创建成功
func TestCreateConnector_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	body := map[string]string{
		"name":     "Windchill",
		"type":     "plm",
		"endpoint": "https://plm.example.com",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/connectors", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["name"] != "Windchill" {
		t.Errorf("期望 name 为 'Windchill'，实际为 '%v'", data["name"])
	}
}

// TestListConnectors GET /connectors 返回列表
func TestListConnectors(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestGetConnector_Success GET /connectors/:id 返回详情
func TestGetConnector_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建连接器
	body := map[string]string{
		"name":     "Windchill",
		"type":     "plm",
		"endpoint": "https://plm.example.com",
	}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/connectors", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	connID := createResp["data"].(map[string]any)["id"].(string)

	// 查询详情
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors/"+connID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestGetConnector_NotFound GET /connectors/:id 不存在返回 404
func TestGetConnector_NotFound(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/connectors/non-existent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestUpdateConnector_Success PUT /connectors/:id 更新成功
func TestUpdateConnector_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建连接器
	body := map[string]string{
		"name":     "OldName",
		"type":     "plm",
		"endpoint": "https://old.example.com",
	}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/connectors", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	connID := createResp["data"].(map[string]any)["id"].(string)

	// 更新
	updateBody := map[string]string{"name": "NewName"}
	updateJSON, _ := json.Marshal(updateBody)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPut, "/api/v1/connectors/"+connID, bytes.NewBuffer(updateJSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var updateResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &updateResp)
	data := updateResp["data"].(map[string]any)
	if data["name"] != "NewName" {
		t.Errorf("期望 name 为 'NewName'，实际为 '%v'", data["name"])
	}
}

// TestDeleteConnector_Success DELETE /connectors/:id 删除成功
func TestDeleteConnector_Success(t *testing.T) {
	_, r, token := setupToolHandlerWithAuth(t)

	// 先创建连接器
	body := map[string]string{
		"name":     "ToDelete",
		"type":     "plm",
		"endpoint": "https://delete.example.com",
	}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/connectors", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	connID := createResp["data"].(map[string]any)["id"].(string)

	// 删除
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodDelete, "/api/v1/connectors/"+connID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestToolEndpoints_RequireAuth 所有端点需要认证中间件
func TestToolEndpoints_RequireAuth(t *testing.T) {
	_, r, _ := setupToolHandlerWithAuth(t)

	// 不带 token 访问
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望未认证返回 401，实际为 %d", w.Code)
	}
}
