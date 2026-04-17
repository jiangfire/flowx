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
