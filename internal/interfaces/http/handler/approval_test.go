package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	approvalservice "git.neolidy.top/neo/flowx/internal/application/approval"
	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupApprovalHandlerTest 创建审批 Handler 测试环境
func setupApprovalHandlerTest(t *testing.T) (*ApprovalHandler, *gin.Engine, string) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&approval.Workflow{}, &approval.WorkflowInstance{}, &approval.Approval{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	repo := persistence.NewApprovalRepository(db)
	svc := approvalservice.NewApprovalService(repo, nil)
	handler := NewApprovalHandler(svc)

	jwtService := auth.NewJWTService("test-secret-key-approval-handler-12345", 24*time.Hour)
	token, _ := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "user-001",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()

	return handler, r, token
}

// registerApprovalRoutes 注册审批路由（带认证中间件）
func registerApprovalRoutes(r *gin.Engine, h *ApprovalHandler, token string) {
	jwtService := auth.NewJWTService("test-secret-key-approval-handler-12345", 24*time.Hour)

	// 工作流路由
	r.POST("/api/v1/workflows", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.CreateWorkflow)
	r.GET("/api/v1/workflows", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.ListWorkflows)
	r.GET("/api/v1/workflows/:id", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.GetWorkflow)
	r.POST("/api/v1/workflows/:id/activate", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.ActivateWorkflow)
	r.POST("/api/v1/workflows/:id/archive", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.ArchiveWorkflow)

	// 审批路由
	r.POST("/api/v1/approvals", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.StartApproval)
	r.GET("/api/v1/approvals", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.ListInstances)
	r.GET("/api/v1/approvals/pending", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.GetMyPendingApprovals)
	r.GET("/api/v1/approvals/:id", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.GetInstance)
	r.POST("/api/v1/approvals/:id/approve", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.Approve)
	r.POST("/api/v1/approvals/:id/reject", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.Reject)
	r.POST("/api/v1/approvals/:id/forward", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.Forward)
	r.POST("/api/v1/approvals/:id/cancel", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.CancelInstance)
	r.GET("/api/v1/approvals/:id/suggestion", middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware(), h.GetSuggestion)
}

// authedRequest 创建带认证的请求
func authedRequest(method, url, token string, body any) (*http.Request, string) {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBody)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return req, "application/json"
}

// createTestWorkflowAndInstance 创建测试用工作流和实例，返回实例 ID
func createTestWorkflowAndInstance(t *testing.T, h *ApprovalHandler, r *gin.Engine, token string) string {
	t.Helper()

	// 创建工作流
	wfBody := map[string]any{
		"name":        "工具部署审批",
		"type":        "tool_deploy",
		"description": "部署工具前需要审批",
		"definition": map[string]any{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
				{"name": "负责人审批", "approvers": []string{"approver-2"}},
			},
		},
	}
	req, _ := authedRequest(http.MethodPost, "/api/v1/workflows", token, wfBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var wfResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &wfResp)
	wfID := wfResp["data"].(map[string]any)["id"].(string)

	// 激活工作流
	reqActivate, _ := authedRequest(http.MethodPost, "/api/v1/workflows/"+wfID+"/activate", token, nil)
	wActivate := httptest.NewRecorder()
	r.ServeHTTP(wActivate, reqActivate)

	// 发起审批
	instBody := map[string]any{
		"workflow_id": wfID,
		"title":       "部署 nginx",
		"context":     map[string]any{"tool_name": "nginx"},
	}
	req2, _ := authedRequest(http.MethodPost, "/api/v1/approvals", token, instBody)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var instResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &instResp)
	return instResp["data"].(map[string]any)["id"].(string)
}

// TestPostWorkflows_Success POST /workflows 创建成功返回 201
func TestPostWorkflows_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	body := map[string]any{
		"name":        "工具部署审批",
		"type":        "tool_deploy",
		"description": "部署工具前需要审批",
		"definition": map[string]any{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
			},
		},
	}
	req, _ := authedRequest(http.MethodPost, "/api/v1/workflows", token, body)
	w := httptest.NewRecorder()
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
	if data["name"] != "工具部署审批" {
		t.Errorf("期望 name 为 '工具部署审批'，实际为 '%v'", data["name"])
	}
}

// TestPostApprovals_Success POST /approvals 发起审批返回 201
func TestPostApprovals_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	// 先创建工作流
	wfBody := map[string]any{
		"name": "工具部署审批",
		"type": "tool_deploy",
		"definition": map[string]any{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
			},
		},
	}
	req, _ := authedRequest(http.MethodPost, "/api/v1/workflows", token, wfBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var wfResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &wfResp)
	wfID := wfResp["data"].(map[string]any)["id"].(string)

	// 激活工作流
	reqActivate, _ := authedRequest(http.MethodPost, "/api/v1/workflows/"+wfID+"/activate", token, nil)
	wActivate := httptest.NewRecorder()
	r.ServeHTTP(wActivate, reqActivate)

	// 发起审批
	instBody := map[string]any{
		"workflow_id": wfID,
		"title":       "部署 nginx",
	}
	req2, _ := authedRequest(http.MethodPost, "/api/v1/approvals", token, instBody)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}
}

