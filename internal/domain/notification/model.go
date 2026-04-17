package notification

import (
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// TableName 指定 Notification 表名
func (Notification) TableName() string {
	return "notifications"
}

// Notification 通知领域模型
type Notification struct {
	base.BaseModel
	Title      string       `gorm:"size:200;not null" json:"title"`       // 通知标题
	Content    string       `gorm:"type:text" json:"content"`             // 通知内容
	Type       string       `gorm:"size:50;not null;index" json:"type"`   // 通知类型（system/reminder/alert/message）
	Category   string       `gorm:"size:50;index" json:"category"`        // 通知分类
	Channel    string       `gorm:"size:50" json:"channel"`               // 通知渠道（in_app/email/sms/webhook）
	SenderID   string       `gorm:"size:26" json:"sender_id"`              // 发送者ID（系统通知为空）
	ReceiverID string       `gorm:"size:26;not null;index" json:"receiver_id"` // 接收者ID
	IsRead     bool         `gorm:"default:false;index" json:"is_read"`   // 是否已读
	ReadAt     *time.Time   `json:"read_at"`                             // 阅读时间
	Status     string       `gorm:"size:20;default:pending;index" json:"status"` // 状态：pending/sent/failed
	RefType    string       `gorm:"size:50" json:"ref_type"`              // 关联类型
	RefID      string       `gorm:"size:26" json:"ref_id"`                // 关联ID
	Extra      base.JSON    `gorm:"type:jsonb" json:"extra"`              // 扩展信息
}

// TableName 指定 NotificationTemplate 表名
func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

// NotificationTemplate 通知模板领域模型
type NotificationTemplate struct {
	base.BaseModel
	Name        string     `gorm:"size:200;not null" json:"name"`             // 模板名称
	Code        string     `gorm:"size:100;not null;uniqueIndex" json:"code"` // 模板编码
	Type        string     `gorm:"size:50;not null;index" json:"type"`        // 模板类型
	Channel     string     `gorm:"size:50;not null" json:"channel"`           // 通知渠道
	TitleTpl    string     `gorm:"size:500;not null" json:"title_tpl"`        // 标题模板
	ContentTpl  string     `gorm:"type:text;not null" json:"content_tpl"`     // 内容模板
	Variables   base.JSON  `gorm:"type:jsonb" json:"variables"`               // 模板变量定义
	Description string     `gorm:"type:text" json:"description"`              // 模板描述
	Status      string     `gorm:"size:20;default:active;index" json:"status"` // 状态：active/inactive
}

// TableName 指定 NotificationPreference 表名
func (NotificationPreference) TableName() string {
	return "notification_preferences"
}

// NotificationPreference 通知偏好领域模型
type NotificationPreference struct {
	base.BaseModel
	UserID    string     `gorm:"size:26;not null;index" json:"user_id"`    // 用户ID
	Type      string     `gorm:"size:50;not null;index" json:"type"`       // 通知类型
	Channel   string     `gorm:"size:50;not null" json:"channel"`          // 通知渠道
	Enabled   bool       `gorm:"not null" json:"enabled"`                        // 是否启用
	MuteUntil *time.Time `json:"mute_until"`                               // 免打扰截止时间
}
