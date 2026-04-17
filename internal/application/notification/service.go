package notification

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	notifdomain "git.neolidy.top/neo/flowx/internal/domain/notification"
	"git.neolidy.top/neo/flowx/pkg/pagination"
)

var (
	ErrNotificationNotFound      = errors.New("通知不存在")
	ErrTemplateNotFound          = errors.New("通知模板不存在")
	ErrPreferenceNotFound        = errors.New("通知偏好不存在")
	ErrTenantMismatch            = errors.New("租户不匹配")
	ErrNotificationTitleRequired = errors.New("通知标题不能为空")
	ErrNotificationTypeRequired  = errors.New("通知类型不能为空")
	ErrTemplateCodeRequired      = errors.New("模板编码不能为空")
	ErrUserMuted                 = errors.New("用户已设置免打扰")
)

// Request DTOs

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	Title      string     `json:"title" binding:"required"`
	Content    string     `json:"content"`
	Type       string     `json:"type" binding:"required"`
	Category   string     `json:"category"`
	Channel    string     `json:"channel"`
	ReceiverID string     `json:"receiver_id" binding:"required"`
	RefType    string     `json:"ref_type"`
	RefID      string     `json:"ref_id"`
	Extra      base.JSON  `json:"extra"`
}

// UpdateNotificationRequest 更新通知请求
type UpdateNotificationRequest struct {
	Title    *string    `json:"title"`
	Content  *string    `json:"content"`
	Type     *string    `json:"type"`
	Category *string    `json:"category"`
	Channel  *string    `json:"channel"`
	RefType  *string    `json:"ref_type"`
	RefID    *string    `json:"ref_id"`
	Extra    *base.JSON `json:"extra"`
}

// ListNotificationsFilter 通知列表过滤条件
type ListNotificationsFilter struct {
	Type     string `form:"type"`
	Category string `form:"category"`
	IsRead   *bool  `form:"is_read"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string    `json:"name" binding:"required"`
	Code        string    `json:"code" binding:"required"`
	Type        string    `json:"type" binding:"required"`
	Channel     string    `json:"channel" binding:"required"`
	TitleTpl    string    `json:"title_tpl" binding:"required"`
	ContentTpl  string    `json:"content_tpl" binding:"required"`
	Variables   base.JSON `json:"variables"`
	Description string    `json:"description"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        *string    `json:"name"`
	Channel     *string    `json:"channel"`
	TitleTpl    *string    `json:"title_tpl"`
	ContentTpl  *string    `json:"content_tpl"`
	Variables   base.JSON  `json:"variables"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
}

// CreatePreferenceRequest 创建偏好请求
type CreatePreferenceRequest struct {
	UserID    string     `json:"user_id" binding:"required"`
	Type      string     `json:"type" binding:"required"`
	Channel   string     `json:"channel" binding:"required"`
	Enabled   *bool      `json:"enabled"`
	MuteUntil *time.Time `json:"mute_until"`
}

// UpdatePreferenceRequest 更新偏好请求
type UpdatePreferenceRequest struct {
	Enabled   *bool      `json:"enabled"`
	MuteUntil *time.Time `json:"mute_until"`
}

// ListPreferencesFilter 通知偏好列表过滤条件
type ListPreferencesFilter struct {
	UserID   string `form:"user_id"`
	Type     string `form:"type"`
	Channel  string `form:"channel"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// SendNotificationRequest 发送通知请求
type SendNotificationRequest struct {
	TemplateCode string            `json:"template_code" binding:"required"`
	ReceiverID   string            `json:"receiver_id" binding:"required"`
	Variables    map[string]string `json:"variables"`
	RefType      string            `json:"ref_type"`
	RefID        string            `json:"ref_id"`
	Channel      string            `json:"channel"`
}

// NotificationService 通知应用服务
type NotificationService struct {
	notifRepo      NotificationRepository
	templateRepo   NotificationTemplateRepository
	preferenceRepo NotificationPreferenceRepository
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(
	notifRepo NotificationRepository,
	templateRepo NotificationTemplateRepository,
	preferenceRepo NotificationPreferenceRepository,
) *NotificationService {
	return &NotificationService{
		notifRepo:      notifRepo,
		templateRepo:   templateRepo,
		preferenceRepo: preferenceRepo,
	}
}

// ==================== Notification CRUD ====================

// CreateNotification 创建通知
func (s *NotificationService) CreateNotification(ctx context.Context, tenantID string, req *CreateNotificationRequest) (*notifdomain.Notification, error) {
	if req.Title == "" {
		return nil, ErrNotificationTitleRequired
	}
	if req.Type == "" {
		return nil, ErrNotificationTypeRequired
	}

	status := "sent"
	n := &notifdomain.Notification{
		BaseModel:  base.BaseModel{TenantID: tenantID},
		Title:      req.Title,
		Content:    req.Content,
		Type:       req.Type,
		Category:   req.Category,
		Channel:    req.Channel,
		ReceiverID: req.ReceiverID,
		Status:     status,
		RefType:    req.RefType,
		RefID:      req.RefID,
		Extra:      req.Extra,
	}

	if err := s.notifRepo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("创建通知失败: %w", err)
	}

	return n, nil
}

// GetNotification 获取通知详情
func (s *NotificationService) GetNotification(ctx context.Context, tenantID string, id string) (*notifdomain.Notification, error) {
	n, err := s.notifRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotificationNotFound
	}

	if n.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return n, nil
}

