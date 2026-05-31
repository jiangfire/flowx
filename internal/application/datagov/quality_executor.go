package datagov

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"

	"github.com/jiangfire/flowx/internal/application/datagov/expression"
	"github.com/jiangfire/flowx/internal/domain/base"
	"github.com/jiangfire/flowx/internal/domain/datagov"
	"gorm.io/gorm"
)

// QualityExecutor 数据质量规则执行器
type QualityExecutor struct {
	db *gorm.DB
}

// NewQualityExecutor 创建质量规则执行器
func NewQualityExecutor(db *gorm.DB) *QualityExecutor {
	return &QualityExecutor{db: db}
}

// CheckResult 单条检查结果
type CheckResult struct {
	TotalRecords int64
	FailedCount  int64
	PassRate     float64
	FailDetails  []map[string]any
	Message      string
}

// Execute 执行质量规则检查
func (e *QualityExecutor) Execute(ctx context.Context, rule *datagov.DataQualityRule, asset *datagov.DataAsset) (*CheckResult, error) {
	if rule == nil || asset == nil {
		return nil, fmt.Errorf("规则和资产不能为空")
	}
	switch rule.Type {
	case "not_null":
		return e.checkNotNull(ctx, rule, asset)
	case "unique":
		return e.checkUnique(ctx, rule, asset)
	case "range":
		return e.checkRange(ctx, rule, asset)
	case "format":
		return e.checkFormat(ctx, rule, asset)
	case "custom":
		return e.checkCustom(ctx, rule, asset)
	default:
		return &CheckResult{
			TotalRecords: asset.RecordCount,
			FailedCount:  0,
			PassRate:     100.0,
			Message:      fmt.Sprintf("不支持的规则类型: %s，默认通过", rule.Type),
		}, nil
	}
}

// checkNotNull 检查目标字段是否非空
func (e *QualityExecutor) checkNotNull(ctx context.Context, rule *datagov.DataQualityRule, asset *datagov.DataAsset) (*CheckResult, error) {
	records, err := e.fetchAssetRecords(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("获取资产数据失败: %w", err)
	}

	field := rule.TargetField
	total := int64(len(records))
	var failed int64
	var failDetails []map[string]any

	for i, record := range records {
		val, _ := getFieldValue(record, field)
		if isEmptyValue(val) {
			failed++
			failDetails = append(failDetails, map[string]any{
				"index":  i,
				"field":  field,
				"reason": "值为空",
			})
		}
	}

	return buildResult(total, failed, failDetails), nil
}

// checkUnique 检查目标字段是否唯一
func (e *QualityExecutor) checkUnique(ctx context.Context, rule *datagov.DataQualityRule, asset *datagov.DataAsset) (*CheckResult, error) {
	records, err := e.fetchAssetRecords(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("获取资产数据失败: %w", err)
	}

	field := rule.TargetField
	total := int64(len(records))
	seen := make(map[string]int)
	var failDetails []map[string]any

	for i, record := range records {
		val, found := getFieldValue(record, field)
		if !found || val == nil {
			continue
		}
		key := fmt.Sprintf("%v", val)
		if prevIdx, exists := seen[key]; exists {
			failDetails = append(failDetails, map[string]any{
				"index":     i,
				"field":     field,
				"value":     val,
				"duplicate": prevIdx,
				"reason":    "值重复",
			})
		} else {
			seen[key] = i
		}
	}

	return buildResult(total, int64(len(failDetails)), failDetails), nil
}

// checkRange 检查数值范围
func (e *QualityExecutor) checkRange(ctx context.Context, rule *datagov.DataQualityRule, asset *datagov.DataAsset) (*CheckResult, error) {
	records, err := e.fetchAssetRecords(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("获取资产数据失败: %w", err)
	}

	field := rule.TargetField
	total := int64(len(records))
	var failed int64
	var failDetails []map[string]any

	minVal, maxVal := getRangeConfig(rule.Config)

	for i, record := range records {
		val, _ := getFieldValue(record, field)
		numVal, ok := toFloat64(val)
		if !ok {
			failed++
			failDetails = append(failDetails, map[string]any{
				"index":  i,
				"field":  field,
				"value":  val,
				"reason": "值无法转换为数值",
			})
			continue
		}
		recordFailed := false
		var reasons []string
		if minVal != nil && numVal < *minVal {
			recordFailed = true
			reasons = append(reasons, fmt.Sprintf("值 %.2f 小于最小值 %.2f", numVal, *minVal))
		}
		if maxVal != nil && numVal > *maxVal {
			recordFailed = true
			reasons = append(reasons, fmt.Sprintf("值 %.2f 大于最大值 %.2f", numVal, *maxVal))
		}
		if recordFailed {
			failed++
			failDetails = append(failDetails, map[string]any{
				"index":  i,
				"field":  field,
				"value":  numVal,
				"min":    *minVal,
				"max":    *maxVal,
				"reason": strings.Join(reasons, "; "),
			})
		}
	}

	return buildResult(total, failed, failDetails), nil
}

// checkFormat 检查字段格式
func (e *QualityExecutor) checkFormat(ctx context.Context, rule *datagov.DataQualityRule, asset *datagov.DataAsset) (*CheckResult, error) {
	pattern := ""
	if v, ok := rule.Config["pattern"].(string); ok {
		pattern = v
	}
	if pattern == "" {
		return &CheckResult{
			TotalRecords: asset.RecordCount,
			FailedCount:  0,
			PassRate:     100.0,
			Message:      "未配置格式模式，默认通过",
		}, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("正则表达式编译失败: %w", err)
	}

	records, err := e.fetchAssetRecords(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("获取资产数据失败: %w", err)
	}

	field := rule.TargetField
	total := int64(len(records))
	var failed int64
	var failDetails []map[string]any

	for i, record := range records {
		val, _ := getFieldValue(record, field)
		strVal := fmt.Sprintf("%v", val)
		if !re.MatchString(strVal) {
			failed++
			failDetails = append(failDetails, map[string]any{
				"index":   i,
				"field":   field,
				"value":   strVal,
				"pattern": pattern,
				"reason":  fmt.Sprintf("值不匹配模式 %s", pattern),
			})
		}
	}

	return buildResult(total, failed, failDetails), nil
}

