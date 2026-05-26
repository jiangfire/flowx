package agent

import "git.neolidy.top/neo/flowx/internal/domain/base"

// AgentTaskLog Agent 任务执行日志
type AgentTaskLog struct {
	base.BaseModel
	TaskID    string `gorm:"size:26;index;not null" json:"task_id"`
	Step      int    `gorm:"default:0" json:"step"`
	AgentName string `gorm:"size:100" json:"agent_name"`
	Status    string `gorm:"size:20;index" json:"status"`
	Output    string `gorm:"type:text" json:"output"`
	Error     string `gorm:"type:text" json:"error"`
}

func (AgentTaskLog) TableName() string {
	return "agent_task_logs"
}
