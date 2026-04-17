package auth

import (
	"context"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tenant"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAuthTestDB 创建认证服务测试用的内存数据库
func setupAuthTestDB(t *testing.T) *gorm.DB {
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

// testUserRepository 基于 GORM 的测试用 UserRepository 实现
type testUserRepository struct {
	db *gorm.DB
}

func newTestUserRepository(db *gorm.DB) UserRepository {
	return &testUserRepository{db: db}
}

func (r *testUserRepository) GetByUsername(ctx context.Context, username string) (*tenant.User, error) {
	var user tenant.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *testUserRepository) GetByID(ctx context.Context, id string) (*tenant.User, error) {
	var user tenant.User
	if err := r.db.WithContext(ctx).Select("id", "tenant_id", "username", "email", "role", "status", "created_at", "updated_at").
		Where("id = ?", id).First(&user).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *testUserRepository) Create(ctx context.Context, user *tenant.User) error {
	if user.ID == "" {
		user.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(user).Error
}

// createTestService 创建测试用认证服务
func createTestService(db *gorm.DB) *AuthService {
	jwtService := NewJWTService("test-secret-key-for-unit-test-12345", 24*time.Hour)
	userRepo := newTestUserRepository(db)
	return NewAuthService(userRepo, jwtService)
}

// TestRegister_CreateUser 验证 Register 创建新用户，密码正确哈希
func TestRegister_CreateUser(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	user, token, err := svc.Register("testuser", "test@example.com", "password123", "tenant-001")
	if err != nil {
		t.Fatalf("Register 返回错误: %v", err)
	}

	// 验证返回的用户信息
	if user.ID == "" {
		t.Error("期望用户 ID 不为空")
	}
	if user.Username != "testuser" {
		t.Errorf("期望 Username 为 'testuser'，实际为 '%s'", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("期望 Email 为 'test@example.com'，实际为 '%s'", user.Email)
	}
	if user.PasswordHash == "" {
		t.Error("期望 PasswordHash 不为空")
	}
	if user.TenantID != "tenant-001" {
		t.Errorf("期望 TenantID 为 'tenant-001'，实际为 '%s'", user.TenantID)
	}
	if user.Role != "user" {
		t.Errorf("期望默认 Role 为 'user'，实际为 '%s'", user.Role)
	}
	if user.Status != "active" {
		t.Errorf("期望默认 Status 为 'active'，实际为 '%s'", user.Status)
	}

	// 验证 token 不为空
	if token == "" {
		t.Error("期望 token 不为空")
	}

	// 验证密码被正确哈希（bcrypt 哈希后不应等于原始密码）
	if user.PasswordHash == "password123" {
		t.Error("期望密码被哈希，但 PasswordHash 等于原始密码")
	}

	// 验证密码哈希可以用 bcrypt 验证
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")); err != nil {
		t.Errorf("密码哈希验证失败: %v", err)
	}

	// 验证用户已保存到数据库
	var dbUser tenant.User
	if err := db.Where("username = ?", "testuser").First(&dbUser).Error; err != nil {
		t.Fatalf("从数据库查询用户失败: %v", err)
	}
	if dbUser.Username != "testuser" {
		t.Errorf("数据库中 Username 不匹配")
	}
}

// TestRegister_DuplicateUsername 验证 Register 重复用户名返回错误
func TestRegister_DuplicateUsername(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	// 创建第一个用户
	_, _, err := svc.Register("testuser", "test1@example.com", "password123", "tenant-001")
	if err != nil {
		t.Fatalf("创建第一个用户失败: %v", err)
	}

	// 尝试用相同用户名创建第二个用户
	_, _, err = svc.Register("testuser", "test2@example.com", "password456", "tenant-001")
	if err == nil {
		t.Error("期望重复用户名返回错误，但返回 nil")
	}
}

// TestLogin_CorrectPassword 验证 Login 正确密码返回 token
func TestLogin_CorrectPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	// 先注册用户
	_, _, err := svc.Register("testuser", "test@example.com", "password123", "tenant-001")
	if err != nil {
		t.Fatalf("注册用户失败: %v", err)
	}

	// 使用正确密码登录
	token, err := svc.Login("testuser", "password123")
	if err != nil {
		t.Fatalf("Login 返回错误: %v", err)
	}
	if token == "" {
		t.Error("期望 token 不为空")
	}
}

// TestLogin_WrongPassword 验证 Login 错误密码返回错误
func TestLogin_WrongPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	// 先注册用户
	_, _, err := svc.Register("testuser", "test@example.com", "password123", "tenant-001")
	if err != nil {
		t.Fatalf("注册用户失败: %v", err)
	}

	// 使用错误密码登录
	_, err = svc.Login("testuser", "wrongpassword")
	if err == nil {
		t.Error("期望错误密码返回错误，但返回 nil")
	}
}

// TestLogin_UserNotFound 验证 Login 不存在的用户返回错误
func TestLogin_UserNotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	// 登录不存在的用户
	_, err := svc.Login("nonexistent", "password123")
	if err == nil {
		t.Error("期望不存在的用户返回错误，但返回 nil")
	}
}

// TestGetProfile_ReturnsUserInfo 验证 GetProfile 返回用户信息（不含密码）
func TestGetProfile_ReturnsUserInfo(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	// 注册用户
	registeredUser, _, err := svc.Register("testuser", "test@example.com", "password123", "tenant-001")
	if err != nil {
		t.Fatalf("注册用户失败: %v", err)
	}

	// 获取用户信息
	profile, err := svc.GetProfile(registeredUser.ID)
	if err != nil {
		t.Fatalf("GetProfile 返回错误: %v", err)
	}

	if profile.ID != registeredUser.ID {
		t.Errorf("期望 ID 匹配")
	}
	if profile.Username != "testuser" {
		t.Errorf("期望 Username 为 'testuser'，实际为 '%s'", profile.Username)
	}
	if profile.Email != "test@example.com" {
		t.Errorf("期望 Email 为 'test@example.com'，实际为 '%s'", profile.Email)
	}
	if profile.PasswordHash != "" {
		t.Error("期望 GetProfile 不返回密码哈希")
	}
	if profile.TenantID != "tenant-001" {
		t.Errorf("期望 TenantID 为 'tenant-001'，实际为 '%s'", profile.TenantID)
	}
}

// TestGetProfile_UserNotFound 验证 GetProfile 用户不存在返回错误
func TestGetProfile_UserNotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	_, err := svc.GetProfile("nonexistent-id")
	if err == nil {
		t.Error("期望不存在的用户返回错误，但返回 nil")
	}
}

// ===================== Boundary Tests =====================

// TestRegister_ShortPassword 短密码应仍然可以注册（当前代码未强制最小长度）
func TestRegister_ShortPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	user, token, err := svc.Register("shortpw", "short@example.com", "ab", "tenant-001")
	if err != nil {
		t.Fatalf("短密码注册失败: %v", err)
	}
	if user.ID == "" {
		t.Error("期望用户 ID 不为空")
	}
	if token == "" {
		t.Error("期望 token 不为空")
	}
}

// TestLogin_EmptyUsername 空用户名登录应返回错误
func TestLogin_EmptyUsername(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	_, err := svc.Login("", "password")
	if err == nil {
		t.Error("期望空用户名返回错误")
	}
}

// TestLogin_EmptyPassword 空密码登录应返回错误
func TestLogin_EmptyPassword(t *testing.T) {
	db := setupAuthTestDB(t)
	svc := createTestService(db)

	_, err := svc.Login("username", "")
	if err == nil {
		t.Error("期望空密码返回错误")
	}
}
