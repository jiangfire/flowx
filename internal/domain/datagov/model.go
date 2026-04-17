package datagov

import (
	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// ==================== DataPolicy 数据治理策略 ====================

// TableName 指定 DataPolicy 表名
func (DataPolicy) TableName() string {
	return "data_policies"
}

// DataPolicy 数据治理策略
type DataPolicy struct {
	base.BaseModel
	Name        string     `gorm:"size:200;not null" json:"name"`              // 策略名称
	Type        string     `gorm:"size:50;not null;index" json:"type"`         // 策略类型（retention/classification/quality/access）
	Description string     `gorm:"type:text" json:"description"`               // 策略描述
	Scope       string     `gorm:"size:100;index" json:"scope"`                // 适用范围（tool_type/category/global）
	ScopeValue  string     `gorm:"size:200" json:"scope_value"`                // 范围值（如 eda/cae 或具体分类名）
	Rules       base.JSON  `gorm:"type:jsonb" json:"rules"`                    // 策略规则（JSON）
	Priority    int        `gorm:"default:0" json:"priority"`                  // 优先级（数值越大优先级越高）
	Status      string     `gorm:"size:20;default:active;index" json:"status"` // 状态：active/inactive/draft
	Version     int        `gorm:"default:1" json:"version"`                   // 版本号
}

// ==================== DataAsset 数据资产 ====================

// TableName 指定 DataAsset 表名
func (DataAsset) TableName() string {
	return "data_assets"
}

// DataAsset 数据资产注册
type DataAsset struct {
	base.BaseModel
	Name           string     `gorm:"size:200;not null" json:"name"`                       // 资产名称
	Type           string     `gorm:"size:50;not null;index" json:"type"`                  // 资产类型（dataset/model/report/config）
	Source         string     `gorm:"size:100" json:"source"`                              // 数据来源（tool名称或系统名）
	SourceID       string     `gorm:"size:26;index" json:"source_id"`                      // 来源实体ID
	Description    string     `gorm:"type:text" json:"description"`                        // 资产描述
	Format         string     `gorm:"size:50" json:"format"`                               // 数据格式（csv/json/parquet/excel）
	Schema         base.JSON  `gorm:"type:jsonb" json:"schema"`                            // 数据结构定义
	Size           int64      `gorm:"default:0" json:"size"`                               // 数据大小（字节）
	RecordCount    int64      `gorm:"default:0" json:"record_count"`                       // 记录数
	Location       string     `gorm:"size:500" json:"location"`                            // 存储位置
	Tags           base.JSON  `gorm:"type:jsonb" json:"tags"`                              // 标签
	Classification string     `gorm:"size:50;default:internal;index" json:"classification"` // 数据分类（public/internal/confidential/restricted）
	OwnerID        string     `gorm:"size:26;index" json:"owner_id"`                       // 负责人ID
	Status         string     `gorm:"size:20;default:active;index" json:"status"`          // 状态：active/archived/deprecated
}

// ==================== DataQualityRule 数据质量规则 ====================

// TableName 指定 DataQualityRule 表名
func (DataQualityRule) TableName() string {
	return "data_quality_rules"
}

// DataQualityRule 数据质量规则
type DataQualityRule struct {
	base.BaseModel
	Name        string     `gorm:"size:200;not null" json:"name"`               // 规则名称
	Type        string     `gorm:"size:50;not null;index" json:"type"`          // 规则类型（not_null/unique/range/format/custom）
	TargetAsset string     `gorm:"size:26;index" json:"target_asset"`           // 目标数据资产ID
	TargetField string     `gorm:"size:200" json:"target_field"`                // 目标字段名
	Description string     `gorm:"type:text" json:"description"`                // 规则描述
	Config      base.JSON  `gorm:"type:jsonb" json:"config"`                    // 规则配置（如范围值、正则表达式等）
	Severity    string     `gorm:"size:20;default:warning;index" json:"severity"` // 严重级别（critical/warning/info）
	Status      string     `gorm:"size:20;default:active;index" json:"status"`  // 状态
}

// ==================== DataQualityCheck 数据质量检查 ====================

// TableName 指定 DataQualityCheck 表名
func (DataQualityCheck) TableName() string {
	return "data_quality_checks"
}

// DataQualityCheck 数据质量检查记录
type DataQualityCheck struct {
	base.BaseModel
	RuleID      string     `gorm:"size:26;index" json:"rule_id"`      // 关联的质量规则ID
	AssetID     string     `gorm:"size:26;index" json:"asset_id"`     // 检查的数据资产ID
	Status      string     `gorm:"size:20;index" json:"status"`       // 检查状态（passed/failed/running/error）
	TotalCount  int64      `gorm:"default:0" json:"total_count"`      // 检查总记录数
	FailCount   int64      `gorm:"default:0" json:"fail_count"`       // 不合格记录数
	PassRate    float64    `gorm:"default:0" json:"pass_rate"`        // 通过率（0-100）
	Result      base.JSON  `gorm:"type:jsonb" json:"result"`          // 检查结果详情
	ErrorMsg    string     `gorm:"type:text" json:"error_msg"`        // 错误信息
	Duration    int64      `gorm:"default:0" json:"duration"`         // 检查耗时（毫秒）
	TriggeredBy string     `gorm:"size:50;index" json:"triggered_by"` // 触发方式（manual/scheduled/agent）
}
