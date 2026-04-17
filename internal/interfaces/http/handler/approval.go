package handler

import (
	"errors"
	"net/http"
	"strconv"

	approvalapp "git.neolidy.top/neo/flowx/internal/application/approval"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// ApprovalHandler 审批处理器
type ApprovalHandler struct {
	svc approvalapp.ApprovalService
}

// NewApprovalHandler 创建审批处理器实例
func NewApprovalHandler(svc approvalapp.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{svc: svc}
}

// ===================== Workflow =====================

// createWorkflowRequest 创建工作流请求参数
type createWorkflowRequest struct {
	Name        string         `json:"name" binding:"required,max=200"`
	Type        string         `json:"type" binding:"required,max=50"`
	Description string         `json:"description"`
	Definition  map[string]any `json:"definition" binding:"required"`
}

// CreateWorkflow 创建工作流
// POST /api/v1/workflows
func (h *ApprovalHandler) CreateWorkflow(c *gin.Context) {
	var req createWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	tenantID := c.GetString("tenant_id")
	w, err := h.svc.CreateWorkflow(c.Request.Context(), tenantID, &approvalapp.CreateWorkflowRequest{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Definition:  req.Definition,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建工作流失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    w,
	})
}

// listWorkflowsRequest 工作流列表请求参数
type listWorkflowsRequest struct {
	Type     string `form:"type"`
	Status   string `form:"status"`
	Page     int    `form:"page,default=1" binding:"min=1"`
	PageSize int    `form:"page_size,default=20" binding:"min=1,max=100"`
}

// ListWorkflows 工作流列表
// GET /api/v1/workflows
func (h *ApprovalHandler) ListWorkflows(c *gin.Context) {
	var req listWorkflowsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	tenantID := c.GetString("tenant_id")
	workflows, result, err := h.svc.ListWorkflows(c.Request.Context(), tenantID, approvalapp.WorkflowFilter{
		TenantID: tenantID,
		Type:     req.Type,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询工作流列表失败")
		return
	}

	response.Paginated(c, workflows, result.Total, result.Page, result.PageSize)
}

// GetWorkflow 工作流详情
// GET /api/v1/workflows/:id
func (h *ApprovalHandler) GetWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	w, err := h.svc.GetWorkflow(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, approvalapp.ErrWorkflowNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "工作流不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询工作流失败")
		return
	}

	response.Success(c, w)
}

// ActivateWorkflow 激活工作流
// POST /api/v1/workflows/:id/activate
func (h *ApprovalHandler) ActivateWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	w, err := h.svc.ActivateWorkflow(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, w)
}

// ArchiveWorkflow 归档工作流
// POST /api/v1/workflows/:id/archive
func (h *ApprovalHandler) ArchiveWorkflow(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	w, err := h.svc.ArchiveWorkflow(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, w)
}

// ===================== Instance =====================

// startApprovalRequest 发起审批请求参数
type startApprovalRequest struct {
	WorkflowID string         `json:"workflow_id" binding:"required"`
	Title      string         `json:"title" binding:"required,max=500"`
	Context    map[string]any `json:"context"`
}

// StartApproval 发起审批
// POST /api/v1/approvals
func (h *ApprovalHandler) StartApproval(c *gin.Context) {
	var req startApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	inst, err := h.svc.StartApproval(c.Request.Context(), tenantID, userID, &approvalapp.StartApprovalRequest{
		WorkflowID: req.WorkflowID,
		Title:      req.Title,
		Context:    req.Context,
	})
	if err != nil {
		if errors.Is(err, approvalapp.ErrWorkflowNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "工作流不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "发起审批失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    inst,
	})
}

// listInstancesRequest 实例列表请求参数
type listInstancesRequest struct {
	Status     string `form:"status"`
	WorkflowID string `form:"workflow_id"`
	Page       int    `form:"page,default=1" binding:"min=1"`
	PageSize   int    `form:"page_size,default=20" binding:"min=1,max=100"`
}

// ListInstances 审批实例列表
// GET /api/v1/approvals
func (h *ApprovalHandler) ListInstances(c *gin.Context) {
	var req listInstancesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	tenantID := c.GetString("tenant_id")
	instances, result, err := h.svc.ListInstances(c.Request.Context(), tenantID, approvalapp.InstanceFilter{
		TenantID:   tenantID,
		Status:     req.Status,
		WorkflowID: req.WorkflowID,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询审批实例列表失败")
		return
	}

	response.Paginated(c, instances, result.Total, result.Page, result.PageSize)
}

// GetInstance 审批实例详情
// GET /api/v1/approvals/:id
func (h *ApprovalHandler) GetInstance(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	inst, err := h.svc.GetInstance(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, approvalapp.ErrInstanceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "审批实例不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询审批实例失败")
		return
	}

	response.Success(c, inst)
}

