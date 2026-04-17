package server

import (
	"log/slog"
	"time"

	"git.neolidy.top/neo/flowx/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// slogLogger Gin 日志中间件，使用 slog 结构化日志
func slogLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}
		if raw != "" {
			attrs = append(attrs, "query", raw)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		// 从 context 获取 tenant_id 和 user_id（如果已认证）
		if tenantID, exists := c.Get("tenant_id"); exists {
			attrs = append(attrs, "tenant_id", tenantID)
		}
		if userID, exists := c.Get("user_id"); exists {
			attrs = append(attrs, "user_id", userID)
		}

		switch {
		case status >= 500:
			slog.Error("请求处理", attrs...)
		case status >= 400:
			slog.Warn("请求处理", attrs...)
		default:
			slog.Info("请求处理", attrs...)
		}
	}
}

// NewServer 创建并配置Gin引擎
func NewServer(cfg config.ServerConfig) *gin.Engine {
	gin.SetMode(cfg.Mode)
	r := gin.New()

	// 使用 slog 结构化日志替代 gin.Logger()
	r.Use(slogLogger())
	r.Use(gin.Recovery())

	// CORS 中间件 - 使用可配置的 origins，默认安全值
	origins := cfg.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:3000", "http://localhost:8080"}
	}
	corsConfig := cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	// 如果 origins 包含 "*"，禁用 credentials（浏览器安全要求）
	for _, o := range origins {
		if o == "*" {
			corsConfig.AllowCredentials = false
			break
		}
	}
	r.Use(cors.New(corsConfig))

	return r
}
