package approval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/application/ai"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupServiceTest 创建审批服务测试环境
func setupServiceTest(t *testing.T) (ApprovalService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&approval.Workflow{}, &approval.WorkflowInstance{}, &approval.Approval{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 使用内存中的 mock repository
	repo := newMockApprovalRepository(db)
	svc := NewApprovalService(repo, nil)
	return svc, db
}

// setupServiceTestWithLLM 创建带 mock LLM 的审批服务测试环境
func setupServiceTestWithLLM(t *testing.T, suggestion string) (ApprovalService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&approval.Workflow{}, &approval.WorkflowInstance{}, &approval.Approval{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建 mock LLM 服务
	respBody := map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": suggestion}},
		},
	}
	respBytes, _ := json.Marshal(respBody)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}))

	llmSvc := ai.NewLLMService(server.URL, "test-key", 5*time.Second)
	repo := newMockApprovalRepository(db)
	svc := NewApprovalService(repo, llmSvc)
	t.Cleanup(func() { server.Close() })
	return svc, db
}

// createTestWorkflow 创建测试用工作流
func createTestWorkflow(t *testing.T, ctx context.Context, svc ApprovalService, tenantID string) *approval.Workflow {
	t.Helper()
	req := &CreateWorkflowRequest{
		Name:        "工具部署审批",
		Type:        "tool_deploy",
		Description: "部署工具前需要审批",
		Definition: base.JSON{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
				{"name": "负责人审批", "approvers": []string{"approver-2"}},
			},
		},
	}
	w, err := svc.CreateWorkflow(ctx, tenantID, req)
	if err != nil {
		t.Fatalf("创建测试工作流失败: %v", err)
	}
	return w
}

// TestCreateWorkflow_Success 创建工作流成功
func TestCreateWorkflow_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	req := &CreateWorkflowRequest{
		Name:        "工具部署审批",
		Type:        "tool_deploy",
		Description: "部署工具前需要审批",
		Definition: base.JSON{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
			},
		},
	}

	w, err := svc.CreateWorkflow(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	if w.ID == "" {
		t.Error("期望 ID 不为空")
	}
	if w.Name != "工具部署审批" {
		t.Errorf("期望 Name 为 '工具部署审批'，实际为 '%s'", w.Name)
	}
	if w.Type != "tool_deploy" {
		t.Errorf("期望 Type 为 'tool_deploy'，实际为 '%s'", w.Type)
	}
	if w.Status != "draft" {
		t.Errorf("期望 Status 为 'draft'，实际为 '%s'", w.Status)
	}
	if w.Version != 1 {
		t.Errorf("期望 Version 为 1，实际为 %d", w.Version)
	}
}

// TestListWorkflows_Success 列出工作流
func TestListWorkflows_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	// 创建工作流
	createTestWorkflow(t, ctx, svc, "tenant-001")

	workflows, result, err := svc.ListWorkflows(ctx, "tenant-001", WorkflowFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("列出工作流失败: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("期望 Total 为 1，实际为 %d", result.Total)
	}
	if len(workflows) != 1 {
		t.Errorf("期望返回 1 条记录，实际为 %d", len(workflows))
	}
}

// TestStartApproval_Success 发起审批创建实例和第一步审批记录
func TestStartApproval_Success(t *testing.T) {
	svc, db := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	req := &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx 工具",
		Context:    base.JSON{"tool_name": "nginx", "version": "1.25"},
	}

	inst, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", req)
	if err != nil {
		t.Fatalf("发起审批失败: %v", err)
	}
	if inst.ID == "" {
		t.Error("期望实例 ID 不为空")
	}
	if inst.Status != "approving" {
		t.Errorf("期望 Status 为 'approving'，实际为 '%s'", inst.Status)
	}
	if inst.CurrentStep != 0 {
		t.Errorf("期望 CurrentStep 为 0，实际为 %d", inst.CurrentStep)
	}

	// 验证第一步审批记录已创建
	var approvals []approval.Approval
	db.Where("instance_id = ?", inst.ID).Find(&approvals)
	if len(approvals) != 1 {
		t.Fatalf("期望创建 1 条审批记录，实际为 %d", len(approvals))
	}
	if approvals[0].Step != 1 {
		t.Errorf("期望审批步骤为 1，实际为 %d", approvals[0].Step)
	}
	if approvals[0].Status != "pending" {
		t.Errorf("期望审批状态为 'pending'，实际为 '%s'", approvals[0].Status)
	}
	if approvals[0].ApproverID != "approver-1" {
		t.Errorf("期望审批人为 'approver-1'，实际为 '%s'", approvals[0].ApproverID)
	}
}

