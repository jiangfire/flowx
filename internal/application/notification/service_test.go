package notification_test

import (
	"context"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/notification"
	notifapp "git.neolidy.top/neo/flowx/internal/application/notification"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupServiceTestDB 创建服务测试数据库
func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&notification.Notification{},
		&notification.NotificationTemplate{},
		&notification.NotificationPreference{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// strPtr 辅助函数：字符串转指针
func strPtr(s string) *string {
	return &s
}

// setupNotificationService 创建通知服务测试环境
func setupNotificationService(t *testing.T) (*notifapp.NotificationService, *gorm.DB) {
	t.Helper()
	db := setupServiceTestDB(t)
	notifRepo := persistence.NewNotificationRepository(db)
	templateRepo := persistence.NewNotificationTemplateRepository(db)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)
	svc := notifapp.NewNotificationService(notifRepo, templateRepo, preferenceRepo)
	return svc, db
}

// ==================== Notification CRUD 测试 ====================

// TestCreateNotification_Success 创建通知成功
func TestCreateNotification_Success(t *testing.T) {
	svc, _ := setupNotificationService(t)

	req := &notifapp.CreateNotificationRequest{
		Title:      "测试通知",
		Content:    "这是一条测试通知",
		Type:       "system",
		ReceiverID: "user-001",
	}

	result, err := svc.CreateNotification(context.Background(), "tenant-001", req)
	if err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}
	if result.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if result.Title != "测试通知" {
		t.Errorf("期望 Title 为 '测试通知'，实际为 '%s'", result.Title)
	}
	if result.Type != "system" {
		t.Errorf("期望 Type 为 'system'，实际为 '%s'", result.Type)
	}
	if result.ReceiverID != "user-001" {
		t.Errorf("期望 ReceiverID 为 'user-001'，实际为 '%s'", result.ReceiverID)
	}
	if result.Status != "sent" {
		t.Errorf("期望 Status 为 'sent'，实际为 '%s'", result.Status)
	}
}

// TestCreateNotification_MissingTitle 缺少标题返回错误
func TestCreateNotification_MissingTitle(t *testing.T) {
	svc, _ := setupNotificationService(t)

	req := &notifapp.CreateNotificationRequest{
		Type:       "system",
		ReceiverID: "user-001",
	}
	_, err := svc.CreateNotification(context.Background(), "tenant-001", req)
	if err != notifapp.ErrNotificationTitleRequired {
		t.Fatalf("期望返回 notifapp.ErrNotificationTitleRequired，实际为 %v", err)
	}
}

// TestCreateNotification_MissingType 缺少类型返回错误
func TestCreateNotification_MissingType(t *testing.T) {
	svc, _ := setupNotificationService(t)

	req := &notifapp.CreateNotificationRequest{
		Title:      "测试通知",
		ReceiverID: "user-001",
	}
	_, err := svc.CreateNotification(context.Background(), "tenant-001", req)
	if err != notifapp.ErrNotificationTypeRequired {
		t.Fatalf("期望返回 notifapp.ErrNotificationTypeRequired，实际为 %v", err)
	}
}

// TestGetNotification_Exists 查询存在的通知
func TestGetNotification_Exists(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Title:      "测试通知",
		Content:    "内容",
		Type:       "system",
		ReceiverID: "user-001",
		Status:     "sent",
	}
	if err := notifRepo.Create(context.Background(), n); err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	result, err := svc.GetNotification(context.Background(), "tenant-001", n.ID)
	if err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if result.Title != "测试通知" {
		t.Errorf("期望 Title 为 '测试通知'，实际为 '%s'", result.Title)
	}
}

// TestGetNotification_NotExists 查询不存在的通知
func TestGetNotification_NotExists(t *testing.T) {
	svc, _ := setupNotificationService(t)

	_, err := svc.GetNotification(context.Background(), "tenant-001", "non-existent-id")
	if err != notifapp.ErrNotificationNotFound {
		t.Fatalf("期望返回 notifapp.ErrNotificationNotFound，实际为 %v", err)
	}
}

