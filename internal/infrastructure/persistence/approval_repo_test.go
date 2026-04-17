package persistence

import (
	"context"
	"testing"

	approvalapp "git.neolidy.top/neo/flowx/internal/application/approval"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/tenant"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupApprovalTestDB 创建审批模块测试数据库
func setupApprovalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&approval.Workflow{}, &approval.WorkflowInstance{}, &approval.Approval{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// ===================== Workflow CRUD 测试 =====================

// TestCreateWorkflow 创建工作流成功
func TestCreateWorkflow(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	w := &approval.Workflow{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Name:        "工具部署审批",
		Type:        "tool_deploy",
		Description: "部署工具前需要审批",
		Definition:  base.JSON{"steps": []any{}},
		Version:     1,
		Status:      "draft",
	}

	err := repo.CreateWorkflow(ctx, w)
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	if w.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestCreateWorkflow_GeneratesUUID 创建工作流时自动生成 UUID
func TestCreateWorkflow_GeneratesUUID(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	w := &approval.Workflow{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Name:        "工具部署审批",
		Type:        "tool_deploy",
		Definition:  base.JSON{"steps": []any{}},
	}

	err := repo.CreateWorkflow(ctx, w)
	if err != nil {
		t.Fatalf("创建工作流失败: %v", err)
	}
	if w.ID == "" {
		t.Error("期望创建后自动生成 ID")
	}
}

// TestGetWorkflowByID 按 ID 获取工作流
func TestGetWorkflowByID(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	w := &approval.Workflow{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Name:        "工具部署审批",
		Type:        "tool_deploy",
		Definition:  base.JSON{"steps": []any{}},
	}
	repo.CreateWorkflow(ctx, w)

	found, err := repo.GetWorkflowByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("获取工作流失败: %v", err)
	}
	if found.Name != "工具部署审批" {
		t.Errorf("期望 Name 为 '工具部署审批'，实际为 '%s'", found.Name)
	}
	if found.Type != "tool_deploy" {
		t.Errorf("期望 Type 为 'tool_deploy'，实际为 '%s'", found.Type)
	}
}

// TestGetWorkflowByID_NotFound 获取不存在的工作流返回错误
func TestGetWorkflowByID_NotFound(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	_, err := repo.GetWorkflowByID(ctx, "non-existent-id")
	if err == nil {
		t.Fatal("期望返回错误，但返回 nil")
	}
}

// TestListWorkflows 列出工作流
func TestListWorkflows(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	// 创建多个工作流
	for i := 0; i < 3; i++ {
		w := &approval.Workflow{
			BaseModel:  base.BaseModel{TenantID: "tenant-001"},
			Name:       "工作流-" + string(rune('A'+i)),
			Type:       "tool_deploy",
			Definition: base.JSON{"steps": []any{}},
		}
		repo.CreateWorkflow(ctx, w)
	}

	workflows, total, err := repo.ListWorkflows(ctx, approvalapp.WorkflowFilter{TenantID: "tenant-001", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("列出工作流失败: %v", err)
	}
	if total != 3 {
		t.Errorf("期望 total 为 3，实际为 %d", total)
	}
	if len(workflows) != 3 {
		t.Errorf("期望返回 3 条记录，实际为 %d", len(workflows))
	}
}

// TestListWorkflows_ByStatus 按状态过滤工作流
func TestListWorkflows_ByStatus(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	// 创建不同状态的工作流
	w1 := &approval.Workflow{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "草稿工作流",
		Type:       "tool_deploy",
		Definition: base.JSON{"steps": []any{}},
		Status:     "draft",
	}
	w2 := &approval.Workflow{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "激活工作流",
		Type:       "data_review",
		Definition: base.JSON{"steps": []any{}},
		Status:     "active",
	}
	repo.CreateWorkflow(ctx, w1)
	repo.CreateWorkflow(ctx, w2)

	workflows, total, err := repo.ListWorkflows(ctx, approvalapp.WorkflowFilter{TenantID: "tenant-001", Status: "draft", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("列出工作流失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望 total 为 1，实际为 %d", total)
	}
	if len(workflows) != 1 || workflows[0].Name != "草稿工作流" {
		t.Errorf("期望返回 '草稿工作流'，实际返回 %d 条", len(workflows))
	}
}

// TestUpdateWorkflow 更新工作流
func TestUpdateWorkflow(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	w := &approval.Workflow{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "原始名称",
		Type:       "tool_deploy",
		Definition: base.JSON{"steps": []any{}},
		Status:     "draft",
	}
	repo.CreateWorkflow(ctx, w)

	w.Name = "更新后名称"
	w.Status = "active"
	err := repo.UpdateWorkflow(ctx, w)
	if err != nil {
		t.Fatalf("更新工作流失败: %v", err)
	}

	found, _ := repo.GetWorkflowByID(ctx, w.ID)
	if found.Name != "更新后名称" {
		t.Errorf("期望 Name 为 '更新后名称'，实际为 '%s'", found.Name)
	}
	if found.Status != "active" {
		t.Errorf("期望 Status 为 'active'，实际为 '%s'", found.Status)
	}
}

// ===================== Instance CRUD 测试 =====================

// TestCreateInstance 创建工作流实例成功
func TestCreateInstance(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	inst := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		WorkflowID:  "wf-001",
		Title:       "部署 nginx",
		Status:      "pending",
		CurrentStep: 0,
		InitiatorID: "user-001",
	}

	err := repo.CreateInstance(ctx, inst)
	if err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	if inst.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestCreateInstance_GeneratesUUID 创建实例时自动生成 UUID
func TestCreateInstance_GeneratesUUID(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	inst := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		WorkflowID:  "wf-001",
		Title:       "部署 nginx",
		Status:      "pending",
		InitiatorID: "user-001",
	}

	err := repo.CreateInstance(ctx, inst)
	if err != nil {
		t.Fatalf("创建实例失败: %v", err)
	}
	if inst.ID == "" {
		t.Error("期望创建后自动生成 ID")
	}
}

// TestGetInstanceByID 按 ID 获取实例
func TestGetInstanceByID(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	inst := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		WorkflowID:  "wf-001",
		Title:       "部署 nginx",
		Status:      "pending",
		InitiatorID: "user-001",
	}
	repo.CreateInstance(ctx, inst)

	found, err := repo.GetInstanceByID(ctx, inst.ID)
	if err != nil {
		t.Fatalf("获取实例失败: %v", err)
	}
	if found.Title != "部署 nginx" {
		t.Errorf("期望 Title 为 '部署 nginx'，实际为 '%s'", found.Title)
	}
}

// TestGetInstanceByID_NotFound 获取不存在的实例返回错误
func TestGetInstanceByID_NotFound(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	_, err := repo.GetInstanceByID(ctx, "non-existent-id")
	if err == nil {
		t.Fatal("期望返回错误，但返回 nil")
	}
}

// TestListInstances 列出实例
func TestListInstances(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		inst := &approval.WorkflowInstance{
			BaseModel:   base.BaseModel{TenantID: "tenant-001"},
			WorkflowID:  "wf-001",
			Title:       "实例-" + string(rune('A'+i)),
			Status:      "pending",
			InitiatorID: "user-001",
		}
		repo.CreateInstance(ctx, inst)
	}

	instances, total, err := repo.ListInstances(ctx, approvalapp.InstanceFilter{TenantID: "tenant-001", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("列出实例失败: %v", err)
	}
	if total != 3 {
		t.Errorf("期望 total 为 3，实际为 %d", total)
	}
	if len(instances) != 3 {
		t.Errorf("期望返回 3 条记录，实际为 %d", len(instances))
	}
}

// TestListInstances_ByStatus 按状态过滤实例
func TestListInstances_ByStatus(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	inst1 := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		WorkflowID:  "wf-001",
		Title:       "待审批实例",
		Status:      "approving",
		InitiatorID: "user-001",
	}
	inst2 := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		WorkflowID:  "wf-001",
		Title:       "已通过实例",
		Status:      "approved",
		InitiatorID: "user-001",
	}
	repo.CreateInstance(ctx, inst1)
	repo.CreateInstance(ctx, inst2)

	instances, total, err := repo.ListInstances(ctx, approvalapp.InstanceFilter{TenantID: "tenant-001", Status: "approving", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("列出实例失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望 total 为 1，实际为 %d", total)
	}
	if len(instances) != 1 || instances[0].Title != "待审批实例" {
		t.Errorf("期望返回 '待审批实例'，实际返回 %d 条", len(instances))
	}
}

// TestUpdateInstance 更新实例
func TestUpdateInstance(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	inst := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		WorkflowID:  "wf-001",
		Title:       "原始标题",
		Status:      "pending",
		InitiatorID: "user-001",
	}
	repo.CreateInstance(ctx, inst)

	inst.Status = "approving"
	inst.CurrentStep = 1
	err := repo.UpdateInstance(ctx, inst)
	if err != nil {
		t.Fatalf("更新实例失败: %v", err)
	}

	found, _ := repo.GetInstanceByID(ctx, inst.ID)
	if found.Status != "approving" {
		t.Errorf("期望 Status 为 'approving'，实际为 '%s'", found.Status)
	}
	if found.CurrentStep != 1 {
		t.Errorf("期望 CurrentStep 为 1，实际为 %d", found.CurrentStep)
	}
}