// TestApprove_Success 审批通过，状态更新
func TestApprove_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	// 发起审批
	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// 审批通过
	approveReq := &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "同意部署",
	}
	ap, err := svc.Approve(ctx, "tenant-001", "approver-1", approveReq)
	if err != nil {
		t.Fatalf("审批通过失败: %v", err)
	}
	if ap.Status != "approved" {
		t.Errorf("期望审批状态为 'approved'，实际为 '%s'", ap.Status)
	}
	if ap.Comment != "同意部署" {
		t.Errorf("期望 Comment 为 '同意部署'，实际为 '%s'", ap.Comment)
	}

	// 验证实例状态更新
	updatedInst, err := svc.GetInstance(ctx, "tenant-001", inst.ID)
	if err != nil {
		t.Fatalf("获取实例失败: %v", err)
	}
	if updatedInst.CurrentStep != 1 {
		t.Errorf("期望 CurrentStep 更新为 1，实际为 %d", updatedInst.CurrentStep)
	}
	if updatedInst.Status != "approving" {
		t.Errorf("期望实例状态仍为 'approving'，实际为 '%s'", updatedInst.Status)
	}
}

// TestApprove_LastStep 最后一步审批通过，实例状态变为 approved
func TestApprove_LastStep(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	// 发起审批
	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// 第一步审批通过
	svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "技术审核通过",
	})

	// 第二步审批通过（最后一步）
	ap, err := svc.Approve(ctx, "tenant-001", "approver-2", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "负责人审批通过",
	})
	if err != nil {
		t.Fatalf("最后一步审批通过失败: %v", err)
	}
	if ap.Status != "approved" {
		t.Errorf("期望审批状态为 'approved'，实际为 '%s'", ap.Status)
	}

	// 验证实例状态变为 approved
	updatedInst, _ := svc.GetInstance(ctx, "tenant-001", inst.ID)
	if updatedInst.Status != "approved" {
		t.Errorf("期望实例状态为 'approved'，实际为 '%s'", updatedInst.Status)
	}
}

// TestReject_Success 审批驳回，实例状态变为 rejected
func TestReject_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	rej, err := svc.Reject(ctx, "tenant-001", "approver-1", &RejectRequest{
		InstanceID: inst.ID,
		Comment:    "工具版本不合规",
	})
	if err != nil {
		t.Fatalf("审批驳回失败: %v", err)
	}
	if rej.Status != "rejected" {
		t.Errorf("期望审批状态为 'rejected'，实际为 '%s'", rej.Status)
	}

	// 验证实例状态变为 rejected
	updatedInst, _ := svc.GetInstance(ctx, "tenant-001", inst.ID)
	if updatedInst.Status != "rejected" {
		t.Errorf("期望实例状态为 'rejected'，实际为 '%s'", updatedInst.Status)
	}
}

// TestForward_Success 转审
func TestForward_Success(t *testing.T) {
	svc, db := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	fwd, err := svc.Forward(ctx, "tenant-001", "approver-1", &ForwardRequest{
		InstanceID:   inst.ID,
		ToApproverID: "approver-3",
		Comment:      "转给更专业的同事",
	})
	if err != nil {
		t.Fatalf("转审失败: %v", err)
	}
	if fwd.Status != "forwarded" {
		t.Errorf("期望审批状态为 'forwarded'，实际为 '%s'", fwd.Status)
	}

	// 验证新的待审批记录已创建
	var newApprovals []approval.Approval
	db.Where("instance_id = ? AND approver_id = ?", inst.ID, "approver-3").Find(&newApprovals)
	if len(newApprovals) != 1 {
		t.Errorf("期望为转审人创建 1 条审批记录，实际为 %d", len(newApprovals))
	}
}

