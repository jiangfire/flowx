package datagov

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	"github.com/xuri/excelize/v2"
)

// ImportResult 导入结果
type ImportResult struct {
	Total   int           `json:"total"`
	Success int           `json:"success"`
	Failed  int           `json:"failed"`
	Errors  []ImportError `json:"errors,omitempty"`
}

// ImportError 导入错误
type ImportError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Error string `json:"error"`
}

// DataGovExcelService 数据治理Excel导入导出服务
type DataGovExcelService struct{}

// NewDataGovExcelService 创建数据治理Excel服务实例
func NewDataGovExcelService() *DataGovExcelService {
	return &DataGovExcelService{}
}

// ==================== Policy Excel ====================

// policyExportColumns 策略导出列定义
var policyExportColumns = []struct {
	header string
	field  string
}{
	{"name", "Name"},
	{"type", "Type"},
	{"description", "Description"},
	{"scope", "Scope"},
	{"scope_value", "ScopeValue"},
	{"priority", "Priority"},
	{"status", "Status"},
}

// ExportPolicies 导出数据策略为 Excel 文件
func (s *DataGovExcelService) ExportPolicies(ctx context.Context, policies []datagov.DataPolicy, columns []string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	exportCols := policyExportColumns
	if len(columns) > 0 {
		colMap := make(map[string]bool)
		for _, c := range columns {
			colMap[strings.ToLower(c)] = true
		}
		var filtered []struct {
			header string
			field  string
		}
		for _, col := range policyExportColumns {
			if colMap[col.header] {
				filtered = append(filtered, col)
			}
		}
		if len(filtered) > 0 {
			exportCols = filtered
		}
	}

	// 写入表头
	headers := make([]string, len(exportCols))
	for i, col := range exportCols {
		headers[i] = col.header
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	_ = f.SetSheetRow(sheet, cell, &headers)

	// 写入数据行
	for rowIdx, p := range policies {
		row := rowIdx + 2
		values := make([]string, len(exportCols))
		for i, col := range exportCols {
			switch col.field {
			case "Name":
				values[i] = p.Name
			case "Type":
				values[i] = p.Type
			case "Description":
				values[i] = p.Description
			case "Scope":
				values[i] = p.Scope
			case "ScopeValue":
				values[i] = p.ScopeValue
			case "Priority":
				values[i] = fmt.Sprintf("%d", p.Priority)
			case "Status":
				values[i] = p.Status
			}
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow(sheet, cell, &values)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成 Excel 文件失败: %w", err)
	}

	return buf, nil
}

// ParsePolicies 从 Excel 文件解析数据策略请求
func (s *DataGovExcelService) ParsePolicies(ctx context.Context, data []byte, tenantID string) ([]*CreatePolicyRequest, *ImportResult, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("文件内容为空")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("打开 Excel 文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, nil, fmt.Errorf("读取 Excel 内容失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil, fmt.Errorf("文件没有数据行")
	}

	// 解析表头
	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	nameIdx, hasName := colIndex["name"]
	_, hasType := colIndex["type"]
	if !hasName || !hasType {
		return nil, nil, fmt.Errorf("缺少必填列: name 或 type")
	}

	result := &ImportResult{}
	var requests []*CreatePolicyRequest

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		result.Total++

		if len(row) == 0 {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "",
				Error: "空行",
			})
			continue
		}

		name := ""
		if nameIdx < len(row) {
			name = strings.TrimSpace(row[nameIdx])
		}
		if name == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "name",
				Error: "策略名称不能为空",
			})
			continue
		}

		typ := ""
		if idx, ok := colIndex["type"]; ok && idx < len(row) {
			typ = strings.TrimSpace(row[idx])
		}
		if typ == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "type",
				Error: "策略类型不能为空",
			})
			continue
		}

		status := "active"
		if idx, ok := colIndex["status"]; ok && idx < len(row) {
			s := strings.TrimSpace(row[idx])
			if s != "" {
				status = s
			}
		}

		description := ""
		if idx, ok := colIndex["description"]; ok && idx < len(row) {
			description = strings.TrimSpace(row[idx])
		}

		scope := ""
		if idx, ok := colIndex["scope"]; ok && idx < len(row) {
			scope = strings.TrimSpace(row[idx])
		}

		req := &CreatePolicyRequest{
			Name:        name,
			Type:        typ,
			Description: description,
			Scope:       scope,
			Status:      status,
		}
		requests = append(requests, req)
		result.Success++
	}

	return requests, result, nil
}

