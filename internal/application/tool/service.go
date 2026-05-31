package tool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	datagovapp "github.com/jiangfire/flowx/internal/application/datagov"
	"github.com/jiangfire/flowx/internal/domain/base"
	domaingov "github.com/jiangfire/flowx/internal/domain/datagov"
	"github.com/jiangfire/flowx/internal/domain/tool"
	bizerrors "github.com/jiangfire/flowx/pkg/errors"
	"github.com/jiangfire/flowx/pkg/pagination"
	"github.com/jiangfire/flowx/pkg/transaction"
	"gorm.io/gorm"
)

// 预定义错误
var (
	ErrToolNotFound      = errors.New("工具不存在")
	ErrConnectorNotFound = errors.New("连接器不存在")
	ErrToolNameRequired  = errors.New("工具名称不能为空")
	ErrToolTypeRequired  = errors.New("工具类型不能为空")
)

// CreateToolRequest 创建工具请求
type CreateToolRequest struct {
	Name        string    `json:"name" binding:"required"`
	Type        string    `json:"type" binding:"required"`
	Description string    `json:"description"`
	ConnectorID string    `json:"connector_id"`
	Config      base.JSON `json:"config"`
	Status      string    `json:"status"`
	Endpoint    string    `json:"endpoint"`
	Icon        string    `json:"icon"`
	Category    string    `json:"category"`
}

// UpdateToolRequest 更新工具请求
type UpdateToolRequest struct {
	Name        *string   `json:"name"`
	Type        *string   `json:"type"`
	Description *string   `json:"description"`
	ConnectorID *string   `json:"connector_id"`
	Config      base.JSON `json:"config"`
	Status      *string   `json:"status"`
	Endpoint    *string   `json:"endpoint"`
	Icon        *string   `json:"icon"`
	Category    *string   `json:"category"`
}

// CreateConnectorRequest 创建连接器请求
type CreateConnectorRequest struct {
	Name       string    `json:"name" binding:"required"`
	Type       string    `json:"type" binding:"required"`
	Endpoint   string    `json:"endpoint" binding:"required"`
	Config     base.JSON `json:"config"`
	Status     string    `json:"status"`
	AuthType   string    `json:"auth_type"`
	AuthConfig base.JSON `json:"auth_config"`
}

// UpdateConnectorRequest 更新连接器请求
type UpdateConnectorRequest struct {
	Name       *string   `json:"name"`
	Type       *string   `json:"type"`
	Endpoint   *string   `json:"endpoint"`
	Config     base.JSON `json:"config"`
	Status     *string   `json:"status"`
	AuthType   *string   `json:"auth_type"`
	AuthConfig base.JSON `json:"auth_config"`
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
	toolRepo        ToolRepository
	connectorRepo   ConnectorRepository
	policyRepo      datagovapp.DataPolicyRepository
	assetRepo       datagovapp.DataAssetRepository
	ruleRepo        datagovapp.DataQualityRuleRepository
	checkRepo       datagovapp.DataQualityCheckRepository
	qualityExecutor *datagovapp.QualityExecutor
	db              *gorm.DB
}

// NewToolService 创建工具服务实例
func NewToolService(
	toolRepo ToolRepository,
	connectorRepo ConnectorRepository,
	policyRepo datagovapp.DataPolicyRepository,
	assetRepo datagovapp.DataAssetRepository,
	ruleRepo datagovapp.DataQualityRuleRepository,
	checkRepo datagovapp.DataQualityCheckRepository,
	db *gorm.DB,
) *ToolService {
	return &ToolService{
		toolRepo:        toolRepo,
		connectorRepo:   connectorRepo,
		policyRepo:      policyRepo,
		assetRepo:       assetRepo,
		ruleRepo:        ruleRepo,
		checkRepo:       checkRepo,
		qualityExecutor: datagovapp.NewQualityExecutor(db),
		db:              db,
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
	var asset *domaingov.DataAsset
	if s.assetRepo != nil {
		asset = &domaingov.DataAsset{
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
		if err := s.assetRepo.Create(ctx, asset); err != nil {
			slog.Warn("自动注册数据资产失败", "error", err, "tool_id", tl.ID)
		}
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
					runQualityCheck(ctx, s.checkRepo, s.qualityExecutor, &rule, asset, tenantID)
				}
			}
		}
	}

	return tl, nil
}