// TestCancelInstance_Success 取消实例
func TestCancelInstance_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	err := svc.CancelInstance(ctx, "tenant-001", inst.ID)
	if err != nil {
		t.Fatalf("取消实例失败: %v", err)
	}

	updatedInst, _ := svc.GetInstance(ctx, "tenant-001", inst.ID)
	if updatedInst.Status != "cancelled" {
		t.Errorf("期望实例状态为 'cancelled'，实际为 '%s'", updatedInst.Status)
	}
}

// TestGetSuggestion_Success 返回 AI 建议
func TestGetSuggestion_Success(t *testing.T) {
	svc, _ := setupServiceTestWithLLM(t, "建议通过该部署请求。")
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	suggestion, err := svc.GetSuggestion(ctx, "tenant-001", inst.ID)
	if err != nil {
		t.Fatalf("获取 AI 建议失败: %v", err)
	}
	if suggestion != "建议通过该部署请求。" {
		t.Errorf("期望建议为 '建议通过该部署请求。'，实际为 '%s'", suggestion)
	}
}

// TestGetMyPendingApprovals_Success 返回待审批列表
func TestGetMyPendingApprovals_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	// 发起两个审批
	inst1, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})
	svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 redis",
	})

	// 审批第一个
	svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst1.ID,
		Comment:    "通过",
	})

	// approver-2 应该有 1 个待审批（inst1 的第二步）
	pending, err := svc.GetMyPendingApprovals(ctx, "tenant-001", "approver-2")
	if err != nil {
		t.Fatalf("获取待审批列表失败: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("期望 approver-2 有 1 个待审批，实际为 %d", len(pending))
	}
}

// TestCrossTenantOperation 跨租户操作返回错误
func TestCrossTenantOperation(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	// 租户 A 创建工作流
	req := &CreateWorkflowRequest{
		Name:       "租户A工作流",
		Type:       "tool_deploy",
		Definition: base.JSON{"steps": []any{}},
	}
	w, _ := svc.CreateWorkflow(ctx, "tenant-A", req)

	// 租户 B 尝试获取租户 A 的工作流
	_, err := svc.GetInstance(ctx, "tenant-B", w.ID)
	if err == nil {
		t.Error("期望跨租户操作返回错误")
	}
}

// TestApproveNonExistentInstance 审批不存在的实例返回错误
func TestApproveNonExistentInstance(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	_, err := svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: "non-existent-id",
		Comment:    "通过",
	})
	if err == nil {
		t.Fatal("期望审批不存在的实例返回错误")
	}
}

// TestDuplicateApproval 重复审批返回错误
func TestDuplicateApproval(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// 第一次审批通过
	_, err := svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "通过",
	})
	if err != nil {
		t.Fatalf("第一次审批失败: %v", err)
	}

	// 重复审批
	_, err = svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "再次通过",
	})
	if err == nil {
		t.Fatal("期望重复审批返回错误")
	}
}

// ===================== Workflow Lifecycle Tests =====================

// TestActivateWorkflow_Success 激活草稿工作流成功
func TestActivateWorkflow_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	if w.Status != "draft" {
		t.Fatalf("期望初始状态为 draft，实际为 '%s'", w.Status)
	}

	activated, err := svc.ActivateWorkflow(ctx, "tenant-001", w.ID)
	if err != nil {
		t.Fatalf("激活工作流失败: %v", err)
	}
	if activated.Status != "active" {
		t.Errorf("期望状态为 'active'，实际为 '%s'", activated.Status)
	}

	// 验证通过 GetWorkflow 获取的状态也是 active
	fetched, err := svc.GetWorkflow(ctx, "tenant-001", w.ID)
	if err != nil {
		t.Fatalf("获取工作流失败: %v", err)
	}
	if fetched.Status != "active" {
		t.Errorf("期望获取的状态为 'active'，实际为 '%s'", fetched.Status)
	}
}