// ListNotifications 查询通知列表
func (s *NotificationService) ListNotifications(ctx context.Context, tenantID string, receiverID string, filter ListNotificationsFilter) ([]notifdomain.Notification, *pagination.PaginatedResult, error) {
	notifications, total, err := s.notifRepo.List(ctx, NotificationFilter{
		TenantID:   tenantID,
		ReceiverID: receiverID,
		Type:       filter.Type,
		Category:   filter.Category,
		IsRead:     filter.IsRead,
		Status:     filter.Status,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询通知列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return notifications, pagination.NewResult(total, page, pageSize), nil
}

// UpdateNotification 更新通知
func (s *NotificationService) UpdateNotification(ctx context.Context, tenantID string, id string, req *UpdateNotificationRequest) (*notifdomain.Notification, error) {
	existing, err := s.notifRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotificationNotFound
	}

	if existing.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Content != nil {
		existing.Content = *req.Content
	}
	if req.Type != nil {
		existing.Type = *req.Type
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Channel != nil {
		existing.Channel = *req.Channel
	}
	if req.RefType != nil {
		existing.RefType = *req.RefType
	}
	if req.RefID != nil {
		existing.RefID = *req.RefID
	}
	if req.Extra != nil {
		existing.Extra = *req.Extra
	}

	if err := s.notifRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新通知失败: %w", err)
	}

	return existing, nil
}

// DeleteNotification 删除通知
func (s *NotificationService) DeleteNotification(ctx context.Context, tenantID string, id string) error {
	existing, err := s.notifRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotificationNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.notifRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除通知失败: %w", err)
	}

	return nil
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(ctx context.Context, tenantID string, id string) error {
	existing, err := s.notifRepo.GetByID(ctx, id)
	if err != nil {
		return ErrNotificationNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.notifRepo.MarkAsRead(ctx, id); err != nil {
		return fmt.Errorf("标记已读失败: %w", err)
	}

	return nil
}

// MarkAllAsRead 标记所有通知为已读
func (s *NotificationService) MarkAllAsRead(ctx context.Context, receiverID string) error {
	if err := s.notifRepo.MarkAllAsRead(ctx, receiverID); err != nil {
		return fmt.Errorf("全部标记已读失败: %w", err)
	}
	return nil
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(ctx context.Context, receiverID string) (int64, error) {
	count, err := s.notifRepo.CountUnread(ctx, receiverID)
	if err != nil {
		return 0, fmt.Errorf("获取未读数量失败: %w", err)
	}
	return count, nil
}

// ==================== Template CRUD ====================

// CreateTemplate 创建通知模板
func (s *NotificationService) CreateTemplate(ctx context.Context, tenantID string, req *CreateTemplateRequest) (*notifdomain.NotificationTemplate, error) {
	if req.Code == "" {
		return nil, ErrTemplateCodeRequired
	}

	status := "active"
	tpl := &notifdomain.NotificationTemplate{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        req.Name,
		Code:        req.Code,
		Type:        req.Type,
		Channel:     req.Channel,
		TitleTpl:    req.TitleTpl,
		ContentTpl:  req.ContentTpl,
		Variables:   req.Variables,
		Description: req.Description,
		Status:      status,
	}

	if err := s.templateRepo.Create(ctx, tpl); err != nil {
		return nil, fmt.Errorf("创建通知模板失败: %w", err)
	}

	return tpl, nil
}

// GetTemplate 获取通知模板详情
func (s *NotificationService) GetTemplate(ctx context.Context, tenantID string, id string) (*notifdomain.NotificationTemplate, error) {
	tpl, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	if tpl.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return tpl, nil
}

// GetTemplateByCode 根据编码获取通知模板
func (s *NotificationService) GetTemplateByCode(ctx context.Context, tenantID string, code string) (*notifdomain.NotificationTemplate, error) {
	tpl, err := s.templateRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	if tpl.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return tpl, nil
}

// ListTemplates 查询通知模板列表
func (s *NotificationService) ListTemplates(ctx context.Context, tenantID string, filter ListNotificationsFilter) ([]notifdomain.NotificationTemplate, *pagination.PaginatedResult, error) {
	templates, total, err := s.templateRepo.List(ctx, NotificationTemplateFilter{
		TenantID: tenantID,
		Type:     filter.Type,
		Status:   filter.Status,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询通知模板列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return templates, pagination.NewResult(total, page, pageSize), nil
}

// UpdateTemplate 更新通知模板
func (s *NotificationService) UpdateTemplate(ctx context.Context, tenantID string, id string, req *UpdateTemplateRequest) (*notifdomain.NotificationTemplate, error) {
	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	if existing.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Channel != nil {
		existing.Channel = *req.Channel
	}
	if req.TitleTpl != nil {
		existing.TitleTpl = *req.TitleTpl
	}
	if req.ContentTpl != nil {
		existing.ContentTpl = *req.ContentTpl
	}
	if req.Variables != nil {
		existing.Variables = req.Variables
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}

	if err := s.templateRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新通知模板失败: %w", err)
	}

	return existing, nil
}

// DeleteTemplate 删除通知模板
func (s *NotificationService) DeleteTemplate(ctx context.Context, tenantID string, id string) error {
	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return ErrTemplateNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.templateRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除通知模板失败: %w", err)
	}

	return nil
}

// ==================== Preference CRUD ====================

// CreatePreference 创建通知偏好
func (s *NotificationService) CreatePreference(ctx context.Context, tenantID string, req *CreatePreferenceRequest) (*notifdomain.NotificationPreference, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	pref := &notifdomain.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: tenantID},
		UserID:    req.UserID,
		Type:      req.Type,
		Channel:   req.Channel,
		Enabled:   enabled,
		MuteUntil: req.MuteUntil,
	}

	if err := s.preferenceRepo.Create(ctx, pref); err != nil {
		return nil, fmt.Errorf("创建通知偏好失败: %w", err)
	}

	return pref, nil
}

// GetPreference 获取通知偏好详情
func (s *NotificationService) GetPreference(ctx context.Context, tenantID string, id string) (*notifdomain.NotificationPreference, error) {
	pref, err := s.preferenceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPreferenceNotFound
	}

	if pref.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return pref, nil
}

