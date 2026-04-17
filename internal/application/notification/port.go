package notification

import (
	"context"

	notifdomain "git.neolidy.top/neo/flowx/internal/domain/notification"
)

// NotificationFilter 通知查询过滤条件
type NotificationFilter struct {
	TenantID   string
	ReceiverID string
	Type       string
	Category   string
	IsRead     *bool
	Status     string
	Page       int
	PageSize   int
}

// NotificationTemplateFilter 通知模板查询过滤条件
type NotificationTemplateFilter struct {
	TenantID string
	Type     string
	Status   string
	Page     int
	PageSize int
}

// NotificationPreferenceFilter 通知偏好查询过滤条件
type NotificationPreferenceFilter struct {
	TenantID string
	UserID   string
	Type     string
	Channel  string
	Page     int
	PageSize int
}

// NotificationRepository 通知仓储接口
type NotificationRepository interface {
	Create(ctx context.Context, n *notifdomain.Notification) error
	GetByID(ctx context.Context, id string) (*notifdomain.Notification, error)
	List(ctx context.Context, filter NotificationFilter) ([]notifdomain.Notification, int64, error)
	Update(ctx context.Context, n *notifdomain.Notification) error
	MarkAsRead(ctx context.Context, id string) error
	MarkAllAsRead(ctx context.Context, receiverID string) error
	Delete(ctx context.Context, id string) error
	CountUnread(ctx context.Context, receiverID string) (int64, error)
}

// NotificationTemplateRepository 通知模板仓储接口
type NotificationTemplateRepository interface {
	Create(ctx context.Context, tpl *notifdomain.NotificationTemplate) error
	GetByID(ctx context.Context, id string) (*notifdomain.NotificationTemplate, error)
	GetByCode(ctx context.Context, code string) (*notifdomain.NotificationTemplate, error)
	List(ctx context.Context, filter NotificationTemplateFilter) ([]notifdomain.NotificationTemplate, int64, error)
	Update(ctx context.Context, tpl *notifdomain.NotificationTemplate) error
	Delete(ctx context.Context, id string) error
}

// NotificationPreferenceRepository 通知偏好仓储接口
type NotificationPreferenceRepository interface {
	Create(ctx context.Context, pref *notifdomain.NotificationPreference) error
	GetByID(ctx context.Context, id string) (*notifdomain.NotificationPreference, error)
	GetByUserAndType(ctx context.Context, userID, typ, channel string) (*notifdomain.NotificationPreference, error)
	List(ctx context.Context, filter NotificationPreferenceFilter) ([]notifdomain.NotificationPreference, int64, error)
	Update(ctx context.Context, pref *notifdomain.NotificationPreference) error
	Delete(ctx context.Context, id string) error
}