// TestActivateWorkflow_AlreadyActive 已激活的工作流不能再次激活
func TestActivateWorkflow_AlreadyActive(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	_, err := svc.ActivateWorkflow(ctx, "tenant-001", w.ID)
	if err == nil {
		t.Fatal("期望已激活的工作流再次激活返回错误")
	}
}

// TestActivateWorkflow_Archived 已归档的工作流不能激活
func TestActivateWorkflow_Archived(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)
	_, _ = svc.ArchiveWorkflow(ctx, "tenant-001", w.ID)

	_, err := svc.ActivateWorkflow(ctx, "tenant-001", w.ID)
	if err == nil {
		t.Fatal("期望已归档的工作流激活返回错误")
	}
}

// TestArchiveWorkflow_Success 归档已激活的工作流成功
func TestArchiveWorkflow_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	archived, err := svc.ArchiveWorkflow(ctx, "tenant-001", w.ID)
	if err != nil {
		t.Fatalf("归档工作流失败: %v", err)
	}
	if archived.Status != "archived" {
		t.Errorf("期望状态为 'archived'，实际为 '%s'", archived.Status)
	}
}

// TestArchiveWorkflow_FromDraft 草稿状态的工作流也可以归档
func TestArchiveWorkflow_FromDraft(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")

	archived, err := svc.ArchiveWorkflow(ctx, "tenant-001", w.ID)
	if err != nil {
		t.Fatalf("归档草稿工作流失败: %v", err)
	}
	if archived.Status != "archived" {
		t.Errorf("期望状态为 'archived'，实际为 '%s'", archived.Status)
	}
}

// TestStartApproval_DraftWorkflow 草稿状态的工作流不能发起审批
func TestStartApproval_DraftWorkflow(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	if w.Status != "draft" {
		t.Fatalf("期望初始状态为 draft，实际为 '%s'", w.Status)
	}

	_, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})
	if err == nil {
		t.Fatal("期望草稿状态的工作流发起审批返回错误")
	}
}

// TestStartApproval_ActiveWorkflow 已激活的工作流可以发起审批
func TestStartApproval_ActiveWorkflow(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})
	if err != nil {
		t.Fatalf("已激活的工作流发起审批失败: %v", err)
	}
	if inst.Status != "approving" {
		t.Errorf("期望实例状态为 'approving'，实际为 '%s'", inst.Status)
	}
}

// TestStartApproval_ArchivedWorkflow 已归档的工作流不能发起审批
func TestStartApproval_ArchivedWorkflow(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)
	_, _ = svc.ArchiveWorkflow(ctx, "tenant-001", w.ID)

	_, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})
	if err == nil {
		t.Fatal("期望已归档的工作流发起审批返回错误")
	}
}

// ===================== Boundary Tests =====================

// TestApprove_EmptyComment 审批通过时使用空评论应仍然成功
func TestApprove_EmptyComment(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	ap, err := svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "",
	})
	if err != nil {
		t.Fatalf("空评论审批通过失败: %v", err)
	}
	if ap.Status != "approved" {
		t.Errorf("期望审批状态为 'approved'，实际为 '%s'", ap.Status)
	}
}

// TestStartApproval_MultiStepWorkflow 多步骤工作流验证第一步审批人
func TestStartApproval_MultiStepWorkflow(t *testing.T) {
	svc, db := setupServiceTest(t)
	ctx := context.Background()

	req := &CreateWorkflowRequest{
		Name:        "三步审批",
		Type:        "multi_step",
		Description: "三步审批工作流",
		Definition: base.JSON{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
				{"name": "安全审核", "approvers": []string{"approver-2"}},
				{"name": "负责人审批", "approvers": []string{"approver-3"}},
			},
		},
	}
	w, err := svc.CreateWorkflow(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "三步审批测试",
	})
	if err != nil {
		t.Fatalf("发起审批失败: %v", err)
	}
	if inst.CurrentStep != 0 {
		t.Errorf("期望 CurrentStep 为 0，实际为 %d", inst.CurrentStep)
	}

	var approvals []approval.Approval
	db.Where("instance_id = ?", inst.ID).Find(&approvals)
	if len(approvals) != 1 {
		t.Fatalf("期望创建 1 条审批记录，实际为 %d", len(approvals))
	}
	if approvals[0].Step != 1 {
		t.Errorf("期望第一步审批步骤为 1，实际为 %d", approvals[0].Step)
	}
	if approvals[0].ApproverID != "approver-1" {
		t.Errorf("期望第一步审批人为 'approver-1'，实际为 '%s'", approvals[0].ApproverID)
	}
}