// GetTool 获取工具详情
func (s *ToolService) GetTool(ctx context.Context, tenantID string, id string) (*tool.Tool, error) {
	tl, err := s.toolRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, ErrToolNotFound
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
	existing, err := s.toolRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, ErrToolNotFound
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
	if _, err := s.toolRepo.GetByID(ctx, tenantID, id); err != nil {
		return err
	}
	if err := s.toolRepo.Delete(ctx, tenantID, id); err != nil {
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
	err := transaction.WithTransaction(ctx, s.db, func(txCtx context.Context) error {
		for _, tl := range tools {
			tl.TenantID = tenantID
			if err := s.toolRepo.Create(txCtx, tl); err != nil {
				return fmt.Errorf("创建工具 '%s' 失败: %w", tl.Name, err)
			}

			// 自动注册数据资产（与手工创建保持一致）
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
				if err := s.assetRepo.Create(txCtx, asset); err != nil {
					return fmt.Errorf("注册工具 '%s' 的数据资产失败: %w", tl.Name, err)
				}
			}

			results = append(results, ImportResult{
				Status:  "success",
				Message: "导入成功",
				ToolID:  tl.ID,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("导入工具失败: %w", err)
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
	conn, err := s.connectorRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, ErrConnectorNotFound
	}

	return conn, nil
}

// UpdateConnector 更新连接器
func (s *ToolService) UpdateConnector(ctx context.Context, tenantID string, id string, req *UpdateConnectorRequest) (*tool.Connector, error) {
	existing, err := s.connectorRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, ErrConnectorNotFound
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
	if _, err := s.connectorRepo.GetByID(ctx, tenantID, id); err != nil {
		return err
	}
	if err := s.connectorRepo.Delete(ctx, tenantID, id); err != nil {
		return fmt.Errorf("删除连接器失败: %w", err)
	}

	return nil
}

// shouldRunRule 判断质量规则是否匹配当前工具
func shouldRunRule(rule *domaingov.DataQualityRule, tl *tool.Tool) bool {
	if rule.Config == nil {
		return false
	}
	matched := false
	hasCondition := false
	if v, ok := rule.Config["tool_type"].(string); ok {
		hasCondition = true
		if v == tl.Type {
			matched = true
		}
	}
	if v, ok := rule.Config["tool_category"].(string); ok {
		hasCondition = true
		if v == tl.Category {
			matched = true
		}
	}
	if !hasCondition {
		return false
	}
	return matched
}

// runQualityCheck 执行质量检查并保存结果
func runQualityCheck(ctx context.Context, checkRepo datagovapp.DataQualityCheckRepository, executor *datagovapp.QualityExecutor, rule *domaingov.DataQualityRule, asset *domaingov.DataAsset, tenantID string) {
	assetID := ""
	if asset != nil {
		assetID = asset.ID
	}

	check := &domaingov.DataQualityCheck{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		RuleID:      rule.ID,
		AssetID:     assetID,
		Status:      "running",
		TotalCount:  1,
		TriggeredBy: "auto",
	}

	if err := checkRepo.Create(ctx, check); err != nil {
		return
	}

	if asset == nil {
		check.Status = "error"
		check.ErrorMsg = "关联的数据资产不存在"
	} else if executor != nil {
		result, err := executor.Execute(ctx, rule, asset)
		if err != nil {
			check.Status = "error"
			check.ErrorMsg = err.Error()
		} else {
			check.TotalCount = result.TotalRecords
			check.FailCount = result.FailedCount
			check.PassRate = result.PassRate
			check.Result = base.JSON{
				"rule_name":    rule.Name,
				"asset_id":     assetID,
				"message":      result.Message,
				"fail_details": result.FailDetails,
			}
			if result.FailedCount > 0 {
				check.Status = "failed"
			} else {
				check.Status = "passed"
			}
		}
	}

	if err := checkRepo.Update(ctx, check); err != nil {
		slog.Error("更新质量检查记录失败", "error", err, "check_id", check.ID, "rule_id", rule.ID)
	}
}
