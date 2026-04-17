package handler

import (
	"errors"
	"net/http"

	"git.neolidy.top/neo/flowx/internal/application/auth"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// 为避免循环导入，在 handler 包中定义响应结构
// 这里使用 response 包的函数和结构体

// apiResponse 内部使用的 API 响应结构（别名）
type apiResponse = response.APIResponse

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *auth.AuthService
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// registerRequest 注册请求参数
type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	TenantID string `json:"tenant_id" binding:"required,min=1,max=26"`
}

// loginRequest 登录请求参数
type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register 用户注册
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	user, token, err := h.authService.Register(req.Username, req.Email, req.Password, req.TenantID)
	if err != nil {
		if errors.Is(err, auth.ErrUserAlreadyExists) {
			response.Error(c, http.StatusConflict, "USER_EXISTS", "用户名已存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
		return
	}

	c.JSON(http.StatusCreated, apiResponse{
		Code:    0,
		Message: "success",
		Data: gin.H{
			"user":  user,
			"token": token,
		},
	})
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名或密码错误")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "登录失败")
		return
	}

	response.Success(c, gin.H{
		"token": token,
	})
}

// Profile 获取用户信息
// GET /api/v1/auth/profile
func (h *AuthHandler) Profile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证")
		return
	}

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "用户不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取用户信息失败")
		return
	}

	response.Success(c, user)
}
