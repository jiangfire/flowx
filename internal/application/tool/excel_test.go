package tool_test

import (
	"bytes"
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupExcelTest 创建 Excel 服务测试环境
func setupExcelTest(t *testing.T) (*toolapp.ExcelService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tool.Tool{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	toolRepo := persistence.NewToolRepository(db)
	svc := toolapp.NewExcelService(toolRepo)
	return svc, db
}

// TestExportTools_Basic 导出为 xlsx 格式
func TestExportTools_Basic(t *testing.T) {
	svc, _ := setupExcelTest(t)

	tools := []tool.Tool{
		{
			BaseModel: base.BaseModel{TenantID: "tenant-001"},
			Name:      "Altium Designer",
			Type:      "eda",
			Status:    "active",
			Endpoint:  "https://eda.example.com",
		},
		{
			BaseModel: base.BaseModel{TenantID: "tenant-001"},
			Name:      "ANSYS Fluent",
			Type:      "cae",
			Status:    "active",
		},
	}

	buf, err := svc.ExportTools(context.Background(), tools, nil)
	if err != nil {
		t.Fatalf("导出工具失败: %v", err)
	}

	// 验证是有效的 xlsx 文件
	_, err = excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("导出文件不是有效的 xlsx 格式: %v", err)
	}
}

// TestExportTools_SpecificColumns 指定列只导出指定字段
func TestExportTools_SpecificColumns(t *testing.T) {
	svc, _ := setupExcelTest(t)

	tools := []tool.Tool{
		{
			BaseModel: base.BaseModel{TenantID: "tenant-001"},
			Name:      "Altium Designer",
			Type:      "eda",
			Status:    "active",
			Endpoint:  "https://eda.example.com",
			Category:  "electronics",
		},
	}

	buf, err := svc.ExportTools(context.Background(), tools, []string{"name", "type"})
	if err != nil {
		t.Fatalf("导出工具失败: %v", err)
	}

	// 验证导出的文件只包含指定的列
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("打开导出文件失败: %v", err)
	}
	defer f.Close()

	// 获取第一行（表头）
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("读取行失败: %v", err)
	}
	if len(rows) < 1 {
		t.Fatal("期望至少有一行数据")
	}

	// 表头应该只有 name 和 type
	header := rows[0]
	if len(header) != 2 {
		t.Errorf("期望表头有 2 列，实际有 %d 列: %v", len(header), header)
	}
}

// TestExportTools_EmptyData 空数据返回只有表头的文件
func TestExportTools_EmptyData(t *testing.T) {
	svc, _ := setupExcelTest(t)

	buf, err := svc.ExportTools(context.Background(), []tool.Tool{}, nil)
	if err != nil {
		t.Fatalf("导出空数据失败: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("打开导出文件失败: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("读取行失败: %v", err)
	}

	// 空数据应该只有表头行
	if len(rows) != 1 {
		t.Errorf("期望只有 1 行（表头），实际有 %d 行", len(rows))
	}
}

// TestImportTools_Normal 正常导入
func TestImportTools_Normal(t *testing.T) {
	svc, db := setupExcelTest(t)

	// 创建一个包含工具数据的 xlsx 文件
	f := excelize.NewFile()
	f.SetSheetRow("Sheet1", "A1", &[]string{"name", "type", "status", "endpoint", "category"})
	f.SetSheetRow("Sheet1", "A2", &[]string{"Altium Designer", "eda", "active", "https://eda.example.com", "electronics"})
	f.SetSheetRow("Sheet1", "A3", &[]string{"ANSYS Fluent", "cae", "active", "https://cae.example.com", "simulation"})

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("创建测试 xlsx 失败: %v", err)
	}

	results, err := svc.ImportTools(context.Background(), buf.Bytes(), "tenant-001")
	if err != nil {
		t.Fatalf("导入工具失败: %v", err)
	}

	// 验证导入结果
	successCount := 0
	for _, r := range results {
		if r.Status == "success" {
			successCount++
		}
	}
	if successCount != 2 {
		t.Errorf("期望成功导入 2 条记录，实际为 %d", successCount)
	}

	// 验证数据库中确实有数据
	toolRepo := persistence.NewToolRepository(db)
	tools, total, err := toolRepo.List(context.Background(), toolapp.ToolFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望数据库中有 2 条记录，实际为 %d", total)
	}
	if len(tools) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(tools))
	}
	_ = tools
}

// TestImportTools_MissingRequired 缺少必填列跳过并报告
func TestImportTools_MissingRequired(t *testing.T) {
	svc, _ := setupExcelTest(t)

	f := excelize.NewFile()
	f.SetSheetRow("Sheet1", "A1", &[]string{"name", "type", "status"})
	f.SetSheetRow("Sheet1", "A2", &[]string{"", "eda", "active"})           // 缺少 name
	f.SetSheetRow("Sheet1", "A3", &[]string{"ValidTool", "eda", "active"})  // 正常行

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("创建测试 xlsx 失败: %v", err)
	}

	results, err := svc.ImportTools(context.Background(), buf.Bytes(), "tenant-001")
	if err != nil {
		t.Fatalf("导入工具失败: %v", err)
	}

	// 验证有一条跳过
	skipCount := 0
	successCount := 0
	for _, r := range results {
		if r.Status == "skip" {
			skipCount++
		}
		if r.Status == "success" {
			successCount++
		}
	}
	if skipCount != 1 {
		t.Errorf("期望跳过 1 条记录，实际为 %d", skipCount)
	}
	if successCount != 1 {
		t.Errorf("期望成功导入 1 条记录，实际为 %d", successCount)
	}
}

// TestImportTools_DuplicateName 重复名称跳过并报告
func TestImportTools_DuplicateName(t *testing.T) {
	svc, db := setupExcelTest(t)

	// 先创建一个同名工具
	toolRepo := persistence.NewToolRepository(db)
	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "Altium Designer",
		Type:      "eda",
		Status:    "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// 导入包含同名工具的文件
	f := excelize.NewFile()
	f.SetSheetRow("Sheet1", "A1", &[]string{"name", "type", "status"})
	f.SetSheetRow("Sheet1", "A2", &[]string{"Altium Designer", "eda", "active"}) // 重复
	f.SetSheetRow("Sheet1", "A3", &[]string{"NewTool", "cae", "active"})         // 新增

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("创建测试 xlsx 失败: %v", err)
	}

	results, err := svc.ImportTools(context.Background(), buf.Bytes(), "tenant-001")
	if err != nil {
		t.Fatalf("导入工具失败: %v", err)
	}

	skipCount := 0
	successCount := 0
	for _, r := range results {
		if r.Status == "skip" {
			skipCount++
		}
		if r.Status == "success" {
			successCount++
		}
	}
	if skipCount != 1 {
		t.Errorf("期望跳过 1 条重复记录，实际为 %d", skipCount)
	}
	if successCount != 1 {
		t.Errorf("期望成功导入 1 条新记录，实际为 %d", successCount)
	}
}

// TestImportTools_EmptyFile 空文件返回错误
func TestImportTools_EmptyFile(t *testing.T) {
	svc, _ := setupExcelTest(t)

	_, err := svc.ImportTools(context.Background(), []byte{}, "tenant-001")
	if err == nil {
		t.Fatal("期望空文件返回错误")
	}
}