// TestListNotifications_Pagination 分页正确
func TestListNotifications_Pagination(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	for i := 1; i <= 5; i++ {
		n := &notification.Notification{
			BaseModel:  base.BaseModel{TenantID: "tenant-001"},
			Title:      "通知" + string(rune('0'+i)),
			Type:       "system",
			ReceiverID: "user-001",
			Status:     "sent",
		}
		if err := notifRepo.Create(context.Background(), n); err != nil {
			t.Fatalf("创建通知失败: %v", err)
		}
	}

	notifications, paginated, err := svc.ListNotifications(context.Background(), "tenant-001", "user-001", notifapp.ListNotificationsFilter{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if paginated.Total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", paginated.Total)
	}
	if len(notifications) != 3 {
		t.Errorf("期望返回 3 条记录，实际为 %d", len(notifications))
	}
	if paginated.TotalPages != 2 {
		t.Errorf("期望总页数为 2，实际为 %d", paginated.TotalPages)
	}
}

// TestUpdateNotification_Success 更新通知成功
func TestUpdateNotification_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Title:      "旧标题",
		Type:       "system",
		ReceiverID: "user-001",
		Status:     "sent",
	}
	if err := notifRepo.Create(context.Background(), n); err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	req := &notifapp.UpdateNotificationRequest{
		Title: strPtr("新标题"),
	}

	result, err := svc.UpdateNotification(context.Background(), "tenant-001", n.ID, req)
	if err != nil {
		t.Fatalf("更新通知失败: %v", err)
	}
	if result.Title != "新标题" {
		t.Errorf("期望 Title 为 '新标题'，实际为 '%s'", result.Title)
	}
}

// TestDeleteNotification_Success 删除通知成功
func TestDeleteNotification_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Title:      "要删除的通知",
		Type:       "system",
		ReceiverID: "user-001",
		Status:     "sent",
	}
	if err := notifRepo.Create(context.Background(), n); err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	err := svc.DeleteNotification(context.Background(), "tenant-001", n.ID)
	if err != nil {
		t.Fatalf("删除通知失败: %v", err)
	}

	_, err = svc.GetNotification(context.Background(), "tenant-001", n.ID)
	if err == nil {
		t.Error("期望删除后查询返回错误")
	}
}

// TestMarkAsRead_Success 标记已读成功
func TestMarkAsRead_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Title:      "未读通知",
		Type:       "system",
		ReceiverID: "user-001",
		Status:     "sent",
	}
	if err := notifRepo.Create(context.Background(), n); err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	err := svc.MarkAsRead(context.Background(), "tenant-001", n.ID)
	if err != nil {
		t.Fatalf("标记已读失败: %v", err)
	}

	updated, err := svc.GetNotification(context.Background(), "tenant-001", n.ID)
	if err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if !updated.IsRead {
		t.Error("期望通知已标记为已读")
	}
	if updated.ReadAt == nil {
		t.Error("期望 ReadAt 不为空")
	}
}

// TestMarkAllAsRead_Success 全部标记已读成功
func TestMarkAllAsRead_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	for i := 0; i < 3; i++ {
		n := &notification.Notification{
			BaseModel:  base.BaseModel{TenantID: "tenant-001"},
			Title:      "未读通知",
			Type:       "system",
			ReceiverID: "user-001",
			Status:     "sent",
		}
		if err := notifRepo.Create(context.Background(), n); err != nil {
			t.Fatalf("创建通知失败: %v", err)
		}
	}

	err := svc.MarkAllAsRead(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("全部标记已读失败: %v", err)
	}

	count, err := svc.GetUnreadCount(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("获取未读数量失败: %v", err)
	}
	if count != 0 {
		t.Errorf("期望未读数量为 0，实际为 %d", count)
	}
}

// TestGetUnreadCount_Success 获取未读数量成功
func TestGetUnreadCount_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	// 创建2条未读通知
	for i := 0; i < 2; i++ {
		n := &notification.Notification{
			BaseModel:  base.BaseModel{TenantID: "tenant-001"},
			Title:      "未读通知",
			Type:       "system",
			ReceiverID: "user-001",
			Status:     "sent",
		}
		if err := notifRepo.Create(context.Background(), n); err != nil {
			t.Fatalf("创建通知失败: %v", err)
		}
	}

	count, err := svc.GetUnreadCount(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("获取未读数量失败: %v", err)
	}
	if count != 2 {
		t.Errorf("期望未读数量为 2，实际为 %d", count)
	}
}

// ==================== Template CRUD 测试 ====================

// TestCreateTemplate_Success 创建模板成功
func TestCreateTemplate_Success(t *testing.T) {
	svc, _ := setupNotificationService(t)

	req := &notifapp.CreateTemplateRequest{
		Name:       "审批通知模板",
		Code:       "approval_notify",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "审批通知 - {{.title}}",
		ContentTpl: "您有一条新的审批任务：{{.content}}",
	}

	result, err := svc.CreateTemplate(context.Background(), "tenant-001", req)
	if err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}
	if result.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if result.Code != "approval_notify" {
		t.Errorf("期望 Code 为 'approval_notify'，实际为 '%s'", result.Code)
	}
	if result.Status != "active" {
		t.Errorf("期望 Status 为 'active'，实际为 '%s'", result.Status)
	}
}