// checkCustom 自定义表达式检查
func (e *QualityExecutor) checkCustom(ctx context.Context, rule *datagov.DataQualityRule, asset *datagov.DataAsset) (*CheckResult, error) {
	expressionStr := ""
	if v, ok := rule.Config["expression"].(string); ok {
		expressionStr = v
	}
	if expressionStr == "" {
		return &CheckResult{
			TotalRecords: asset.RecordCount,
			FailedCount:  0,
			PassRate:     100.0,
			Message:      "未配置自定义表达式，默认通过",
		}, nil
	}

	records, err := e.fetchAssetRecords(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("获取资产数据失败: %w", err)
	}

	total := int64(len(records))
	var failed int64
	var failDetails []map[string]any

	for i, record := range records {
		evalCtx := &expression.Context{
			Tool: recordToToolContext(record),
		}
		result, err := expression.Evaluate(evalCtx, expressionStr)
		if err != nil {
			failed++
			failDetails = append(failDetails, map[string]any{
				"index":      i,
				"expression": expressionStr,
				"error":      err.Error(),
				"reason":     "表达式求值失败",
			})
			continue
		}
		if !result {
			failed++
			failDetails = append(failDetails, map[string]any{
				"index":      i,
				"expression": expressionStr,
				"reason":     "表达式检查未通过",
			})
		}
	}

	return buildResult(total, failed, failDetails), nil
}

// fetchAssetRecords 从资产获取数据记录
func (e *QualityExecutor) fetchAssetRecords(ctx context.Context, asset *datagov.DataAsset) ([]map[string]any, error) {
	if asset.Source == "tool" && asset.SourceID != "" {
		return e.fetchToolRecords(ctx, asset)
	}
	if len(asset.Schema) > 0 {
		return []map[string]any{asset.Schema}, nil
	}
	return []map[string]any{}, nil
}

// fetchToolRecords 从工具获取数据记录，强制执行租户隔离
func (e *QualityExecutor) fetchToolRecords(ctx context.Context, asset *datagov.DataAsset) ([]map[string]any, error) {
	type toolRow struct {
		Type        string
		Category    string
		Name        string
		Status      string
		Config      base.JSON
		Description string
	}

	var tools []toolRow
	if err := e.db.WithContext(ctx).
		Table("tools").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", asset.SourceID, asset.TenantID).
		Find(&tools).Error; err != nil {
		return nil, fmt.Errorf("查询工具数据失败: %w", err)
	}

	records := make([]map[string]any, len(tools))
	for i, t := range tools {
		record := map[string]any{
			"type":        t.Type,
			"category":    t.Category,
			"name":        t.Name,
			"status":      t.Status,
			"description": t.Description,
		}
		if t.Config != nil {
			for k, v := range t.Config {
				record["config."+k] = v
			}
		}
		records[i] = record
	}
	return records, nil
}

// getFieldValue 从记录中获取字段值，支持点号分隔的嵌套路径
func getFieldValue(record map[string]any, field string) (any, bool) {
	parts := strings.Split(field, ".")
	var current any = record
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// isEmptyValue 判断值是否为空
func isEmptyValue(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

// toFloat64 将值转换为 float64
func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	}
	return 0, false
}

// getRangeConfig 从规则配置中获取范围值
func getRangeConfig(config base.JSON) (*float64, *float64) {
	var min, max *float64
	if v, ok := config["min"]; ok {
		if f, ok := toFloat64(v); ok {
			min = &f
		}
	}
	if v, ok := config["max"]; ok {
		if f, ok := toFloat64(v); ok {
			max = &f
		}
	}
	return min, max
}

// recordToToolContext 将记录转为表达式求值所需的工具上下文
func recordToToolContext(record map[string]any) expression.ToolContext {
	tc := expression.ToolContext{
		Name:   getStringField(record, "name"),
		Type:   getStringField(record, "type"),
		Status: getStringField(record, "status"),
		Config: make(map[string]any),
	}
	for k, v := range record {
		if strings.HasPrefix(k, "config.") {
			tc.Config[strings.TrimPrefix(k, "config.")] = v
		}
	}
	return tc
}

func getStringField(record map[string]any, field string) string {
	if v, ok := record[field].(string); ok {
		return v
	}
	return ""
}

func buildResult(total, failed int64, failDetails []map[string]any) *CheckResult {
	if failed > total {
		slog.Warn("质量检查失败数超过总数，已修正", "total", total, "failed", failed)
		failed = total
	}
	passRate := 100.0
	if total > 0 {
		passRate = math.Round(float64(total-failed)/float64(total)*10000) / 100
	}
	if passRate < 0 {
		passRate = 0
	}
	if passRate > 100 {
		passRate = 100
	}
	var message string
	if failed == 0 {
		message = "质量检查通过"
	} else {
		message = fmt.Sprintf("发现 %d 条不满足规则的数据", failed)
	}
	return &CheckResult{
		TotalRecords: total,
		FailedCount:  failed,
		PassRate:     passRate,
		FailDetails:  failDetails,
		Message:      message,
	}
}
