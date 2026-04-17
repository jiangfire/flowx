package handler

import (
	"errors"
	"net/http"
	"strconv"

	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"git.neolidy.top/neo/flowx/pkg/response"
	"github.com/gin-gonic/gin"
)

// DataGovHandler 数据治理 HTTP 处理器
type DataGovHandler struct {
	service      *datagovapp.DataGovService
	excelService *datagovapp.DataGovExcelService
}

// NewDataGovHandler 创建数据治理处理器实例
func NewDataGovHandler(service *datagovapp.DataGovService, excelService *datagovapp.DataGovExcelService) *DataGovHandler {
	return &DataGovHandler{
		service:      service,
		excelService: excelService,
	}
}

// ==================== 数据策略 CRUD ====================

// CreatePolicy 创建数据策略
// POST /api/v1/data-policies
func (h *DataGovHandler) CreatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req datagovapp.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.CreatePolicy(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrPolicyNameRequired) || errors.Is(err, datagovapp.ErrPolicyTypeRequired) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建数据策略失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// ListPolicies 数据策略列表
// GET /api/v1/data-policies
func (h *DataGovHandler) ListPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := datagovapp.ListPoliciesFilter{
		Type:     c.Query("type"),
		Status:   c.Query("status"),
		Scope:    c.Query("scope"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}

	policies, paginated, err := h.service.ListPolicies(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据策略列表失败")
		return
	}

	response.Paginated(c, policies, paginated.Total, paginated.Page, paginated.PageSize)
}

// GetPolicy 数据策略详情
// GET /api/v1/data-policies/:id
func (h *DataGovHandler) GetPolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.service.GetPolicy(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrPolicyNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据策略不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据策略")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据策略失败")
		return
	}

	response.Success(c, result)
}

// UpdatePolicy 更新数据策略
// PUT /api/v1/data-policies/:id
func (h *DataGovHandler) UpdatePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req datagovapp.UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.UpdatePolicy(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrPolicyNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据策略不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据策略")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新数据策略失败")
		return
	}

	response.Success(c, result)
}

// DeletePolicy 删除数据策略
// DELETE /api/v1/data-policies/:id
func (h *DataGovHandler) DeletePolicy(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.DeletePolicy(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrPolicyNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据策略不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据策略")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除数据策略失败")
		return
	}

	response.Success(c, nil)
}

// ==================== 数据资产 CRUD ====================

// CreateAsset 创建数据资产
// POST /api/v1/data-assets
func (h *DataGovHandler) CreateAsset(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req datagovapp.CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.CreateAsset(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrAssetNameRequired) || errors.Is(err, datagovapp.ErrAssetTypeRequired) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建数据资产失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// ListAssets 数据资产列表
// GET /api/v1/data-assets
func (h *DataGovHandler) ListAssets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := datagovapp.ListAssetsFilter{
		Type:           c.Query("type"),
		Status:         c.Query("status"),
		Classification: c.Query("classification"),
		Source:         c.Query("source"),
		Keyword:        c.Query("keyword"),
		Page:           page,
		PageSize:       pageSize,
	}

	assets, paginated, err := h.service.ListAssets(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据资产列表失败")
		return
	}

	response.Paginated(c, assets, paginated.Total, paginated.Page, paginated.PageSize)
}

// GetAsset 数据资产详情
// GET /api/v1/data-assets/:id
func (h *DataGovHandler) GetAsset(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.service.GetAsset(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrAssetNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据资产不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据资产")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据资产失败")
		return
	}

	response.Success(c, result)
}

// UpdateAsset 更新数据资产
// PUT /api/v1/data-assets/:id
func (h *DataGovHandler) UpdateAsset(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req datagovapp.UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.UpdateAsset(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrAssetNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据资产不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据资产")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新数据资产失败")
		return
	}

	response.Success(c, result)
}

// DeleteAsset 删除数据资产
// DELETE /api/v1/data-assets/:id
func (h *DataGovHandler) DeleteAsset(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.DeleteAsset(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrAssetNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据资产不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据资产")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除数据资产失败")
		return
	}

	response.Success(c, nil)
}

// ==================== 数据质量规则 CRUD ====================

// CreateRule 创建数据质量规则
// POST /api/v1/data-quality/rules
func (h *DataGovHandler) CreateRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req datagovapp.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.CreateRule(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrRuleNameRequired) || errors.Is(err, datagovapp.ErrRuleTypeRequired) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建数据质量规则失败")
		return
	}

	c.JSON(http.StatusCreated, response.APIResponse{
		Code:    0,
		Message: "success",
		Data:    result,
	})
}

// ListRules 数据质量规则列表
// GET /api/v1/data-quality/rules
func (h *DataGovHandler) ListRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := datagovapp.ListRulesFilter{
		Type:        c.Query("type"),
		Status:      c.Query("status"),
		Severity:    c.Query("severity"),
		TargetAsset: c.Query("target_asset"),
		Keyword:     c.Query("keyword"),
		Page:        page,
		PageSize:    pageSize,
	}

	rules, paginated, err := h.service.ListRules(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据质量规则列表失败")
		return
	}

	response.Paginated(c, rules, paginated.Total, paginated.Page, paginated.PageSize)
}

