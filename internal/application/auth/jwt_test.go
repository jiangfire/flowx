package auth

import (
	"testing"
	"time"
)

// TestGenerateToken 验证 GenerateToken 正确生成 JWT
func TestGenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	claims := &TokenClaims{
		UserID:   "user-001",
		TenantID: "tenant-001",
		Roles:    []string{"admin"},
	}

	token, err := service.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}
	if token == "" {
		t.Fatal("期望 token 不为空")
	}

	// 验证生成的 token 可以被解析
	parsed, err := service.ParseToken(token)
	if err != nil {
		t.Fatalf("解析生成的 token 失败: %v", err)
	}
	if parsed.UserID != "user-001" {
		t.Errorf("期望 UserID 为 'user-001'，实际为 '%s'", parsed.UserID)
	}
	if parsed.TenantID != "tenant-001" {
		t.Errorf("期望 TenantID 为 'tenant-001'，实际为 '%s'", parsed.TenantID)
	}
	if len(parsed.Roles) != 1 || parsed.Roles[0] != "admin" {
		t.Errorf("期望 Roles 为 ['admin']，实际为 %v", parsed.Roles)
	}
}

// TestParseToken 验证 ParseToken 正确解析 JWT
func TestParseToken(t *testing.T) {
	service := NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	claims := &TokenClaims{
		UserID:   "user-002",
		TenantID: "tenant-002",
		Roles:    []string{"user", "editor"},
	}

	token, err := service.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}

	parsed, err := service.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken 返回错误: %v", err)
	}
	if parsed.UserID != "user-002" {
		t.Errorf("期望 UserID 为 'user-002'，实际为 '%s'", parsed.UserID)
	}
	if parsed.TenantID != "tenant-002" {
		t.Errorf("期望 TenantID 为 'tenant-002'，实际为 '%s'", parsed.TenantID)
	}
	if len(parsed.Roles) != 2 {
		t.Errorf("期望 Roles 长度为 2，实际为 %d", len(parsed.Roles))
	}
}

// TestParseTokenExpired 验证 ParseToken 过期 token 返回错误
func TestParseTokenExpired(t *testing.T) {
	// 使用极短的过期时间
	service := NewJWTService("test-secret-key-1234567890123456", -1*time.Hour)

	claims := &TokenClaims{
		UserID:   "user-003",
		TenantID: "tenant-003",
		Roles:    []string{"user"},
	}

	token, err := service.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}

	_, err = service.ParseToken(token)
	if err == nil {
		t.Error("期望 ParseToken 对过期 token 返回错误，但返回 nil")
	}
}

// TestParseTokenInvalid 验证 ParseToken 无效 token 返回错误
func TestParseTokenInvalid(t *testing.T) {
	service := NewJWTService("test-secret-key-1234567890123456", 24*time.Hour)

	// 使用无效 token
	_, err := service.ParseToken("invalid.token.string")
	if err == nil {
		t.Error("期望 ParseToken 对无效 token 返回错误，但返回 nil")
	}
}

// TestParseTokenWrongSecret 验证使用错误密钥解析 token 返回错误
func TestParseTokenWrongSecret(t *testing.T) {
	service1 := NewJWTService("secret-key-one-1234567890123456", 24*time.Hour)
	service2 := NewJWTService("secret-key-two-1234567890123456", 24*time.Hour)

	claims := &TokenClaims{
		UserID:   "user-004",
		TenantID: "tenant-004",
		Roles:    []string{"admin"},
	}

	token, err := service1.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken 返回错误: %v", err)
	}

	// 用不同密钥的服务解析
	_, err = service2.ParseToken(token)
	if err == nil {
		t.Error("期望使用错误密钥解析 token 返回错误，但返回 nil")
	}
}