// ==================== Asset Excel ====================

// assetExportColumns 资产导出列定义
var assetExportColumns = []struct {
	header string
	field  string
}{
	{"name", "Name"},
	{"type", "Type"},
	{"source", "Source"},
	{"description", "Description"},
	{"format", "Format"},
	{"classification", "Classification"},
	{"owner_id", "OwnerID"},
	{"status", "Status"},
}

// ExportAssets 导出数据资产为 Excel 文件
func (s *DataGovExcelService) ExportAssets(ctx context.Context, assets []datagov.DataAsset, columns []string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	exportCols := assetExportColumns
	if len(columns) > 0 {
		colMap := make(map[string]bool)
		for _, c := range columns {
			colMap[strings.ToLower(c)] = true
		}
		var filtered []struct {
			header string
			field  string
		}
		for _, col := range assetExportColumns {
			if colMap[col.header] {
				filtered = append(filtered, col)
			}
		}
		if len(filtered) > 0 {
			exportCols = filtered
		}
	}

	// 写入表头
	headers := make([]string, len(exportCols))
	for i, col := range exportCols {
		headers[i] = col.header
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	_ = f.SetSheetRow(sheet, cell, &headers)

	// 写入数据行
	for rowIdx, a := range assets {
		row := rowIdx + 2
		values := make([]string, len(exportCols))
		for i, col := range exportCols {
			switch col.field {
			case "Name":
				values[i] = a.Name
			case "Type":
				values[i] = a.Type
			case "Source":
				values[i] = a.Source
			case "Description":
				values[i] = a.Description
			case "Format":
				values[i] = a.Format
			case "Classification":
				values[i] = a.Classification
			case "OwnerID":
				values[i] = a.OwnerID
			case "Status":
				values[i] = a.Status
			}
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow(sheet, cell, &values)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成 Excel 文件失败: %w", err)
	}

	return buf, nil
}

// ParseAssets 从 Excel 文件解析数据资产请求
func (s *DataGovExcelService) ParseAssets(ctx context.Context, data []byte, tenantID string) ([]*CreateAssetRequest, *ImportResult, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("文件内容为空")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("打开 Excel 文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, nil, fmt.Errorf("读取 Excel 内容失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil, fmt.Errorf("文件没有数据行")
	}

	// 解析表头
	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	nameIdx, hasName := colIndex["name"]
	_, hasType := colIndex["type"]
	if !hasName || !hasType {
		return nil, nil, fmt.Errorf("缺少必填列: name 或 type")
	}

	result := &ImportResult{}
	var requests []*CreateAssetRequest

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		result.Total++

		if len(row) == 0 {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "",
				Error: "空行",
			})
			continue
		}

		name := ""
		if nameIdx < len(row) {
			name = strings.TrimSpace(row[nameIdx])
		}
		if name == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "name",
				Error: "资产名称不能为空",
			})
			continue
		}

		typ := ""
		if idx, ok := colIndex["type"]; ok && idx < len(row) {
			typ = strings.TrimSpace(row[idx])
		}
		if typ == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "type",
				Error: "资产类型不能为空",
			})
			continue
		}

		status := "active"
		if idx, ok := colIndex["status"]; ok && idx < len(row) {
			s := strings.TrimSpace(row[idx])
			if s != "" {
				status = s
			}
		}

		description := ""
		if idx, ok := colIndex["description"]; ok && idx < len(row) {
			description = strings.TrimSpace(row[idx])
		}

		source := ""
		if idx, ok := colIndex["source"]; ok && idx < len(row) {
			source = strings.TrimSpace(row[idx])
		}

		req := &CreateAssetRequest{
			Name:           name,
			Type:           typ,
			Source:         source,
			Description:    description,
			Status:         status,
			Classification: "internal",
		}
		requests = append(requests, req)
		result.Success++
	}

	return requests, result, nil
}

// ==================== Rule Excel ====================

