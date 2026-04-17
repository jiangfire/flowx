package persistence

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/notification"
	notifapp "git.neolidy.top/neo/flowx/internal/application/notification"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupNotificationRepoTestDB 创建通知 Repository 测试数据库
func setupNotificationRepoTestDB(t *testing.T) *gorm.DB {
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

// createTestNotification 创建测试用通知
func createTestNotification(t *testing.T, db *gorm.DB, tenantID, receiverID, typ, category, status string, isRead bool) *notification.Notification {
	t.Helper()
	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Title:      "测试通知",
		Content:    "测试通知内容",
		Type:       typ,
		Category:   category,
		Channel:    "in_app",
		ReceiverID: receiverID,
		IsRead:     isRead,
		Status:     status,
	}
	if err := db.Create(n).Error; err != nil {
		t.Fatalf("创建测试通知失败: %v", err)
	}
	return n
}

// createTestNotificationTemplate 创建测试用通知模板
func createTestNotificationTemplate(t *testing.T, db *gorm.DB, tenantID, name, code, typ, channel string) *notification.NotificationTemplate {
	t.Helper()
	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Name:       name,
		Code:       code,
		Type:       typ,
		Channel:    channel,
		TitleTpl:   "标题模板：{{.title}}",
		ContentTpl: "内容模板：{{.content}}",
		Status:     "active",
	}
	if err := db.Create(tpl).Error; err != nil {
		t.Fatalf("创建测试通知模板失败: %v", err)
	}
	return tpl
}

// createTestNotificationPreference 创建测试用通知偏好
func createTestNotificationPreference(t *testing.T, db *gorm.DB, tenantID, userID, typ, channel string, enabled bool) *notification.NotificationPreference {
	t.Helper()
	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		UserID:    userID,
		Type:      typ,
		Channel:   channel,
		Enabled:   enabled,
	}
	if err := db.Create(pref).Error; err != nil {
		t.Fatalf("创建测试通知偏好失败: %v", err)
	}
	return pref
}

// ==================== NotificationRepository 测试 ====================

// TestNotificationRepository_Create 创建通知成功
func TestNotificationRepository_Create(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	n := &notification.Notification{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Title:      "系统通知",
		Content:    "系统升级通知",
		Type:       "system",
		Category:   "info",
		ReceiverID: "user-001",
		Status:     "pending",
	}

	err := repo.Create(context.Background(), n)
	if err != nil {
		t.Fatalf("创建通知失败: %v", err)
	}
	if n.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if n.Title != "系统通知" {
		t.Errorf("期望 Title 为 '系统通知'，实际为 '%s'", n.Title)
	}
}

// TestNotificationRepository_GetByID_Exists 查询存在的通知
func TestNotificationRepository_GetByID_Exists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	created := createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询通知失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 匹配")
	}
	if found.Title != "测试通知" {
		t.Errorf("期望 Title 为 '测试通知'，实际为 '%s'", found.Title)
	}
}

// TestNotificationRepository_GetByID_NotExists 查询不存在的通知
func TestNotificationRepository_GetByID_NotExists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的通知返回错误")
	}
}

// TestNotificationRepository_List_NoFilter 无过滤返回全部
func TestNotificationRepository_List_NoFilter(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", true)

	items, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(items))
	}
}

// TestNotificationRepository_List_FilterByReceiverID 按接收者过滤
func TestNotificationRepository_List_FilterByReceiverID(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-002", "task", "warning", "sent", true)
	createTestNotification(t, db, "tenant-001", "user-001", "reminder", "info", "pending", false)

	items, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001", ReceiverID: "user-001"})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, item := range items {
		if item.ReceiverID != "user-001" {
			t.Errorf("期望 ReceiverID 为 'user-001'，实际为 '%s'", item.ReceiverID)
		}
	}
}

// TestNotificationRepository_List_FilterByType 按类型过滤
func TestNotificationRepository_List_FilterByType(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", true)
	createTestNotification(t, db, "tenant-001", "user-001", "system", "error", "failed", false)

	items, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001", Type: "system"})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, item := range items {
		if item.Type != "system" {
			t.Errorf("期望 Type 为 'system'，实际为 '%s'", item.Type)
		}
	}
}

