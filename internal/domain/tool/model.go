package tool

import (
	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// TableName 指定 Tool 表名
func (Tool) TableName() string {
	return "tools"
}

// Tool 工具领域模型
type Tool struct {
	base.BaseModel
	Name        string       `gorm:"size:200;not null" json:"name"`             // 工具名称
	Type        string       `gorm:"size:50;not null;index" json:"type"`        // 工具类型（eda/cae/custom）
	Description string       `gorm:"type:text" json:"description"`              // 工具描述
	ConnectorID string       `gorm:"size:26;index" json:"connector_id"`         // 关联的连接器
	Config      base.JSON    `gorm:"type:jsonb" json:"config"`                  // 工具配置（JSON）
	Status      string       `gorm:"size:20;default:active;index" json:"status"` // 状态：active/inactive/maintenance
	Endpoint    string       `gorm:"size:500" json:"endpoint"`                  // 工具访问地址
	Icon        string       `gorm:"size:100" json:"icon"`                      // 工具图标
	Category    string       `gorm:"size:50;index" json:"category"`             // 工具分类
}

// TableName 指定 Connector 表名
func (Connector) TableName() string {
	return "connectors"
}

// Connector 连接器领域模型
type Connector struct {
	base.BaseModel
	Name       string    `gorm:"size:200;not null" json:"name"`        // 连接器名称
	Type       string    `gorm:"size:50;not null;index" json:"type"`   // 连接器类型（plm/eda/cae/custom）
	Description string   `gorm:"type:text" json:"description"`         // 连接器描述
	Endpoint   string    `gorm:"size:500;not null" json:"endpoint"`    // 连接器访问地址
	Config     base.JSON `gorm:"type:jsonb" json:"config"`             // 连接器配置
	Status     string    `gorm:"size:20;default:active" json:"status"` // 状态
	AuthType   string    `gorm:"size:50" json:"auth_type"`             // 认证类型（none/api_key/oauth/token）
	AuthConfig base.JSON `gorm:"type:jsonb" json:"auth_config"`        // 认证配置
}