// ruleExportColumns 规则导出列定义
var ruleExportColumns = []struct {
	header string
	field  string
}{
	{"name", "Name"},
	{"type", "Type"},
	{"target_asset", "TargetAsset"},
	{"target_field", "TargetField"},
	{"description", "Description"},
	{"severity", "Severity"},
	{"status", "Status"},
}

// ExportRules 导出数据质量规则为 Excel 文件
func (s *DataGovExcelService) ExportRules(ctx context.Context, rules []datagov.DataQualityRule, columns []string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	exportCols := ruleExportColumns
	if len(columns) > 0 {
		colMap := make(map[string]bool)
		for _, c := range columns {
			colMap[strings.ToLower(c)] = true
		}
		var filtered []struct {
			header string
			field  string
		}
		for _, col := range ruleExportColumns {
			if colMap[col.header] {
				filtered = append(filtered, col)
			}
		}
		if len(filtered) > 0 {
			exportCols = filtered
		}
	}

	// 写入表头
	headers := make([]string, len(exportCols))
	for i, col := range exportCols {
		headers[i] = col.header
	}
	cell, _ := excelize.CoordinatesToCellName(1, 1)
	_ = f.SetSheetRow(sheet, cell, &headers)

	// 写入数据行
	for rowIdx, r := range rules {
		row := rowIdx + 2
		values := make([]string, len(exportCols))
		for i, col := range exportCols {
			switch col.field {
			case "Name":
				values[i] = r.Name
			case "Type":
				values[i] = r.Type
			case "TargetAsset":
				values[i] = r.TargetAsset
			case "TargetField":
				values[i] = r.TargetField
			case "Description":
				values[i] = r.Description
			case "Severity":
				values[i] = r.Severity
			case "Status":
				values[i] = r.Status
			}
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow(sheet, cell, &values)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成 Excel 文件失败: %w", err)
	}

	return buf, nil
}

// ParseRules 从 Excel 文件解析数据质量规则请求
func (s *DataGovExcelService) ParseRules(ctx context.Context, data []byte, tenantID string) ([]*CreateRuleRequest, *ImportResult, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("文件内容为空")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("打开 Excel 文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, nil, fmt.Errorf("读取 Excel 内容失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil, fmt.Errorf("文件没有数据行")
	}

	// 解析表头
	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	nameIdx, hasName := colIndex["name"]
	_, hasType := colIndex["type"]
	if !hasName || !hasType {
		return nil, nil, fmt.Errorf("缺少必填列: name 或 type")
	}

	result := &ImportResult{}
	var requests []*CreateRuleRequest

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		result.Total++

		if len(row) == 0 {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "",
				Error: "空行",
			})
			continue
		}

		name := ""
		if nameIdx < len(row) {
			name = strings.TrimSpace(row[nameIdx])
		}
		if name == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "name",
				Error: "规则名称不能为空",
			})
			continue
		}

		typ := ""
		if idx, ok := colIndex["type"]; ok && idx < len(row) {
			typ = strings.TrimSpace(row[idx])
		}
		if typ == "" {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowIdx + 1,
				Field: "type",
				Error: "规则类型不能为空",
			})
			continue
		}

		status := "active"
		if idx, ok := colIndex["status"]; ok && idx < len(row) {
			s := strings.TrimSpace(row[idx])
			if s != "" {
				status = s
			}
		}

		severity := "warning"
		if idx, ok := colIndex["severity"]; ok && idx < len(row) {
			s := strings.TrimSpace(row[idx])
			if s != "" {
				severity = s
			}
		}

		description := ""
		if idx, ok := colIndex["description"]; ok && idx < len(row) {
			description = strings.TrimSpace(row[idx])
		}

		targetAsset := ""
		if idx, ok := colIndex["target_asset"]; ok && idx < len(row) {
			targetAsset = strings.TrimSpace(row[idx])
		}

		targetField := ""
		if idx, ok := colIndex["target_field"]; ok && idx < len(row) {
			targetField = strings.TrimSpace(row[idx])
		}

		req := &CreateRuleRequest{
			Name:        name,
			Type:        typ,
			TargetAsset: targetAsset,
			TargetField: targetField,
			Description: description,
			Severity:    severity,
			Status:      status,
		}
		requests = append(requests, req)
		result.Success++
	}

	return requests, result, nil
}