// TestNotificationRepository_List_FilterByCategory 按分类过滤
func TestNotificationRepository_List_FilterByCategory(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", true)
	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)

	_, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001", Category: "info"})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
}

// TestNotificationRepository_List_FilterByIsRead 按已读状态过滤
func TestNotificationRepository_List_FilterByIsRead(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", true)
	createTestNotification(t, db, "tenant-001", "user-001", "reminder", "info", "pending", false)

	isRead := true
	items, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001", IsRead: &isRead})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	for _, item := range items {
		if !item.IsRead {
			t.Error("期望所有通知已读")
		}
	}
}

// TestNotificationRepository_List_FilterByStatus 按状态过滤
func TestNotificationRepository_List_FilterByStatus(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", true)
	createTestNotification(t, db, "tenant-001", "user-001", "system", "error", "failed", false)

	items, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001", Status: "sent"})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(items) != 1 || items[0].Status != "sent" {
		t.Error("期望返回状态为 sent 的通知")
	}
}

// TestNotificationRepository_List_Pagination 分页正确
func TestNotificationRepository_List_Pagination(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	// 创建 5 条记录
	for i := 1; i <= 5; i++ {
		createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	}

	// 查询第 2 页，每页 2 条
	items, total, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询通知列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(items))
	}
}

// TestNotificationRepository_Update 更新成功
func TestNotificationRepository_Update(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	created := createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)

	created.Title = "更新后的标题"
	created.Status = "sent"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新通知失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的通知失败: %v", err)
	}
	if updated.Title != "更新后的标题" {
		t.Errorf("期望 Title 为 '更新后的标题'，实际为 '%s'", updated.Title)
	}
	if updated.Status != "sent" {
		t.Errorf("期望 Status 为 'sent'，实际为 '%s'", updated.Status)
	}
}

// TestNotificationRepository_MarkAsRead 标记已读成功
func TestNotificationRepository_MarkAsRead(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	created := createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)

	err := repo.MarkAsRead(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("标记通知已读失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的通知失败: %v", err)
	}
	if !updated.IsRead {
		t.Error("期望通知已读")
	}
	if updated.ReadAt == nil {
		t.Error("期望 ReadAt 不为空")
	}
}

// TestNotificationRepository_MarkAsRead_NotExists 标记不存在的通知
func TestNotificationRepository_MarkAsRead_NotExists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	err := repo.MarkAsRead(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望标记不存在的通知返回错误")
	}
}

// TestNotificationRepository_MarkAllAsRead 批量标记已读
func TestNotificationRepository_MarkAllAsRead(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", false)
	// 已读通知不应被重复标记
	createTestNotification(t, db, "tenant-001", "user-001", "reminder", "info", "pending", true)

	err := repo.MarkAllAsRead(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("批量标记已读失败: %v", err)
	}

	// 验证未读数量为 0
	count, err := repo.CountUnread(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("统计未读数量失败: %v", err)
	}
	if count != 0 {
		t.Errorf("期望未读数量为 0，实际为 %d", count)
	}
}

// TestNotificationRepository_Delete 软删除成功
func TestNotificationRepository_Delete(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	created := createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除通知失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// TestNotificationRepository_CountUnread 统计未读数量
func TestNotificationRepository_CountUnread(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-001", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-001", "user-001", "task", "warning", "sent", false)
	createTestNotification(t, db, "tenant-001", "user-001", "reminder", "info", "pending", true)
	// 其他用户的通知不应计入
	createTestNotification(t, db, "tenant-001", "user-002", "system", "info", "pending", false)

	count, err := repo.CountUnread(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("统计未读数量失败: %v", err)
	}
	if count != 2 {
		t.Errorf("期望未读数量为 2，实际为 %d", count)
	}
}

