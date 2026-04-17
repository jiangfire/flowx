package datagov_test

import (
	"bytes"
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestExcelService 创建测试用 DataGovExcelService
func setupTestExcelService(t *testing.T) (*datagovapp.DataGovExcelService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	err = db.AutoMigrate(
		&datagov.DataPolicy{},
		&datagov.DataAsset{},
		&datagov.DataQualityRule{},
		&datagov.DataQualityCheck{},
	)
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	policyRepo := persistence.NewDataPolicyRepository(db)
	assetRepo := persistence.NewDataAssetRepository(db)
	ruleRepo := persistence.NewDataQualityRuleRepository(db)

	svc := datagovapp.NewDataGovExcelService(policyRepo, assetRepo, ruleRepo)
	return svc, db
}

// createTestExcelFile 创建测试用 Excel 文件
func createTestExcelFile(headers []string, rows [][]string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	f.SetSheetRow(sheet, cell, &headers)

	for i, row := range rows {
		rowNum := i + 2
		cell, _ := excelize.CoordinatesToCellName(1, rowNum)
		f.SetSheetRow(sheet, cell, &row)
	}

	return f.WriteToBuffer()
}

// ==================== ExportPolicies Tests ====================

func TestExportPolicies(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	policies := []datagov.DataPolicy{
		{
			BaseModel:   base.BaseModel{TenantID: "tenant_excel_001"},
			Name:        "保留策略",
			Type:        "retention",
			Description: "数据保留30天",
			Status:      "active",
			Version:     1,
		},
		{
			BaseModel:   base.BaseModel{TenantID: "tenant_excel_001"},
			Name:        "分类策略",
			Type:        "classification",
			Description: "数据分类管理",
			Status:      "active",
			Version:     1,
		},
	}

	buf, err := svc.ExportPolicies(ctx, policies, nil)
	if err != nil {
		t.Fatalf("导出策略失败: %v", err)
	}
	if buf == nil {
		t.Fatal("导出结果不应为空")
	}
	if buf.Len() == 0 {
		t.Fatal("导出文件大小不应为0")
	}

	// 验证导出内容
	f, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("读取行数据失败: %v", err)
	}

	// 应有1行表头 + 2行数据 = 3行
	if len(rows) != 3 {
		t.Fatalf("期望3行(1表头+2数据), got %d", len(rows))
	}

	// 验证表头
	if rows[0][0] != "name" {
		t.Fatalf("表头第一列应为 name, got: %s", rows[0][0])
	}
}

func TestExportPolicies_Empty(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := svc.ExportPolicies(ctx, []datagov.DataPolicy{}, nil)
	if err != nil {
		t.Fatalf("导出策略失败: %v", err)
	}

	// 空数据也应成功导出（只有表头）
	f, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("读取行数据失败: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("期望1行(只有表头), got %d", len(rows))
	}
}

// ==================== ImportPolicies Tests ====================

