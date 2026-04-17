package handler

import (
	"errors"
	"net/http"
	"strconv"

	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	bizerrors "git.neolidy.top/neo/flowx/pkg/errors"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// ToolHandler 工具 HTTP 处理器
type ToolHandler struct {
	toolService  *toolapp.ToolService
	excelService *toolapp.ExcelService
}

// NewToolHandler 创建工具处理器实例
func NewToolHandler(toolService *toolapp.ToolService, excelService *toolapp.ExcelService) *ToolHandler {
	return &ToolHandler{
		toolService:  toolService,
		excelService: excelService,
	}
}

// getUserRole 从 JWT 上下文中提取用户角色
func getUserRole(c *gin.Context) string {
	roles, exists := c.Get("roles")
	if !exists {
		return ""
	}
	roleSlice, ok := roles.([]string)
	if !ok || len(roleSlice) == 0 {
		return ""
	}
	return roleSlice[0]
}

// ==================== 工具 CRUD ====================

// @Summary      创建工具
// @Description  创建新的工具
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        request  body  toolapp.CreateToolRequest  true  "创建工具请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools [post]
//
// CreateTool 创建工具
// POST /api/v1/tools
func (h *ToolHandler) CreateTool(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userRole := getUserRole(c)

	var req toolapp.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.toolService.CreateTool(c.Request.Context(), tenantID, &req, userRole)
	if err != nil {
		if errors.Is(err, toolapp.ErrToolNameRequired) || errors.Is(err, toolapp.ErrToolTypeRequired) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		var policyErr *bizerrors.PolicyViolationError
		if errors.As(err, &policyErr) {
			c.JSON(policyErr.StatusCode(), response.APIResponse{
				Code:    -1,
				Message: "策略校验失败",
				Data:    policyErr.Violations,
			})
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建工具失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// @Summary      工具列表
// @Description  分页查询工具列表，支持按类型、状态、分类筛选
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"           default(1)
// @Param        page_size  query  int     false  "每页数量"       default(20)
// @Param        type       query  string  false  "工具类型"
// @Param        status     query  string  false  "工具状态"
// @Param        category   query  string  false  "工具分类"
// @Param        keyword    query  string  false  "关键词搜索"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools [get]
//
// ListTools 工具列表
// GET /api/v1/tools
func (h *ToolHandler) ListTools(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := toolapp.ListToolsFilter{
		Type:     c.Query("type"),
		Status:   c.Query("status"),
		Category: c.Query("category"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}

	tools, paginated, err := h.toolService.ListTools(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询工具列表失败")
		return
	}

	response.Paginated(c, tools, paginated.Total, paginated.Page, paginated.PageSize)
}

// @Summary      工具详情
// @Description  根据ID获取工具详细信息
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "工具ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools/{id} [get]
//
// GetTool 工具详情
// GET /api/v1/tools/:id
func (h *ToolHandler) GetTool(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.toolService.GetTool(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, toolapp.ErrToolNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "工具不存在")
			return
		}
		if errors.Is(err, toolapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该工具")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询工具失败")
		return
	}

	response.Success(c, result)
}

// @Summary      更新工具
// @Description  根据ID更新工具信息
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        id       path  string                       true  "工具ID"
// @Param        request  body  toolapp.UpdateToolRequest    true  "更新工具请求参数"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools/{id} [put]
//
// UpdateTool 更新工具
// PUT /api/v1/tools/:id
func (h *ToolHandler) UpdateTool(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")
	userRole := getUserRole(c)

	var req toolapp.UpdateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.toolService.UpdateTool(c.Request.Context(), tenantID, id, &req, userRole)
	if err != nil {
		if errors.Is(err, toolapp.ErrToolNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "工具不存在")
			return
		}
		if errors.Is(err, toolapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该工具")
			return
		}
		var policyErr *bizerrors.PolicyViolationError
		if errors.As(err, &policyErr) {
			c.JSON(policyErr.StatusCode(), response.APIResponse{
				Code:    -1,
				Message: "策略校验失败",
				Data:    policyErr.Violations,
			})
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新工具失败")
		return
	}

	response.Success(c, result)
}

// @Summary      删除工具
// @Description  根据ID删除工具
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "工具ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools/{id} [delete]
//
// DeleteTool 删除工具
// DELETE /api/v1/tools/:id
func (h *ToolHandler) DeleteTool(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.toolService.DeleteTool(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, toolapp.ErrToolNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "工具不存在")
			return
		}
		if errors.Is(err, toolapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该工具")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除工具失败")
		return
	}

	response.Success(c, nil)
}

// ==================== Excel 导入导出 ====================

// @Summary      导出工具
// @Description  导出工具列表为Excel文件
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        request  body  object  false  "导出列配置 {columns: []string}"
// @Success      200  {file}  file
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools/export [post]
//
// ExportTools 创建导出任务
// POST /api/v1/tools/export
func (h *ToolHandler) ExportTools(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	// 解析请求中的列参数
	var req struct {
		Columns []string `json:"columns"`
	}
	_ = c.ShouldBindJSON(&req)

	// 查询所有工具
	tools, _, err := h.toolService.ListTools(c.Request.Context(), tenantID, toolapp.ListToolsFilter{PageSize: 100000})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询工具列表失败")
		return
	}

	// 生成 Excel
	buf, err := h.excelService.ExportTools(c.Request.Context(), tools, req.Columns)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "导出失败")
		return
	}

	// 直接返回文件
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=tools_export.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// @Summary      导入工具
// @Description  通过Excel文件批量导入工具
// @Tags         工具管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Excel文件"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools/import [post]
//
// ImportTools 导入 Excel
// POST /api/v1/tools/import
func (h *ToolHandler) ImportTools(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "请上传文件")
		return
	}
	defer file.Close()

	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "读取文件失败")
		return
	}

	results, err := h.excelService.ImportTools(c.Request.Context(), data, tenantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "导入失败: "+err.Error())
		return
	}

	response.Success(c, results)
}