// TestNotificationRepository_TenantIsolation 多租户隔离
func TestNotificationRepository_TenantIsolation(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationRepository(db)

	createTestNotification(t, db, "tenant-a", "user-001", "system", "info", "pending", false)
	createTestNotification(t, db, "tenant-b", "user-001", "system", "info", "pending", false)

	itemsA, totalA, err := repo.List(context.Background(), notifapp.NotificationFilter{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("查询租户A通知列表失败: %v", err)
	}
	if totalA != 1 {
		t.Errorf("期望租户A总数为 1，实际为 %d", totalA)
	}
	if len(itemsA) != 1 || itemsA[0].TenantID != "tenant-a" {
		t.Error("期望租户A只能看到自己的通知")
	}
}

// ==================== NotificationTemplateRepository 测试 ====================

// TestNotificationTemplateRepository_Create 创建通知模板成功
func TestNotificationTemplateRepository_Create(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	tpl := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001"},
		Name:       "审批通知模板",
		Code:       "approval_notify",
		Type:       "approval",
		Channel:    "in_app",
		TitleTpl:   "您有一条新的审批请求",
		ContentTpl: "审批详情如下",
		Status:     "active",
	}

	err := repo.Create(context.Background(), tpl)
	if err != nil {
		t.Fatalf("创建通知模板失败: %v", err)
	}
	if tpl.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestNotificationTemplateRepository_GetByID_Exists 查询存在的通知模板
func TestNotificationTemplateRepository_GetByID_Exists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	created := createTestNotificationTemplate(t, db, "tenant-001", "审批模板", "approval_notify", "approval", "in_app")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询通知模板失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 匹配")
	}
	if found.Name != "审批模板" {
		t.Errorf("期望 Name 为 '审批模板'，实际为 '%s'", found.Name)
	}
}

// TestNotificationTemplateRepository_GetByID_NotExists 查询不存在的通知模板
func TestNotificationTemplateRepository_GetByID_NotExists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的通知模板返回错误")
	}
}

// TestNotificationTemplateRepository_GetByCode_Exists 根据编码查询存在的模板
func TestNotificationTemplateRepository_GetByCode_Exists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	createTestNotificationTemplate(t, db, "tenant-001", "审批模板", "approval_notify", "approval", "in_app")

	found, err := repo.GetByCode(context.Background(), "approval_notify")
	if err != nil {
		t.Fatalf("根据编码查询通知模板失败: %v", err)
	}
	if found.Code != "approval_notify" {
		t.Errorf("期望 Code 为 'approval_notify'，实际为 '%s'", found.Code)
	}
}

// TestNotificationTemplateRepository_GetByCode_NotExists 根据编码查询不存在的模板
func TestNotificationTemplateRepository_GetByCode_NotExists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	_, err := repo.GetByCode(context.Background(), "non_existent_code")
	if err == nil {
		t.Fatal("期望根据编码查询不存在的模板返回错误")
	}
}

// TestNotificationTemplateRepository_List 无过滤返回全部
func TestNotificationTemplateRepository_List(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	createTestNotificationTemplate(t, db, "tenant-001", "审批模板", "approval_notify", "approval", "in_app")
	createTestNotificationTemplate(t, db, "tenant-001", "任务模板", "task_notify", "task", "email")

	items, total, err := repo.List(context.Background(), notifapp.NotificationTemplateFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询通知模板列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(items))
	}
}