// approveRequest 审批通过请求参数
type approveRequest struct {
	Comment string `json:"comment"`
}

// Approve 审批通过
// POST /api/v1/approvals/:id/approve
func (h *ApprovalHandler) Approve(c *gin.Context) {
	instanceID := c.Param("id")
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req approveRequest
	c.ShouldBindJSON(&req) // comment 非必填

	ap, err := h.svc.Approve(c.Request.Context(), tenantID, userID, &approvalapp.ApproveRequest{
		InstanceID: instanceID,
		Comment:    req.Comment,
	})
	if err != nil {
		if errors.Is(err, approvalapp.ErrInstanceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "审批实例不存在")
			return
		}
		if errors.Is(err, approvalapp.ErrApprovalNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "待审批记录不存在")
			return
		}
		if errors.Is(err, approvalapp.ErrInstanceFinished) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "审批已结束")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "审批操作失败")
		return
	}

	response.Success(c, ap)
}

// rejectRequest 审批驳回请求参数
type rejectRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// Reject 审批驳回
// POST /api/v1/approvals/:id/reject
func (h *ApprovalHandler) Reject(c *gin.Context) {
	instanceID := c.Param("id")
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req rejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	ap, err := h.svc.Reject(c.Request.Context(), tenantID, userID, &approvalapp.RejectRequest{
		InstanceID: instanceID,
		Comment:    req.Comment,
	})
	if err != nil {
		if errors.Is(err, approvalapp.ErrInstanceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "审批实例不存在")
			return
		}
		if errors.Is(err, approvalapp.ErrApprovalNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "待审批记录不存在")
			return
		}
		if errors.Is(err, approvalapp.ErrInstanceFinished) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "审批已结束")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "审批操作失败")
		return
	}

	response.Success(c, ap)
}

// forwardRequest 转审请求参数
type forwardRequest struct {
	ToApproverID string `json:"to_approver_id" binding:"required"`
	Comment      string `json:"comment"`
}

// Forward 转审
// POST /api/v1/approvals/:id/forward
func (h *ApprovalHandler) Forward(c *gin.Context) {
	instanceID := c.Param("id")
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req forwardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	ap, err := h.svc.Forward(c.Request.Context(), tenantID, userID, &approvalapp.ForwardRequest{
		InstanceID:   instanceID,
		ToApproverID: req.ToApproverID,
		Comment:      req.Comment,
	})
	if err != nil {
		if errors.Is(err, approvalapp.ErrInstanceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "审批实例不存在")
			return
		}
		if errors.Is(err, approvalapp.ErrApprovalNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "待审批记录不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "转审操作失败")
		return
	}

	response.Success(c, ap)
}

// CancelInstance 取消审批
// POST /api/v1/approvals/:id/cancel
func (h *ApprovalHandler) CancelInstance(c *gin.Context) {
	instanceID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	err := h.svc.CancelInstance(c.Request.Context(), tenantID, instanceID)
	if err != nil {
		if errors.Is(err, approvalapp.ErrInstanceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "审批实例不存在")
			return
		}
		if errors.Is(err, approvalapp.ErrInstanceFinished) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "审批已结束，无法取消")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "取消审批失败")
		return
	}

	response.Success(c, gin.H{"message": "取消成功"})
}

// GetSuggestion 获取 AI 审批建议
// GET /api/v1/approvals/:id/suggestion
func (h *ApprovalHandler) GetSuggestion(c *gin.Context) {
	instanceID := c.Param("id")
	tenantID := c.GetString("tenant_id")

	suggestion, err := h.svc.GetSuggestion(c.Request.Context(), tenantID, instanceID)
	if err != nil {
		if errors.Is(err, approvalapp.ErrInstanceNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "审批实例不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取审批建议失败")
		return
	}

	response.Success(c, gin.H{"suggestion": suggestion})
}

// GetMyPendingApprovals 我的待审批列表
// GET /api/v1/approvals/pending
func (h *ApprovalHandler) GetMyPendingApprovals(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	instances, err := h.svc.GetMyPendingApprovals(c.Request.Context(), tenantID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询待审批列表失败")
		return
	}

	response.Success(c, instances)
}

// parseIntParam 辅助函数：解析路径参数中的整数
func parseIntParam(c *gin.Context, name string, defaultValue int) int {
	val := c.Param(name)
	if val == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return result
}
