package tool

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	"github.com/xuri/excelize/v2"
)

// ImportResult 导入结果
type ImportResult struct {
	Row     int    `json:"row"`
	Status  string `json:"status"` // success/skip/error
	Message string `json:"message"`
	ToolID  string `json:"tool_id,omitempty"`
}

// ExcelService Excel 导入导出服务
type ExcelService struct {
	toolRepo ToolRepository
}

// NewExcelService 创建 Excel 服务实例
func NewExcelService(toolRepo ToolRepository) *ExcelService {
	return &ExcelService{toolRepo: toolRepo}
}

// 默认导出列及其对应的字段名映射
var defaultExportColumns = []struct {
	header string
	field  string
}{
	{"name", "Name"},
	{"type", "Type"},
	{"description", "Description"},
	{"status", "Status"},
	{"endpoint", "Endpoint"},
	{"category", "Category"},
	{"connector_id", "ConnectorID"},
}

// toolFieldMapper 工具字段映射函数表
var toolFieldMapper = map[string]func(*tool.Tool) string{
	"Name":        func(t *tool.Tool) string { return t.Name },
	"Type":        func(t *tool.Tool) string { return t.Type },
	"Description": func(t *tool.Tool) string { return t.Description },
	"Status":      func(t *tool.Tool) string { return t.Status },
	"Endpoint":    func(t *tool.Tool) string { return t.Endpoint },
	"Category":    func(t *tool.Tool) string { return t.Category },
	"ConnectorID": func(t *tool.Tool) string { return t.ConnectorID },
}

// ExportTools 导出工具为 Excel 文件
func (s *ExcelService) ExportTools(ctx context.Context, tools []tool.Tool, columns []string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 确定导出列
	exportCols := defaultExportColumns
	if len(columns) > 0 {
		// 根据用户指定的列过滤
		colMap := make(map[string]bool)
		for _, c := range columns {
			colMap[strings.ToLower(c)] = true
		}
		var filtered []struct {
			header string
			field  string
		}
		for _, col := range defaultExportColumns {
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
	for rowIdx, tl := range tools {
		row := rowIdx + 2 // 从第2行开始
		values := make([]string, len(exportCols))
		for i, col := range exportCols {
			if fn, ok := toolFieldMapper[col.field]; ok {
				values[i] = fn(&tl)
			}
		}
		cell, err := excelize.CoordinatesToCellName(1, row)
		if err != nil {
			return nil, fmt.Errorf("生成单元格坐标失败: %w", err)
		}
		_ = f.SetSheetRow(sheet, cell, &values)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("生成 Excel 文件失败: %w", err)
	}

	return buf, nil
}

// ImportTools 从 Excel 文件导入工具
func (s *ExcelService) ImportTools(ctx context.Context, file []byte, tenantID string) ([]ImportResult, error) {
	if len(file) == 0 {
		return nil, fmt.Errorf("文件内容为空")
	}

	f, err := excelize.OpenReader(bytes.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("打开 Excel 文件失败: %w", err)
	}
	defer f.Close()

	// 获取所有行
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("读取 Excel 内容失败: %w", err)
	}

	if len(rows) < 2 {
		// 只有表头或空文件
		return nil, fmt.Errorf("文件没有数据行")
	}

	// 解析表头，建立列索引映射
	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// 检查必填列
	nameIdx, hasName := colIndex["name"]
	_, hasType := colIndex["type"]
	if !hasName || !hasType {
		return nil, fmt.Errorf("缺少必填列: name 或 type")
	}

	var results []ImportResult

	// 获取现有工具名称集合（用于去重）
	existingTools, _, err := s.toolRepo.List(ctx, ToolFilter{TenantID: tenantID, PageSize: 10000})
	if err != nil {
		return nil, fmt.Errorf("查询现有工具失败: %w", err)
	}
	existingNames := make(map[string]bool)
	for _, tl := range existingTools {
		existingNames[tl.Name] = true
	}

	// 逐行导入
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		result := ImportResult{Row: rowIdx + 1} // Excel 行号从1开始

		// 跳过空行
		if len(row) == 0 {
			result.Status = "skip"
			result.Message = "空行"
			results = append(results, result)
			continue
		}

		// 获取 name 值
		name := ""
		if nameIdx < len(row) {
			name = strings.TrimSpace(row[nameIdx])
		}
		if name == "" {
			result.Status = "skip"
			result.Message = "缺少工具名称"
			results = append(results, result)
			continue
		}

		// 检查重复
		if existingNames[name] {
			result.Status = "skip"
			result.Message = fmt.Sprintf("工具名称 '%s' 已存在", name)
			results = append(results, result)
			continue
		}

		// 获取其他字段
		typ := ""
		if idx, ok := colIndex["type"]; ok && idx < len(row) {
			typ = strings.TrimSpace(row[idx])
		}
		status := "active"
		if idx, ok := colIndex["status"]; ok && idx < len(row) {
			s := strings.TrimSpace(row[idx])
			if s != "" {
				status = s
			}
		}
		endpoint := ""
		if idx, ok := colIndex["endpoint"]; ok && idx < len(row) {
			endpoint = strings.TrimSpace(row[idx])
		}
		category := ""
		if idx, ok := colIndex["category"]; ok && idx < len(row) {
			category = strings.TrimSpace(row[idx])
		}

		// 创建工具
		tl := &tool.Tool{
			BaseModel: base.BaseModel{TenantID: tenantID},
			Name:      name,
			Type:      typ,
			Status:    status,
			Endpoint:  endpoint,
			Category:  category,
		}

		if err := s.toolRepo.Create(ctx, tl); err != nil {
			result.Status = "error"
			result.Message = fmt.Sprintf("创建失败: %v", err)
			results = append(results, result)
			continue
		}

		result.Status = "success"
		result.Message = "导入成功"
		result.ToolID = tl.ID
		results = append(results, result)

		// 添加到已存在集合，防止同文件内重复
		existingNames[name] = true
	}

	return results, nil
}
