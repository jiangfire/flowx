package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims JWT 自定义声明
type TokenClaims struct {
	UserID   string   `json:"user_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
}

// jwtClaims 内部 JWT 声明，包含标准声明和自定义声明
type jwtClaims struct {
	TokenClaims
	jwt.RegisteredClaims
}

// JWTService JWT 令牌服务接口
type JWTService interface {
	// GenerateToken 生成 JWT 令牌
	GenerateToken(claims *TokenClaims) (string, error)
	// ParseToken 解析 JWT 令牌
	ParseToken(tokenString string) (*TokenClaims, error)
}

// jwtService JWT 令牌服务实现
type jwtService struct {
	secret      []byte
	expireHours time.Duration
}

// NewJWTService 创建 JWT 服务实例
func NewJWTService(secret string, expireHours time.Duration) JWTService {
	return &jwtService{
		secret:      []byte(secret),
		expireHours: expireHours,
	}
}

// GenerateToken 生成 JWT 令牌
func (s *jwtService) GenerateToken(claims *TokenClaims) (string, error) {
	now := time.Now()
	jwtClaims := jwtClaims{
		TokenClaims: *claims,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expireHours)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "flowx",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString(s.secret)
}

// ParseToken 解析 JWT 令牌
func (s *jwtService) ParseToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return &claims.TokenClaims, nil
}
