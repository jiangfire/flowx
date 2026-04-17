package datagov

import (
	"context"
	"errors"
	"fmt"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/pkg/pagination"
)

// 预定义错误
var (
	ErrPolicyNotFound       = errors.New("数据策略不存在")
	ErrAssetNotFound        = errors.New("数据资产不存在")
	ErrQualityRuleNotFound  = errors.New("数据质量规则不存在")
	ErrQualityCheckNotFound = errors.New("数据质量检查不存在")
	ErrTenantMismatch       = errors.New("租户不匹配")
	ErrPolicyNameRequired   = errors.New("策略名称不能为空")
	ErrPolicyTypeRequired   = errors.New("策略类型不能为空")
	ErrAssetNameRequired    = errors.New("资产名称不能为空")
	ErrAssetTypeRequired    = errors.New("资产类型不能为空")
	ErrRuleNameRequired     = errors.New("规则名称不能为空")
	ErrRuleTypeRequired     = errors.New("规则类型不能为空")
)

// ==================== 请求类型 ====================

// CreatePolicyRequest 创建数据策略请求
type CreatePolicyRequest struct {
	Name        string     `json:"name" binding:"required"`
	Type        string     `json:"type" binding:"required"`
	Description string     `json:"description"`
	Scope       string     `json:"scope"`
	ScopeValue  string     `json:"scope_value"`
	Rules       base.JSON  `json:"rules"`
	Priority    int        `json:"priority"`
	Status      string     `json:"status"`
}

// UpdatePolicyRequest 更新数据策略请求
type UpdatePolicyRequest struct {
	Name        *string     `json:"name"`
	Type        *string     `json:"type"`
	Description *string     `json:"description"`
	Scope       *string     `json:"scope"`
	ScopeValue  *string     `json:"scope_value"`
	Rules       *base.JSON  `json:"rules"`
	Priority    *int        `json:"priority"`
	Status      *string     `json:"status"`
}

