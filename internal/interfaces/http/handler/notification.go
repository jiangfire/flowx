package handler

import (
	"errors"
	"net/http"
	"strconv"

	notificationapp "git.neolidy.top/neo/flowx/internal/application/notification"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// NotificationHandler 通知 HTTP 处理器
type NotificationHandler struct {
	service *notificationapp.NotificationService
}

// NewNotificationHandler 创建通知处理器实例
func NewNotificationHandler(service *notificationapp.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// ==================== Notification CRUD ====================

// @Summary      创建通知
// @Description  创建新的通知
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        request  body  notificationapp.CreateNotificationRequest  true  "创建通知请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications [post]
//
// CreateNotification 创建通知
// POST /api/v1/notifications
func (h *NotificationHandler) CreateNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req notificationapp.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.CreateNotification(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, notificationapp.ErrNotificationTitleRequired) || errors.Is(err, notificationapp.ErrNotificationTypeRequired) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建通知失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// @Summary      通知列表
// @Description  分页查询通知列表，支持按类型、分类、已读状态筛选
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        type       query  string  false  "通知类型"
// @Param        category   query  string  false  "通知分类"
// @Param        is_read    query  string  false  "已读状态"
// @Param        status     query  string  false  "通知状态"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications [get]
//
// ListNotifications 通知列表
// GET /api/v1/notifications
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var isRead *bool
	if c.Query("is_read") != "" {
		r := c.Query("is_read") == "true"
		isRead = &r
	}

	filter := notificationapp.ListNotificationsFilter{
		Type:     c.Query("type"),
		Category: c.Query("category"),
		IsRead:   isRead,
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	}

	notifications, paginated, err := h.service.ListNotifications(c.Request.Context(), tenantID, userID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询通知列表失败")
		return
	}

	response.Paginated(c, notifications, paginated.Total, paginated.Page, paginated.PageSize)
}

// @Summary      通知详情
// @Description  根据ID获取通知详细信息
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "通知ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/{id} [get]
//
// GetNotification 通知详情
// GET /api/v1/notifications/:id
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.service.GetNotification(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, notificationapp.ErrNotificationNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该通知")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询通知失败")
		return
	}

	response.Success(c, result)
}

// @Summary      更新通知
// @Description  根据ID更新通知信息
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id       path  string                                    true  "通知ID"
// @Param        request  body  notificationapp.UpdateNotificationRequest  true  "更新通知请求参数"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/{id} [put]
//
// UpdateNotification 更新通知
// PUT /api/v1/notifications/:id
func (h *NotificationHandler) UpdateNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req notificationapp.UpdateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.UpdateNotification(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, notificationapp.ErrNotificationNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该通知")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新通知失败")
		return
	}

	response.Success(c, result)
}

// @Summary      删除通知
// @Description  根据ID删除通知
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "通知ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/{id} [delete]
//
// DeleteNotification 删除通知
// DELETE /api/v1/notifications/:id
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.DeleteNotification(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, notificationapp.ErrNotificationNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该通知")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除通知失败")
		return
	}

	response.Success(c, nil)
}

// @Summary      标记通知为已读
// @Description  将指定通知标记为已读
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "通知ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/{id}/read [put]
//
// MarkAsRead 标记通知为已读
// PUT /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.MarkAsRead(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, notificationapp.ErrNotificationNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该通知")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "标记已读失败")
		return
	}

	response.Success(c, nil)
}

// @Summary      标记所有通知为已读
// @Description  将当前用户所有通知标记为已读
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/read-all [put]
//
// MarkAllAsRead 标记所有通知为已读
// PUT /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetString("user_id")

	err := h.service.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "全部标记已读失败")
		return
	}

	response.Success(c, nil)
}

// @Summary      获取未读通知数量
// @Description  获取当前用户的未读通知数量
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/unread-count [get]
//
// GetUnreadCount 获取未读通知数量
// GET /api/v1/notifications/unread-count
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.GetString("user_id")

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取未读数量失败")
		return
	}

	response.Success(c, gin.H{"count": count})
}