// TestGetTemplate_Exists 查询存在的模板
func TestGetTemplate_Exists(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "测试模板",
		Code:       "test_tpl",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "标题",
		ContentTpl: "内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	result, err := svc.GetTemplate(context.Background(), "tenant-001", tpl.ID)
	if err != nil {
		t.Fatalf("查询模板失败: %v", err)
	}
	if result.Name != "测试模板" {
		t.Errorf("期望 Name 为 '测试模板'，实际为 '%s'", result.Name)
	}
}

// TestGetTemplateByCode_Success 根据编码查询模板
func TestGetTemplateByCode_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "测试模板",
		Code:       "test_by_code",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "标题",
		ContentTpl: "内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	result, err := svc.GetTemplateByCode(context.Background(), "tenant-001", "test_by_code")
	if err != nil {
		t.Fatalf("根据编码查询模板失败: %v", err)
	}
	if result.Code != "test_by_code" {
		t.Errorf("期望 Code 为 'test_by_code'，实际为 '%s'", result.Code)
	}
}

// TestListTemplates_Success 列出模板成功
func TestListTemplates_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "测试模板",
		Code:       "list_test",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "标题",
		ContentTpl: "内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	templates, paginated, err := svc.ListTemplates(context.Background(), "tenant-001", notifapp.ListNotificationsFilter{})
	if err != nil {
		t.Fatalf("查询模板列表失败: %v", err)
	}
	if paginated.Total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", paginated.Total)
	}
	if len(templates) != 1 {
		t.Errorf("期望返回 1 条记录，实际为 %d", len(templates))
	}
}

// TestUpdateTemplate_Success 更新模板成功
func TestUpdateTemplate_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "旧名称",
		Code:       "update_test",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "旧标题",
		ContentTpl: "旧内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	newName := "新名称"
	inactive := "inactive"
	req := &notifapp.UpdateTemplateRequest{
		Name:   &newName,
		Status: &inactive,
	}

	result, err := svc.UpdateTemplate(context.Background(), "tenant-001", tpl.ID, req)
	if err != nil {
		t.Fatalf("更新模板失败: %v", err)
	}
	if result.Name != "新名称" {
		t.Errorf("期望 Name 为 '新名称'，实际为 '%s'", result.Name)
	}
	if result.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", result.Status)
	}
}

// TestDeleteTemplate_Success 删除模板成功
func TestDeleteTemplate_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "要删除的模板",
		Code:       "delete_test",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "标题",
		ContentTpl: "内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	err := svc.DeleteTemplate(context.Background(), "tenant-001", tpl.ID)
	if err != nil {
		t.Fatalf("删除模板失败: %v", err)
	}

	_, err = svc.GetTemplate(context.Background(), "tenant-001", tpl.ID)
	if err == nil {
		t.Error("期望删除后查询返回错误")
	}
}

// ==================== Preference CRUD 测试 ====================

// TestCreatePreference_Success 创建偏好成功
func TestCreatePreference_Success(t *testing.T) {
	svc, _ := setupNotificationService(t)

	req := &notifapp.CreatePreferenceRequest{
		UserID:  "user-001",
		Type:    "system",
		Channel: "in_app",
	}

	result, err := svc.CreatePreference(context.Background(), "tenant-001", req)
	if err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}
	if result.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if !result.Enabled {
		t.Error("期望默认 Enabled 为 true")
	}
}

// TestGetPreference_Exists 查询存在的偏好
func TestGetPreference_Exists(t *testing.T) {
	svc, db := setupNotificationService(t)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "system",
		Channel:   "in_app",
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	result, err := svc.GetPreference(context.Background(), "tenant-001", pref.ID)
	if err != nil {
		t.Fatalf("查询偏好失败: %v", err)
	}
	if result.UserID != "user-001" {
		t.Errorf("期望 UserID 为 'user-001'，实际为 '%s'", result.UserID)
	}
}

// TestListPreferences_Success 列出偏好成功
func TestListPreferences_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "system",
		Channel:   "in_app",
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	preferences, paginated, err := svc.ListPreferences(context.Background(), "tenant-001", notifapp.ListPreferencesFilter{})
	if err != nil {
		t.Fatalf("查询偏好列表失败: %v", err)
	}
	if paginated.Total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", paginated.Total)
	}
	if len(preferences) != 1 {
		t.Errorf("期望返回 1 条记录，实际为 %d", len(preferences))
	}
}

