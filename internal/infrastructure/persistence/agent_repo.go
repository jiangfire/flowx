package persistence

import (
	"context"
	"fmt"

	agentapp "git.neolidy.top/neo/flowx/internal/application/agent"
	"git.neolidy.top/neo/flowx/internal/domain/agent"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/pagination"
	"gorm.io/gorm"
)

type agentTaskRepository struct {
	db *gorm.DB
}

func NewAgentTaskRepository(db *gorm.DB) agentapp.AgentTaskRepository {
	return &agentTaskRepository{db: db}
}

func (r *agentTaskRepository) Create(ctx context.Context, task *agent.AgentTask) error {
	if task.ID == "" {
		task.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *agentTaskRepository) GetByID(ctx context.Context, id string) (*agent.AgentTask, error) {
	var task agent.AgentTask
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return &task, nil
}

func (r *agentTaskRepository) Update(ctx context.Context, task *agent.AgentTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *agentTaskRepository) List(ctx context.Context, tenantID, status string, page, pageSize int) ([]agent.AgentTask, int64, error) {
	page, pageSize = pagination.NormalizePage(page, pageSize)
	query := r.db.WithContext(ctx).Model(&agent.AgentTask{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计任务数量失败: %w", err)
	}
	var tasks []agent.AgentTask
	offset := pagination.Offset(page, pageSize)
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败: %w", err)
	}
	return tasks, total, nil
}