// TestNotificationTemplateRepository_List_FilterByType 按类型过滤
func TestNotificationTemplateRepository_List_FilterByType(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	createTestNotificationTemplate(t, db, "tenant-001", "审批模板", "approval_notify", "approval", "in_app")
	createTestNotificationTemplate(t, db, "tenant-001", "任务模板", "task_notify", "task", "email")
	createTestNotificationTemplate(t, db, "tenant-001", "系统模板", "system_notify", "system", "in_app")

	items, total, err := repo.List(context.Background(), notifapp.NotificationTemplateFilter{TenantID: "tenant-001", Type: "approval"})
	if err != nil {
		t.Fatalf("查询通知模板列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(items) != 1 || items[0].Type != "approval" {
		t.Error("期望返回类型为 approval 的模板")
	}
}

// TestNotificationTemplateRepository_List_FilterByStatus 按状态过滤
func TestNotificationTemplateRepository_List_FilterByStatus(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	createTestNotificationTemplate(t, db, "tenant-001", "审批模板", "approval_notify", "approval", "in_app")
	// 创建一个 inactive 模板
	inactive := &notification.NotificationTemplate{
		BaseModel:  base.BaseModel{TenantID: "tenant-001", ID: base.GenerateUUID()},
		Name:       "停用模板",
		Code:       "inactive_tpl",
		Type:       "system",
		Channel:    "in_app",
		TitleTpl:   "停用标题",
		ContentTpl: "停用内容",
		Status:     "inactive",
	}
	if err := db.Create(inactive).Error; err != nil {
		t.Fatalf("创建停用模板失败: %v", err)
	}

	_, total, err := repo.List(context.Background(), notifapp.NotificationTemplateFilter{TenantID: "tenant-001", Status: "active"})
	if err != nil {
		t.Fatalf("查询通知模板列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
}

// TestNotificationTemplateRepository_List_Pagination 分页正确
func TestNotificationTemplateRepository_List_Pagination(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	for i := 1; i <= 5; i++ {
		createTestNotificationTemplate(t, db, "tenant-001", "模板"+string(rune('0'+i)), "tpl_"+string(rune('0'+i)), "system", "in_app")
	}

	items, total, err := repo.List(context.Background(), notifapp.NotificationTemplateFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询通知模板列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(items))
	}
}

// TestNotificationTemplateRepository_Update 更新成功
func TestNotificationTemplateRepository_Update(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	created := createTestNotificationTemplate(t, db, "tenant-001", "旧名称", "old_code", "approval", "in_app")

	created.Name = "新名称"
	created.Status = "inactive"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新通知模板失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的模板失败: %v", err)
	}
	if updated.Name != "新名称" {
		t.Errorf("期望 Name 为 '新名称'，实际为 '%s'", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", updated.Status)
	}
}

// TestNotificationTemplateRepository_Delete 软删除成功
func TestNotificationTemplateRepository_Delete(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	created := createTestNotificationTemplate(t, db, "tenant-001", "删除模板", "delete_tpl", "system", "in_app")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除通知模板失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// TestNotificationTemplateRepository_TenantIsolation 多租户隔离
func TestNotificationTemplateRepository_TenantIsolation(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationTemplateRepository(db)

	createTestNotificationTemplate(t, db, "tenant-a", "模板A", "tpl_a", "system", "in_app")
	createTestNotificationTemplate(t, db, "tenant-b", "模板B", "tpl_b", "system", "in_app")

	itemsA, totalA, err := repo.List(context.Background(), notifapp.NotificationTemplateFilter{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("查询租户A通知模板列表失败: %v", err)
	}
	if totalA != 1 {
		t.Errorf("期望租户A总数为 1，实际为 %d", totalA)
	}
	if len(itemsA) != 1 || itemsA[0].TenantID != "tenant-a" {
		t.Error("期望租户A只能看到自己的通知模板")
	}
}

// ==================== NotificationPreferenceRepository 测试 ====================

// TestNotificationPreferenceRepository_Create 创建通知偏好成功
func TestNotificationPreferenceRepository_Create(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	pref := &notification.NotificationPreference{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		UserID:    "user-001",
		Type:      "system",
		Channel:   "in_app",
		Enabled:   true,
	}

	err := repo.Create(context.Background(), pref)
	if err != nil {
		t.Fatalf("创建通知偏好失败: %v", err)
	}
	if pref.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestNotificationPreferenceRepository_GetByID_Exists 查询存在的通知偏好
func TestNotificationPreferenceRepository_GetByID_Exists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	created := createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询通知偏好失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 匹配")
	}
	if found.UserID != "user-001" {
		t.Errorf("期望 UserID 为 'user-001'，实际为 '%s'", found.UserID)
	}
}

// TestNotificationPreferenceRepository_GetByID_NotExists 查询不存在的通知偏好
func TestNotificationPreferenceRepository_GetByID_NotExists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的通知偏好返回错误")
	}
}

// TestNotificationPreferenceRepository_GetByUserAndType_Exists 根据用户和类型查询存在的偏好
func TestNotificationPreferenceRepository_GetByUserAndType_Exists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)

	found, err := repo.GetByUserAndType(context.Background(), "user-001", "system", "in_app")
	if err != nil {
		t.Fatalf("根据用户和类型查询通知偏好失败: %v", err)
	}
	if found.UserID != "user-001" {
		t.Errorf("期望 UserID 为 'user-001'，实际为 '%s'", found.UserID)
	}
	if found.Type != "system" {
		t.Errorf("期望 Type 为 'system'，实际为 '%s'", found.Type)
	}
	if found.Channel != "in_app" {
		t.Errorf("期望 Channel 为 'in_app'，实际为 '%s'", found.Channel)
	}
}

