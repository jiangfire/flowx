package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	agentapp "git.neolidy.top/neo/flowx/internal/application/agent"
	approvalapp "git.neolidy.top/neo/flowx/internal/application/approval"
	"git.neolidy.top/neo/flowx/internal/domain/agent"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupE2E 创建 E2E 测试环境：使用 SQLite 内存数据库和真实服务实例
func setupE2E(t *testing.T) (*agentapp.AgentService, approvalapp.ApprovalService, *gorm.DB) {
	t.Helper()

	// 创建 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 自动迁移所有相关模型
	if err := db.AutoMigrate(
		&agent.AgentTask{},
		&approval.Workflow{},
		&approval.WorkflowInstance{},
		&approval.Approval{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建审批服务（无 LLM，E2E 测试不需要 AI 建议）
	approvalRepo := persistence.NewApprovalRepository(db)
	approvalSvc := approvalapp.NewApprovalService(approvalRepo, nil)

	// 创建 Agent 引擎并注册内置 Agent
	registry := mcpif.NewToolRegistry()
	engine := agentapp.NewAgentEngine(registry)
	engine.RegisterAgent(agentapp.NewToolOrchestrationAgent(), 1)
	engine.RegisterAgent(agentapp.NewApprovalAgent(), 1)
	engine.RegisterAgent(agentapp.NewDataQualityAgent(), 1)

	// 创建 Agent 任务仓储和服务
	agentTaskRepo := persistence.NewAgentTaskRepository(db)
	agentSvc := agentapp.NewAgentService(engine, agentTaskRepo, approvalSvc)

	return agentSvc, approvalSvc, db
}

// TestE2E_AgentApproval_FullWorkflow 验证完整的 Agent-Approval 协作流程：
// 创建审批任务 -> 任务进入待审批状态 -> 审批通过 -> 验证审批信息
func TestE2E_AgentApproval_FullWorkflow(t *testing.T) {
	agentSvc, _, _ := setupE2E(t)
	ctx := context.Background()

	const tenantID = "tenant-e2e-001"
	const userID = "user-001"

	// 创建审批任务：ApprovalAgent 处理 type="approval_review"，返回 pending_approval
	task := &agentapp.Task{
		Type:        "approval_review",
		Description: "工具部署审批",
		RequireApproval: true,
		Context: map[string]any{
			"requester": "user-001",
			"reason":    "部署工具",
		},
		Steps: []agentapp.TaskStep{
			{Type: "approval_review", Description: "审批审核"},
		},
	}

	result, err := agentSvc.CreateAndExecuteTask(ctx, tenantID, userID, task)
	if err != nil {
		t.Fatalf("创建并执行审批任务失败: %v", err)
	}

	// 验证任务状态为 pending_approval
	if result.Status != "pending_approval" {
		t.Fatalf("期望任务状态为 'pending_approval'，实际为 '%s'", result.Status)
	}
	if result.TaskID == "" {
		t.Fatal("期望 TaskID 不为空")
	}

	// 审批通过任务
	approvedTask, err := agentSvc.ApproveTask(ctx, tenantID, "approver-001", result.TaskID, "同意")
	if err != nil {
		t.Fatalf("审批通过任务失败: %v", err)
	}

	// 验证审批后状态
	if approvedTask.Status != "approved" {
		t.Errorf("期望任务状态为 'approved'，实际为 '%s'", approvedTask.Status)
	}

	// 通过 GetTask 再次查询，验证持久化结果
	fetchedTask, err := agentSvc.GetTask(ctx, tenantID, result.TaskID)
	if err != nil {
		t.Fatalf("查询已审批任务失败: %v", err)
	}
	if fetchedTask.Status != "approved" {
		t.Errorf("期望查询到的任务状态为 'approved'，实际为 '%s'", fetchedTask.Status)
	}
	if fetchedTask.ApprovedBy != "approver-001" {
		t.Errorf("期望审批人为 'approver-001'，实际为 '%s'", fetchedTask.ApprovedBy)
	}
	if fetchedTask.ApprovalComment != "同意" {
		t.Errorf("期望审批意见为 '同意'，实际为 '%s'", fetchedTask.ApprovalComment)
	}
}

// TestE2E_AgentApproval_RejectTask 验证拒绝审批任务流程：
// 创建审批任务 -> 拒绝 -> 验证拒绝信息
func TestE2E_AgentApproval_RejectTask(t *testing.T) {
	agentSvc, _, _ := setupE2E(t)
	ctx := context.Background()

	const tenantID = "tenant-e2e-002"
	const userID = "user-002"

	// 创建审批任务
	task := &agentapp.Task{
		Type:        "approval_review",
		Description: "工具部署审批",
		RequireApproval: true,
		Context: map[string]any{
			"requester": "user-002",
			"reason":    "部署工具",
		},
		Steps: []agentapp.TaskStep{
			{Type: "approval_review", Description: "审批审核"},
		},
	}

	result, err := agentSvc.CreateAndExecuteTask(ctx, tenantID, userID, task)
	if err != nil {
		t.Fatalf("创建并执行审批任务失败: %v", err)
	}

	// 验证任务状态为 pending_approval
	if result.Status != "pending_approval" {
		t.Fatalf("期望任务状态为 'pending_approval'，实际为 '%s'", result.Status)
	}

	// 拒绝任务
	rejectedTask, err := agentSvc.RejectTask(ctx, tenantID, "approver-002", result.TaskID, "不符合规范")
	if err != nil {
		t.Fatalf("拒绝任务失败: %v", err)
	}

	// 验证拒绝后状态
	if rejectedTask.Status != "rejected" {
		t.Errorf("期望任务状态为 'rejected'，实际为 '%s'", rejectedTask.Status)
	}

	// 通过 GetTask 再次查询，验证持久化结果
	fetchedTask, err := agentSvc.GetTask(ctx, tenantID, result.TaskID)
	if err != nil {
		t.Fatalf("查询已拒绝任务失败: %v", err)
	}
	if fetchedTask.Status != "rejected" {
		t.Errorf("期望查询到的任务状态为 'rejected'，实际为 '%s'", fetchedTask.Status)
	}
	if fetchedTask.ApprovedBy != "approver-002" {
		t.Errorf("期望审批人为 'approver-002'，实际为 '%s'", fetchedTask.ApprovedBy)
	}
	if fetchedTask.ApprovalComment != "不符合规范" {
		t.Errorf("期望审批意见为 '不符合规范'，实际为 '%s'", fetchedTask.ApprovalComment)
	}
}

// TestE2E_AgentApproval_ApproveNonPendingTask 验证对非待审批状态的任务执行审批操作应返回错误：
// DataQualityAgent 处理 type="data_check"，直接返回 completed
func TestE2E_AgentApproval_ApproveNonPendingTask(t *testing.T) {
	agentSvc, _, _ := setupE2E(t)
	ctx := context.Background()

	const tenantID = "tenant-e2e-003"
	const userID = "user-003"

	// 创建数据检查任务：DataQualityAgent 处理 type="data_check"，直接返回 completed
	task := &agentapp.Task{
		Type:        "data_check",
		Description: "数据质量检查",
		RequireApproval: false,
		Context: map[string]any{
			"check_type": "full",
			"target":     "tools",
		},
		Steps: []agentapp.TaskStep{
			{Type: "data_check", Description: "数据质量检查"},
		},
	}

	result, err := agentSvc.CreateAndExecuteTask(ctx, tenantID, userID, task)
	if err != nil {
		t.Fatalf("创建并执行数据检查任务失败: %v", err)
	}

	// 验证任务状态为 completed（DataQualityAgent 自动完成）
	if result.Status != "completed" {
		t.Fatalf("期望任务状态为 'completed'，实际为 '%s'", result.Status)
	}

	// 尝试审批已完成的任务，应返回 ErrTaskNotPending
	_, err = agentSvc.ApproveTask(ctx, tenantID, "approver-003", result.TaskID, "同意")
	if err == nil {
		t.Fatal("期望对已完成任务审批返回错误，实际为 nil")
	}
	if !errors.Is(err, agentapp.ErrTaskNotPending) {
		t.Errorf("期望返回 ErrTaskNotPending，实际为 '%v'", err)
	}
}

// TestE2E_AgentApproval_CrossTenantIsolation 验证跨租户隔离：
// 租户 A 创建的任务，租户 B 无法查询和审批
func TestE2E_AgentApproval_CrossTenantIsolation(t *testing.T) {
	agentSvc, _, _ := setupE2E(t)
	ctx := context.Background()

	const tenantA = "tenant-A"
	const tenantB = "tenant-B"
	const userID = "user-isolation"

	// 为租户 A 创建审批任务
	task := &agentapp.Task{
		Type:        "approval_review",
		Description: "租户A的审批任务",
		RequireApproval: true,
		Context: map[string]any{
			"requester": userID,
			"reason":    "跨租户隔离测试",
		},
		Steps: []agentapp.TaskStep{
			{Type: "approval_review", Description: "审批审核"},
		},
	}

	result, err := agentSvc.CreateAndExecuteTask(ctx, tenantA, userID, task)
	if err != nil {
		t.Fatalf("租户A创建任务失败: %v", err)
	}

	// 租户 B 尝试查询租户 A 的任务，应返回 ErrTaskNotFound
	_, err = agentSvc.GetTask(ctx, tenantB, result.TaskID)
	if err == nil {
		t.Fatal("期望跨租户查询返回错误，实际为 nil")
	}
	if !errors.Is(err, agentapp.ErrTaskNotFound) {
		t.Errorf("期望返回 ErrTaskNotFound，实际为 '%v'", err)
	}

	// 租户 B 尝试审批租户 A 的任务，应返回 ErrTaskNotFound
	_, err = agentSvc.ApproveTask(ctx, tenantB, "approver-B", result.TaskID, "同意")
	if err == nil {
		t.Fatal("期望跨租户审批返回错误，实际为 nil")
	}
	if !errors.Is(err, agentapp.ErrTaskNotFound) {
		t.Errorf("期望返回 ErrTaskNotFound，实际为 '%v'", err)
	}
}

// TestE2E_AgentApproval_ListTasksWithFilter 验证任务列表过滤功能：
// 创建多个不同状态的任务，按状态过滤查询
func TestE2E_AgentApproval_ListTasksWithFilter(t *testing.T) {
	agentSvc, _, _ := setupE2E(t)
	ctx := context.Background()

	const tenantID = "tenant-e2e-005"
	const userID = "user-005"

	// 创建 2 个待审批任务
	for i := 0; i < 2; i++ {
		task := &agentapp.Task{
			Type:        "approval_review",
			Description: "审批任务",
			RequireApproval: true,
			Context: map[string]any{
				"requester": userID,
				"reason":    "列表过滤测试",
			},
			Steps: []agentapp.TaskStep{
				{Type: "approval_review", Description: "审批审核"},
			},
		}
		_, err := agentSvc.CreateAndExecuteTask(ctx, tenantID, userID, task)
		if err != nil {
			t.Fatalf("创建待审批任务 %d 失败: %v", i, err)
		}
	}

	// 创建 1 个已完成任务
	completedTask := &agentapp.Task{
		Type:        "data_check",
		Description: "数据质量检查",
		RequireApproval: false,
		Context: map[string]any{
			"check_type": "full",
			"target":     "tools",
		},
		Steps: []agentapp.TaskStep{
			{Type: "data_check", Description: "数据质量检查"},
		},
	}
	_, err := agentSvc.CreateAndExecuteTask(ctx, tenantID, userID, completedTask)
	if err != nil {
		t.Fatalf("创建已完成任务失败: %v", err)
	}

	// 按状态过滤：查询 pending_approval
	pendingTasks, pendingPage, err := agentSvc.ListTasks(ctx, tenantID, "pending_approval", 1, 10)
	if err != nil {
		t.Fatalf("查询待审批任务列表失败: %v", err)
	}
	if len(pendingTasks) != 2 {
		t.Errorf("期望返回 2 条待审批任务，实际为 %d", len(pendingTasks))
	}
	if pendingPage.Total != 2 {
		t.Errorf("期望待审批任务总数为 2，实际为 %d", pendingPage.Total)
	}

	// 按状态过滤：查询 completed
	completedTasks, completedPage, err := agentSvc.ListTasks(ctx, tenantID, "completed", 1, 10)
	if err != nil {
		t.Fatalf("查询已完成任务列表失败: %v", err)
	}
	if len(completedTasks) != 1 {
		t.Errorf("期望返回 1 条已完成任务，实际为 %d", len(completedTasks))
	}
	if completedPage.Total != 1 {
		t.Errorf("期望已完成任务总数为 1，实际为 %d", completedPage.Total)
	}

	// 无状态过滤：查询全部
	allTasks, allPage, err := agentSvc.ListTasks(ctx, tenantID, "", 1, 10)
	if err != nil {
		t.Fatalf("查询全部任务列表失败: %v", err)
	}
	if len(allTasks) != 3 {
		t.Errorf("期望返回 3 条任务，实际为 %d", len(allTasks))
	}
	if allPage.Total != 3 {
		t.Errorf("期望任务总数为 3，实际为 %d", allPage.Total)
	}
}

// TestE2E_AgentApproval_DataQualityAgent 验证 DataQualityAgent 自动完成任务：
// 创建数据检查任务 -> 验证状态为 completed -> 验证结果包含 quality_score
func TestE2E_AgentApproval_DataQualityAgent(t *testing.T) {
	agentSvc, _, _ := setupE2E(t)
	ctx := context.Background()

	const tenantID = "tenant-e2e-006"
	const userID = "user-006"

	// 创建数据检查任务
	task := &agentapp.Task{
		Type:        "data_check",
		Description: "数据质量检查",
		RequireApproval: false,
		Context: map[string]any{
			"check_type": "full",
			"target":     "tools",
		},
		Steps: []agentapp.TaskStep{
			{Type: "data_check", Description: "数据质量检查"},
		},
	}

	result, err := agentSvc.CreateAndExecuteTask(ctx, tenantID, userID, task)
	if err != nil {
		t.Fatalf("创建并执行数据检查任务失败: %v", err)
	}

	// 验证任务状态为 completed（DataQualityAgent 自动完成）
	if result.Status != "completed" {
		t.Fatalf("期望任务状态为 'completed'，实际为 '%s'", result.Status)
	}

	// 验证步骤执行结果
	if len(result.Steps) != 1 {
		t.Fatalf("期望有 1 个步骤结果，实际为 %d", len(result.Steps))
	}
	if result.Steps[0].Status != "completed" {
		t.Errorf("期望步骤状态为 'completed'，实际为 '%s'", result.Steps[0].Status)
	}

	// 验证输出包含 quality_score
	// 检查步骤输出中是否包含 quality_score
	stepsJSON, err := json.Marshal(result.Steps[0].Output)
	if err != nil {
		t.Fatalf("序列化步骤输出失败: %v", err)
	}
	var stepOutput map[string]any
	if err := json.Unmarshal(stepsJSON, &stepOutput); err != nil {
		t.Fatalf("解析步骤输出失败: %v", err)
	}
	if _, ok := stepOutput["quality_score"]; !ok {
		t.Error("期望步骤输出中包含 'quality_score' 字段")
	}

	// 通过持久化查询验证
	fetchedTask, err := agentSvc.GetTask(ctx, tenantID, result.TaskID)
	if err != nil {
		t.Fatalf("查询数据检查任务失败: %v", err)
	}
	if fetchedTask.Status != "completed" {
		t.Errorf("期望持久化状态为 'completed'，实际为 '%s'", fetchedTask.Status)
	}

	// 验证持久化的 Result JSON 包含 quality_score
	if fetchedTask.Result == "" {
		t.Fatal("期望持久化的 Result 字段不为空")
	}
	var resultData map[string]any
	if err := json.Unmarshal([]byte(fetchedTask.Result), &resultData); err != nil {
		t.Fatalf("解析持久化 Result 失败: %v", err)
	}
	// Result 中应包含 steps 数组
	stepsVal, ok := resultData["steps"]
	if !ok {
		t.Fatal("期望 Result 中包含 'steps' 字段")
	}
	stepsArr, ok := stepsVal.([]any)
	if !ok {
		t.Fatal("期望 Result.steps 为数组类型")
	}
	if len(stepsArr) == 0 {
		t.Fatal("期望 Result.steps 不为空")
	}
	firstStep, ok := stepsArr[0].(map[string]any)
	if !ok {
		t.Fatal("期望步骤元素为 map 类型")
	}
	stepOutputVal, ok := firstStep["output"]
	if !ok {
		t.Fatal("期望步骤中包含 'output' 字段")
	}
	stepOutputMap, ok := stepOutputVal.(map[string]any)
	if !ok {
		t.Fatal("期望步骤 output 为 map 类型")
	}
	if _, ok := stepOutputMap["quality_score"]; !ok {
		t.Error("期望持久化的步骤输出中包含 'quality_score' 字段")
	}
}
