package handler

import (
	"git.neolidy.top/neo/flowx/pkg/response"
	"git.neolidy.top/neo/flowx/pkg/version"

	"github.com/gin-gonic/gin"
)

// @Summary      健康检查
// @Description  检查服务运行状态和版本信息
// @Tags         系统
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any
// @Router       /health [get]
//
// HealthCheck 健康检查接口
func HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status":  "ok",
		"version": version.Version,
	})
}
