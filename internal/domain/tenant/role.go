package tenant

import (
	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// Role 角色域模型
type Role struct {
	base.BaseModel
	Name        string `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	DisplayName string `gorm:"type:varchar(255);not null" json:"display_name"`
	Description string `gorm:"type:varchar(500)" json:"description"`
}

// TableName 指定 Role 表名
func (Role) TableName() string {
	return "roles"
}

// Permission 权限域模型
type Permission struct {
	base.BaseModel
	Name        string `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	DisplayName string `gorm:"type:varchar(255);not null" json:"display_name"`
	Resource    string `gorm:"type:varchar(100);not null" json:"resource"`
	Action      string `gorm:"type:varchar(50);not null" json:"action"`
}

// TableName 指定 Permission 表名
func (Permission) TableName() string {
	return "permissions"
}