// GetRule 数据质量规则详情
// GET /api/v1/data-quality/rules/:id
func (h *DataGovHandler) GetRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.service.GetRule(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrQualityRuleNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据质量规则不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据质量规则")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据质量规则失败")
		return
	}

	response.Success(c, result)
}

// UpdateRule 更新数据质量规则
// PUT /api/v1/data-quality/rules/:id
func (h *DataGovHandler) UpdateRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	var req datagovapp.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.UpdateRule(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrQualityRuleNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据质量规则不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据质量规则")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新数据质量规则失败")
		return
	}

	response.Success(c, result)
}

// DeleteRule 删除数据质量规则
// DELETE /api/v1/data-quality/rules/:id
func (h *DataGovHandler) DeleteRule(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	err := h.service.DeleteRule(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrQualityRuleNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据质量规则不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据质量规则")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除数据质量规则失败")
		return
	}

	response.Success(c, nil)
}

// ==================== 数据质量检查 ====================

// ListChecks 数据质量检查列表
// GET /api/v1/data-quality/checks
func (h *DataGovHandler) ListChecks(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := datagovapp.ListChecksFilter{
		RuleID:      c.Query("rule_id"),
		AssetID:     c.Query("asset_id"),
		Status:      c.Query("status"),
		TriggeredBy: c.Query("triggered_by"),
		Page:        page,
		PageSize:    pageSize,
	}

	checks, paginated, err := h.service.ListChecks(c.Request.Context(), tenantID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据质量检查列表失败")
		return
	}

	response.Paginated(c, checks, paginated.Total, paginated.Page, paginated.PageSize)
}

// GetCheck 数据质量检查详情
// GET /api/v1/data-quality/checks/:id
func (h *DataGovHandler) GetCheck(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	id := c.Param("id")

	result, err := h.service.GetCheck(c.Request.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, datagovapp.ErrQualityCheckNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "数据质量检查不存在")
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权访问该数据质量检查")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据质量检查失败")
		return
	}

	response.Success(c, result)
}

// RunQualityCheck 执行数据质量检查
// POST /api/v1/data-quality/checks/run
func (h *DataGovHandler) RunQualityCheck(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req datagovapp.RunQualityCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "请求参数校验失败: "+err.Error())
		return
	}

	result, err := h.service.RunQualityCheck(c.Request.Context(), tenantID, &req)
	if err != nil {
		if errors.Is(err, datagovapp.ErrQualityRuleNotFound) || errors.Is(err, datagovapp.ErrAssetNotFound) {
			response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		if errors.Is(err, datagovapp.ErrTenantMismatch) {
			response.Error(c, http.StatusForbidden, "FORBIDDEN", "无权执行该操作")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "执行数据质量检查失败")
		return
	}

	response.Success(c, result)
}

// ==================== Excel 导入导出 ====================

// ExportPolicies 导出数据策略
// POST /api/v1/data-policies/export
func (h *DataGovHandler) ExportPolicies(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		Columns []string `json:"columns"`
	}
	_ = c.ShouldBindJSON(&req)

	policies, _, err := h.service.ListPolicies(c.Request.Context(), tenantID, datagovapp.ListPoliciesFilter{PageSize: 100000})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据策略列表失败")
		return
	}

	buf, err := h.excelService.ExportPolicies(c.Request.Context(), policies, req.Columns)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "导出失败")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=data_policies_export.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportPolicies 导入数据策略
// POST /api/v1/data-policies/import
func (h *DataGovHandler) ImportPolicies(c *gin.Context) {
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

	results, err := h.excelService.ImportPolicies(c.Request.Context(), data, tenantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "导入失败: "+err.Error())
		return
	}

	response.Success(c, results)
}

// ExportAssets 导出数据资产
// POST /api/v1/data-assets/export
func (h *DataGovHandler) ExportAssets(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		Columns []string `json:"columns"`
	}
	_ = c.ShouldBindJSON(&req)

	assets, _, err := h.service.ListAssets(c.Request.Context(), tenantID, datagovapp.ListAssetsFilter{PageSize: 100000})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据资产列表失败")
		return
	}

	buf, err := h.excelService.ExportAssets(c.Request.Context(), assets, req.Columns)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "导出失败")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=data_assets_export.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportAssets 导入数据资产
// POST /api/v1/data-assets/import
func (h *DataGovHandler) ImportAssets(c *gin.Context) {
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

	results, err := h.excelService.ImportAssets(c.Request.Context(), data, tenantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "导入失败: "+err.Error())
		return
	}

	response.Success(c, results)
}

// ExportRules 导出数据质量规则
// POST /api/v1/data-quality/rules/export
func (h *DataGovHandler) ExportRules(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req struct {
		Columns []string `json:"columns"`
	}
	_ = c.ShouldBindJSON(&req)

	rules, _, err := h.service.ListRules(c.Request.Context(), tenantID, datagovapp.ListRulesFilter{PageSize: 100000})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "查询数据质量规则列表失败")
		return
	}

	buf, err := h.excelService.ExportRules(c.Request.Context(), rules, req.Columns)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "导出失败")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=data_quality_rules_export.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ImportRules 导入数据质量规则
// POST /api/v1/data-quality/rules/import
func (h *DataGovHandler) ImportRules(c *gin.Context) {
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

	results, err := h.excelService.ImportRules(c.Request.Context(), data, tenantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "导入失败: "+err.Error())
		return
	}

	response.Success(c, results)
}