// TestForward_ToNonExistentUser 转审给不在工作流定义中的用户应仍然成功
func TestForward_ToNonExistentUser(t *testing.T) {
	svc, db := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	fwd, err := svc.Forward(ctx, "tenant-001", "approver-1", &ForwardRequest{
		InstanceID:   inst.ID,
		ToApproverID: "nonexistent-user-999",
		Comment:      "转给不在定义中的用户",
	})
	if err != nil {
		t.Fatalf("转审给不存在的用户应仍然成功，实际失败: %v", err)
	}
	if fwd.Status != "forwarded" {
		t.Errorf("期望审批状态为 'forwarded'，实际为 '%s'", fwd.Status)
	}

	// 验证新的待审批记录已创建
	var newApprovals []approval.Approval
	db.Where("instance_id = ? AND approver_id = ?", inst.ID, "nonexistent-user-999").Find(&newApprovals)
	if len(newApprovals) != 1 {
		t.Errorf("期望为转审人创建 1 条审批记录，实际为 %d", len(newApprovals))
	}
}

// ===================== 高价值业务逻辑测试 =====================

// TestListInstances_Success 列出工作流实例，验证分页和数量
func TestListInstances_Success(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	// 创建并激活工作流
	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	// 发起两个审批实例
	_, _ = svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})
	_, _ = svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 redis",
	})

	// 列出实例
	instances, result, err := svc.ListInstances(ctx, "tenant-001", InstanceFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("列出工作流实例失败: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("期望 Total 为 2，实际为 %d", result.Total)
	}
	if len(instances) != 2 {
		t.Errorf("期望返回 2 条实例记录，实际为 %d", len(instances))
	}
}

// TestApprove_WrongApprover 非审批人尝试审批通过，返回 ErrApprovalNotFound
func TestApprove_WrongApprover(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// approver-2 不是当前步骤的审批人，尝试审批应失败
	_, err := svc.Approve(ctx, "tenant-001", "approver-2", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "越权审批",
	})
	if err != ErrApprovalNotFound {
		t.Errorf("期望返回 ErrApprovalNotFound，实际为 %v", err)
	}
}

// TestReject_WrongApprover 非审批人尝试驳回，返回 ErrApprovalNotFound
func TestReject_WrongApprover(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// approver-2 不是当前步骤的审批人，尝试驳回应失败
	_, err := svc.Reject(ctx, "tenant-001", "approver-2", &RejectRequest{
		InstanceID: inst.ID,
		Comment:    "越权驳回",
	})
	if err != ErrApprovalNotFound {
		t.Errorf("期望返回 ErrApprovalNotFound，实际为 %v", err)
	}
}

// TestApprove_AlreadyFinished 对已完成的实例再次审批，返回 ErrInstanceFinished
func TestApprove_AlreadyFinished(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// 完成所有审批步骤
	_, _ = svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "第一步通过",
	})
	_, _ = svc.Approve(ctx, "tenant-001", "approver-2", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "第二步通过",
	})

	// 实例已完成，再次审批应失败
	_, err := svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "重复审批",
	})
	if err != ErrInstanceFinished {
		t.Errorf("期望返回 ErrInstanceFinished，实际为 %v", err)
	}
}

