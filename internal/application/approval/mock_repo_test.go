package approval

import (
	"context"
	"fmt"

	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/pagination"
	"git.neolidy.top/neo/flowx/pkg/tenant"
	"gorm.io/gorm"
)

// mockApprovalRepository 基于 GORM 的测试用仓储实现（使用 ApprovalRepository 接口）
type mockApprovalRepository struct {
	db *gorm.DB
}

func newMockApprovalRepository(db *gorm.DB) ApprovalRepository {
	return &mockApprovalRepository{db: db}
}

func (r *mockApprovalRepository) CreateWorkflow(ctx context.Context, w *approval.Workflow) error {
	if w.ID == "" {
		w.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(w).Error
}

func (r *mockApprovalRepository) GetWorkflowByID(ctx context.Context, id string) (*approval.Workflow, error) {
	var w approval.Workflow
	tenantID := tenant.TenantIDFromContext(ctx)
	q := r.db.WithContext(ctx).Where("id = ?", id)
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.First(&w).Error; err != nil {
		return nil, fmt.Errorf("工作流不存在: %w", err)
	}
	return &w, nil
}

func (r *mockApprovalRepository) ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]approval.Workflow, int64, error) {
	var workflows []approval.Workflow
	var total int64
	q := r.db.WithContext(ctx).Model(&approval.Workflow{}).Where("tenant_id = ?", filter.TenantID)
	if filter.Type != "" {
		q = q.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计工作流数量失败: %w", err)
	}
	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	offset := pagination.Offset(page, pageSize)
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&workflows).Error; err != nil {
		return nil, 0, fmt.Errorf("查询工作流列表失败: %w", err)
	}
	return workflows, total, nil
}

func (r *mockApprovalRepository) UpdateWorkflow(ctx context.Context, w *approval.Workflow) error {
	return r.db.WithContext(ctx).Save(w).Error
}

func (r *mockApprovalRepository) UpdateWorkflowStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&approval.Workflow{}).Where("id = ?", id).Update("status", status).Error
}

func (r *mockApprovalRepository) CreateInstance(ctx context.Context, inst *approval.WorkflowInstance) error {
	if inst.ID == "" {
		inst.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(inst).Error
}

func (r *mockApprovalRepository) GetInstanceByID(ctx context.Context, id string) (*approval.WorkflowInstance, error) {
	var inst approval.WorkflowInstance
	tenantID := tenant.TenantIDFromContext(ctx)
	q := r.db.WithContext(ctx).Where("id = ?", id)
	if tenantID != "" {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if err := q.First(&inst).Error; err != nil {
		return nil, fmt.Errorf("工作流实例不存在: %w", err)
	}
	return &inst, nil
}

func (r *mockApprovalRepository) ListInstances(ctx context.Context, filter InstanceFilter) ([]approval.WorkflowInstance, int64, error) {
	var instances []approval.WorkflowInstance
	var total int64
	q := r.db.WithContext(ctx).Model(&approval.WorkflowInstance{}).Where("tenant_id = ?", filter.TenantID)
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.WorkflowID != "" {
		q = q.Where("workflow_id = ?", filter.WorkflowID)
	}
	if filter.InitiatorID != "" {
		q = q.Where("initiator_id = ?", filter.InitiatorID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计实例数量失败: %w", err)
	}
	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	offset := pagination.Offset(page, pageSize)
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&instances).Error; err != nil {
		return nil, 0, fmt.Errorf("查询实例列表失败: %w", err)
	}
	return instances, total, nil
}

func (r *mockApprovalRepository) UpdateInstance(ctx context.Context, inst *approval.WorkflowInstance) error {
	return r.db.WithContext(ctx).Save(inst).Error
}

func (r *mockApprovalRepository) CreateApproval(ctx context.Context, a *approval.Approval) error {
	if a.ID == "" {
		a.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *mockApprovalRepository) ListApprovalsByInstance(ctx context.Context, instanceID string) ([]approval.Approval, error) {
	var approvals []approval.Approval
	if err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("step ASC").Find(&approvals).Error; err != nil {
		return nil, fmt.Errorf("查询审批记录失败: %w", err)
	}
	return approvals, nil
}

func (r *mockApprovalRepository) GetPendingApproval(ctx context.Context, instanceID string, step int) (*approval.Approval, error) {
	var a approval.Approval
	if err := r.db.WithContext(ctx).Where("instance_id = ? AND step = ? AND status = ?", instanceID, step, "pending").First(&a).Error; err != nil {
		return nil, fmt.Errorf("待审批记录不存在: %w", err)
	}
	return &a, nil
}

func (r *mockApprovalRepository) UpdateApproval(ctx context.Context, a *approval.Approval) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *mockApprovalRepository) GetPendingApprovalsByApprover(ctx context.Context, tenantID, approverID string) ([]approval.WorkflowInstance, error) {
	var instances []approval.WorkflowInstance
	err := r.db.WithContext(ctx).
		Where(`tenant_id = ? AND status = ? AND id IN (
			SELECT DISTINCT instance_id FROM approvals
			WHERE step = (workflow_instances.current_step + 1)
			AND approver_id = ? AND status = 'pending'
		)`, tenantID, "approving", approverID).
		Order("created_at DESC").
		Find(&instances).Error
	if err != nil {
		return nil, fmt.Errorf("查询待审批列表失败: %w", err)
	}
	return instances, nil
}
