package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	notifdomain "github.com/jiangfire/flowx/internal/domain/notification"
)

// WebhookSender webhook 通知发送器
type WebhookSender struct {
	webhookURL string
	timeout    time.Duration
	httpClient *http.Client
}

// WebhookConfig webhook 发送器配置
type WebhookConfig struct {
	URL        string
	TimeoutSec int
}

// NewWebhookSender 创建 webhook 通知发送器
func NewWebhookSender(cfg WebhookConfig) *WebhookSender {
	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookSender{
		webhookURL: cfg.URL,
		timeout:    timeout,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Channel 返回渠道名称
func (s *WebhookSender) Channel() string {
	return "webhook"
}

// webhookPayload webhook 请求体
type webhookPayload struct {
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	Type       string         `json:"type"`
	Category   string         `json:"category,omitempty"`
	ReceiverID string         `json:"receiver_id"`
	RefType    string         `json:"ref_type,omitempty"`
	RefID      string         `json:"ref_id,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
	Timestamp  string         `json:"timestamp"`
}

// Send 通过 webhook 发送通知
func (s *WebhookSender) Send(ctx context.Context, n *notifdomain.Notification) error {
	if s.webhookURL == "" {
		slog.Warn("webhook URL 未配置，跳过发送", "notification_id", n.ID)
		return nil
	}

	payload := webhookPayload{
		Title:      n.Title,
		Content:    n.Content,
		Type:       n.Type,
		Category:   n.Category,
		ReceiverID: n.ReceiverID,
		RefType:    n.RefType,
		RefID:      n.RefID,
		Extra:      n.Extra,
		Timestamp:  n.CreatedAt.Format(time.RFC3339),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook 序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("webhook 创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 返回异常状态码: %d", resp.StatusCode)
	}

	slog.Info("webhook 通知已发送",
		"id", n.ID,
		"receiver_id", n.ReceiverID,
		"webhook_url", s.webhookURL,
	)
	return nil
}