// TestReject_AlreadyFinished 对已驳回的实例再次驳回，返回 ErrInstanceFinished
func TestReject_AlreadyFinished(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// 第一步驳回
	_, _ = svc.Reject(ctx, "tenant-001", "approver-1", &RejectRequest{
		InstanceID: inst.ID,
		Comment:    "第一步驳回",
	})

	// 实例已驳回，再次驳回应失败
	_, err := svc.Reject(ctx, "tenant-001", "approver-1", &RejectRequest{
		InstanceID: inst.ID,
		Comment:    "重复驳回",
	})
	if err != ErrInstanceFinished {
		t.Errorf("期望返回 ErrInstanceFinished，实际为 %v", err)
	}
}

// TestCancelInstance_AlreadyFinished 取消已完成的实例，返回 ErrInstanceFinished
func TestCancelInstance_AlreadyFinished(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// 完成所有审批步骤
	_, _ = svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "第一步通过",
	})
	_, _ = svc.Approve(ctx, "tenant-001", "approver-2", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "第二步通过",
	})

	// 实例已完成，取消应失败
	err := svc.CancelInstance(ctx, "tenant-001", inst.ID)
	if err != ErrInstanceFinished {
		t.Errorf("期望返回 ErrInstanceFinished，实际为 %v", err)
	}
}

// TestForward_WrongApprover 非审批人尝试转审，返回 ErrApprovalNotFound
func TestForward_WrongApprover(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// approver-2 不是当前步骤的审批人，尝试转审应失败
	_, err := svc.Forward(ctx, "tenant-001", "approver-2", &ForwardRequest{
		InstanceID:   inst.ID,
		ToApproverID: "approver-3",
		Comment:      "越权转审",
	})
	if err != ErrApprovalNotFound {
		t.Errorf("期望返回 ErrApprovalNotFound，实际为 %v", err)
	}
}

// TestGetSuggestion_NoLLM 未配置 LLM 服务时获取建议，返回错误
func TestGetSuggestion_NoLLM(t *testing.T) {
	svc, _ := setupServiceTest(t)
	ctx := context.Background()

	w := createTestWorkflow(t, ctx, svc, "tenant-001")
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	inst, _ := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "部署 nginx",
	})

	// setupServiceTest 创建的服务 LLM 为 nil，应返回错误
	_, err := svc.GetSuggestion(ctx, "tenant-001", inst.ID)
	if err == nil {
		t.Fatal("期望未配置 LLM 时返回错误")
	}
	if err.Error() != "LLM 服务未配置" {
		t.Errorf("期望错误信息为 'LLM 服务未配置'，实际为 '%s'", err.Error())
	}
}

// TestStartApproval_NoApproversInDefinition 工作流定义中步骤无审批人，实例创建但不生成审批记录
func TestStartApproval_NoApproversInDefinition(t *testing.T) {
	svc, db := setupServiceTest(t)
	ctx := context.Background()

	// 创建步骤中审批人列表为空的工作流
	req := &CreateWorkflowRequest{
		Name:        "无审批人工作流",
		Type:        "no_approver",
		Description: "步骤中未配置审批人",
		Definition: base.JSON{
			"steps": []map[string]any{
				{"name": "空审批人步骤", "approvers": []string{}},
			},
		},
	}
	w, err := svc.CreateWorkflow(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	// 发起审批，实例应正常创建
	inst, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "无审批人测试",
	})
	if err != nil {
		t.Fatalf("发起审批失败: %v", err)
	}
	if inst.ID == "" {
		t.Fatal("期望实例 ID 不为空")
	}
	if inst.Status != "approving" {
		t.Errorf("期望实例状态为 'approving'，实际为 '%s'", inst.Status)
	}

	// 验证没有创建审批记录
	var approvals []approval.Approval
	db.Where("instance_id = ?", inst.ID).Find(&approvals)
	if len(approvals) != 0 {
		t.Errorf("期望创建 0 条审批记录，实际为 %d", len(approvals))
	}
}