// TestUpdatePreference_Success 更新偏好成功
func TestUpdatePreference_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "system",
		Channel:   "in_app",
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	disabled := false
	req := &notifapp.UpdatePreferenceRequest{
		Enabled: &disabled,
	}

	result, err := svc.UpdatePreference(context.Background(), "tenant-001", pref.ID, req)
	if err != nil {
		t.Fatalf("更新偏好失败: %v", err)
	}
	if result.Enabled {
		t.Error("期望 Enabled 为 false")
	}
}

// TestDeletePreference_Success 删除偏好成功
func TestDeletePreference_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "system",
		Channel:   "in_app",
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	err := svc.DeletePreference(context.Background(), "tenant-001", pref.ID)
	if err != nil {
		t.Fatalf("删除偏好失败: %v", err)
	}

	_, err = svc.GetPreference(context.Background(), "tenant-001", pref.ID)
	if err == nil {
		t.Error("期望删除后查询返回错误")
	}
}

// ==================== SendNotification 测试 ====================

// TestSendNotification_Success 通过模板发送通知成功
func TestSendNotification_Success(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "审批通知",
		Code:       "approval_notify",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "审批通知 - {{.title}}",
		ContentTpl: "您有一条新的审批任务：{{.content}}",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	req := &notifapp.SendNotificationRequest{
		TemplateCode: "approval_notify",
		ReceiverID:   "user-001",
		Variables: map[string]string{
			"title":   "采购审批",
			"content": "请审批采购申请单 #12345",
		},
	}

	result, err := svc.SendNotification(context.Background(), "tenant-001", req)
	if err != nil {
		t.Fatalf("发送通知失败: %v", err)
	}
	if result.Title != "审批通知 - 采购审批" {
		t.Errorf("期望 Title 为 '审批通知 - 采购审批'，实际为 '%s'", result.Title)
	}
	if result.Content != "您有一条新的审批任务：请审批采购申请单 #12345" {
		t.Errorf("期望 Content 包含渲染后的内容，实际为 '%s'", result.Content)
	}
	if result.Channel != "in_app" {
		t.Errorf("期望 Channel 为 'in_app'，实际为 '%s'", result.Channel)
	}
}

// TestSendNotification_TemplateNotFound 模板不存在
func TestSendNotification_TemplateNotFound(t *testing.T) {
	svc, _ := setupNotificationService(t)

	req := &notifapp.SendNotificationRequest{
		TemplateCode: "non_existent_code",
		ReceiverID:   "user-001",
	}

	_, err := svc.SendNotification(context.Background(), "tenant-001", req)
	if err != notifapp.ErrTemplateNotFound {
		t.Fatalf("期望返回 notifapp.ErrTemplateNotFound，实际为 %v", err)
	}
}

// TestSendNotification_UserMuted 用户已设置免打扰
func TestSendNotification_UserMuted(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "系统通知",
		Code:       "system_notify",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "系统通知",
		ContentTpl: "系统消息内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	// 设置用户偏好为禁用
	disabled := false
	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "system",
		Channel:   "in_app",
		Enabled:   disabled,
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	req := &notifapp.SendNotificationRequest{
		TemplateCode: "system_notify",
		ReceiverID:   "user-001",
	}

	_, err := svc.SendNotification(context.Background(), "tenant-001", req)
	if err != notifapp.ErrUserMuted {
		t.Fatalf("期望返回 notifapp.ErrUserMuted，实际为 %v", err)
	}
}

// TestSendNotification_UserMutedByTime 用户在免打扰时间内
func TestSendNotification_UserMutedByTime(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "提醒通知",
		Code:       "reminder_notify",
		Type:       "reminder",
		Channel:    "in_app",
		TitleTpl:   "提醒",
		ContentTpl: "提醒内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	// 设置免打扰到未来
	muteUntil := time.Now().Add(24 * time.Hour)
	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "reminder",
		Channel:   "in_app",
		Enabled:   true,
		MuteUntil: &muteUntil,
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	req := &notifapp.SendNotificationRequest{
		TemplateCode: "reminder_notify",
		ReceiverID:   "user-001",
	}

	_, err := svc.SendNotification(context.Background(), "tenant-001", req)
	if err != notifapp.ErrUserMuted {
		t.Fatalf("期望返回 notifapp.ErrUserMuted，实际为 %v", err)
	}
}

