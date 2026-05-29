package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jiangfire/flowx/pkg/response"
	"github.com/jiangfire/flowx/pkg/version"
	"gorm.io/gorm"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// @Summary      健康检查
// @Description  检查服务运行状态和版本信息（liveness probe）
// @Tags         系统
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any
// @Router       /health [get]
//
// HealthCheck 健康检查接口
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status":  "ok",
		"version": version.Version,
	})
}

// @Summary      就绪检查
// @Description  检查服务是否已准备好接收流量（readiness probe），包含数据库连接检查
// @Tags         系统
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      503  {object}  response.APIResponse
// @Router       /ready [get]
//
// ReadinessCheck 就绪检查接口
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	if h.db == nil {
		response.Error(c, http.StatusServiceUnavailable, "NOT_READY", "数据库连接未初始化")
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "NOT_READY", "数据库连接异常")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		response.Error(c, http.StatusServiceUnavailable, "NOT_READY", "数据库连接失败")
		return
	}

	response.Success(c, gin.H{
		"status":   "ready",
		"version":  version.Version,
		"database": "connected",
	})
}