// ListPoliciesFilter 数据策略列表过滤条件
type ListPoliciesFilter struct {
	Type     string `form:"type"`
	Status   string `form:"status"`
	Scope    string `form:"scope"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// CreateAssetRequest 创建数据资产请求
type CreateAssetRequest struct {
	Name           string     `json:"name" binding:"required"`
	Type           string     `json:"type" binding:"required"`
	Source         string     `json:"source"`
	SourceID       string     `json:"source_id"`
	Description    string     `json:"description"`
	Format         string     `json:"format"`
	Schema         base.JSON  `json:"schema"`
	Size           int64      `json:"size"`
	RecordCount    int64      `json:"record_count"`
	Location       string     `json:"location"`
	Tags           base.JSON  `json:"tags"`
	Classification string     `json:"classification"`
	OwnerID        string     `json:"owner_id"`
	Status         string     `json:"status"`
}

// UpdateAssetRequest 更新数据资产请求
type UpdateAssetRequest struct {
	Name           *string     `json:"name"`
	Type           *string     `json:"type"`
	Source         *string     `json:"source"`
	SourceID       *string     `json:"source_id"`
	Description    *string     `json:"description"`
	Format         *string     `json:"format"`
	Schema         *base.JSON  `json:"schema"`
	Size           *int64      `json:"size"`
	RecordCount    *int64      `json:"record_count"`
	Location       *string     `json:"location"`
	Tags           *base.JSON  `json:"tags"`
	Classification *string     `json:"classification"`
	OwnerID        *string     `json:"owner_id"`
	Status         *string     `json:"status"`
}

// ListAssetsFilter 数据资产列表过滤条件
type ListAssetsFilter struct {
	Type           string `form:"type"`
	Status         string `form:"status"`
	Classification string `form:"classification"`
	Source         string `form:"source"`
	Keyword        string `form:"keyword"`
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
}

// CreateRuleRequest 创建数据质量规则请求
type CreateRuleRequest struct {
	Name        string     `json:"name" binding:"required"`
	Type        string     `json:"type" binding:"required"`
	TargetAsset string     `json:"target_asset"`
	TargetField string     `json:"target_field"`
	Description string     `json:"description"`
	Config      base.JSON  `json:"config"`
	Severity    string     `json:"severity"`
	Status      string     `json:"status"`
}

// UpdateRuleRequest 更新数据质量规则请求
type UpdateRuleRequest struct {
	Name        *string     `json:"name"`
	Type        *string     `json:"type"`
	TargetAsset *string     `json:"target_asset"`
	TargetField *string     `json:"target_field"`
	Description *string     `json:"description"`
	Config      *base.JSON  `json:"config"`
	Severity    *string     `json:"severity"`
	Status      *string     `json:"status"`
}

// ListRulesFilter 数据质量规则列表过滤条件
type ListRulesFilter struct {
	Type        string `form:"type"`
	Status      string `form:"status"`
	Severity    string `form:"severity"`
	TargetAsset string `form:"target_asset"`
	Keyword     string `form:"keyword"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

// ListChecksFilter 数据质量检查列表过滤条件
type ListChecksFilter struct {
	RuleID      string `form:"rule_id"`
	AssetID     string `form:"asset_id"`
	Status      string `form:"status"`
	TriggeredBy string `form:"triggered_by"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

// RunQualityCheckRequest 执行质量检查请求
type RunQualityCheckRequest struct {
	RuleID  string `json:"rule_id" binding:"required"`
	AssetID string `json:"asset_id" binding:"required"`
}

// ==================== DataGovService ====================

// DataGovService 数据治理应用服务
type DataGovService struct {
	policyRepo DataPolicyRepository
	assetRepo  DataAssetRepository
	ruleRepo   DataQualityRuleRepository
	checkRepo  DataQualityCheckRepository
}

// NewDataGovService 创建数据治理服务实例
func NewDataGovService(
	policyRepo DataPolicyRepository,
	assetRepo DataAssetRepository,
	ruleRepo DataQualityRuleRepository,
	checkRepo DataQualityCheckRepository,
) *DataGovService {
	return &DataGovService{
		policyRepo: policyRepo,
		assetRepo:  assetRepo,
		ruleRepo:   ruleRepo,
		checkRepo:  checkRepo,
	}
}

// ==================== 数据策略 CRUD ====================

// CreatePolicy 创建数据策略
func (s *DataGovService) CreatePolicy(ctx context.Context, tenantID string, req *CreatePolicyRequest) (*datagov.DataPolicy, error) {
	if req.Name == "" {
		return nil, ErrPolicyNameRequired
	}
	if req.Type == "" {
		return nil, ErrPolicyTypeRequired
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	policy := &datagov.DataPolicy{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Scope:       req.Scope,
		ScopeValue:  req.ScopeValue,
		Rules:       req.Rules,
		Priority:    req.Priority,
		Status:      status,
		Version:     1,
	}

	if err := s.policyRepo.Create(ctx, policy); err != nil {
		return nil, fmt.Errorf("创建数据策略失败: %w", err)
	}

	return policy, nil
}

// GetPolicy 获取数据策略详情
func (s *DataGovService) GetPolicy(ctx context.Context, tenantID string, id string) (*datagov.DataPolicy, error) {
	policy, err := s.policyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPolicyNotFound
	}

	if policy.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return policy, nil
}

// ListPolicies 查询数据策略列表
func (s *DataGovService) ListPolicies(ctx context.Context, tenantID string, filter ListPoliciesFilter) ([]datagov.DataPolicy, *pagination.PaginatedResult, error) {
	policies, total, err := s.policyRepo.List(ctx, DataPolicyFilter{
		TenantID: tenantID,
		Type:     filter.Type,
		Status:   filter.Status,
		Scope:    filter.Scope,
		Keyword:  filter.Keyword,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询数据策略列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return policies, pagination.NewResult(total, page, pageSize), nil
}

// UpdatePolicy 更新数据策略
func (s *DataGovService) UpdatePolicy(ctx context.Context, tenantID string, id string, req *UpdatePolicyRequest) (*datagov.DataPolicy, error) {
	existing, err := s.policyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPolicyNotFound
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
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Scope != nil {
		existing.Scope = *req.Scope
	}
	if req.ScopeValue != nil {
		existing.ScopeValue = *req.ScopeValue
	}
	if req.Rules != nil {
		existing.Rules = *req.Rules
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	existing.Version++

	if err := s.policyRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新数据策略失败: %w", err)
	}

	return existing, nil
}

// DeletePolicy 删除数据策略
func (s *DataGovService) DeletePolicy(ctx context.Context, tenantID string, id string) error {
	existing, err := s.policyRepo.GetByID(ctx, id)
	if err != nil {
		return ErrPolicyNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.policyRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除数据策略失败: %w", err)
	}

	return nil
}

// ==================== 数据资产 CRUD ====================

// CreateAsset 创建数据资产
func (s *DataGovService) CreateAsset(ctx context.Context, tenantID string, req *CreateAssetRequest) (*datagov.DataAsset, error) {
	if req.Name == "" {
		return nil, ErrAssetNameRequired
	}
	if req.Type == "" {
		return nil, ErrAssetTypeRequired
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	classification := req.Classification
	if classification == "" {
		classification = "internal"
	}

	asset := &datagov.DataAsset{
		BaseModel:      base.BaseModel{TenantID: tenantID},
		Name:           req.Name,
		Type:           req.Type,
		Source:         req.Source,
		SourceID:       req.SourceID,
		Description:    req.Description,
		Format:         req.Format,
		Schema:         req.Schema,
		Size:           req.Size,
		RecordCount:    req.RecordCount,
		Location:       req.Location,
		Tags:           req.Tags,
		Classification: classification,
		OwnerID:        req.OwnerID,
		Status:         status,
	}

	if err := s.assetRepo.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("创建数据资产失败: %w", err)
	}

	return asset, nil
}

// GetAsset 获取数据资产详情
func (s *DataGovService) GetAsset(ctx context.Context, tenantID string, id string) (*datagov.DataAsset, error) {
	asset, err := s.assetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrAssetNotFound
	}

	if asset.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return asset, nil
}

// ListAssets 查询数据资产列表
func (s *DataGovService) ListAssets(ctx context.Context, tenantID string, filter ListAssetsFilter) ([]datagov.DataAsset, *pagination.PaginatedResult, error) {
	assets, total, err := s.assetRepo.List(ctx, DataAssetFilter{
		TenantID:       tenantID,
		Type:           filter.Type,
		Status:         filter.Status,
		Classification: filter.Classification,
		Source:         filter.Source,
		Keyword:        filter.Keyword,
		Page:           filter.Page,
		PageSize:       filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询数据资产列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return assets, pagination.NewResult(total, page, pageSize), nil
}

// UpdateAsset 更新数据资产
func (s *DataGovService) UpdateAsset(ctx context.Context, tenantID string, id string, req *UpdateAssetRequest) (*datagov.DataAsset, error) {
	existing, err := s.assetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrAssetNotFound
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
	if req.Source != nil {
		existing.Source = *req.Source
	}
	if req.SourceID != nil {
		existing.SourceID = *req.SourceID
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Format != nil {
		existing.Format = *req.Format
	}
	if req.Schema != nil {
		existing.Schema = *req.Schema
	}
	if req.Size != nil {
		existing.Size = *req.Size
	}
	if req.RecordCount != nil {
		existing.RecordCount = *req.RecordCount
	}
	if req.Location != nil {
		existing.Location = *req.Location
	}
	if req.Tags != nil {
		existing.Tags = *req.Tags
	}
	if req.Classification != nil {
		existing.Classification = *req.Classification
	}
	if req.OwnerID != nil {
		existing.OwnerID = *req.OwnerID
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}

	if err := s.assetRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新数据资产失败: %w", err)
	}

	return existing, nil
}

// DeleteAsset 删除数据资产
func (s *DataGovService) DeleteAsset(ctx context.Context, tenantID string, id string) error {
	existing, err := s.assetRepo.GetByID(ctx, id)
	if err != nil {
		return ErrAssetNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.assetRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除数据资产失败: %w", err)
	}

	return nil
}

// ==================== 数据质量规则 CRUD ====================

// CreateRule 创建数据质量规则
func (s *DataGovService) CreateRule(ctx context.Context, tenantID string, req *CreateRuleRequest) (*datagov.DataQualityRule, error) {
	if req.Name == "" {
		return nil, ErrRuleNameRequired
	}
	if req.Type == "" {
		return nil, ErrRuleTypeRequired
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	severity := req.Severity
	if severity == "" {
		severity = "warning"
	}

	rule := &datagov.DataQualityRule{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        req.Name,
		Type:        req.Type,
		TargetAsset: req.TargetAsset,
		TargetField: req.TargetField,
		Description: req.Description,
		Config:      req.Config,
		Severity:    severity,
		Status:      status,
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("创建数据质量规则失败: %w", err)
	}

	return rule, nil
}

// GetRule 获取数据质量规则详情
func (s *DataGovService) GetRule(ctx context.Context, tenantID string, id string) (*datagov.DataQualityRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrQualityRuleNotFound
	}

	if rule.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return rule, nil
}

// ListRules 查询数据质量规则列表
func (s *DataGovService) ListRules(ctx context.Context, tenantID string, filter ListRulesFilter) ([]datagov.DataQualityRule, *pagination.PaginatedResult, error) {
	rules, total, err := s.ruleRepo.List(ctx, DataQualityRuleFilter{
		TenantID:    tenantID,
		Type:        filter.Type,
		Status:      filter.Status,
		Severity:    filter.Severity,
		TargetAsset: filter.TargetAsset,
		Keyword:     filter.Keyword,
		Page:        filter.Page,
		PageSize:    filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询数据质量规则列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return rules, pagination.NewResult(total, page, pageSize), nil
}

// UpdateRule 更新数据质量规则
func (s *DataGovService) UpdateRule(ctx context.Context, tenantID string, id string, req *UpdateRuleRequest) (*datagov.DataQualityRule, error) {
	existing, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrQualityRuleNotFound
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
	if req.TargetAsset != nil {
		existing.TargetAsset = *req.TargetAsset
	}
	if req.TargetField != nil {
		existing.TargetField = *req.TargetField
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Config != nil {
		existing.Config = *req.Config
	}
	if req.Severity != nil {
		existing.Severity = *req.Severity
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}

	if err := s.ruleRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新数据质量规则失败: %w", err)
	}

	return existing, nil
}

// DeleteRule 删除数据质量规则
func (s *DataGovService) DeleteRule(ctx context.Context, tenantID string, id string) error {
	existing, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return ErrQualityRuleNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.ruleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除数据质量规则失败: %w", err)
	}

	return nil
}

// ==================== 数据质量检查 ====================

// GetCheck 获取数据质量检查详情
func (s *DataGovService) GetCheck(ctx context.Context, tenantID string, id string) (*datagov.DataQualityCheck, error) {
	check, err := s.checkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrQualityCheckNotFound
	}

	if check.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	return check, nil
}

// ListChecks 查询数据质量检查列表
func (s *DataGovService) ListChecks(ctx context.Context, tenantID string, filter ListChecksFilter) ([]datagov.DataQualityCheck, *pagination.PaginatedResult, error) {
	checks, total, err := s.checkRepo.List(ctx, DataQualityCheckFilter{
		TenantID:    tenantID,
		RuleID:      filter.RuleID,
		AssetID:     filter.AssetID,
		Status:      filter.Status,
		TriggeredBy: filter.TriggeredBy,
		Page:        filter.Page,
		PageSize:    filter.PageSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("查询数据质量检查列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return checks, pagination.NewResult(total, page, pageSize), nil
}

// DeleteCheck 删除数据质量检查
func (s *DataGovService) DeleteCheck(ctx context.Context, tenantID string, id string) error {
	existing, err := s.checkRepo.GetByID(ctx, id)
	if err != nil {
		return ErrQualityCheckNotFound
	}

	if existing.TenantID != tenantID {
		return ErrTenantMismatch
	}

	if err := s.checkRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除数据质量检查失败: %w", err)
	}

	return nil
}

// RunQualityCheck 执行数据质量检查
func (s *DataGovService) RunQualityCheck(ctx context.Context, tenantID string, req *RunQualityCheckRequest) (*datagov.DataQualityCheck, error) {
	// 校验规则存在
	rule, err := s.ruleRepo.GetByID(ctx, req.RuleID)
	if err != nil {
		return nil, ErrQualityRuleNotFound
	}
	if rule.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	// 校验资产存在
	asset, err := s.assetRepo.GetByID(ctx, req.AssetID)
	if err != nil {
		return nil, ErrAssetNotFound
	}
	if asset.TenantID != tenantID {
		return nil, ErrTenantMismatch
	}

	// 创建检查记录
	startTime := time.Now()
	check := &datagov.DataQualityCheck{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		RuleID:      req.RuleID,
		AssetID:     req.AssetID,
		Status:      "running",
		TotalCount:  asset.RecordCount,
		TriggeredBy: "manual",
	}

	if err := s.checkRepo.Create(ctx, check); err != nil {
		return nil, fmt.Errorf("创建质量检查记录失败: %w", err)
	}

	// 模拟质量检查执行（简化版）
	duration := time.Since(startTime).Milliseconds()
	check.Status = "passed"
	check.PassRate = 100.0
	check.Duration = duration
	check.Result = base.JSON{
		"rule_name":  rule.Name,
		"asset_name": asset.Name,
		"message":    "质量检查通过",
	}

	if err := s.checkRepo.Update(ctx, check); err != nil {
		return nil, fmt.Errorf("更新质量检查记录失败: %w", err)
	}

	return check, nil
}
