package approval

import (
	"context"

	"git.neolidy.top/neo/flowx/internal/domain/approval"
)

// WorkflowFilter 工作流查询过滤条件
type WorkflowFilter struct {
	TenantID string
	Type     string
	Status   string
	Page     int
	PageSize int
}

// InstanceFilter 实例查询过滤条件
type InstanceFilter struct {
	TenantID    string
	Status      string
	WorkflowID  string
	InitiatorID string
	Page        int
	PageSize    int
}

// ApprovalFilter 审批查询过滤条件
type ApprovalFilter struct {
	TenantID    string
	InstanceID  string
	ApproverID  string
	Status      string
	Page        int
	PageSize    int
}

// ApprovalRepository 审批仓储接口
type ApprovalRepository interface {
	// Workflow
	CreateWorkflow(ctx context.Context, w *approval.Workflow) error
	GetWorkflowByID(ctx context.Context, id string) (*approval.Workflow, error)
	ListWorkflows(ctx context.Context, filter WorkflowFilter) ([]approval.Workflow, int64, error)
	UpdateWorkflow(ctx context.Context, w *approval.Workflow) error
	UpdateWorkflowStatus(ctx context.Context, id string, status string) error

	// Instance
	CreateInstance(ctx context.Context, inst *approval.WorkflowInstance) error
	GetInstanceByID(ctx context.Context, id string) (*approval.WorkflowInstance, error)
	ListInstances(ctx context.Context, filter InstanceFilter) ([]approval.WorkflowInstance, int64, error)
	UpdateInstance(ctx context.Context, inst *approval.WorkflowInstance) error

	// Approval
	CreateApproval(ctx context.Context, a *approval.Approval) error
	ListApprovalsByInstance(ctx context.Context, instanceID string) ([]approval.Approval, error)
	GetPendingApproval(ctx context.Context, instanceID string, step int) (*approval.Approval, error)
	UpdateApproval(ctx context.Context, a *approval.Approval) error

	// Pending approvals by approver
	GetPendingApprovalsByApprover(ctx context.Context, tenantID, approverID string) ([]approval.WorkflowInstance, error)
}
