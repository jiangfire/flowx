package tenant

import (
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"gorm.io/gorm"
)

// Tenant 租户域模型（预留）
type Tenant struct {
	base.BaseModel
	Name string `gorm:"type:varchar(255);not null" json:"name"`
}

// User 用户域模型
type User struct {
	base.BaseModel
	Username     string `gorm:"type:varchar(100);not null;index" json:"username"`
	Email        string `gorm:"type:varchar(255);not null;index" json:"email"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"`
	Role         string `gorm:"type:varchar(50);not null;default:user" json:"role"`
	Status       string `gorm:"type:varchar(20);not null;default:active" json:"status"`
}

// TableName 指定 User 表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前钩子，自动生成 UUID v7
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = base.GenerateUUID()
	}
	return nil
}