// TestPostApprove_Success POST /approvals/:id/approve 审批通过
func TestPostApprove_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	instID := createTestWorkflowAndInstance(t, h, r, token)

	// 审批通过（注意：approver-1 是审批人，但当前用户是 user-001）
	// 需要用 approver-1 的 token
	jwtService := auth.NewJWTService("test-secret-key-approval-handler-12345", 24*time.Hour)
	approverToken, _ := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "approver-1",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})

	body := map[string]any{"comment": "同意部署"}
	req, _ := authedRequest(http.MethodPost, "/api/v1/approvals/"+instID+"/approve", approverToken, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "approved" {
		t.Errorf("期望审批状态为 'approved'，实际为 '%v'", data["status"])
	}
}

// TestPostReject_Success POST /approvals/:id/reject 审批驳回
func TestPostReject_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	instID := createTestWorkflowAndInstance(t, h, r, token)

	jwtService := auth.NewJWTService("test-secret-key-approval-handler-12345", 24*time.Hour)
	approverToken, _ := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "approver-1",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})

	body := map[string]any{"comment": "工具版本不合规"}
	req, _ := authedRequest(http.MethodPost, "/api/v1/approvals/"+instID+"/reject", approverToken, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "rejected" {
		t.Errorf("期望审批状态为 'rejected'，实际为 '%v'", data["status"])
	}
}

// TestPostForward_Success POST /approvals/:id/forward 转审
func TestPostForward_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	instID := createTestWorkflowAndInstance(t, h, r, token)

	jwtService := auth.NewJWTService("test-secret-key-approval-handler-12345", 24*time.Hour)
	approverToken, _ := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "approver-1",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})

	body := map[string]any{"to_approver_id": "approver-3", "comment": "转给更专业的同事"}
	req, _ := authedRequest(http.MethodPost, "/api/v1/approvals/"+instID+"/forward", approverToken, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "forwarded" {
		t.Errorf("期望审批状态为 'forwarded'，实际为 '%v'", data["status"])
	}
}

// TestPostCancel_Success POST /approvals/:id/cancel 取消
func TestPostCancel_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	instID := createTestWorkflowAndInstance(t, h, r, token)

	req, _ := authedRequest(http.MethodPost, "/api/v1/approvals/"+instID+"/cancel", token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestGetSuggestion_NoLLM GET /approvals/:id/suggestion 无 LLM 服务返回错误
func TestGetSuggestion_NoLLM(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	instID := createTestWorkflowAndInstance(t, h, r, token)

	req, _ := authedRequest(http.MethodGet, "/api/v1/approvals/"+instID+"/suggestion", token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// LLM 未配置应返回 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望状态码 500，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestGetPendingApprovals_Success GET /approvals/pending 返回待审批列表
func TestGetPendingApprovals_Success(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	createTestWorkflowAndInstance(t, h, r, token)

	jwtService := auth.NewJWTService("test-secret-key-approval-handler-12345", 24*time.Hour)
	approverToken, _ := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "approver-1",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})

	req, _ := authedRequest(http.MethodGet, "/api/v1/approvals/pending", approverToken, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("期望 approver-1 有 1 个待审批，实际为 %d", len(data))
	}
}

// TestPostWorkflows_Validation 参数校验（422）
func TestPostWorkflows_Validation(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	// 缺少必填字段 name
	body := map[string]any{
		"type": "tool_deploy",
		"definition": map[string]any{
			"steps": []any{},
		},
	}
	req, _ := authedRequest(http.MethodPost, "/api/v1/workflows", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestGetWorkflow_NotFound 不存在返回 404
func TestGetWorkflow_NotFound(t *testing.T) {
	h, r, token := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, token)

	req, _ := authedRequest(http.MethodGet, "/api/v1/workflows/non-existent-id", token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestPostWorkflows_Unauthorized 未认证返回 401
func TestPostWorkflows_Unauthorized(t *testing.T) {
	h, r, _ := setupApprovalHandlerTest(t)
	registerApprovalRoutes(r, h, "dummy-token")

	body := map[string]any{
		"name":       "工具部署审批",
		"type":       "tool_deploy",
		"definition": map[string]any{"steps": []any{}},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(mustMarshal(t, body)))
	req.Header.Set("Content-Type", "application/json")
	// 不设置 Authorization header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}

// mustMarshal 辅助函数：JSON 序列化
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}
	return b
}
