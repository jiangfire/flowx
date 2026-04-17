package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/application/auth"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDatagovHandlerTest 创建数据治理 Handler 测试环境
func setupDatagovHandlerTest(t *testing.T) (*DataGovHandler, *gin.Engine, auth.JWTService) {
	t.Helper()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&datagov.DataPolicy{}, &datagov.DataAsset{}, &datagov.DataQualityRule{}, &datagov.DataQualityCheck{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建服务
	policyRepo := persistence.NewDataPolicyRepository(db)
	assetRepo := persistence.NewDataAssetRepository(db)
	ruleRepo := persistence.NewDataQualityRuleRepository(db)
	checkRepo := persistence.NewDataQualityCheckRepository(db)

	service := datagovapp.NewDataGovService(policyRepo, assetRepo, ruleRepo, checkRepo)
	excelService := datagovapp.NewDataGovExcelService(policyRepo, assetRepo, ruleRepo)
	handler := NewDataGovHandler(service, excelService)

	// 创建 JWT 服务
	jwtService := auth.NewJWTService("test-secret-key-for-datagov-handler", 24*time.Hour)

	// 创建 Gin 引擎
	gin.SetMode(gin.TestMode)
	r := gin.New()

	return handler, r, jwtService
}

// setupDatagovHandlerWithAuth 创建带认证的数据治理 Handler 测试环境
func setupDatagovHandlerWithAuth(t *testing.T) (*DataGovHandler, *gin.Engine, string) {
	t.Helper()
	h, r, jwtService := setupDatagovHandlerTest(t)
	token, err := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "test-user-id",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("生成测试 token 失败: %v", err)
	}

	// 注册路由（带认证中间件）
	authMiddleware := middleware.AuthMiddleware(jwtService)
	tenantMiddleware := middleware.TenantMiddleware()

	// 数据策略路由
	policies := r.Group("/api/v1/data-policies")
	policies.Use(authMiddleware, tenantMiddleware)
	{
		policies.POST("", h.CreatePolicy)
		policies.GET("", h.ListPolicies)
		policies.GET("/:id", h.GetPolicy)
		policies.PUT("/:id", h.UpdatePolicy)
		policies.DELETE("/:id", h.DeletePolicy)
		policies.POST("/export", h.ExportPolicies)
		policies.POST("/import", h.ImportPolicies)
	}

	// 数据资产路由
	assets := r.Group("/api/v1/data-assets")
	assets.Use(authMiddleware, tenantMiddleware)
	{
		assets.POST("", h.CreateAsset)
		assets.GET("", h.ListAssets)
		assets.GET("/:id", h.GetAsset)
		assets.PUT("/:id", h.UpdateAsset)
		assets.DELETE("/:id", h.DeleteAsset)
		assets.POST("/export", h.ExportAssets)
		assets.POST("/import", h.ImportAssets)
	}

	// 数据质量规则路由
	rules := r.Group("/api/v1/data-quality/rules")
	rules.Use(authMiddleware, tenantMiddleware)
	{
		rules.POST("", h.CreateRule)
		rules.GET("", h.ListRules)
		rules.GET("/:id", h.GetRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
		rules.POST("/export", h.ExportRules)
		rules.POST("/import", h.ImportRules)
	}

	// 数据质量检查路由
	checks := r.Group("/api/v1/data-quality/checks")
	checks.Use(authMiddleware, tenantMiddleware)
	{
		checks.GET("", h.ListChecks)
		checks.GET("/:id", h.GetCheck)
		checks.POST("/run", h.RunQualityCheck)
	}

	return h, r, token
}

// ==================== 数据策略 Handler 测试 ====================

// TestCreatePolicy_Success POST /data-policies 创建成功返回 201
func TestCreatePolicy_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	body := map[string]string{
		"name": "数据保留策略",
		"type": "retention",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies", bytes.NewBuffer(jsonBody))
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
	if data["name"] != "数据保留策略" {
		t.Errorf("期望 name 为 '数据保留策略'，实际为 '%v'", data["name"])
	}
}

// TestCreatePolicy_MissingName POST /data-policies 缺少 name 返回 422
func TestCreatePolicy_MissingName(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	body := map[string]string{
		"type": "retention",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestListPolicies_Success GET /data-policies 返回分页列表
func TestListPolicies_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-policies", nil)
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

// TestGetPolicy_Success GET /data-policies/:id 返回详情
func TestGetPolicy_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	// 先创建策略
	body := map[string]string{"name": "TestPolicy", "type": "retention"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	policyID := createResp["data"].(map[string]any)["id"].(string)

	// 查询详情
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/data-policies/"+policyID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var getResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &getResp)
	data := getResp["data"].(map[string]any)
	if data["name"] != "TestPolicy" {
		t.Errorf("期望 name 为 'TestPolicy'，实际为 '%v'", data["name"])
	}
}

// TestGetPolicy_NotFound GET /data-policies/:id 不存在返回 404
func TestGetPolicy_NotFound(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-policies/non-existent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestUpdatePolicy_Success PUT /data-policies/:id 更新成功
func TestUpdatePolicy_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	// 先创建策略
	body := map[string]string{"name": "OldName", "type": "retention"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	policyID := createResp["data"].(map[string]any)["id"].(string)

	// 更新
	updateBody := map[string]string{"name": "NewName"}
	updateJSON, _ := json.Marshal(updateBody)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPut, "/api/v1/data-policies/"+policyID, bytes.NewBuffer(updateJSON))
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

// TestDeletePolicy_Success DELETE /data-policies/:id 删除成功
func TestDeletePolicy_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	// 先创建策略
	body := map[string]string{"name": "ToDelete", "type": "retention"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var createResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &createResp)
	policyID := createResp["data"].(map[string]any)["id"].(string)

	// 删除
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodDelete, "/api/v1/data-policies/"+policyID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// ==================== 数据资产 Handler 测试 ====================

// TestCreateAsset_Success POST /data-assets 创建成功返回 201
func TestCreateAsset_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	body := map[string]string{
		"name": "EDA设计数据集",
		"type": "dataset",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/data-assets", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["name"] != "EDA设计数据集" {
		t.Errorf("期望 name 为 'EDA设计数据集'，实际为 '%v'", data["name"])
	}
}

// TestListAssets_Success GET /data-assets 返回分页列表
func TestListAssets_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-assets", nil)
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

// TestGetAsset_NotFound GET /data-assets/:id 不存在返回 404
func TestGetAsset_NotFound(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-assets/non-existent-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// ==================== 数据质量规则 Handler 测试 ====================

// TestCreateRule_Success POST /data-quality/rules 创建成功返回 201
func TestCreateRule_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	body := map[string]string{
		"name": "非空检查规则",
		"type": "not_null",
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/data-quality/rules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["name"] != "非空检查规则" {
		t.Errorf("期望 name 为 '非空检查规则'，实际为 '%v'", data["name"])
	}
}

// TestListRules_Success GET /data-quality/rules 返回分页列表
func TestListRules_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-quality/rules", nil)
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

// ==================== 数据质量检查 Handler 测试 ====================

// TestListChecks_Success GET /data-quality/checks 返回分页列表
func TestListChecks_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-quality/checks", nil)
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

// TestRunQualityCheck_Success POST /data-quality/checks/run 执行质量检查
func TestRunQualityCheck_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	// 先创建资产
	assetBody := map[string]string{"name": "TestAsset", "type": "dataset"}
	assetJSON, _ := json.Marshal(assetBody)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/data-assets", bytes.NewBuffer(assetJSON))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	var assetResp map[string]any
	json.Unmarshal(w1.Body.Bytes(), &assetResp)
	assetID := assetResp["data"].(map[string]any)["id"].(string)

	// 先创建规则
	ruleBody := map[string]string{"name": "TestRule", "type": "not_null"}
	ruleJSON, _ := json.Marshal(ruleBody)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/data-quality/rules", bytes.NewBuffer(ruleJSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	var ruleResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &ruleResp)
	ruleID := ruleResp["data"].(map[string]any)["id"].(string)

	// 执行质量检查
	checkBody := map[string]string{"rule_id": ruleID, "asset_id": assetID}
	checkJSON, _ := json.Marshal(checkBody)
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/api/v1/data-quality/checks/run", bytes.NewBuffer(checkJSON))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w3.Code, w3.Body.String())
	}

	var checkResp map[string]any
	json.Unmarshal(w3.Body.Bytes(), &checkResp)
	data := checkResp["data"].(map[string]any)
	if data["status"] != "passed" {
		t.Errorf("期望 status 为 'passed'，实际为 '%v'", data["status"])
	}
}

// ==================== Excel 导入导出测试 ====================

// TestExportPolicies_Success POST /data-policies/export 导出 xlsx
func TestExportPolicies_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	// 先创建一个策略
	body := map[string]string{"name": "TestPolicy", "type": "retention"}
	jsonBody, _ := json.Marshal(body)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)

	// 导出
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("期望 Content-Type 为 xlsx，实际为 '%s'", contentType)
	}
}

// TestImportPolicies_Success POST /data-policies/import 导入 Excel
func TestImportPolicies_Success(t *testing.T) {
	_, r, token := setupDatagovHandlerWithAuth(t)

	// 创建测试 xlsx 文件
	f := excelize.NewFile()
	f.SetSheetRow("Sheet1", "A1", &[]string{"name", "type", "status"})
	f.SetSheetRow("Sheet1", "A2", &[]string{"ImportedPolicy", "retention", "active"})
	buf, _ := f.WriteToBuffer()

	// 创建 multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, _ := writer.CreateFormFile("file", "policies.xlsx")
	part.Write(buf.Bytes())
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/data-policies/import", &requestBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// ==================== 认证测试 ====================

// TestDataGovEndpoints_RequireAuth 所有端点需要认证中间件
func TestDataGovEndpoints_RequireAuth(t *testing.T) {
	_, r, _ := setupDatagovHandlerWithAuth(t)

	// 不带 token 访问
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/data-policies", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望未认证返回 401，实际为 %d", w.Code)
	}
}
