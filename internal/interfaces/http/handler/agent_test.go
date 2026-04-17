package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agentapp "git.neolidy.top/neo/flowx/internal/application/agent"
	"git.neolidy.top/neo/flowx/internal/application/auth"
	domainagent "git.neolidy.top/neo/flowx/internal/domain/agent"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAgentHandlerTest 创建 Agent Handler 测试环境
func setupAgentHandlerTest(t *testing.T) (*AgentHandler, *gin.Engine, string) {
	t.Helper()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&domainagent.AgentTask{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建工具注册器并注册测试工具
	registry := mcpif.NewToolRegistry()
	registry.RegisterToolWithHandler(
		&tool.Tool{Name: "echo", Type: "custom", Description: "回显工具", Status: "active"},
		func(ctx context.Context, args map[string]any) (any, error) {
			return map[string]any{"echoed": args["message"]}, nil
		},
	)

	// 创建 Agent 引擎并注册内置 Agent
	engine := agentapp.NewAgentEngine(registry)
	engine.RegisterAgent(agentapp.NewToolOrchestrationAgent())
	engine.RegisterAgent(agentapp.NewApprovalAgent())
	engine.RegisterAgent(agentapp.NewDataQualityAgent())

	// 创建 AgentTaskRepository
	agentTaskRepo := persistence.NewAgentTaskRepository(db)

	// 创建 AgentService
	agentSvc := agentapp.NewAgentService(engine, agentTaskRepo, nil)

	// 创建 Handler
	handler := NewAgentHandler(agentSvc)

	// 创建 JWT 服务
	jwtService := auth.NewJWTService("test-secret-key-agent-handler-12345", 24*time.Hour)
	token, _ := jwtService.GenerateToken(&auth.TokenClaims{
		UserID:   "user-001",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()

	return handler, r, token
}

// registerAgentRoutes 注册 Agent 路由（带认证中间件）
func registerAgentRoutes(r *gin.Engine, h *AgentHandler, token string) {
	jwtService := auth.NewJWTService("test-secret-key-agent-handler-12345", 24*time.Hour)

	agentGroup := r.Group("/api/v1/agent")
	agentGroup.Use(middleware.AuthMiddleware(jwtService), middleware.TenantMiddleware())
	{
		agentGroup.GET("/tools", h.ListTools)
		agentGroup.POST("/tasks", h.CreateTask)
		agentGroup.GET("/tasks", h.ListTasks)
		agentGroup.GET("/tasks/:id", h.GetTask)
		agentGroup.POST("/tasks/:id/approve", h.ApproveTask)
		agentGroup.POST("/tasks/:id/reject", h.RejectTask)
	}
}

// TestGetAgentTools_Success GET /agent/tools 返回工具列表
func TestGetAgentTools_Success(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	req, _ := authedRequest(http.MethodGet, "/api/v1/agent/tools", token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]any)
	if len(data) < 1 {
		t.Error("期望至少返回 1 个工具")
	}
}

// TestCreateTask_Success POST /agent/tasks 创建任务并执行
func TestCreateTask_Success(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	body := map[string]any{
		"type":        "tool_execute",
		"description": "执行回显工具",
		"steps": []map[string]any{
			{"type": "tool_execute"},
		},
		"context": map[string]any{
			"tool_name": "echo",
			"args":      map[string]any{"message": "hello"},
		},
	}

	req, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%v'", data["status"])
	}
}

// TestCreateTask_RequireApproval POST /agent/tasks 需要审批的任务返回 pending_approval
func TestCreateTask_RequireApproval(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	body := map[string]any{
		"type":             "approval_review",
		"description":      "审批测试",
		"require_approval": true,
		"steps": []map[string]any{
			{"type": "approval_review"},
		},
		"context": map[string]any{
			"requester": "user-001",
			"reason":    "工具部署审批",
		},
	}

	req, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("期望状态码 201，实际为 %d，响应: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	if data["status"] != "pending_approval" {
		t.Errorf("期望状态为 'pending_approval'，实际为 '%v'", data["status"])
	}
}

// TestGetTask_Success GET /agent/tasks/:id 返回任务详情
func TestGetTask_Success(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	// 先创建任务
	body := map[string]any{
		"type":        "data_check",
		"description": "数据质量检查",
		"steps": []map[string]any{
			{"type": "data_check"},
		},
		"context": map[string]any{
			"check_type": "completeness",
			"target":     "tools",
		},
	}

	req, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	taskID := createResp["data"].(map[string]any)["task_id"].(string)

	// 查询详情
	req2, _ := authedRequest(http.MethodGet, "/api/v1/agent/tasks/"+taskID, token, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var getResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &getResp)
	data := getResp["data"].(map[string]any)
	if data["id"] != taskID {
		t.Errorf("期望任务 ID 为 '%s'，实际为 '%v'", taskID, data["id"])
	}
}

// TestApproveTask_Success POST /agent/tasks/:id/approve 审批通过
func TestApproveTask_Success(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	// 创建需要审批的任务
	body := map[string]any{
		"type":             "approval_review",
		"description":      "审批测试",
		"require_approval": true,
		"steps": []map[string]any{
			{"type": "approval_review"},
		},
		"context": map[string]any{
			"requester": "user-001",
			"reason":    "工具部署审批",
		},
	}

	req, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	taskID := createResp["data"].(map[string]any)["task_id"].(string)

	// 审批通过
	approveBody := map[string]any{"comment": "同意"}
	req2, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks/"+taskID+"/approve", token, approveBody)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var approveResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &approveResp)
	data := approveResp["data"].(map[string]any)
	if data["status"] != "approved" {
		t.Errorf("期望状态为 'approved'，实际为 '%v'", data["status"])
	}
}

// TestRejectTask_Success POST /agent/tasks/:id/reject 拒绝
func TestRejectTask_Success(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	// 创建需要审批的任务
	body := map[string]any{
		"type":             "approval_review",
		"description":      "审批测试",
		"require_approval": true,
		"steps": []map[string]any{
			{"type": "approval_review"},
		},
		"context": map[string]any{
			"requester": "user-001",
			"reason":    "工具部署审批",
		},
	}

	req, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	taskID := createResp["data"].(map[string]any)["task_id"].(string)

	// 拒绝
	rejectBody := map[string]any{"comment": "不符合规范"}
	req2, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks/"+taskID+"/reject", token, rejectBody)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w2.Code, w2.Body.String())
	}

	var rejectResp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &rejectResp)
	data := rejectResp["data"].(map[string]any)
	if data["status"] != "rejected" {
		t.Errorf("期望状态为 'rejected'，实际为 '%v'", data["status"])
	}
}

// TestCreateTask_Validation POST /agent/tasks 参数校验（422）
func TestCreateTask_Validation(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	// 缺少必填字段 type
	body := map[string]any{
		"description": "缺少 type",
	}

	req, _ := authedRequest(http.MethodPost, "/api/v1/agent/tasks", token, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("期望状态码 422，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}

// TestAgentEndpoints_Unauthorized 未认证返回 401
func TestAgentEndpoints_Unauthorized(t *testing.T) {
	h, r, _ := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, "dummy-token")

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/agent/tools", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401，实际为 %d", w.Code)
	}
}

// TestListTasks_Success GET /agent/tasks 返回任务列表
func TestListTasks_Success(t *testing.T) {
	h, r, token := setupAgentHandlerTest(t)
	registerAgentRoutes(r, h, token)

	req, _ := authedRequest(http.MethodGet, "/api/v1/agent/tasks", token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，实际为 %d，响应: %s", w.Code, w.Body.String())
	}
}
