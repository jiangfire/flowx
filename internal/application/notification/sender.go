package notification

import (
	"context"

	notifdomain "github.com/jiangfire/flowx/internal/domain/notification"
)

// ChannelSender 通知渠道发送器接口
type ChannelSender interface {
	Channel() string
	Send(ctx context.Context, n *notifdomain.Notification) error
}

// Notifier 简化通知接口，供其他模块使用
type Notifier interface {
	SendSimple(ctx context.Context, tenantID string, req *SendSimpleRequest) (*notifdomain.Notification, error)
}
