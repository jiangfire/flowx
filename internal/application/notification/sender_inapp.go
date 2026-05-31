package notification

import (
	"context"
	"log/slog"

	notifdomain "github.com/jiangfire/flowx/internal/domain/notification"
)

// InAppSender 站内通知发送器，通知已通过 DB 持久化，此处仅记录日志
type InAppSender struct{}

// NewInAppSender 创建站内通知发送器
func NewInAppSender() *InAppSender {
	return &InAppSender{}
}

// Channel 返回渠道名称
func (s *InAppSender) Channel() string {
	return "in_app"
}

// Send 发送站内通知
func (s *InAppSender) Send(ctx context.Context, n *notifdomain.Notification) error {
	slog.Info("站内通知已发送",
		"id", n.ID,
		"receiver_id", n.ReceiverID,
		"type", n.Type,
	)
	return nil
}