// ===================== Approval CRUD 测试 =====================

// TestCreateApproval 创建审批记录成功
func TestCreateApproval(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	a := &approval.Approval{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		InstanceID:  "inst-001",
		Step:        1,
		ApproverID:  "user-002",
		Status:      "pending",
		Comment:     "请审核",
	}

	err := repo.CreateApproval(ctx, a)
	if err != nil {
		t.Fatalf("创建审批记录失败: %v", err)
	}
	if a.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestCreateApproval_GeneratesUUID 创建审批记录时自动生成 UUID
func TestCreateApproval_GeneratesUUID(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	a := &approval.Approval{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		InstanceID: "inst-001",
		Step:       1,
		ApproverID: "user-002",
		Status:     "pending",
	}

	err := repo.CreateApproval(ctx, a)
	if err != nil {
		t.Fatalf("创建审批记录失败: %v", err)
	}
	if a.ID == "" {
		t.Error("期望创建后自动生成 ID")
	}
}

// TestListApprovalsByInstance 按实例列出审批记录
func TestListApprovalsByInstance(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	instID := "inst-001"
	for i := 0; i < 3; i++ {
		a := &approval.Approval{
			BaseModel:  base.BaseModel{TenantID: "tenant-001"},
			InstanceID: instID,
			Step:       i + 1,
			ApproverID: "user-002",
			Status:     "pending",
		}
		repo.CreateApproval(ctx, a)
	}

	approvals, err := repo.ListApprovalsByInstance(ctx, instID)
	if err != nil {
		t.Fatalf("列出审批记录失败: %v", err)
	}
	if len(approvals) != 3 {
		t.Errorf("期望返回 3 条记录，实际为 %d", len(approvals))
	}
}

// TestGetPendingApproval 获取待审批记录
func TestGetPendingApproval(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	a := &approval.Approval{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		InstanceID: "inst-001",
		Step:       1,
		ApproverID: "user-002",
		Status:     "pending",
	}
	repo.CreateApproval(ctx, a)

	found, err := repo.GetPendingApproval(ctx, "inst-001", 1)
	if err != nil {
		t.Fatalf("获取待审批记录失败: %v", err)
	}
	if found.Status != "pending" {
		t.Errorf("期望 Status 为 'pending'，实际为 '%s'", found.Status)
	}
	if found.Step != 1 {
		t.Errorf("期望 Step 为 1，实际为 %d", found.Step)
	}
}

// TestGetPendingApproval_NotFound 获取不存在的待审批记录返回错误
func TestGetPendingApproval_NotFound(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	_, err := repo.GetPendingApproval(ctx, "inst-001", 99)
	if err == nil {
		t.Fatal("期望返回错误，但返回 nil")
	}
}

// TestUpdateApproval 更新审批记录
func TestUpdateApproval(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	a := &approval.Approval{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		InstanceID: "inst-001",
		Step:       1,
		ApproverID: "user-002",
		Status:     "pending",
	}
	repo.CreateApproval(ctx, a)

	a.Status = "approved"
	a.Comment = "同意"
	err := repo.UpdateApproval(ctx, a)
	if err != nil {
		t.Fatalf("更新审批记录失败: %v", err)
	}

	// 验证更新：通过 ListApprovalsByInstance 查询
	approvals, err := repo.ListApprovalsByInstance(ctx, "inst-001")
	if err != nil {
		t.Fatalf("查询审批记录失败: %v", err)
	}
	if len(approvals) != 1 {
		t.Fatalf("期望 1 条记录，实际为 %d", len(approvals))
	}
	if approvals[0].Status != "approved" {
		t.Errorf("期望 Status 为 'approved'，实际为 '%s'", approvals[0].Status)
	}
	if approvals[0].Comment != "同意" {
		t.Errorf("期望 Comment 为 '同意'，实际为 '%s'", approvals[0].Comment)
	}
}

// ===================== 多租户隔离测试 =====================

// TestTenantIsolation_Workflow 工作流多租户隔离
func TestTenantIsolation_Workflow(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	// 租户 A 创建工作流
	wA := &approval.Workflow{
		BaseModel:  base.BaseModel{TenantID: "tenant-A"},
		Name:       "租户A工作流",
		Type:       "tool_deploy",
		Definition: base.JSON{"steps": []any{}},
	}
	repo.CreateWorkflow(ctx, wA)

	// 租户 B 创建工作流
	wB := &approval.Workflow{
		BaseModel:  base.BaseModel{TenantID: "tenant-B"},
		Name:       "租户B工作流",
		Type:       "data_review",
		Definition: base.JSON{"steps": []any{}},
	}
	repo.CreateWorkflow(ctx, wB)

	// 租户 A 只能看到自己的工作流
	workflows, total, _ := repo.ListWorkflows(ctx, approvalapp.WorkflowFilter{TenantID: "tenant-A", Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("期望租户A有 1 条工作流，实际为 %d", total)
	}
	if len(workflows) != 1 || workflows[0].Name != "租户A工作流" {
		t.Errorf("期望返回 '租户A工作流'，实际返回 %d 条", len(workflows))
	}

	// 租户 A 无法通过 ID 获取租户 B 的工作流
	ctxA := tenant.WithTenantID(ctx, "tenant-A")
	_, err := repo.GetWorkflowByID(ctxA, wB.ID)
	if err == nil {
		t.Error("期望跨租户获取工作流返回错误")
	}
}

// TestTenantIsolation_Instance 实例多租户隔离
func TestTenantIsolation_Instance(t *testing.T) {
	db := setupApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	instA := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-A"},
		WorkflowID:  "wf-001",
		Title:       "租户A实例",
		Status:      "pending",
		InitiatorID: "user-001",
	}
	repo.CreateInstance(ctx, instA)

	instB := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: "tenant-B"},
		WorkflowID:  "wf-001",
		Title:       "租户B实例",
		Status:      "pending",
		InitiatorID: "user-001",
	}
	repo.CreateInstance(ctx, instB)

	instances, total, _ := repo.ListInstances(ctx, approvalapp.InstanceFilter{TenantID: "tenant-A", Page: 1, PageSize: 10})
	if total != 1 {
		t.Errorf("期望租户A有 1 条实例，实际为 %d", total)
	}
	if len(instances) != 1 || instances[0].Title != "租户A实例" {
		t.Errorf("期望返回 '租户A实例'，实际返回 %d 条", len(instances))
	}

	ctxB := tenant.WithTenantID(ctx, "tenant-B")
	_, err := repo.GetInstanceByID(ctxB, instA.ID)
	if err == nil {
		t.Error("期望跨租户获取实例返回错误")
	}
}
