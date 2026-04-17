package persistence

import (
	"context"
	"fmt"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"gorm.io/gorm"
)

// ==================== DataPolicyRepository ====================

// dataPolicyRepository 数据策略仓储实现
type dataPolicyRepository struct {
	db *gorm.DB
}

// NewDataPolicyRepository 创建数据策略仓储实例
func NewDataPolicyRepository(db *gorm.DB) datagovapp.DataPolicyRepository {
	return &dataPolicyRepository{db: db}
}

// Create 创建数据策略
func (r *dataPolicyRepository) Create(ctx context.Context, policy *datagov.DataPolicy) error {
	if policy.ID == "" {
		policy.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(policy).Error
}

// GetByID 根据 ID 查询数据策略
func (r *dataPolicyRepository) GetByID(ctx context.Context, id string) (*datagov.DataPolicy, error) {
	var policy datagov.DataPolicy
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&policy).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("数据策略不存在: %s", id)
		}
		return nil, fmt.Errorf("查询数据策略失败: %w", err)
	}
	return &policy, nil
}

// List 查询数据策略列表（支持过滤和分页）
func (r *dataPolicyRepository) List(ctx context.Context, filter datagovapp.DataPolicyFilter) ([]datagov.DataPolicy, int64, error) {
	var policies []datagov.DataPolicy
	var total int64

	query := r.db.WithContext(ctx).Model(&datagov.DataPolicy{}).Where("tenant_id = ?", filter.TenantID)

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Scope != "" {
		query = query.Where("scope = ?", filter.Scope)
	}
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计数据策略数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&policies).Error; err != nil {
		return nil, 0, fmt.Errorf("查询数据策略列表失败: %w", err)
	}

	return policies, total, nil
}

// Update 更新数据策略
func (r *dataPolicyRepository) Update(ctx context.Context, policy *datagov.DataPolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

// Delete 软删除数据策略
func (r *dataPolicyRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&datagov.DataPolicy{}, "id = ?", id).Error
}

// ==================== DataAssetRepository ====================

// dataAssetRepository 数据资产仓储实现
type dataAssetRepository struct {
	db *gorm.DB
}

// NewDataAssetRepository 创建数据资产仓储实例
func NewDataAssetRepository(db *gorm.DB) datagovapp.DataAssetRepository {
	return &dataAssetRepository{db: db}
}

// Create 创建数据资产
func (r *dataAssetRepository) Create(ctx context.Context, asset *datagov.DataAsset) error {
	if asset.ID == "" {
		asset.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(asset).Error
}

// GetByID 根据 ID 查询数据资产
func (r *dataAssetRepository) GetByID(ctx context.Context, id string) (*datagov.DataAsset, error) {
	var asset datagov.DataAsset
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&asset).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("数据资产不存在: %s", id)
		}
		return nil, fmt.Errorf("查询数据资产失败: %w", err)
	}
	return &asset, nil
}

// List 查询数据资产列表（支持过滤和分页）
func (r *dataAssetRepository) List(ctx context.Context, filter datagovapp.DataAssetFilter) ([]datagov.DataAsset, int64, error) {
	var assets []datagov.DataAsset
	var total int64

	query := r.db.WithContext(ctx).Model(&datagov.DataAsset{}).Where("tenant_id = ?", filter.TenantID)

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Classification != "" {
		query = query.Where("classification = ?", filter.Classification)
	}
	if filter.Source != "" {
		query = query.Where("source = ?", filter.Source)
	}
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计数据资产数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&assets).Error; err != nil {
		return nil, 0, fmt.Errorf("查询数据资产列表失败: %w", err)
	}

	return assets, total, nil
}

// Update 更新数据资产
func (r *dataAssetRepository) Update(ctx context.Context, asset *datagov.DataAsset) error {
	return r.db.WithContext(ctx).Save(asset).Error
}

// Delete 软删除数据资产
func (r *dataAssetRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&datagov.DataAsset{}, "id = ?", id).Error
}

// ==================== DataQualityRuleRepository ====================

// dataQualityRuleRepository 数据质量规则仓储实现
type dataQualityRuleRepository struct {
	db *gorm.DB
}

// NewDataQualityRuleRepository 创建数据质量规则仓储实例
func NewDataQualityRuleRepository(db *gorm.DB) datagovapp.DataQualityRuleRepository {
	return &dataQualityRuleRepository{db: db}
}

