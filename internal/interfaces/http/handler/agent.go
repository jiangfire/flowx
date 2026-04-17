package handler

import (
	"errors"
	"net/http"
	"strconv"

	agentapp "git.neolidy.top/neo/flowx/internal/application/agent"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// AgentHandler Agent HTTP 处理器
type AgentHandler struct {
	service *agentapp.AgentService
}

// NewAgentHandler 创建 Agent 处理器实例
func NewAgentHandler(service *agentapp.AgentService) *AgentHandler {
	return &AgentHandler{service: service}
}

// ==================== 工具列表 ====================

// @Summary      获取可用工具列表
// @Description  获取Agent可用的工具列表
// @Tags         Agent智能体
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /agent/tools [get]
//
// ListTools 获取可用工具列表
// GET /api/v1/agent/tools
func (h *AgentHandler) ListTools(c *gin.Context) {
	tools, err := h.service.ListAvailableTools(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取工具列表失败")
		return
	}

	response.Success(c, tools)
}

// ==================== 任务管理 ====================

// createTaskRequest 创建任务请求参数
type createTaskRequest struct {
	Type            string         `json:"type" binding:"required,max=50"`
	Description     string         `json:"description" binding:"required,max=500"`
	Context         map[string]any `json:"context"`
	Steps           []taskStepReq  `json:"steps" binding:"required,min=1"`
	RequireApproval bool           `json:"require_approval"`
	WorkflowID      string         `json:"workflow_id"` // 关联的工作流定义 ID（可选）
}

// taskStepReq 任务步骤请求参数
type taskStepReq struct {
	Type        string         `json:"type" binding:"required"`
	Description string         `json:"description"`
	Params      map[string]any `json:"params"`
}

// @Summary      创建并执行任务
// @Description  创建Agent任务并自动执行
// @Tags         Agent智能体
// @Accept       json
// @Produce      json
// @Param        request  body  createTaskRequest  true  "创建任务请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /agent/tasks [post]
//
// CreateTask 创建并执行任务
// POST /api/v1/agent/tasks
func (h *AgentHandler) CreateTask(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	// 转换步骤
	steps := make([]agentapp.TaskStep, len(req.Steps))
	for i, s := range req.Steps {
		steps[i] = agentapp.TaskStep{
			Type:        s.Type,
			Description: s.Description,
			Params:      s.Params,
		}
	}

	// 创建任务对象
	task := &agentapp.Task{
		ID:              base.GenerateUUID(),
		Type:            req.Type,
		Description:     req.Description,
		Context:         req.Context,
		Steps:           steps,
		RequireApproval: req.RequireApproval,
		WorkflowID:      req.WorkflowID,
	}

	result, err := h.service.CreateAndExecuteTask(c.Request.Context(), tenantID, userID, task)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// @Summary      任务列表
// @Description  分页查询Agent任务列表，支持按状态筛选
// @Tags         Agent智能体
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        status     query  string  false  "任务状态"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /agent/tasks [get]
//
// ListTasks 任务列表
// GET /api/v1/agent/tasks
func (h *AgentHandler) ListTasks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	tasks, paginated, err := h.service.ListTasks(c.Request.Context(), tenantID, status, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询任务列表失败")
		return
	}

	// 转换为响应格式
	items := make([]map[string]any, len(tasks))
	for i, t := range tasks {
		items[i] = agentapp.TaskToResponse(t)
	}

	response.Paginated(c, items, paginated.Total, paginated.Page, paginated.PageSize)
}

// @Summary      任务详情
// @Description  根据ID获取Agent任务详细信息
// @Tags         Agent智能体
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "任务ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /agent/tasks/{id} [get]
//
// GetTask 任务详情
// GET /api/v1/agent/tasks/:id
func (h *AgentHandler) GetTask(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	taskID := c.Param("id")

	task, err := h.service.GetTask(c.Request.Context(), tenantID, taskID)
	if err != nil {
		if errors.Is(err, agentapp.ErrTaskNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询任务失败")
		return
	}

	response.Success(c, agentapp.TaskToResponse(*task))
}

// @Summary      审批通过任务
// @Description  审批通过指定的Agent任务
// @Tags         Agent智能体
// @Accept       json
// @Produce      json
// @Param        id       path  string          true  "任务ID"
// @Param        request  body  object          false  "审批请求 {comment: string}"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /agent/tasks/{id}/approve [post]
//
// ApproveTask 审批通过
// POST /api/v1/agent/tasks/:id/approve
func (h *AgentHandler) ApproveTask(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	taskID := c.Param("id")

	var req struct {
		Comment string `json:"comment"`
	}
	c.ShouldBindJSON(&req) // comment 非必填

	task, err := h.service.ApproveTask(c.Request.Context(), tenantID, userID, taskID, req.Comment)
	if err != nil {
		if errors.Is(err, agentapp.ErrTaskNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, agentapp.ErrTaskNotPending) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "审批操作失败")
		return
	}

	response.Success(c, agentapp.TaskToResponse(*task))
}

// @Summary      拒绝任务
// @Description  拒绝指定的Agent任务
// @Tags         Agent智能体
// @Accept       json
// @Produce      json
// @Param        id       path  string  true  "任务ID"
// @Param        request  body  object  true  "拒绝请求 {comment: string}"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /agent/tasks/{id}/reject [post]
//
// RejectTask 拒绝任务
// POST /api/v1/agent/tasks/:id/reject
func (h *AgentHandler) RejectTask(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	taskID := c.Param("id")

	var req struct {
		Comment string `json:"comment" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	task, err := h.service.RejectTask(c.Request.Context(), tenantID, userID, taskID, req.Comment)
	if err != nil {
		if errors.Is(err, agentapp.ErrTaskNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, agentapp.ErrTaskNotPending) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "拒绝操作失败")
		return
	}

	response.Success(c, agentapp.TaskToResponse(*task))
}
