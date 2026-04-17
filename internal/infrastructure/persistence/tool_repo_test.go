package persistence

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupToolRepoTestDB 创建工具 Repository 测试数据库
func setupToolRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tool.Tool{}, &tool.Connector{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// createTestTool 创建测试用工具
func createTestTool(t *testing.T, db *gorm.DB, tenantID, name, typ, status string) *tool.Tool {
	t.Helper()
	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Name:      name,
		Type:      typ,
		Status:    status,
	}
	if err := db.Create(tl).Error; err != nil {
		t.Fatalf("创建测试工具失败: %v", err)
	}
	return tl
}

// createTestConnector 创建测试用连接器
func createTestConnector(t *testing.T, db *gorm.DB, tenantID, name, typ, endpoint string) *tool.Connector {
	t.Helper()
	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Name:      name,
		Type:      typ,
		Endpoint:  endpoint,
	}
	if err := db.Create(conn).Error; err != nil {
		t.Fatalf("创建测试连接器失败: %v", err)
	}
	return conn
}

// ==================== ToolRepository 测试 ====================

// TestToolRepository_Create 创建工具成功
func TestToolRepository_Create(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "Altium Designer",
		Type:      "eda",
		Status:    "active",
	}

	err := repo.Create(context.Background(), tl)
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}
	if tl.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if tl.Name != "Altium Designer" {
		t.Errorf("期望 Name 为 'Altium Designer'，实际为 '%s'", tl.Name)
	}
}

// TestToolRepository_GetByID_Exists 查询存在的工具
func TestToolRepository_GetByID_Exists(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	created := createTestTool(t, db, "tenant-001", "TestTool", "eda", "active")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询工具失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 为 '%s'，实际为 '%s'", created.ID, found.ID)
	}
	if found.Name != "TestTool" {
		t.Errorf("期望 Name 为 'TestTool'，实际为 '%s'", found.Name)
	}
}

// TestToolRepository_GetByID_NotExists 查询不存在的工具
func TestToolRepository_GetByID_NotExists(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的工具返回错误")
	}
}

// TestToolRepository_List_NoFilter 无过滤返回全部
func TestToolRepository_List_NoFilter(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	createTestTool(t, db, "tenant-001", "Tool1", "eda", "active")
	createTestTool(t, db, "tenant-001", "Tool2", "cae", "active")

	tools, total, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(tools) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(tools))
	}
}

// TestToolRepository_List_FilterByType 按类型过滤
func TestToolRepository_List_FilterByType(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	createTestTool(t, db, "tenant-001", "Tool1", "eda", "active")
	createTestTool(t, db, "tenant-001", "Tool2", "cae", "active")
	createTestTool(t, db, "tenant-001", "Tool3", "eda", "inactive")

	tools, total, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-001", Type: "eda"})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, tl := range tools {
		if tl.Type != "eda" {
			t.Errorf("期望类型为 eda，实际为 '%s'", tl.Type)
		}
	}
}

// TestToolRepository_List_FilterByStatus 按状态过滤
func TestToolRepository_List_FilterByStatus(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	createTestTool(t, db, "tenant-001", "Tool1", "eda", "active")
	createTestTool(t, db, "tenant-001", "Tool2", "cae", "inactive")
	createTestTool(t, db, "tenant-001", "Tool3", "custom", "maintenance")

	tools, total, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-001", Status: "active"})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(tools) != 1 || tools[0].Status != "active" {
		t.Error("期望返回状态为 active 的工具")
	}
}

// TestToolRepository_List_FilterByKeyword 按关键词搜索
func TestToolRepository_List_FilterByKeyword(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	createTestTool(t, db, "tenant-001", "Altium Designer", "eda", "active")
	createTestTool(t, db, "tenant-001", "ANSYS Fluent", "cae", "active")
	createTestTool(t, db, "tenant-001", "MATLAB", "custom", "active")

	tools, total, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-001", Keyword: "altium"})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(tools) != 1 || tools[0].Name != "Altium Designer" {
		t.Error("期望返回名称包含 'altium' 的工具")
	}
}

// TestToolRepository_List_Pagination 分页正确
func TestToolRepository_List_Pagination(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	// 创建 5 条记录
	for i := 1; i <= 5; i++ {
		createTestTool(t, db, "tenant-001", "Tool"+string(rune('0'+i)), "eda", "active")
	}

	// 查询第 2 页，每页 2 条
	tools, total, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(tools) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(tools))
	}
}

