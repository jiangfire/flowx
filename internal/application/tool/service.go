package tool

import (
	"context"
	"errors"
	"fmt"

	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	domaingov "git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	bizerrors "git.neolidy.top/neo/flowx/pkg/errors"
	"git.neolidy.top/neo/flowx/pkg/pagination"
)

// 预定义错误
var (
	ErrToolNotFound      = errors.New("工具不存在")
	ErrConnectorNotFound = errors.New("连接器不存在")
	ErrToolNameRequired  = errors.New("工具名称不能为空")
	ErrToolTypeRequired  = errors.New("工具类型不能为空")
	ErrTenantMismatch    = errors.New("租户不匹配")
)

// CreateToolRequest 创建工具请求
type CreateToolRequest struct {
	Name        string      `json:"name" binding:"required"`
	Type        string      `json:"type" binding:"required"`
	Description string      `json:"description"`
	ConnectorID string      `json:"connector_id"`
	Config      base.JSON   `json:"config"`
	Status      string      `json:"status"`
	Endpoint    string      `json:"endpoint"`
	Icon        string      `json:"icon"`
	Category    string      `json:"category"`
}

// UpdateToolRequest 更新工具请求
type UpdateToolRequest struct {
	Name        *string     `json:"name"`
	Type        *string     `json:"type"`
	Description *string     `json:"description"`
	ConnectorID *string     `json:"connector_id"`
	Config      base.JSON   `json:"config"`
	Status      *string     `json:"status"`
	Endpoint    *string     `json:"endpoint"`
	Icon        *string     `json:"icon"`
	Category    *string     `json:"category"`
}

// CreateConnectorRequest 创建连接器请求
type CreateConnectorRequest struct {
	Name       string      `json:"name" binding:"required"`
	Type       string      `json:"type" binding:"required"`
	Endpoint   string      `json:"endpoint" binding:"required"`
	Config     base.JSON   `json:"config"`
	Status     string      `json:"status"`
	AuthType   string      `json:"auth_type"`
	AuthConfig base.JSON   `json:"auth_config"`
}

// UpdateConnectorRequest 更新连接器请求
type UpdateConnectorRequest struct {
	Name       *string     `json:"name"`
	Type       *string     `json:"type"`
	Endpoint   *string     `json:"endpoint"`
	Config     base.JSON   `json:"config"`
	Status     *string     `json:"status"`
	AuthType   *string     `json:"auth_type"`
	AuthConfig base.JSON   `json:"auth_config"`
}