// TestSendNotification_VariableRendering 变量渲染正确
func TestSendNotification_VariableRendering(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "变量测试模板",
		Code:       "var_test",
		Type:       "alert",
		Channel:    "email",
		TitleTpl:   "{{.greeting}}，{{.name}}",
		ContentTpl: "任务 {{.task}} 状态已更新为 {{.status}}",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	req := &notifapp.SendNotificationRequest{
		TemplateCode: "var_test",
		ReceiverID:   "user-001",
		Variables: map[string]string{
			"greeting": "你好",
			"name":     "张三",
			"task":     "设计评审",
			"status":   "已完成",
		},
		Channel: "email",
	}

	result, err := svc.SendNotification(context.Background(), "tenant-001", req)
	if err != nil {
		t.Fatalf("发送通知失败: %v", err)
	}
	if result.Title != "你好，张三" {
		t.Errorf("期望 Title 为 '你好，张三'，实际为 '%s'", result.Title)
	}
	if result.Content != "任务 设计评审 状态已更新为 已完成" {
		t.Errorf("期望 Content 为 '任务 设计评审 状态已更新为 已完成'，实际为 '%s'", result.Content)
	}
}

// ==================== 跨租户测试 ====================

// TestNotificationCrossTenantOperation 通知跨租户操作返回错误
func TestNotificationCrossTenantOperation(t *testing.T) {
	svc, db := setupNotificationService(t)
	notifRepo := persistence.NewNotificationRepository(db)

	// 租户 A 创建通知
	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: "tenant-a"},
		Title:      "租户A的通知",
		Type:       "system",
		ReceiverID: "user-a",
		Status:     "sent",
	}
	if err := notifRepo.Create(context.Background(), n); err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}

	// 租户 B 尝试查询
	_, err := svc.GetNotification(context.Background(), "tenant-b", n.ID)
	if err != notifapp.ErrTenantMismatch {
		t.Fatalf("期望跨租户查询返回 notifapp.ErrTenantMismatch，实际为 %v", err)
	}

	// 租户 B 尝试更新
	_, err = svc.UpdateNotification(context.Background(), "tenant-b", n.ID, &notifapp.UpdateNotificationRequest{Title: strPtr("Hacked")})
	if err != notifapp.ErrTenantMismatch {
		t.Fatalf("期望跨租户更新返回 notifapp.ErrTenantMismatch，实际为 %v", err)
	}

	// 租户 B 尝试删除
	err = svc.DeleteNotification(context.Background(), "tenant-b", n.ID)
	if err != notifapp.ErrTenantMismatch {
		t.Fatalf("期望跨租户删除返回 notifapp.ErrTenantMismatch，实际为 %v", err)
	}

	// 租户 B 尝试标记已读
	err = svc.MarkAsRead(context.Background(), "tenant-b", n.ID)
	if err != notifapp.ErrTenantMismatch {
		t.Fatalf("期望跨租户标记已读返回 notifapp.ErrTenantMismatch，实际为 %v", err)
	}
}

// TestTemplateCrossTenantOperation 模板跨租户操作返回错误
func TestTemplateCrossTenantOperation(t *testing.T) {
	svc, db := setupNotificationService(t)
	templateRepo := persistence.NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-a"},
		Name:       "租户A的模板",
		Code:       "tenant_a_tpl",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "标题",
		ContentTpl: "内容",
	}
	if err := templateRepo.Create(context.Background(), tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}

	_, err := svc.GetTemplate(context.Background(), "tenant-b", tpl.ID)
	if err != notifapp.ErrTenantMismatch {
		t.Fatalf("期望跨租户查询返回 notifapp.ErrTenantMismatch，实际为 %v", err)
	}
}

// TestPreferenceCrossTenantOperation 偏好跨租户操作返回错误
func TestPreferenceCrossTenantOperation(t *testing.T) {
	svc, db := setupNotificationService(t)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)

	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-a"},
		UserID:    "user-a",
		Type:      "system",
		Channel:   "in_app",
	}
	if err := preferenceRepo.Create(context.Background(), pref); err != nil {
		t.Fatalf("创建偏好失败: %v", err)
	}

	_, err := svc.GetPreference(context.Background(), "tenant-b", pref.ID)
	if err != notifapp.ErrTenantMismatch {
		t.Fatalf("期望跨租户查询返回 notifapp.ErrTenantMismatch，实际为 %v", err)
	}
}
