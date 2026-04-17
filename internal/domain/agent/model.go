package agent

import (
	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// AgentTask Agent 任务
type AgentTask struct {
	base.BaseModel
	Type            string `gorm:"type:varchar(50);not null;index" json:"type"`
	Description     string `gorm:"type:text" json:"description"`
	Context         string `gorm:"type:text" json:"context"`
	Steps           string `gorm:"type:text" json:"steps"`
	Status          string `gorm:"type:varchar(20);default:created;index" json:"status"`
	Result          string `gorm:"type:text" json:"result"`
	RequireApproval bool   `gorm:"default:false" json:"require_approval"`
	CreatedBy          string `gorm:"type:varchar(26)" json:"created_by"`
	ApprovedBy         string `gorm:"type:varchar(26)" json:"approved_by"`
	ApprovalComment    string `gorm:"type:text" json:"approval_comment"`
	WorkflowInstanceID string `gorm:"size:26;index" json:"workflow_instance_id"`
}

// TableName 指定表名
func (AgentTask) TableName() string {
	return "agent_tasks"
}
