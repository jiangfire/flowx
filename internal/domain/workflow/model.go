package workflow

import (
	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// Workflow 工作流域模型（预留）
type Workflow struct {
	base.BaseModel
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
}