// TestApprove_ThreeStepWorkflow 完整三步审批流程验证
func TestApprove_ThreeStepWorkflow(t *testing.T) {
	svc, db := setupServiceTest(t)
	ctx := context.Background()

	// 创建三步审批工作流
	req := &CreateWorkflowRequest{
		Name:        "三步审批流程",
		Type:        "three_step",
		Description: "完整三步审批流程测试",
		Definition: base.JSON{
			"steps": []map[string]any{
				{"name": "技术审核", "approvers": []string{"approver-1"}},
				{"name": "安全审核", "approvers": []string{"approver-2"}},
				{"name": "负责人审批", "approvers": []string{"approver-3"}},
			},
		},
	}
	w, err := svc.CreateWorkflow(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	_, _ = svc.ActivateWorkflow(ctx, "tenant-001", w.ID)

	// 发起审批
	inst, err := svc.StartApproval(ctx, "tenant-001", "initiator-001", &StartApprovalRequest{
		WorkflowID: w.ID,
		Title:      "三步审批测试",
	})
	if err != nil {
		t.Fatalf("发起审批失败: %v", err)
	}

	// 验证初始状态：只有第一步审批记录
	var approvals []approval.Approval
	db.Where("instance_id = ?", inst.ID).Find(&approvals)
	if len(approvals) != 1 {
		t.Fatalf("期望初始创建 1 条审批记录，实际为 %d", len(approvals))
	}

	// 第一步审批通过
	ap1, err := svc.Approve(ctx, "tenant-001", "approver-1", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "技术审核通过",
	})
	if err != nil {
		t.Fatalf("第一步审批失败: %v", err)
	}
	if ap1.Status != "approved" {
		t.Errorf("期望第一步状态为 'approved'，实际为 '%s'", ap1.Status)
	}

	// 验证实例状态和第二步审批记录
	updatedInst, _ := svc.GetInstance(ctx, "tenant-001", inst.ID)
	if updatedInst.CurrentStep != 1 {
		t.Errorf("期望 CurrentStep 为 1，实际为 %d", updatedInst.CurrentStep)
	}
	if updatedInst.Status != "approving" {
		t.Errorf("期望实例状态仍为 'approving'，实际为 '%s'", updatedInst.Status)
	}
	db.Where("instance_id = ?", inst.ID).Find(&approvals)
	if len(approvals) != 2 {
		t.Fatalf("期望创建 2 条审批记录，实际为 %d", len(approvals))
	}

	// 第二步审批通过
	ap2, err := svc.Approve(ctx, "tenant-001", "approver-2", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "安全审核通过",
	})
	if err != nil {
		t.Fatalf("第二步审批失败: %v", err)
	}
	if ap2.Status != "approved" {
		t.Errorf("期望第二步状态为 'approved'，实际为 '%s'", ap2.Status)
	}

	// 验证实例状态和第三步审批记录
	updatedInst, _ = svc.GetInstance(ctx, "tenant-001", inst.ID)
	if updatedInst.CurrentStep != 2 {
		t.Errorf("期望 CurrentStep 为 2，实际为 %d", updatedInst.CurrentStep)
	}
	if updatedInst.Status != "approving" {
		t.Errorf("期望实例状态仍为 'approving'，实际为 '%s'", updatedInst.Status)
	}
	db.Where("instance_id = ?", inst.ID).Find(&approvals)
	if len(approvals) != 3 {
		t.Fatalf("期望创建 3 条审批记录，实际为 %d", len(approvals))
	}

	// 第三步审批通过（最后一步）
	ap3, err := svc.Approve(ctx, "tenant-001", "approver-3", &ApproveRequest{
		InstanceID: inst.ID,
		Comment:    "负责人审批通过",
	})
	if err != nil {
		t.Fatalf("第三步审批失败: %v", err)
	}
	if ap3.Status != "approved" {
		t.Errorf("期望第三步状态为 'approved'，实际为 '%s'", ap3.Status)
	}

	// 验证实例最终状态为 approved
	finalInst, _ := svc.GetInstance(ctx, "tenant-001", inst.ID)
	if finalInst.Status != "approved" {
		t.Errorf("期望实例最终状态为 'approved'，实际为 '%s'", finalInst.Status)
	}
	if finalInst.CurrentStep != 3 {
		t.Errorf("期望最终 CurrentStep 为 3，实际为 %d", finalInst.CurrentStep)
	}
}