func TestImportPolicies_Success(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type", "description", "status"},
		[][]string{
			{"保留策略", "retention", "数据保留30天", "active"},
			{"分类策略", "classification", "数据分类管理", "draft"},
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	result, err := svc.ImportPolicies(ctx, buf.Bytes(), "tenant_import_001")
	if err != nil {
		t.Fatalf("导入策略失败: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("期望总数2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Fatalf("期望成功2, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Fatalf("期望失败0, got %d", result.Failed)
	}
}

func TestImportPolicies_Validation(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type", "description", "status"},
		[][]string{
			{"", "retention", "名称为空"},        // 名称为空
			{"有效策略", "", "类型为空"},           // 类型为空
			{"有效策略2", "quality", "有效描述"},   // 有效行
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	result, err := svc.ImportPolicies(ctx, buf.Bytes(), "tenant_import_002")
	if err != nil {
		t.Fatalf("导入策略失败: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("期望总数3, got %d", result.Total)
	}
	if result.Success != 1 {
		t.Fatalf("期望成功1, got %d", result.Success)
	}
	if result.Failed != 2 {
		t.Fatalf("期望失败2, got %d", result.Failed)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("期望2条错误, got %d", len(result.Errors))
	}
}

func TestImportPolicies_EmptyFile(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	_, err := svc.ImportPolicies(ctx, []byte{}, "tenant_import_003")
	if err == nil {
		t.Fatal("空文件应返回错误")
	}
}

func TestImportPolicies_OnlyHeader(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type"},
		[][]string{},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	_, err = svc.ImportPolicies(ctx, buf.Bytes(), "tenant_import_004")
	if err == nil {
		t.Fatal("只有表头应返回错误")
	}
}

func TestImportPolicies_MissingRequiredColumns(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"description", "status"},
		[][]string{
			{"描述", "active"},
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	_, err = svc.ImportPolicies(ctx, buf.Bytes(), "tenant_import_005")
	if err == nil {
		t.Fatal("缺少必填列应返回错误")
	}
}

// ==================== ExportAssets Tests ====================

func TestExportAssets(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	assets := []datagov.DataAsset{
		{
			BaseModel:      base.BaseModel{TenantID: "tenant_excel_001"},
			Name:           "用户数据集",
			Type:           "dataset",
			Source:         "mysql",
			Description:    "用户信息数据",
			Format:         "csv",
			Classification: "internal",
			Status:         "active",
		},
		{
			BaseModel:      base.BaseModel{TenantID: "tenant_excel_001"},
			Name:           "模型文件",
			Type:           "model",
			Source:         "s3",
			Description:    "训练模型",
			Format:         "parquet",
			Classification: "confidential",
			Status:         "active",
		},
	}

	buf, err := svc.ExportAssets(ctx, assets, nil)
	if err != nil {
		t.Fatalf("导出资产失败: %v", err)
	}
	if buf == nil || buf.Len() == 0 {
		t.Fatal("导出结果不应为空")
	}

	// 验证导出内容
	f, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("读取行数据失败: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("期望3行(1表头+2数据), got %d", len(rows))
	}

	// 验证表头
	if rows[0][0] != "name" {
		t.Fatalf("表头第一列应为 name, got: %s", rows[0][0])
	}
}

// ==================== ImportAssets Tests ====================

func TestImportAssets_Success(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type", "source", "description", "status"},
		[][]string{
			{"用户数据集", "dataset", "mysql", "用户信息", "active"},
			{"模型文件", "model", "s3", "训练模型", "active"},
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	result, err := svc.ImportAssets(ctx, buf.Bytes(), "tenant_import_001")
	if err != nil {
		t.Fatalf("导入资产失败: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("期望总数2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Fatalf("期望成功2, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Fatalf("期望失败0, got %d", result.Failed)
	}
}

func TestImportAssets_Validation(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type", "source", "description", "status"},
		[][]string{
			{"", "dataset", "mysql", "名称为空"},    // 名称为空
			{"有效资产", "", "s3", "类型为空"},       // 类型为空
			{"有效资产2", "report", "api", "有效描述"}, // 有效行
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	result, err := svc.ImportAssets(ctx, buf.Bytes(), "tenant_import_002")
	if err != nil {
		t.Fatalf("导入资产失败: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("期望总数3, got %d", result.Total)
	}
	if result.Success != 1 {
		t.Fatalf("期望成功1, got %d", result.Success)
	}
	if result.Failed != 2 {
		t.Fatalf("期望失败2, got %d", result.Failed)
	}
}

// ==================== ExportRules Tests ====================

func TestExportRules(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	rules := []datagov.DataQualityRule{
		{
			BaseModel:   base.BaseModel{TenantID: "tenant_excel_001"},
			Name:        "非空检查",
			Type:        "not_null",
			TargetField: "email",
			Description: "邮箱不能为空",
			Severity:    "critical",
			Status:      "active",
		},
		{
			BaseModel:   base.BaseModel{TenantID: "tenant_excel_001"},
			Name:        "唯一性检查",
			Type:        "unique",
			TargetField: "username",
			Description: "用户名必须唯一",
			Severity:    "warning",
			Status:      "active",
		},
	}

	buf, err := svc.ExportRules(ctx, rules, nil)
	if err != nil {
		t.Fatalf("导出规则失败: %v", err)
	}
	if buf == nil || buf.Len() == 0 {
		t.Fatal("导出结果不应为空")
	}

	// 验证导出内容
	f, err := excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatalf("读取行数据失败: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("期望3行(1表头+2数据), got %d", len(rows))
	}

	// 验证表头
	if rows[0][0] != "name" {
		t.Fatalf("表头第一列应为 name, got: %s", rows[0][0])
	}
}

// ==================== ImportRules Tests ====================

func TestImportRules_Success(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type", "target_field", "description", "severity", "status"},
		[][]string{
			{"非空检查", "not_null", "email", "邮箱不能为空", "critical", "active"},
			{"唯一性检查", "unique", "username", "用户名唯一", "warning", "active"},
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	result, err := svc.ImportRules(ctx, buf.Bytes(), "tenant_import_001")
	if err != nil {
		t.Fatalf("导入规则失败: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("期望总数2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Fatalf("期望成功2, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Fatalf("期望失败0, got %d", result.Failed)
	}
}

func TestImportRules_Validation(t *testing.T) {
	svc, _ := setupTestExcelService(t)
	ctx := context.Background()

	buf, err := createTestExcelFile(
		[]string{"name", "type", "target_field", "description", "severity", "status"},
		[][]string{
			{"", "not_null", "email", "名称为空"},       // 名称为空
			{"有效规则", "", "username", "类型为空"},     // 类型为空
			{"有效规则2", "range", "age", "有效描述"},    // 有效行
		},
	)
	if err != nil {
		t.Fatalf("创建测试Excel文件失败: %v", err)
	}

	result, err := svc.ImportRules(ctx, buf.Bytes(), "tenant_import_002")
	if err != nil {
		t.Fatalf("导入规则失败: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("期望总数3, got %d", result.Total)
	}
	if result.Success != 1 {
		t.Fatalf("期望成功1, got %d", result.Success)
	}
	if result.Failed != 2 {
		t.Fatalf("期望失败2, got %d", result.Failed)
	}
}
