package bpmn

import (
	"context"

	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
)

// ProcessDefinitionFilter 流程定义查询过滤
type ProcessDefinitionFilter struct {
	TenantID string
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

// ProcessInstanceFilter 流程实例查询过滤
type ProcessInstanceFilter struct {
	TenantID string
	Status   string
	Page     int
	PageSize int
}

// ProcessDefinitionRepository 流程定义仓储
type ProcessDefinitionRepository interface {
	Create(ctx context.Context, def *bpmn.ProcessDefinition) error
	GetByID(ctx context.Context, id string) (*bpmn.ProcessDefinition, error)
	List(ctx context.Context, filter ProcessDefinitionFilter) ([]*bpmn.ProcessDefinition, int64, error)
	Update(ctx context.Context, def *bpmn.ProcessDefinition) error
	Delete(ctx context.Context, id string) error
}

// ProcessInstanceRepository 流程实例仓储
type ProcessInstanceRepository interface {
	Create(ctx context.Context, inst *bpmn.ProcessInstance) error
	GetByID(ctx context.Context, id string) (*bpmn.ProcessInstance, error)
	List(ctx context.Context, filter ProcessInstanceFilter) ([]*bpmn.ProcessInstance, int64, error)
	Update(ctx context.Context, inst *bpmn.ProcessInstance) error
}

// ProcessTaskRepository 流程任务仓储
type ProcessTaskRepository interface {
	Create(ctx context.Context, task *bpmn.ProcessTask) error
	GetByID(ctx context.Context, id string) (*bpmn.ProcessTask, error)
	ListByInstance(ctx context.Context, instanceID string) ([]*bpmn.ProcessTask, error)
	ListPending(ctx context.Context, tenantID, assignee string) ([]*bpmn.ProcessTask, error)
	Update(ctx context.Context, task *bpmn.ProcessTask) error
}

// ExecutionHistoryRepository 执行历史仓储
type ExecutionHistoryRepository interface {
	Create(ctx context.Context, h *bpmn.ExecutionHistory) error
	ListByInstance(ctx context.Context, instanceID string) ([]*bpmn.ExecutionHistory, error)
}