// Create 创建数据质量规则
func (r *dataQualityRuleRepository) Create(ctx context.Context, rule *datagov.DataQualityRule) error {
	if rule.ID == "" {
		rule.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(rule).Error
}

// GetByID 根据 ID 查询数据质量规则
func (r *dataQualityRuleRepository) GetByID(ctx context.Context, id string) (*datagov.DataQualityRule, error) {
	var rule datagov.DataQualityRule
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("数据质量规则不存在: %s", id)
		}
		return nil, fmt.Errorf("查询数据质量规则失败: %w", err)
	}
	return &rule, nil
}

// List 查询数据质量规则列表（支持过滤和分页）
func (r *dataQualityRuleRepository) List(ctx context.Context, filter datagovapp.DataQualityRuleFilter) ([]datagov.DataQualityRule, int64, error) {
	var rules []datagov.DataQualityRule
	var total int64

	query := r.db.WithContext(ctx).Model(&datagov.DataQualityRule{}).Where("tenant_id = ?", filter.TenantID)

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	if filter.TargetAsset != "" {
		query = query.Where("target_asset = ?", filter.TargetAsset)
	}
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计数据质量规则数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rules).Error; err != nil {
		return nil, 0, fmt.Errorf("查询数据质量规则列表失败: %w", err)
	}

	return rules, total, nil
}

// Update 更新数据质量规则
func (r *dataQualityRuleRepository) Update(ctx context.Context, rule *datagov.DataQualityRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete 软删除数据质量规则
func (r *dataQualityRuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&datagov.DataQualityRule{}, "id = ?", id).Error
}

// ==================== DataQualityCheckRepository ====================

// dataQualityCheckRepository 数据质量检查仓储实现
type dataQualityCheckRepository struct {
	db *gorm.DB
}

// NewDataQualityCheckRepository 创建数据质量检查仓储实例
func NewDataQualityCheckRepository(db *gorm.DB) datagovapp.DataQualityCheckRepository {
	return &dataQualityCheckRepository{db: db}
}

// Create 创建数据质量检查
func (r *dataQualityCheckRepository) Create(ctx context.Context, check *datagov.DataQualityCheck) error {
	if check.ID == "" {
		check.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(check).Error
}

// GetByID 根据 ID 查询数据质量检查
func (r *dataQualityCheckRepository) GetByID(ctx context.Context, id string) (*datagov.DataQualityCheck, error) {
	var check datagov.DataQualityCheck
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&check).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("数据质量检查不存在: %s", id)
		}
		return nil, fmt.Errorf("查询数据质量检查失败: %w", err)
	}
	return &check, nil
}

// List 查询数据质量检查列表（支持过滤和分页）
func (r *dataQualityCheckRepository) List(ctx context.Context, filter datagovapp.DataQualityCheckFilter) ([]datagov.DataQualityCheck, int64, error) {
	var checks []datagov.DataQualityCheck
	var total int64

	query := r.db.WithContext(ctx).Model(&datagov.DataQualityCheck{}).Where("tenant_id = ?", filter.TenantID)

	if filter.RuleID != "" {
		query = query.Where("rule_id = ?", filter.RuleID)
	}
	if filter.AssetID != "" {
		query = query.Where("asset_id = ?", filter.AssetID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.TriggeredBy != "" {
		query = query.Where("triggered_by = ?", filter.TriggeredBy)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计数据质量检查数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&checks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询数据质量检查列表失败: %w", err)
	}

	return checks, total, nil
}

// Update 更新数据质量检查
func (r *dataQualityCheckRepository) Update(ctx context.Context, check *datagov.DataQualityCheck) error {
	return r.db.WithContext(ctx).Save(check).Error
}

// Delete 软删除数据质量检查
func (r *dataQualityCheckRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&datagov.DataQualityCheck{}, "id = ?", id).Error
}

// GetByRuleAndAsset 根据规则ID和资产ID查询最新的数据质量检查
func (r *dataQualityCheckRepository) GetByRuleAndAsset(ctx context.Context, ruleID, assetID string) (*datagov.DataQualityCheck, error) {
	var check datagov.DataQualityCheck
	if err := r.db.WithContext(ctx).Where("rule_id = ? AND asset_id = ?", ruleID, assetID).Order("created_at DESC").First(&check).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("数据质量检查不存在: rule_id=%s, asset_id=%s", ruleID, assetID)
		}
		return nil, fmt.Errorf("查询数据质量检查失败: %w", err)
	}
	return &check, nil
}
