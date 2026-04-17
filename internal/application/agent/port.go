package agent

import (
	"context"

	"git.neolidy.top/neo/flowx/internal/domain/agent"
)

// AgentTaskRepository Agent 任务仓储接口
type AgentTaskRepository interface {
	Create(ctx context.Context, task *agent.AgentTask) error
	GetByID(ctx context.Context, id string) (*agent.AgentTask, error)
	Update(ctx context.Context, task *agent.AgentTask) error
	List(ctx context.Context, tenantID, status string, page, pageSize int) ([]agent.AgentTask, int64, error)
}
