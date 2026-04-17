package persistence

import (
	"context"
	"fmt"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/notification"
	notifapp "git.neolidy.top/neo/flowx/internal/application/notification"
	"gorm.io/gorm"
)

// ==================== NotificationRepository ====================

// notificationRepository 通知仓储实现
type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(db *gorm.DB) notifapp.NotificationRepository {
	return &notificationRepository{db: db}
}

// Create 创建通知
func (r *notificationRepository) Create(ctx context.Context, n *notification.Notification) error {
	if n.ID == "" {
		n.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(n).Error
}

// GetByID 根据 ID 查询通知
func (r *notificationRepository) GetByID(ctx context.Context, id string) (*notification.Notification, error) {
	var n notification.Notification
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&n).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("通知不存在: %s", id)
		}
		return nil, fmt.Errorf("查询通知失败: %w", err)
	}
	return &n, nil
}

// List 查询通知列表（支持过滤和分页）
func (r *notificationRepository) List(ctx context.Context, filter notifapp.NotificationFilter) ([]notification.Notification, int64, error) {
	var items []notification.Notification
	var total int64

	query := r.db.WithContext(ctx).Model(&notification.Notification{}).Where("tenant_id = ?", filter.TenantID)

	if filter.ReceiverID != "" {
		query = query.Where("receiver_id = ?", filter.ReceiverID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.IsRead != nil {
		query = query.Where("is_read = ?", *filter.IsRead)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计通知数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通知列表失败: %w", err)
	}

	return items, total, nil
}

// Update 更新通知
func (r *notificationRepository) Update(ctx context.Context, n *notification.Notification) error {
	return r.db.WithContext(ctx).Save(n).Error
}

// MarkAsRead 标记通知为已读
func (r *notificationRepository) MarkAsRead(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&notification.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("标记通知已读失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("通知不存在: %s", id)
	}
	return nil
}

// MarkAllAsRead 标记接收者的所有通知为已读
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, receiverID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&notification.Notification{}).
		Where("receiver_id = ? AND is_read = ?", receiverID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("批量标记通知已读失败: %w", result.Error)
	}
	return nil
}

// Delete 软删除通知
func (r *notificationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&notification.Notification{}, "id = ?", id).Error
}

// CountUnread 统计未读通知数量
func (r *notificationRepository) CountUnread(ctx context.Context, receiverID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&notification.Notification{}).
		Where("receiver_id = ? AND is_read = ?", receiverID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计未读通知数量失败: %w", err)
	}
	return count, nil
}

// ==================== NotificationTemplateRepository ====================

// notificationTemplateRepository 通知模板仓储实现
type notificationTemplateRepository struct {
	db *gorm.DB
}

// NewNotificationTemplateRepository 创建通知模板仓储实例
func NewNotificationTemplateRepository(db *gorm.DB) notifapp.NotificationTemplateRepository {
	return &notificationTemplateRepository{db: db}
}

// Create 创建通知模板
func (r *notificationTemplateRepository) Create(ctx context.Context, tpl *notification.NotificationTemplate) error {
	if tpl.ID == "" {
		tpl.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(tpl).Error
}

// GetByID 根据 ID 查询通知模板
func (r *notificationTemplateRepository) GetByID(ctx context.Context, id string) (*notification.NotificationTemplate, error) {
	var tpl notification.NotificationTemplate
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tpl).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("通知模板不存在: %s", id)
		}
		return nil, fmt.Errorf("查询通知模板失败: %w", err)
	}
	return &tpl, nil
}

// GetByCode 根据编码查询通知模板
func (r *notificationTemplateRepository) GetByCode(ctx context.Context, code string) (*notification.NotificationTemplate, error) {
	var tpl notification.NotificationTemplate
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&tpl).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("通知模板不存在: %s", code)
		}
		return nil, fmt.Errorf("查询通知模板失败: %w", err)
	}
	return &tpl, nil
}

// List 查询通知模板列表（支持过滤和分页）
func (r *notificationTemplateRepository) List(ctx context.Context, filter notifapp.NotificationTemplateFilter) ([]notification.NotificationTemplate, int64, error) {
	var items []notification.NotificationTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&notification.NotificationTemplate{}).Where("tenant_id = ?", filter.TenantID)

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计通知模板数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通知模板列表失败: %w", err)
	}

	return items, total, nil
}

// Update 更新通知模板
func (r *notificationTemplateRepository) Update(ctx context.Context, tpl *notification.NotificationTemplate) error {
	return r.db.WithContext(ctx).Save(tpl).Error
}

// Delete 软删除通知模板
func (r *notificationTemplateRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&notification.NotificationTemplate{}, "id = ?", id).Error
}

// ==================== NotificationPreferenceRepository ====================

// notificationPreferenceRepository 通知偏好仓储实现
type notificationPreferenceRepository struct {
	db *gorm.DB
}

// NewNotificationPreferenceRepository 创建通知偏好仓储实例
func NewNotificationPreferenceRepository(db *gorm.DB) notifapp.NotificationPreferenceRepository {
	return &notificationPreferenceRepository{db: db}
}

// Create 创建通知偏好
func (r *notificationPreferenceRepository) Create(ctx context.Context, pref *notification.NotificationPreference) error {
	if pref.ID == "" {
		pref.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Select("id", "tenant_id", "created_at", "updated_at", "user_id", "type", "channel", "enabled", "mute_until").Create(pref).Error
}

// GetByID 根据 ID 查询通知偏好
func (r *notificationPreferenceRepository) GetByID(ctx context.Context, id string) (*notification.NotificationPreference, error) {
	var pref notification.NotificationPreference
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&pref).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("通知偏好不存在: %s", id)
		}
		return nil, fmt.Errorf("查询通知偏好失败: %w", err)
	}
	return &pref, nil
}

// GetByUserAndType 根据用户ID、通知类型和渠道查询通知偏好
func (r *notificationPreferenceRepository) GetByUserAndType(ctx context.Context, userID, typ, channel string) (*notification.NotificationPreference, error) {
	var pref notification.NotificationPreference
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND channel = ?", userID, typ, channel).
		First(&pref).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("通知偏好不存在: user_id=%s, type=%s, channel=%s", userID, typ, channel)
		}
		return nil, fmt.Errorf("查询通知偏好失败: %w", err)
	}
	return &pref, nil
}

// List 查询通知偏好列表（支持过滤和分页）
func (r *notificationPreferenceRepository) List(ctx context.Context, filter notifapp.NotificationPreferenceFilter) ([]notification.NotificationPreference, int64, error) {
	var items []notification.NotificationPreference
	var total int64

	query := r.db.WithContext(ctx).Model(&notification.NotificationPreference{}).Where("tenant_id = ?", filter.TenantID)

	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Channel != "" {
		query = query.Where("channel = ?", filter.Channel)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计通知偏好数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("查询通知偏好列表失败: %w", err)
	}

	return items, total, nil
}

// Update 更新通知偏好
func (r *notificationPreferenceRepository) Update(ctx context.Context, pref *notification.NotificationPreference) error {
	return r.db.WithContext(ctx).Save(pref).Error
}

// Delete 软删除通知偏好
func (r *notificationPreferenceRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&notification.NotificationPreference{}, "id = ?", id).Error
}
