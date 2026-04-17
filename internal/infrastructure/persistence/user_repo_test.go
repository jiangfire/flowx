package persistence

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/tenant"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupUserTestDB 创建用户模块测试数据库
func setupUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tenant.User{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// TestGetByUsername_Success 按用户名查询用户成功
func TestGetByUsername_Success(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &tenant.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Status:       "active",
	}
	repo.Create(ctx, user)

	found, err := repo.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if found.Username != "testuser" {
		t.Errorf("期望 Username 为 'testuser'，实际为 '%s'", found.Username)
	}
	if found.Email != "test@example.com" {
		t.Errorf("期望 Email 为 'test@example.com'，实际为 '%s'", found.Email)
	}
}

// TestGetByUsername_NotFound 用户不存在返回错误
func TestGetByUsername_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByUsername(ctx, "nonexistent")
	if err == nil {
		t.Fatal("期望返回错误，但返回 nil")
	}
}

// TestUserGetByID_Success 按 ID 查询用户成功
func TestUserGetByID_Success(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &tenant.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Status:       "active",
	}
	repo.Create(ctx, user)

	found, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("期望 ID 匹配")
	}
	if found.Username != "testuser" {
		t.Errorf("期望 Username 为 'testuser'，实际为 '%s'", found.Username)
	}
}

// TestUserGetByID_NotFound 用户不存在返回错误
func TestUserGetByID_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("期望返回错误，但返回 nil")
	}
}

// TestCreate_Success 创建用户成功
func TestCreate_Success(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &tenant.User{
		Username:     "newuser",
		Email:        "new@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if user.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}

	// 验证可以从数据库查询到
	found, err := repo.GetByUsername(ctx, "newuser")
	if err != nil {
		t.Fatalf("查询创建的用户失败: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("期望 ID 匹配")
	}
}

// TestCreate_GeneratesUUID 创建用户时自动生成 UUID
func TestCreate_GeneratesUUID(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &tenant.User{
		Username:     "uuiduser",
		Email:        "uuid@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if user.ID == "" {
		t.Error("期望创建后自动生成 ID")
	}
}