// ListPreferences 查询通知偏好列表
func (s *NotificationService) ListPreferences(ctx context.Context, tenantID string, filter ListPreferencesFilter) ([]notifdomain.NotificationPreference, *pagination.PaginatedResult, error) {
	preferences, total, err := s.preferenceRepo.List(ctx, NotificationPreferenceFilter{
		TenantID: tenantID,
		UserID:   filter.UserID,
		Type:     filter.Type,
		Channel:  filter.Channel,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询通知偏好列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return preferences, pagination.NewResult(total, page, pageSize), nil
}

// UpdatePreference 更新通知偏好
func (s *NotificationService) UpdatePreference(ctx context.Context, tenantID string, id string, req *UpdatePreferenceRequest) (*notifdomain.NotificationPreference, error) {
	existing, err := s.preferenceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPreferenceNotFound
	}

	if existing.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.MuteUntil != nil {
		existing.MuteUntil = req.MuteUntil
	}

	if err := s.preferenceRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新通知偏好失败: %w", err)
	}

	return existing, nil
}

// DeletePreference 删除通知偏好
func (s *NotificationService) DeletePreference(ctx context.Context, tenantID string, id string) error {
	existing, err := s.preferenceRepo.GetByID(ctx, id)
	if err != nil {
		return ErrPreferenceNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.preferenceRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除通知偏好失败: %w", err)
	}

	return nil
}

// ==================== Business: SendNotification ====================

// renderTemplate 渲染模板，替换 {{.var}} 占位符
var placeholderRegex = regexp.MustCompile(`\{\{\.(\w+)\}\}`)

func renderTemplate(tpl string, variables map[string]string) string {
	result := tpl
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{{."+key+"}}", value)
	}
	// 清理未匹配的占位符，替换为空字符串
	result = placeholderRegex.ReplaceAllString(result, "")
	return result
}

// SendNotification 通过模板发送通知
func (s *NotificationService) SendNotification(ctx context.Context, tenantID string, req *SendNotificationRequest) (*notifdomain.Notification, error) {
	// 查找模板
	tpl, err := s.templateRepo.GetByCode(ctx, req.TemplateCode)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	// 多租户校验
	if tpl.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	// 检查用户偏好（是否被免打扰）
	pref, err := s.preferenceRepo.GetByUserAndType(ctx, req.ReceiverID, tpl.Type, tpl.Channel)
	if err == nil {
		// 偏好存在，检查是否被免打扰
		if pref.MuteUntil != nil && pref.MuteUntil.After(time.Now()) {
			return nil, ErrUserMuted
		}
		if !pref.Enabled {
			return nil, ErrUserMuted
		}
	}

	// 渲染模板
	variables := req.Variables
	if variables == nil {
		variables = make(map[string]string)
	}
	title := renderTemplate(tpl.TitleTpl, variables)
	content := renderTemplate(tpl.ContentTpl, variables)

	// 确定渠道
	channel := req.Channel
	if channel == "" {
		channel = tpl.Channel
	}

	// 创建通知
	n := &notifdomain.Notification{
		BaseModel:  base.BaseModel{TenantID: tenantID},
		Title:      title,
		Content:    content,
		Type:       tpl.Type,
		Channel:    channel,
		ReceiverID: req.ReceiverID,
		Status:     "sent",
		RefType:    req.RefType,
		RefID:      req.RefID,
	}

	if err := s.notifRepo.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("创建通知失败: %w", err)
	}

	return n, nil
}
