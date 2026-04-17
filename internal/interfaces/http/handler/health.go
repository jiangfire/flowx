package handler

import (
	"git.neolidy.top/neo/flowx/pkg/response"
	"git.neolidy.top/neo/flowx/pkg/version"

	"github.com/gin-gonic/gin"
)

// HealthCheck 健康检查接口
func HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status":  "ok",
		"version": version.Version,
	})
}