// TestToolRepository_Update 更新成功
func TestToolRepository_Update(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	created := createTestTool(t, db, "tenant-001", "OldName", "eda", "active")

	created.Name = "NewName"
	created.Status = "inactive"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新工具失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的工具失败: %v", err)
	}
	if updated.Name != "NewName" {
		t.Errorf("期望 Name 为 'NewName'，实际为 '%s'", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", updated.Status)
	}
}

// TestToolRepository_Delete 软删除成功
func TestToolRepository_Delete(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	created := createTestTool(t, db, "tenant-001", "ToDelete", "eda", "active")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除工具失败: %v", err)
	}

	// 软删除后查询应返回错误
	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// TestToolRepository_TenantIsolation 多租户隔离
func TestToolRepository_TenantIsolation(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewToolRepository(db)

	// 租户 A 创建工具
	createTestTool(t, db, "tenant-a", "ToolA", "eda", "active")
	// 租户 B 创建工具
	createTestTool(t, db, "tenant-b", "ToolB", "cae", "active")

	// 租户 A 查询不应看到租户 B 的数据
	toolsA, totalA, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("查询租户A工具列表失败: %v", err)
	}
	if totalA != 1 {
		t.Errorf("期望租户A总数为 1，实际为 %d", totalA)
	}
	if len(toolsA) != 1 || toolsA[0].Name != "ToolA" {
		t.Error("期望租户A只能看到自己的工具")
	}

	// 租户 B 查询不应看到租户 A 的数据
	toolsB, totalB, err := repo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-b"})
	if err != nil {
		t.Fatalf("查询租户B工具列表失败: %v", err)
	}
	if totalB != 1 {
		t.Errorf("期望租户B总数为 1，实际为 %d", totalB)
	}
	if len(toolsB) != 1 || toolsB[0].Name != "ToolB" {
		t.Error("期望租户B只能看到自己的工具")
	}
}

// ==================== ConnectorRepository 测试 ====================

// TestConnectorRepository_Create 创建连接器成功
func TestConnectorRepository_Create(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "Windchill",
		Type:      "plm",
		Endpoint:  "https://plm.example.com",
	}

	err := repo.Create(context.Background(), conn)
	if err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}
	if conn.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestConnectorRepository_GetByID_Exists 查询存在的连接器
func TestConnectorRepository_GetByID_Exists(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	created := createTestConnector(t, db, "tenant-001", "Windchill", "plm", "https://plm.example.com")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询连接器失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 匹配")
	}
	if found.Name != "Windchill" {
		t.Errorf("期望 Name 为 'Windchill'，实际为 '%s'", found.Name)
	}
}

// TestConnectorRepository_GetByID_NotExists 查询不存在的连接器
func TestConnectorRepository_GetByID_NotExists(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的连接器返回错误")
	}
}

// TestConnectorRepository_List 无过滤返回全部
func TestConnectorRepository_List(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	createTestConnector(t, db, "tenant-001", "Windchill", "plm", "https://plm.example.com")
	createTestConnector(t, db, "tenant-001", "Jira", "custom", "https://jira.example.com")

	connectors, total, err := repo.List(context.Background(), toolapp.ConnectorFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询连接器列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(connectors) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(connectors))
	}
}

// TestConnectorRepository_Update 更新成功
func TestConnectorRepository_Update(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	created := createTestConnector(t, db, "tenant-001", "OldName", "plm", "https://old.example.com")

	created.Name = "NewName"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新连接器失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的连接器失败: %v", err)
	}
	if updated.Name != "NewName" {
		t.Errorf("期望 Name 为 'NewName'，实际为 '%s'", updated.Name)
	}
}

// TestConnectorRepository_Delete 软删除成功
func TestConnectorRepository_Delete(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	created := createTestConnector(t, db, "tenant-001", "ToDelete", "plm", "https://delete.example.com")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除连接器失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// TestConnectorRepository_TenantIsolation 多租户隔离
func TestConnectorRepository_TenantIsolation(t *testing.T) {
	db := setupToolRepoTestDB(t)
	repo := NewConnectorRepository(db)

	createTestConnector(t, db, "tenant-a", "ConnA", "plm", "https://a.example.com")
	createTestConnector(t, db, "tenant-b", "ConnB", "eda", "https://b.example.com")

	connectorsA, totalA, err := repo.List(context.Background(), toolapp.ConnectorFilter{TenantID: "tenant-a"})
	if err != nil {
		t.Fatalf("查询租户A连接器列表失败: %v", err)
	}
	if totalA != 1 {
		t.Errorf("期望租户A总数为 1，实际为 %d", totalA)
	}
	if len(connectorsA) != 1 || connectorsA[0].Name != "ConnA" {
		t.Error("期望租户A只能看到自己的连接器")
	}
}