// TestNotificationPreferenceRepository_GetByUserAndType_NotExists 根据用户和类型查询不存在的偏好
func TestNotificationPreferenceRepository_GetByUserAndType_NotExists(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	_, err := repo.GetByUserAndType(context.Background(), "user-001", "system", "email")
	if err == nil {
		t.Fatal("期望根据用户和类型查询不存在的偏好返回错误")
	}
}

// TestNotificationPreferenceRepository_List 无过滤返回全部
func TestNotificationPreferenceRepository_List(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)
	createTestNotificationPreference(t, db, "tenant-001", "user-001", "task", "email", false)

	items, total, err := repo.List(context.Background(), notifapp.NotificationPreferenceFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询通知偏好列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(items))
	}
}

// TestNotificationPreferenceRepository_List_FilterByUserID 按用户过滤
func TestNotificationPreferenceRepository_List_FilterByUserID(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)
	createTestNotificationPreference(t, db, "tenant-001", "user-002", "system", "in_app", true)
	createTestNotificationPreference(t, db, "tenant-001", "user-001", "task", "email", false)

	items, total, err := repo.List(context.Background(), notifapp.NotificationPreferenceFilter{TenantID: "tenant-001", UserID: "user-001"})
	if err != nil {
		t.Fatalf("查询通知偏好列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, item := range items {
		if item.UserID != "user-001" {
			t.Errorf("期望 UserID 为 'user-001'，实际为 '%s'", item.UserID)
		}
	}
}

// TestNotificationPreferenceRepository_List_FilterByType 按类型过滤
func TestNotificationPreferenceRepository_List_FilterByType(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)
	createTestNotificationPreference(t, db, "tenant-001", "user-001", "task", "email", false)
	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "email", true)

	_, total, err := repo.List(context.Background(), notifapp.NotificationPreferenceFilter{TenantID: "tenant-001", Type: "system"})
	if err != nil {
		t.Fatalf("查询通知偏好列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
}

// TestNotificationPreferenceRepository_List_FilterByChannel 按渠道过滤
func TestNotificationPreferenceRepository_List_FilterByChannel(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)
	createTestNotificationPreference(t, db, "tenant-001", "user-001", "task", "email", false)
	createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "email", true)

	_, total, err := repo.List(context.Background(), notifapp.NotificationPreferenceFilter{TenantID: "tenant-001", Channel: "email"})
	if err != nil {
		t.Fatalf("查询通知偏好列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
}

// TestNotificationPreferenceRepository_List_Pagination 分页正确
func TestNotificationPreferenceRepository_List_Pagination(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	for i := 1; i <= 5; i++ {
		createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)
	}

	items, total, err := repo.List(context.Background(), notifapp.NotificationPreferenceFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询通知偏好列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(items) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(items))
	}
}

// TestNotificationPreferenceRepository_Update 更新成功
func TestNotificationPreferenceRepository_Update(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	created := createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)

	created.Enabled = false
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新通知偏好失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的偏好失败: %v", err)
	}
	if updated.Enabled {
		t.Error("期望 Enabled 为 false")
	}
}

// TestNotificationPreferenceRepository_Delete 软删除成功
func TestNotificationPreferenceRepository_Delete(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	created := createTestNotificationPreference(t, db, "tenant-001", "user-001", "system", "in_app", true)

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除通知偏好失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// TestNotificationPreferenceRepository_TenantIsolation 多租户隔离
func TestNotificationPreferenceRepository_TenantIsolation(t *testing.T) {
	db := setupNotificationRepoTestDB(t)
	repo := NewNotificationPreferenceRepository(db)

	createTestNotificationPreference(t, db, "tenant-a", "user-001", "system", "in_app", true)
	createTestNotificationPreference(t, db, "tenant-b", "user-001", "system", "in_app", true)

	itemsA, totalA, err := repo.List(context.Background(), notifapp.NotificationPreferenceFilter{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("查询租户A通知偏好列表失败: %v", err)
	}
	if totalA != 1 {
		t.Errorf("期望租户A总数为 1，实际为 %d", totalA)
	}
	if len(itemsA) != 1 || itemsA[0].TenantID != "tenant-a" {
		t.Error("期望租户A只能看到自己的通知偏好")
	}
}