// @Summary      查询导出任务状态
// @Description  查询工具导出任务的状态
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        task_id  path  string  true  "任务ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /tools/export/{task_id} [get]
//
// GetExportStatus 查询导出任务状态（预留接口）
// GET /api/v1/tools/export/:task_id
func (h *ToolHandler) GetExportStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	// 目前导出是同步的，直接返回已完成状态
	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  "completed",
	})
}

// ==================== 连接器 CRUD ====================

// @Summary      创建连接器
// @Description  创建新的连接器
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        request  body  toolapp.CreateConnectorRequest  true  "创建连接器请求参数"
// @Success      201  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /connectors [post]
//
// CreateConnector 创建连接器
// POST /api/v1/connectors
func (h *ToolHandler) CreateConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req toolapp.CreateConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.toolService.CreateConnector(c.Request.Context(), tenantID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建连接器失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// @Summary      连接器列表
// @Description  分页查询连接器列表，支持按类型、状态筛选
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Param        type       query  string  false  "连接器类型"
// @Param        status     query  string  false  "连接器状态"
// @Param        keyword    query  string  false  "关键词搜索"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /connectors [get]
//
// ListConnectors 连接器列表
// GET /api/v1/connectors
func (h *ToolHandler) ListConnectors(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := toolapp.ListConnectorsFilter{
		Type:     c.Query("type"),
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}

	connectors, paginated, err := h.toolService.ListConnectors(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询连接器列表失败")
		return
	}

	response.Paginated(c, connectors, paginated.Total, paginated.Page, paginated.PageSize)
}

// @Summary      连接器详情
// @Description  根据ID获取连接器详细信息
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "连接器ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /connectors/{id} [get]
//
// GetConnector 连接器详情
// GET /api/v1/connectors/:id
func (h *ToolHandler) GetConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.toolService.GetConnector(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, toolapp.ErrConnectorNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "连接器不存在")
			return
		}
		if errors.Is(err, toolapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该连接器")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询连接器失败")
		return
	}

	response.Success(c, result)
}

// @Summary      更新连接器
// @Description  根据ID更新连接器信息
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        id       path  string                          true  "连接器ID"
// @Param        request  body  toolapp.UpdateConnectorRequest  true  "更新连接器请求参数"
// @Success      200  {object}  map[string]any
// @Failure      400  {object}  response.APIResponse
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /connectors/{id} [put]
//
// UpdateConnector 更新连接器
// PUT /api/v1/connectors/:id
func (h *ToolHandler) UpdateConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req toolapp.UpdateConnectorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.toolService.UpdateConnector(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, toolapp.ErrConnectorNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "连接器不存在")
			return
		}
		if errors.Is(err, toolapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该连接器")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新连接器失败")
		return
	}

	response.Success(c, result)
}

// @Summary      删除连接器
// @Description  根据ID删除连接器
// @Tags         工具管理
// @Accept       json
// @Produce      json
// @Param        id  path  string  true  "连接器ID"
// @Success      200  {object}  map[string]any
// @Failure      401  {object}  response.APIResponse
// @Failure      404  {object}  response.APIResponse
// @Failure      500  {object}  response.APIResponse
// @Security     BearerAuth
// @Router       /connectors/{id} [delete]
//
// DeleteConnector 删除连接器
// DELETE /api/v1/connectors/:id
func (h *ToolHandler) DeleteConnector(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.toolService.DeleteConnector(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, toolapp.ErrConnectorNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "连接器不存在")
			return
		}
		if errors.Is(err, toolapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该连接器")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除连接器失败")
		return
	}

	response.Success(c, nil)
}