// @Summary      通过模板发送通知
// @Description  使用通知模板发送通知给指定用户
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        request  body  notificationapp.SendNotificationRequest  true  "发送通知请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notifications/send [post]
//
// SendNotification 通过模板发送通知
// POST /api/v1/notifications/send
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req notificationapp.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.SendNotification(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, notificationapp.ErrTemplateNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知模板不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该模板")
			return
		}
		if errors.Is(err, notificationapp.ErrUserMuted) {
			response.Error(c, http.StatusOK, "MUTED", "用户已设置免打扰")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "发送通知失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// ==================== Template CRUD ====================

// @Summary      创建通知模板
// @Description  创建新的通知模板
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        request  body  notificationapp.CreateTemplateRequest  true  "创建通知模板请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-templates [post]
//
// CreateTemplate 创建通知模板
// POST /api/v1/notification-templates
func (h *NotificationHandler) CreateTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req notificationapp.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.CreateTemplate(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, notificationapp.ErrTemplateCodeRequired) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建通知模板失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// @Summary      通知模板列表
// @Description  分页查询通知模板列表，支持按类型、状态筛选
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        type       query  string  false  "模板类型"
// @Param        status     query  string  false  "模板状态"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-templates [get]
//
// ListTemplates 通知模板列表
// GET /api/v1/notification-templates
func (h *NotificationHandler) ListTemplates(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := notificationapp.ListNotificationsFilter{
		Type:     c.Query("type"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	}

	templates, paginated, err := h.service.ListTemplates(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询通知模板列表失败")
		return
	}

	response.Paginated(c, templates, paginated.Total, paginated.Page, paginated.PageSize)
}

// @Summary      通知模板详情
// @Description  根据ID获取通知模板详细信息
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "通知模板ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-templates/{id} [get]
//
// GetTemplate 通知模板详情
// GET /api/v1/notification-templates/:id
func (h *NotificationHandler) GetTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.service.GetTemplate(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, notificationapp.ErrTemplateNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知模板不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该模板")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询通知模板失败")
		return
	}

	response.Success(c, result)
}

// @Summary      更新通知模板
// @Description  根据ID更新通知模板信息
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id       path  string                                    true  "通知模板ID"
// @Param        request  body  notificationapp.UpdateTemplateRequest      true  "更新通知模板请求参数"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-templates/{id} [put]
//
// UpdateTemplate 更新通知模板
// PUT /api/v1/notification-templates/:id
func (h *NotificationHandler) UpdateTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req notificationapp.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.UpdateTemplate(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, notificationapp.ErrTemplateNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知模板不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该模板")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新通知模板失败")
		return
	}

	response.Success(c, result)
}

// @Summary      删除通知模板
// @Description  根据ID删除通知模板
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "通知模板ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-templates/{id} [delete]
//
// DeleteTemplate 删除通知模板
// DELETE /api/v1/notification-templates/:id
func (h *NotificationHandler) DeleteTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.DeleteTemplate(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, notificationapp.ErrTemplateNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知模板不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该模板")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除通知模板失败")
		return
	}

	response.Success(c, nil)
}

// ==================== Preference CRUD ====================

// @Summary      创建通知偏好
// @Description  创建新的通知偏好设置
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        request  body  notificationapp.CreatePreferenceRequest  true  "创建通知偏好请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-preferences [post]
//
// CreatePreference 创建通知偏好
// POST /api/v1/notification-preferences
func (h *NotificationHandler) CreatePreference(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req notificationapp.CreatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.CreatePreference(c.Request.Context(), tenantID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建通知偏好失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// @Summary      通知偏好列表
// @Description  分页查询通知偏好列表，支持按用户、类型、渠道筛选
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        user_id    query  string  false  "用户ID"
// @Param        type       query  string  false  "偏好类型"
// @Param        channel    query  string  false  "通知渠道"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-preferences [get]
//
// ListPreferences 通知偏好列表
// GET /api/v1/notification-preferences
func (h *NotificationHandler) ListPreferences(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := notificationapp.ListPreferencesFilter{
		UserID:   c.Query("user_id"),
		Type:     c.Query("type"),
		Channel:  c.Query("channel"),
		Page:     page,
		PageSize: pageSize,
	}

	preferences, paginated, err := h.service.ListPreferences(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询通知偏好列表失败")
		return
	}

	response.Paginated(c, preferences, paginated.Total, paginated.Page, paginated.PageSize)
}

// @Summary      更新通知偏好
// @Description  根据ID更新通知偏好设置
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id       path  string                                      true  "通知偏好ID"
// @Param        request  body  notificationapp.UpdatePreferenceRequest      true  "更新通知偏好请求参数"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-preferences/{id} [put]
//
// UpdatePreference 更新通知偏好
// PUT /api/v1/notification-preferences/:id
func (h *NotificationHandler) UpdatePreference(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req notificationapp.UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.UpdatePreference(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, notificationapp.ErrPreferenceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知偏好不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该偏好")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新通知偏好失败")
		return
	}

	response.Success(c, result)
}

// @Summary      删除通知偏好
// @Description  根据ID删除通知偏好设置
// @Tags         通知系统
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "通知偏好ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /notification-preferences/{id} [delete]
//
// DeletePreference 删除通知偏好
// DELETE /api/v1/notification-preferences/:id
func (h *NotificationHandler) DeletePreference(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.DeletePreference(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, notificationapp.ErrPreferenceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "通知偏好不存在")
			return
		}
		if errors.Is(err, notificationapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该偏好")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除通知偏好失败")
		return
	}

	response.Success(c, nil)
}