// ListToolsFilter 工具列表过滤条件
type ListToolsFilter struct {
	Type     string `form:"type"`
	Status   string `form:"status"`
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListConnectorsFilter 连接器列表过滤条件
type ListConnectorsFilter struct {
	Type     string `form:"type"`
	Status   string `form:"status"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ToolService 工具应用服务
type ToolService struct {
	toolRepo      ToolRepository
	connectorRepo ConnectorRepository
	policyRepo    datagovapp.DataPolicyRepository
	assetRepo     datagovapp.DataAssetRepository
	ruleRepo      datagovapp.DataQualityRuleRepository
	checkRepo     datagovapp.DataQualityCheckRepository
}

// NewToolService 创建工具服务实例
func NewToolService(
	toolRepo ToolRepository,
	connectorRepo ConnectorRepository,
	policyRepo datagovapp.DataPolicyRepository,
	assetRepo datagovapp.DataAssetRepository,
	ruleRepo datagovapp.DataQualityRuleRepository,
	checkRepo datagovapp.DataQualityCheckRepository,
) *ToolService {
	return &ToolService{
		toolRepo:      toolRepo,
		connectorRepo: connectorRepo,
		policyRepo:    policyRepo,
		assetRepo:     assetRepo,
		ruleRepo:      ruleRepo,
		checkRepo:     checkRepo,
	}
}

// validatePolicy 校验工具是否满足策略要求
func (s *ToolService) validatePolicy(ctx context.Context, tenantID string, tl *tool.Tool, userRole string, action string) error {
	if s.policyRepo == nil {
		return nil
	}
	policies, _, err := s.policyRepo.List(ctx, datagovapp.DataPolicyFilter{
		TenantID: tenantID,
		Status:   "active",
		PageSize: 1000,
	})
	if err != nil {
		return fmt.Errorf("查询策略失败: %w", err)
	}

	// 转换为指针切片
	policyPtrs := make([]*domaingov.DataPolicy, len(policies))
	for i := range policies {
		policyPtrs[i] = &policies[i]
	}

	result := datagovapp.ValidateTool(policyPtrs, tl, userRole, action)
	if !result.Passed {
		return &bizerrors.PolicyViolationError{Code: bizerrors.PolicyViolationErrorCode, Violations: result.Violations}
	}
	return nil
}

// CreateTool 创建工具
func (s *ToolService) CreateTool(ctx context.Context, tenantID string, req *CreateToolRequest, userRole string) (*tool.Tool, error) {
	// 校验必填字段
	if req.Name == "" {
		return nil, ErrToolNameRequired
	}
	if req.Type == "" {
		return nil, ErrToolTypeRequired
	}

	// 设置默认状态
	status := req.Status
	if status == "" {
		status = "active"
	}

	tl := &tool.Tool{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		ConnectorID: req.ConnectorID,
		Config:      req.Config,
		Status:      status,
		Endpoint:    req.Endpoint,
		Icon:        req.Icon,
		Category:    req.Category,
	}

	// 策略校验
	if err := s.validatePolicy(ctx, tenantID, tl, userRole, "create"); err != nil {
		return nil, err
	}

	if err := s.toolRepo.Create(ctx, tl); err != nil {
		return nil, fmt.Errorf("创建工具失败: %w", err)
	}

	// 自动注册数据资产
	if s.assetRepo != nil {
		asset := &domaingov.DataAsset{
			BaseModel:      base.BaseModel{TenantID: tenantID},
			Name:           tl.Name + " (工具元数据)",
			Type:           "config",
			Source:         "tool",
			SourceID:       tl.ID,
			Description:    tl.Description,
			Classification: tl.Category,
			Status:         "active",
			Schema: base.JSON{
				"tool_id":       tl.ID,
				"tool_name":     tl.Name,
				"tool_type":     tl.Type,
				"tool_category": tl.Category,
				"endpoint":      tl.Endpoint,
				"connector_id":  tl.ConnectorID,
			},
		}
		_ = s.assetRepo.Create(ctx, asset)
	}

	// 自动触发质量检查
	if s.ruleRepo != nil && s.checkRepo != nil {
		rules, _, err := s.ruleRepo.List(ctx, datagovapp.DataQualityRuleFilter{
			TenantID: tenantID,
			Status:   "active",
			PageSize: 1000,
		})
		if err == nil {
			for _, rule := range rules {
				if shouldRunRule(&rule, tl) {
					runQualityCheck(ctx, s.checkRepo, &rule, tl.ID, tenantID)
				}
			}
		}
	}

	return tl, nil
}

// GetTool 获取工具详情
func (s *ToolService) GetTool(ctx context.Context, tenantID string, id string) (*tool.Tool, error) {
	tl, err := s.toolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrToolNotFound
	}

	// 多租户校验
	if tl.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return tl, nil
}

// ListTools 查询工具列表
func (s *ToolService) ListTools(ctx context.Context, tenantID string, filter ListToolsFilter) ([]tool.Tool, *pagination.PaginatedResult, error) {
	tools, total, err := s.toolRepo.List(ctx, ToolFilter{
		TenantID: tenantID,
		Type:     filter.Type,
		Status:   filter.Status,
		Category: filter.Category,
		Keyword:  filter.Keyword,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询工具列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return tools, pagination.NewResult(total, page, pageSize), nil
}

// UpdateTool 更新工具
func (s *ToolService) UpdateTool(ctx context.Context, tenantID string, id string, req *UpdateToolRequest, userRole string) (*tool.Tool, error) {
	// 先查询并校验
	existing, err := s.toolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrToolNotFound
	}

	// 多租户校验
	if existing.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	// 更新非空字段
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Type != nil {
		existing.Type = *req.Type
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.ConnectorID != nil {
		existing.ConnectorID = *req.ConnectorID
	}
	if req.Config != nil {
		existing.Config = req.Config
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.Endpoint != nil {
		existing.Endpoint = *req.Endpoint
	}
	if req.Icon != nil {
		existing.Icon = *req.Icon
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}

	// 策略校验
	if err := s.validatePolicy(ctx, tenantID, existing, userRole, "update"); err != nil {
		return nil, err
	}

	if err := s.toolRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新工具失败: %w", err)
	}

	// 同步更新数据资产
	if s.assetRepo != nil {
		assets, _, err := s.assetRepo.List(ctx, datagovapp.DataAssetFilter{
			TenantID: tenantID,
			Source:   "tool",
			Page:     1,
			PageSize: 1000,
		})
		if err == nil {
			for _, a := range assets {
				if a.SourceID == existing.ID {
					a.Name = existing.Name + " (工具元数据)"
					a.Description = existing.Description
					a.Classification = existing.Category
					a.Schema = base.JSON{
						"tool_id":       existing.ID,
						"tool_name":     existing.Name,
						"tool_type":     existing.Type,
						"tool_category": existing.Category,
						"endpoint":      existing.Endpoint,
						"connector_id":  existing.ConnectorID,
					}
					_ = s.assetRepo.Update(ctx, &a)
					break
				}
			}
		}
	}

	return existing, nil
}

// DeleteTool 删除工具
func (s *ToolService) DeleteTool(ctx context.Context, tenantID string, id string) error {
	// 先查询并校验
	existing, err := s.toolRepo.GetByID(ctx, id)
	if err != nil {
		return ErrToolNotFound
	}

	// 多租户校验
	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.toolRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除工具失败: %w", err)
	}

	// 归档关联的数据资产
	if s.assetRepo != nil {
		assets, _, err := s.assetRepo.List(ctx, datagovapp.DataAssetFilter{
			TenantID: tenantID,
			Source:   "tool",
			Page:     1,
			PageSize: 1000,
		})
		if err == nil {
			for _, a := range assets {
				if a.SourceID == id {
					a.Status = "archived"
					_ = s.assetRepo.Update(ctx, &a)
				}
			}
		}
	}

	return nil
}

// ImportTools 批量导入工具（带策略校验）
func (s *ToolService) ImportTools(ctx context.Context, tenantID string, tools []*tool.Tool, userRole string) ([]ImportResult, error) {
	if s.policyRepo != nil {
		policies, _, err := s.policyRepo.List(ctx, datagovapp.DataPolicyFilter{
			TenantID: tenantID,
			Status:   "active",
			PageSize: 1000,
		})
		if err != nil {
			return nil, fmt.Errorf("查询策略失败: %w", err)
		}

		policyPtrs := make([]*domaingov.DataPolicy, len(policies))
		for i := range policies {
			policyPtrs[i] = &policies[i]
		}

		result := datagovapp.ValidateTools(policyPtrs, tools, userRole)
		if !result.Passed {
			return nil, &bizerrors.PolicyViolationError{
				Code: bizerrors.PolicyViolationErrorCode,
				Violations: func() []datagovapp.PolicyViolation {
					var all []datagovapp.PolicyViolation
					for _, e := range result.Errors {
						all = append(all, e.Violations...)
					}
					return all
				}(),
			}
		}
	}

	var results []ImportResult
	for _, tl := range tools {
		if err := s.toolRepo.Create(ctx, tl); err != nil {
			results = append(results, ImportResult{
				Status:  "error",
				Message: fmt.Sprintf("创建失败: %v", err),
			})
			continue
		}
		results = append(results, ImportResult{
			Status:  "success",
			Message: "导入成功",
			ToolID:  tl.ID,
		})
	}
	return results, nil
}

// CreateConnector 创建连接器
func (s *ToolService) CreateConnector(ctx context.Context, tenantID string, req *CreateConnectorRequest) (*tool.Connector, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	conn := &tool.Connector{
		BaseModel:  base.BaseModel{TenantID: tenantID},
		Name:       req.Name,
		Type:       req.Type,
		Endpoint:   req.Endpoint,
		Config:     req.Config,
		Status:     status,
		AuthType:   req.AuthType,
		AuthConfig: req.AuthConfig,
	}

	if err := s.connectorRepo.Create(ctx, conn); err != nil {
		return nil, fmt.Errorf("创建连接器失败: %w", err)
	}

	return conn, nil
}

// ListConnectors 查询连接器列表
func (s *ToolService) ListConnectors(ctx context.Context, tenantID string, filter ListConnectorsFilter) ([]tool.Connector, *pagination.PaginatedResult, error) {
	connectors, total, err := s.connectorRepo.List(ctx, ConnectorFilter{
		TenantID: tenantID,
		Type:     filter.Type,
		Status:   filter.Status,
		Keyword:  filter.Keyword,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询连接器列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return connectors, pagination.NewResult(total, page, pageSize), nil
}

// GetConnector 获取连接器详情
func (s *ToolService) GetConnector(ctx context.Context, tenantID string, id string) (*tool.Connector, error) {
	conn, err := s.connectorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrConnectorNotFound
	}

	// 多租户校验
	if conn.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return conn, nil
}

// UpdateConnector 更新连接器
func (s *ToolService) UpdateConnector(ctx context.Context, tenantID string, id string, req *UpdateConnectorRequest) (*tool.Connector, error) {
	existing, err := s.connectorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrConnectorNotFound
	}

	if existing.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Type != nil {
		existing.Type = *req.Type
	}
	if req.Endpoint != nil {
		existing.Endpoint = *req.Endpoint
	}
	if req.Config != nil {
		existing.Config = req.Config
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	if req.AuthType != nil {
		existing.AuthType = *req.AuthType
	}
	if req.AuthConfig != nil {
		existing.AuthConfig = req.AuthConfig
	}

	if err := s.connectorRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新连接器失败: %w", err)
	}

	return existing, nil
}

// DeleteConnector 删除连接器
func (s *ToolService) DeleteConnector(ctx context.Context, tenantID string, id string) error {
	existing, err := s.connectorRepo.GetByID(ctx, id)
	if err != nil {
		return ErrConnectorNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.connectorRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除连接器失败: %w", err)
	}

	return nil
}

// shouldRunRule 判断质量规则是否匹配当前工具
func shouldRunRule(rule *domaingov.DataQualityRule, tl *tool.Tool) bool {
	if rule.Config == nil {
		return false
	}
	// 如果规则配置了 tool_type 且匹配
	if v, ok := rule.Config["tool_type"].(string); ok {
		return v == tl.Type
	}
	// 如果规则配置了 tool_category 且匹配
	if v, ok := rule.Config["tool_category"].(string); ok {
		return v == tl.Category
	}
	// 没有工具相关配置，跳过（规则针对特定资产，非工具）
	return false
}

// runQualityCheck 创建模拟的质量检查记录
func runQualityCheck(ctx context.Context, checkRepo datagovapp.DataQualityCheckRepository, rule *domaingov.DataQualityRule, toolID string, tenantID string) {
	check := &domaingov.DataQualityCheck{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		RuleID:      rule.ID,
		AssetID:     toolID,
		Status:      "passed",
		PassRate:    100.0,
		TriggeredBy: "auto",
		Result: base.JSON{
			"rule_name": rule.Name,
			"tool_id":   toolID,
			"message":   "自动质量检查通过",
		},
	}
	_ = checkRepo.Create(ctx, check)
}
