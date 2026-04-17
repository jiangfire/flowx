package persistence

import (
	"context"
	"fmt"

	approvalapp "git.neolidy.top/neo/flowx/internal/application/approval"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/pagination"
	"git.neolidy.top/neo/flowx/pkg/tenant"
	"gorm.io/gorm"
)

// approvalRepository 审批仓储实现
type approvalRepository struct {
	db *gorm.DB
}

// NewApprovalRepository 创建审批仓储实例
func NewApprovalRepository(db *gorm.DB) approvalapp.ApprovalRepository {
	return &approvalRepository{db: db}
}

// ===================== Workflow =====================

// CreateWorkflow 创建工作流
func (r *approvalRepository) CreateWorkflow(ctx context.Context, w *approval.Workflow) error {
	if w.ID == "" {
		w.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(w).Error
}

// GetWorkflowByID 按 ID 获取工作流（多租户隔离）
func (r *approvalRepository) GetWorkflowByID(ctx context.Context, id string) (*approval.Workflow, error) {
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

// ListWorkflows 列出工作流（支持过滤和分页）
func (r *approvalRepository) ListWorkflows(ctx context.Context, filter approvalapp.WorkflowFilter) ([]approval.Workflow, int64, error) {
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

// UpdateWorkflow 更新工作流
func (r *approvalRepository) UpdateWorkflow(ctx context.Context, w *approval.Workflow) error {
	return r.db.WithContext(ctx).Save(w).Error
}

// UpdateWorkflowStatus 更新工作流状态
func (r *approvalRepository) UpdateWorkflowStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&approval.Workflow{}).Where("id = ?", id).Update("status", status).Error
}

// ===================== Instance =====================

// CreateInstance 创建工作流实例
func (r *approvalRepository) CreateInstance(ctx context.Context, inst *approval.WorkflowInstance) error {
	if inst.ID == "" {
		inst.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(inst).Error
}

// GetInstanceByID 按 ID 获取实例（多租户隔离）
func (r *approvalRepository) GetInstanceByID(ctx context.Context, id string) (*approval.WorkflowInstance, error) {
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

// ListInstances 列出实例（支持过滤和分页）
func (r *approvalRepository) ListInstances(ctx context.Context, filter approvalapp.InstanceFilter) ([]approval.WorkflowInstance, int64, error) {
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

// UpdateInstance 更新实例
func (r *approvalRepository) UpdateInstance(ctx context.Context, inst *approval.WorkflowInstance) error {
	return r.db.WithContext(ctx).Save(inst).Error
}

// ===================== Approval =====================

// CreateApproval 创建审批记录
func (r *approvalRepository) CreateApproval(ctx context.Context, a *approval.Approval) error {
	if a.ID == "" {
		a.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(a).Error
}

// ListApprovalsByInstance 按实例列出审批记录
func (r *approvalRepository) ListApprovalsByInstance(ctx context.Context, instanceID string) ([]approval.Approval, error) {
	var approvals []approval.Approval
	if err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("step ASC").Find(&approvals).Error; err != nil {
		return nil, fmt.Errorf("查询审批记录失败: %w", err)
	}
	return approvals, nil
}

// GetPendingApproval 获取指定步骤的待审批记录
func (r *approvalRepository) GetPendingApproval(ctx context.Context, instanceID string, step int) (*approval.Approval, error) {
	var a approval.Approval
	if err := r.db.WithContext(ctx).Where("instance_id = ? AND step = ? AND status = ?", instanceID, step, "pending").First(&a).Error; err != nil {
		return nil, fmt.Errorf("待审批记录不存在: %w", err)
	}
	return &a, nil
}

// UpdateApproval 更新审批记录
func (r *approvalRepository) UpdateApproval(ctx context.Context, a *approval.Approval) error {
	return r.db.WithContext(ctx).Save(a).Error
}

// GetPendingApprovalsByApprover 获取审批人的待审批实例列表（单次查询，无 N+1）
func (r *approvalRepository) GetPendingApprovalsByApprover(ctx context.Context, tenantID, approverID string) ([]approval.WorkflowInstance, error) {
	var instances []approval.WorkflowInstance
	// 使用子查询一次性找出该审批人有待审批记录的实例
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
