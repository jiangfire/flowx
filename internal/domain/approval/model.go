package approval

import (
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// Workflow 工作流定义模型
type Workflow struct {
	base.BaseModel
	Name        string    `gorm:"size:200;not null" json:"name"`
	Type        string    `gorm:"size:50;not null;index" json:"type"` // tool_deploy/data_review/change_request/custom
	Description string    `gorm:"type:text" json:"description"`
	Definition  base.JSON `gorm:"type:jsonb;not null" json:"definition"` // 工作流步骤定义（JSON）
	Version     int       `gorm:"default:1" json:"version"`
	Status      string    `gorm:"size:20;default:draft;index" json:"status"` // draft/active/archived
}

// TableName 指定 Workflow 表名
func (Workflow) TableName() string {
	return "workflows"
}

// WorkflowInstance 工作流实例模型
type WorkflowInstance struct {
	base.BaseModel
	WorkflowID  string    `gorm:"size:26;not null;index" json:"workflow_id"`
	Title       string    `gorm:"size:500;not null" json:"title"`
	Status      string    `gorm:"size:20;not null;index" json:"status"` // pending/approving/rejected/approved/cancelled
	CurrentStep int       `gorm:"default:0" json:"current_step"`
	InitiatorID string    `gorm:"size:26;not null;index" json:"initiator_id"`
	Context     base.JSON `gorm:"type:jsonb" json:"context"`  // 审批上下文数据
	Result      base.JSON `gorm:"type:jsonb" json:"result"`    // 审批结果数据
	AgentTaskID string    `gorm:"size:26;index" json:"agent_task_id"`
}

// TableName 指定 WorkflowInstance 表名
func (WorkflowInstance) TableName() string {
	return "workflow_instances"
}

// Approval 审批记录模型
type Approval struct {
	base.BaseModel
	InstanceID   string     `gorm:"size:26;not null;index" json:"instance_id"`
	Step         int        `gorm:"not null" json:"step"`
	ApproverID   string     `gorm:"size:26;not null;index" json:"approver_id"`
	Status       string     `gorm:"size:20;not null;index" json:"status"` // pending/approved/rejected/forwarded
	Comment      string     `gorm:"type:text" json:"comment"`
	AISuggestion string     `gorm:"type:text" json:"ai_suggestion"` // AI 的审批建议
	ReviewedAt   *time.Time `json:"reviewed_at"`
}

// TableName 指定 Approval 表名
func (Approval) TableName() string {
	return "approvals"
}
