package datagov

import (
	"context"

	domaintool "git.neolidy.top/neo/flowx/internal/domain/datagov"
)

// DataPolicyFilter 数据策略查询过滤条件
type DataPolicyFilter struct {
	TenantID string
	Type     string
	Status   string
	Scope    string
	Keyword  string
	Page     int
	PageSize int
}

// DataAssetFilter 数据资产查询过滤条件
type DataAssetFilter struct {
	TenantID       string
	Type           string
	Status         string
	Classification string
	Source         string
	Keyword        string
	Page           int
	PageSize       int
}

// DataQualityRuleFilter 数据质量规则查询过滤条件
type DataQualityRuleFilter struct {
	TenantID    string
	Type        string
	Status      string
	Severity    string
	TargetAsset string
	Keyword     string
	Page        int
	PageSize    int
}

// DataQualityCheckFilter 数据质量检查查询过滤条件
type DataQualityCheckFilter struct {
	TenantID    string
	RuleID      string
	AssetID     string
	Status      string
	TriggeredBy string
	Page        int
	PageSize    int
}

// DataPolicyRepository 数据策略仓储接口
type DataPolicyRepository interface {
	Create(ctx context.Context, policy *domaintool.DataPolicy) error
	GetByID(ctx context.Context, id string) (*domaintool.DataPolicy, error)
	List(ctx context.Context, filter DataPolicyFilter) ([]domaintool.DataPolicy, int64, error)
	Update(ctx context.Context, policy *domaintool.DataPolicy) error
	Delete(ctx context.Context, id string) error
}

// DataAssetRepository 数据资产仓储接口
type DataAssetRepository interface {
	Create(ctx context.Context, asset *domaintool.DataAsset) error
	GetByID(ctx context.Context, id string) (*domaintool.DataAsset, error)
	List(ctx context.Context, filter DataAssetFilter) ([]domaintool.DataAsset, int64, error)
	Update(ctx context.Context, asset *domaintool.DataAsset) error
	Delete(ctx context.Context, id string) error
}

// DataQualityRuleRepository 数据质量规则仓储接口
type DataQualityRuleRepository interface {
	Create(ctx context.Context, rule *domaintool.DataQualityRule) error
	GetByID(ctx context.Context, id string) (*domaintool.DataQualityRule, error)
	List(ctx context.Context, filter DataQualityRuleFilter) ([]domaintool.DataQualityRule, int64, error)
	Update(ctx context.Context, rule *domaintool.DataQualityRule) error
	Delete(ctx context.Context, id string) error
}

// DataQualityCheckRepository 数据质量检查仓储接口
type DataQualityCheckRepository interface {
	Create(ctx context.Context, check *domaintool.DataQualityCheck) error
	GetByID(ctx context.Context, id string) (*domaintool.DataQualityCheck, error)
	List(ctx context.Context, filter DataQualityCheckFilter) ([]domaintool.DataQualityCheck, int64, error)
	Update(ctx context.Context, check *domaintool.DataQualityCheck) error
	Delete(ctx context.Context, id string) error
	GetByRuleAndAsset(ctx context.Context, ruleID, assetID string) (*domaintool.DataQualityCheck, error)
}
