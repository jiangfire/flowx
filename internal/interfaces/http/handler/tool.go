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
