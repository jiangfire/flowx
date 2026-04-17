package handler

import (
	"io"
	"net/http"
	"strconv"

	bpmnapp "git.neolidy.top/neo/flowx/internal/application/bpmn"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// BPMNHandler BPMN 流程 HTTP 处理器
type BPMNHandler struct {
	service *bpmnapp.ProcessService
}

// NewBPMNHandler 创建 BPMN 处理器实例
func NewBPMNHandler(service *bpmnapp.ProcessService) *BPMNHandler {
	return &BPMNHandler{service: service}
}

// ==================== 流程定义 ====================

// @Summary      部署流程定义
// @Description  部署新的BPMN流程定义（YAML格式）
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        body  body  string  true  "YAML格式的流程定义"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-definitions [post]
//
// DeployDefinition 部署流程定义
// POST /api/v1/process-definitions
func (h *BPMNHandler) DeployDefinition(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	yamlData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "读取请求体失败")
		return
	}

	def, err := h.service.DeployDefinition(c.Request.Context(), tenantID, yamlData)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    def,
	})
}

// @Summary      流程定义列表
// @Description  分页查询流程定义列表，支持按状态、关键词筛选
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        status     query  string  false  "定义状态"
// @Param        keyword    query  string  false  "关键词搜索"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-definitions [get]
//
// ListDefinitions 流程定义列表
// GET /api/v1/process-definitions
func (h *BPMNHandler) ListDefinitions(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := bpmnapp.ProcessDefinitionFilter{
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}

	defs, total, err := h.service.ListDefinitions(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询流程定义列表失败")
		return
	}

	response.Paginated(c, defs, total, filter.Page, filter.PageSize)
}

// @Summary      流程定义详情
// @Description  根据ID获取流程定义详细信息
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "流程定义ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-definitions/{id} [get]
//
// GetDefinition 流程定义详情
// GET /api/v1/process-definitions/:id
func (h *BPMNHandler) GetDefinition(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	def, err := h.service.GetDefinition(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "流程定义不存在")
		return
	}

	response.Success(c, def)
}

// ==================== 流程实例 ====================

// @Summary      启动流程实例
// @Description  根据流程定义启动新的流程实例
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        request  body  object  true  "启动流程请求 {definition_id: string, variables: map}"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances [post]
//
// StartProcess 启动流程实例
// POST /api/v1/process-instances
func (h *BPMNHandler) StartProcess(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	var req struct {
		DefinitionID string         `json:"definition_id" binding:"required"`
		Variables    map[string]any `json:"variables"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	inst, err := h.service.StartProcess(c.Request.Context(), tenantID, req.DefinitionID, userID, req.Variables)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    inst,
	})
}

// @Summary      流程实例列表
// @Description  分页查询流程实例列表，支持按状态筛选
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        status     query  string  false  "实例状态"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances [get]
//
// ListProcessInstances 流程实例列表
// GET /api/v1/process-instances
func (h *BPMNHandler) ListProcessInstances(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := bpmnapp.ProcessInstanceFilter{
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	}

	instances, total, err := h.service.ListProcessInstances(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询流程实例列表失败")
		return
	}

	response.Paginated(c, instances, total, filter.Page, filter.PageSize)
}

// @Summary      流程实例详情
// @Description  根据ID获取流程实例详细信息
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "流程实例ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances/{id} [get]
//
// GetProcessInstance 流程实例详情
// GET /api/v1/process-instances/:id
func (h *BPMNHandler) GetProcessInstance(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	inst, err := h.service.GetProcessInstance(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "流程实例不存在")
		return
	}

	response.Success(c, inst)
}

// @Summary      挂起流程实例
// @Description  挂起指定流程实例
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "流程实例ID"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances/{id}/suspend [post]
//
// SuspendProcess 挂起流程实例
// POST /api/v1/process-instances/:id/suspend
func (h *BPMNHandler) SuspendProcess(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.SuspendProcess(c.Request.Context(), tenantID, id); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary      恢复流程实例
// @Description  恢复已挂起的流程实例
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "流程实例ID"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances/{id}/resume [post]
//
// ResumeProcess 恢复流程实例
// POST /api/v1/process-instances/:id/resume
func (h *BPMNHandler) ResumeProcess(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.ResumeProcess(c.Request.Context(), tenantID, id); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary      取消流程实例
// @Description  取消指定流程实例
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "流程实例ID"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances/{id}/cancel [post]
//
// CancelProcess 取消流程实例
// POST /api/v1/process-instances/:id/cancel
func (h *BPMNHandler) CancelProcess(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	if err := h.service.CancelProcess(c.Request.Context(), tenantID, id); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary      获取流程实例的任务列表
// @Description  获取指定流程实例的所有任务
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "流程实例ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances/{id}/tasks [get]
//
// GetProcessTasks 获取流程实例的任务列表
// GET /api/v1/process-instances/:id/tasks
func (h *BPMNHandler) GetProcessTasks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	instanceID := c.Param("id")

	tasks, err := h.service.GetProcessTasks(c.Request.Context(), tenantID, instanceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询任务列表失败")
		return
	}

	response.Success(c, tasks)
}

// @Summary      完成任务
// @Description  完成指定的流程任务
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        id       path  string  true  "流程实例ID"
// @Param        taskId   path  string  true  "任务ID"
// @Param        request  body  object  false  "提交数据 {submitted_data: map}"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-instances/{id}/tasks/{taskId}/complete [post]
//
// CompleteTask 完成任务
// POST /api/v1/process-instances/:id/tasks/:taskId/complete
func (h *BPMNHandler) CompleteTask(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	taskID := c.Param("taskId")
	userID := c.GetString("user_id")

	var req struct {
		SubmittedData map[string]any `json:"submitted_data"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := h.service.CompleteTask(c.Request.Context(), tenantID, taskID, userID, req.SubmittedData); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(c, nil)
}

// @Summary      获取待办任务
// @Description  获取指定处理人的待办任务列表
// @Tags         BPMN流程
// @Accept       json
// @Produce      json
// @Param        assignee  query  string  false  "处理人"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /process-tasks/pending [get]
//
// GetPendingTasks 获取待办任务
// GET /api/v1/process-tasks/pending
func (h *BPMNHandler) GetPendingTasks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	assignee := c.Query("assignee")

	tasks, err := h.service.GetPendingTasks(c.Request.Context(), tenantID, assignee)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询待办任务失败")
		return
	}

	response.Success(c, tasks)
}
