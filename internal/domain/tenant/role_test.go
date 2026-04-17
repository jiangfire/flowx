package tenant

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试用内存数据库
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	return db
}

// TestRoleTableName 验证 Role 表名正确
func TestRoleTableName(t *testing.T) {
	role := Role{}
	if role.TableName() != "roles" {
		t.Errorf("期望 Role 表名为 'roles'，实际为 '%s'", role.TableName())
	}
}

// TestPermissionTableName 验证 Permission 表名正确
func TestPermissionTableName(t *testing.T) {
	perm := Permission{}
	if perm.TableName() != "permissions" {
		t.Errorf("期望 Permission 表名为 'permissions'，实际为 '%s'", perm.TableName())
	}
}

// TestRoleFields 验证 Role 结构体字段
func TestRoleFields(t *testing.T) {
	role := Role{
		Name:        "admin",
		DisplayName: "管理员",
		Description: "系统管理员角色",
	}

	if role.Name != "admin" {
		t.Errorf("期望 Name 为 'admin'，实际为 '%s'", role.Name)
	}
	if role.DisplayName != "管理员" {
		t.Errorf("期望 DisplayName 为 '管理员'，实际为 '%s'", role.DisplayName)
	}
	if role.Description != "系统管理员角色" {
		t.Errorf("期望 Description 为 '系统管理员角色'，实际为 '%s'", role.Description)
	}
}

// TestPermissionFields 验证 Permission 结构体字段
func TestPermissionFields(t *testing.T) {
	perm := Permission{
		Name:        "user:read",
		DisplayName: "查看用户",
		Resource:    "user",
		Action:      "read",
	}

	if perm.Name != "user:read" {
		t.Errorf("期望 Name 为 'user:read'，实际为 '%s'", perm.Name)
	}
	if perm.DisplayName != "查看用户" {
		t.Errorf("期望 DisplayName 为 '查看用户'，实际为 '%s'", perm.DisplayName)
	}
	if perm.Resource != "user" {
		t.Errorf("期望 Resource 为 'user'，实际为 '%s'", perm.Resource)
	}
	if perm.Action != "read" {
		t.Errorf("期望 Action 为 'read'，实际为 '%s'", perm.Action)
	}
}

// TestRoleAutoMigrate 验证 Role 和 Permission 模型能正确迁移
func TestRoleAutoMigrate(t *testing.T) {
	db := setupTestDB(t)

	err := db.AutoMigrate(&Role{}, &Permission{})
	if err != nil {
		t.Fatalf("Role/Permission 自动迁移失败: %v", err)
	}

	if !db.Migrator().HasTable("roles") {
		t.Error("期望 'roles' 表已创建")
	}
	if !db.Migrator().HasTable("permissions") {
		t.Error("期望 'permissions' 表已创建")
	}
}
