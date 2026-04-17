package persistence

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDatagovTestDB 创建数据治理 Repository 测试数据库
func setupDatagovTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&datagov.DataPolicy{}, &datagov.DataAsset{}, &datagov.DataQualityRule{}, &datagov.DataQualityCheck{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// ==================== 测试辅助函数 ====================

// createTestDataPolicy 创建测试用数据策略
func createTestDataPolicy(t *testing.T, db *gorm.DB, tenantID, name, typ, status, scope string) *datagov.DataPolicy {
	t.Helper()
	policy := &datagov.DataPolicy{
		BaseModel:   base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Name:        name,
		Type:        typ,
		Status:      status,
		Scope:       scope,
		Description: "测试描述",
	}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("创建测试数据策略失败: %v", err)
	}
	return policy
}

// createTestDataAsset 创建测试用数据资产
func createTestDataAsset(t *testing.T, db *gorm.DB, tenantID, name, typ, status, classification, source string) *datagov.DataAsset {
	t.Helper()
	asset := &datagov.DataAsset{
		BaseModel:     base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Name:          name,
		Type:          typ,
		Status:        status,
		Classification: classification,
		Source:        source,
		Description:   "测试描述",
	}
	if err := db.Create(asset).Error; err != nil {
		t.Fatalf("创建测试数据资产失败: %v", err)
	}
	return asset
}

// createTestDataQualityRule 创建测试用数据质量规则
func createTestDataQualityRule(t *testing.T, db *gorm.DB, tenantID, name, typ, status, severity, targetAsset string) *datagov.DataQualityRule {
	t.Helper()
	rule := &datagov.DataQualityRule{
		BaseModel:   base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		Name:        name,
		Type:        typ,
		Status:      status,
		Severity:    severity,
		TargetAsset: targetAsset,
		Description: "测试描述",
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("创建测试数据质量规则失败: %v", err)
	}
	return rule
}

// createTestDataQualityCheck 创建测试用数据质量检查
func createTestDataQualityCheck(t *testing.T, db *gorm.DB, tenantID, ruleID, assetID, status, triggeredBy string) *datagov.DataQualityCheck {
	t.Helper()
	check := &datagov.DataQualityCheck{
		BaseModel:   base.BaseModel{TenantID: tenantID, ID: base.GenerateUUID()},
		RuleID:      ruleID,
		AssetID:     assetID,
		Status:      status,
		TriggeredBy: triggeredBy,
		Result:      base.JSON{"message": "测试结果"},
	}
	if err := db.Create(check).Error; err != nil {
		t.Fatalf("创建测试数据质量检查失败: %v", err)
	}
	return check
}

// ==================== DataPolicyRepository 测试 ====================

// TestDataPolicyRepository_Create 创建数据策略成功
func TestDataPolicyRepository_Create(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	policy := &datagov.DataPolicy{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Name:        "数据保留策略",
		Type:        "retention",
		Status:      "active",
		Scope:       "global",
		Description: "全局数据保留策略",
	}

	err := repo.Create(context.Background(), policy)
	if err != nil {
		t.Fatalf("创建数据策略失败: %v", err)
	}
	if policy.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if policy.Name != "数据保留策略" {
		t.Errorf("期望 Name 为 '数据保留策略'，实际为 '%s'", policy.Name)
	}
}

// TestDataPolicyRepository_GetByID_Success 查询存在的数据策略
func TestDataPolicyRepository_GetByID_Success(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	created := createTestDataPolicy(t, db, "tenant-001", "测试策略", "retention", "active", "global")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询数据策略失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 为 '%s'，实际为 '%s'", created.ID, found.ID)
	}
	if found.Name != "测试策略" {
		t.Errorf("期望 Name 为 '测试策略'，实际为 '%s'", found.Name)
	}
}

// TestDataPolicyRepository_GetByID_NotFound 查询不存在的数据策略
func TestDataPolicyRepository_GetByID_NotFound(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的数据策略返回错误")
	}
}

// TestDataPolicyRepository_List_All 无过滤返回全部
func TestDataPolicyRepository_List_All(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	createTestDataPolicy(t, db, "tenant-001", "策略1", "retention", "active", "global")
	createTestDataPolicy(t, db, "tenant-001", "策略2", "access", "active", "project")

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(policies) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(policies))
	}
}

// TestDataPolicyRepository_List_FilterByType 按类型过滤
func TestDataPolicyRepository_List_FilterByType(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	createTestDataPolicy(t, db, "tenant-001", "策略1", "retention", "active", "global")
	createTestDataPolicy(t, db, "tenant-001", "策略2", "access", "active", "project")
	createTestDataPolicy(t, db, "tenant-001", "策略3", "retention", "inactive", "global")

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-001", Type: "retention"})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, p := range policies {
		if p.Type != "retention" {
			t.Errorf("期望类型为 retention，实际为 '%s'", p.Type)
		}
	}
}

// TestDataPolicyRepository_List_FilterByStatus 按状态过滤
func TestDataPolicyRepository_List_FilterByStatus(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	createTestDataPolicy(t, db, "tenant-001", "策略1", "retention", "active", "global")
	createTestDataPolicy(t, db, "tenant-001", "策略2", "access", "inactive", "project")
	createTestDataPolicy(t, db, "tenant-001", "策略3", "masking", "draft", "global")

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-001", Status: "active"})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(policies) != 1 || policies[0].Status != "active" {
		t.Error("期望返回状态为 active 的数据策略")
	}
}

// TestDataPolicyRepository_List_FilterByScope 按范围过滤
func TestDataPolicyRepository_List_FilterByScope(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	createTestDataPolicy(t, db, "tenant-001", "策略1", "retention", "active", "global")
	createTestDataPolicy(t, db, "tenant-001", "策略2", "access", "active", "project")
	createTestDataPolicy(t, db, "tenant-001", "策略3", "masking", "active", "project")

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-001", Scope: "project"})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, p := range policies {
		if p.Scope != "project" {
			t.Errorf("期望范围为 project，实际为 '%s'", p.Scope)
		}
	}
}

// TestDataPolicyRepository_List_FilterByKeyword 按关键词搜索
func TestDataPolicyRepository_List_FilterByKeyword(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	createTestDataPolicy(t, db, "tenant-001", "数据保留策略", "retention", "active", "global")
	createTestDataPolicy(t, db, "tenant-001", "数据访问策略", "access", "active", "project")
	createTestDataPolicy(t, db, "tenant-001", "脱敏规则", "masking", "active", "global")

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-001", Keyword: "保留"})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(policies) != 1 || policies[0].Name != "数据保留策略" {
		t.Error("期望返回名称包含 '保留' 的数据策略")
	}
}

// TestDataPolicyRepository_List_WithPagination 分页正确
func TestDataPolicyRepository_List_WithPagination(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	for i := 1; i <= 5; i++ {
		createTestDataPolicy(t, db, "tenant-001", "策略"+string(rune('0'+i)), "retention", "active", "global")
	}

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(policies) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(policies))
	}
}

// TestDataPolicyRepository_List_EmptyResult 空结果
func TestDataPolicyRepository_List_EmptyResult(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	policies, total, err := repo.List(context.Background(), datagovapp.DataPolicyFilter{TenantID: "tenant-empty"})
	if err != nil {
		t.Fatalf("查询数据策略列表失败: %v", err)
	}
	if total != 0 {
		t.Errorf("期望总数为 0，实际为 %d", total)
	}
	if len(policies) != 0 {
		t.Errorf("期望返回 0 条记录，实际为 %d", len(policies))
	}
}

// TestDataPolicyRepository_Update 更新成功
func TestDataPolicyRepository_Update(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	created := createTestDataPolicy(t, db, "tenant-001", "旧名称", "retention", "active", "global")

	created.Name = "新名称"
	created.Status = "inactive"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新数据策略失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的数据策略失败: %v", err)
	}
	if updated.Name != "新名称" {
		t.Errorf("期望 Name 为 '新名称'，实际为 '%s'", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", updated.Status)
	}
}

// TestDataPolicyRepository_Delete 软删除成功
func TestDataPolicyRepository_Delete(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataPolicyRepository(db)

	created := createTestDataPolicy(t, db, "tenant-001", "待删除", "retention", "active", "global")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除数据策略失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// ==================== DataAssetRepository 测试 ====================

// TestDataAssetRepository_Create 创建数据资产成功
func TestDataAssetRepository_Create(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	asset := &datagov.DataAsset{
		BaseModel:     base.BaseModel{TenantID: "tenant-001"},
		Name:          "BOM数据表",
		Type:          "table",
		Status:        "active",
		Classification: "confidential",
		Source:        "ERP",
		Description:   "物料清单数据表",
	}

	err := repo.Create(context.Background(), asset)
	if err != nil {
		t.Fatalf("创建数据资产失败: %v", err)
	}
	if asset.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if asset.Name != "BOM数据表" {
		t.Errorf("期望 Name 为 'BOM数据表'，实际为 '%s'", asset.Name)
	}
}

// TestDataAssetRepository_GetByID_Success 查询存在的数据资产
func TestDataAssetRepository_GetByID_Success(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	created := createTestDataAsset(t, db, "tenant-001", "测试资产", "table", "active", "confidential", "ERP")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询数据资产失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 为 '%s'，实际为 '%s'", created.ID, found.ID)
	}
	if found.Name != "测试资产" {
		t.Errorf("期望 Name 为 '测试资产'，实际为 '%s'", found.Name)
	}
}

// TestDataAssetRepository_GetByID_NotFound 查询不存在的数据资产
func TestDataAssetRepository_GetByID_NotFound(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的数据资产返回错误")
	}
}

// TestDataAssetRepository_List_All 无过滤返回全部
func TestDataAssetRepository_List_All(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	createTestDataAsset(t, db, "tenant-001", "资产1", "table", "active", "confidential", "ERP")
	createTestDataAsset(t, db, "tenant-001", "资产2", "file", "active", "internal", "PLM")

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(assets))
	}
}

// TestDataAssetRepository_List_FilterByType 按类型过滤
func TestDataAssetRepository_List_FilterByType(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	createTestDataAsset(t, db, "tenant-001", "资产1", "table", "active", "confidential", "ERP")
	createTestDataAsset(t, db, "tenant-001", "资产2", "file", "active", "internal", "PLM")
	createTestDataAsset(t, db, "tenant-001", "资产3", "table", "inactive", "public", "MES")

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001", Type: "table"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, a := range assets {
		if a.Type != "table" {
			t.Errorf("期望类型为 table，实际为 '%s'", a.Type)
		}
	}
}

// TestDataAssetRepository_List_FilterByStatus 按状态过滤
func TestDataAssetRepository_List_FilterByStatus(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	createTestDataAsset(t, db, "tenant-001", "资产1", "table", "active", "confidential", "ERP")
	createTestDataAsset(t, db, "tenant-001", "资产2", "file", "inactive", "internal", "PLM")
	createTestDataAsset(t, db, "tenant-001", "资产3", "api", "draft", "public", "MES")

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001", Status: "active"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(assets) != 1 || assets[0].Status != "active" {
		t.Error("期望返回状态为 active 的数据资产")
	}
}

// TestDataAssetRepository_List_FilterByClassification 按分类过滤
func TestDataAssetRepository_List_FilterByClassification(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	createTestDataAsset(t, db, "tenant-001", "资产1", "table", "active", "confidential", "ERP")
	createTestDataAsset(t, db, "tenant-001", "资产2", "file", "active", "internal", "PLM")
	createTestDataAsset(t, db, "tenant-001", "资产3", "api", "active", "confidential", "MES")

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001", Classification: "confidential"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, a := range assets {
		if a.Classification != "confidential" {
			t.Errorf("期望分类为 confidential，实际为 '%s'", a.Classification)
		}
	}
}

// TestDataAssetRepository_List_FilterBySource 按来源过滤
func TestDataAssetRepository_List_FilterBySource(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	createTestDataAsset(t, db, "tenant-001", "资产1", "table", "active", "confidential", "ERP")
	createTestDataAsset(t, db, "tenant-001", "资产2", "file", "active", "internal", "PLM")
	createTestDataAsset(t, db, "tenant-001", "资产3", "api", "active", "public", "ERP")

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001", Source: "ERP"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, a := range assets {
		if a.Source != "ERP" {
			t.Errorf("期望来源为 ERP，实际为 '%s'", a.Source)
		}
	}
}

// TestDataAssetRepository_List_FilterByKeyword 按关键词搜索
func TestDataAssetRepository_List_FilterByKeyword(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	createTestDataAsset(t, db, "tenant-001", "BOM数据表", "table", "active", "confidential", "ERP")
	createTestDataAsset(t, db, "tenant-001", "工艺参数文件", "file", "active", "internal", "PLM")
	createTestDataAsset(t, db, "tenant-001", "质量检测API", "api", "active", "public", "MES")

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001", Keyword: "BOM"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(assets) != 1 || assets[0].Name != "BOM数据表" {
		t.Error("期望返回名称包含 'BOM' 的数据资产")
	}
}

// TestDataAssetRepository_List_WithPagination 分页正确
func TestDataAssetRepository_List_WithPagination(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	for i := 1; i <= 5; i++ {
		createTestDataAsset(t, db, "tenant-001", "资产"+string(rune('0'+i)), "table", "active", "internal", "ERP")
	}

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(assets))
	}
}

// TestDataAssetRepository_List_EmptyResult 空结果
func TestDataAssetRepository_List_EmptyResult(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	assets, total, err := repo.List(context.Background(), datagovapp.DataAssetFilter{TenantID: "tenant-empty"})
	if err != nil {
		t.Fatalf("查询数据资产列表失败: %v", err)
	}
	if total != 0 {
		t.Errorf("期望总数为 0，实际为 %d", total)
	}
	if len(assets) != 0 {
		t.Errorf("期望返回 0 条记录，实际为 %d", len(assets))
	}
}

// TestDataAssetRepository_Update 更新成功
func TestDataAssetRepository_Update(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	created := createTestDataAsset(t, db, "tenant-001", "旧名称", "table", "active", "confidential", "ERP")

	created.Name = "新名称"
	created.Status = "inactive"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新数据资产失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的数据资产失败: %v", err)
	}
	if updated.Name != "新名称" {
		t.Errorf("期望 Name 为 '新名称'，实际为 '%s'", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", updated.Status)
	}
}

// TestDataAssetRepository_Delete 软删除成功
func TestDataAssetRepository_Delete(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataAssetRepository(db)

	created := createTestDataAsset(t, db, "tenant-001", "待删除", "table", "active", "confidential", "ERP")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除数据资产失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// ==================== DataQualityRuleRepository 测试 ====================

// TestDataQualityRuleRepository_Create 创建数据质量规则成功
func TestDataQualityRuleRepository_Create(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	rule := &datagov.DataQualityRule{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Name:        "BOM完整性检查",
		Type:        "completeness",
		Status:      "active",
		Severity:    "high",
		TargetAsset: "asset-001",
		Description: "检查BOM数据完整性",
	}

	err := repo.Create(context.Background(), rule)
	if err != nil {
		t.Fatalf("创建数据质量规则失败: %v", err)
	}
	if rule.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if rule.Name != "BOM完整性检查" {
		t.Errorf("期望 Name 为 'BOM完整性检查'，实际为 '%s'", rule.Name)
	}
}

// TestDataQualityRuleRepository_GetByID_Success 查询存在的数据质量规则
func TestDataQualityRuleRepository_GetByID_Success(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	created := createTestDataQualityRule(t, db, "tenant-001", "测试规则", "completeness", "active", "high", "asset-001")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询数据质量规则失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 为 '%s'，实际为 '%s'", created.ID, found.ID)
	}
	if found.Name != "测试规则" {
		t.Errorf("期望 Name 为 '测试规则'，实际为 '%s'", found.Name)
	}
}

// TestDataQualityRuleRepository_GetByID_NotFound 查询不存在的数据质量规则
func TestDataQualityRuleRepository_GetByID_NotFound(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的数据质量规则返回错误")
	}
}

// TestDataQualityRuleRepository_List_All 无过滤返回全部
func TestDataQualityRuleRepository_List_All(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	createTestDataQualityRule(t, db, "tenant-001", "规则1", "completeness", "active", "high", "asset-001")
	createTestDataQualityRule(t, db, "tenant-001", "规则2", "accuracy", "active", "medium", "asset-002")

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(rules) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(rules))
	}
}

// TestDataQualityRuleRepository_List_FilterByType 按类型过滤
func TestDataQualityRuleRepository_List_FilterByType(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	createTestDataQualityRule(t, db, "tenant-001", "规则1", "completeness", "active", "high", "asset-001")
	createTestDataQualityRule(t, db, "tenant-001", "规则2", "accuracy", "active", "medium", "asset-002")
	createTestDataQualityRule(t, db, "tenant-001", "规则3", "completeness", "inactive", "low", "asset-003")

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001", Type: "completeness"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, r := range rules {
		if r.Type != "completeness" {
			t.Errorf("期望类型为 completeness，实际为 '%s'", r.Type)
		}
	}
}

// TestDataQualityRuleRepository_List_FilterByStatus 按状态过滤
func TestDataQualityRuleRepository_List_FilterByStatus(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	createTestDataQualityRule(t, db, "tenant-001", "规则1", "completeness", "active", "high", "asset-001")
	createTestDataQualityRule(t, db, "tenant-001", "规则2", "accuracy", "inactive", "medium", "asset-002")
	createTestDataQualityRule(t, db, "tenant-001", "规则3", "consistency", "draft", "low", "asset-003")

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001", Status: "active"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(rules) != 1 || rules[0].Status != "active" {
		t.Error("期望返回状态为 active 的数据质量规则")
	}
}

// TestDataQualityRuleRepository_List_FilterBySeverity 按严重程度过滤
func TestDataQualityRuleRepository_List_FilterBySeverity(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	createTestDataQualityRule(t, db, "tenant-001", "规则1", "completeness", "active", "high", "asset-001")
	createTestDataQualityRule(t, db, "tenant-001", "规则2", "accuracy", "active", "medium", "asset-002")
	createTestDataQualityRule(t, db, "tenant-001", "规则3", "consistency", "active", "high", "asset-003")

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001", Severity: "high"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, r := range rules {
		if r.Severity != "high" {
			t.Errorf("期望严重程度为 high，实际为 '%s'", r.Severity)
		}
	}
}

// TestDataQualityRuleRepository_List_FilterByTargetAsset 按目标资产过滤
func TestDataQualityRuleRepository_List_FilterByTargetAsset(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	createTestDataQualityRule(t, db, "tenant-001", "规则1", "completeness", "active", "high", "asset-001")
	createTestDataQualityRule(t, db, "tenant-001", "规则2", "accuracy", "active", "medium", "asset-002")
	createTestDataQualityRule(t, db, "tenant-001", "规则3", "consistency", "active", "low", "asset-001")

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001", TargetAsset: "asset-001"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, r := range rules {
		if r.TargetAsset != "asset-001" {
			t.Errorf("期望目标资产为 asset-001，实际为 '%s'", r.TargetAsset)
		}
	}
}

// TestDataQualityRuleRepository_List_FilterByKeyword 按关键词搜索
func TestDataQualityRuleRepository_List_FilterByKeyword(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	createTestDataQualityRule(t, db, "tenant-001", "BOM完整性检查", "completeness", "active", "high", "asset-001")
	createTestDataQualityRule(t, db, "tenant-001", "数据准确性验证", "accuracy", "active", "medium", "asset-002")
	createTestDataQualityRule(t, db, "tenant-001", "一致性校验", "consistency", "active", "low", "asset-003")

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001", Keyword: "BOM"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(rules) != 1 || rules[0].Name != "BOM完整性检查" {
		t.Error("期望返回名称包含 'BOM' 的数据质量规则")
	}
}

// TestDataQualityRuleRepository_List_WithPagination 分页正确
func TestDataQualityRuleRepository_List_WithPagination(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	for i := 1; i <= 5; i++ {
		createTestDataQualityRule(t, db, "tenant-001", "规则"+string(rune('0'+i)), "completeness", "active", "high", "asset-001")
	}

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(rules) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(rules))
	}
}

// TestDataQualityRuleRepository_List_EmptyResult 空结果
func TestDataQualityRuleRepository_List_EmptyResult(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	rules, total, err := repo.List(context.Background(), datagovapp.DataQualityRuleFilter{TenantID: "tenant-empty"})
	if err != nil {
		t.Fatalf("查询数据质量规则列表失败: %v", err)
	}
	if total != 0 {
		t.Errorf("期望总数为 0，实际为 %d", total)
	}
	if len(rules) != 0 {
		t.Errorf("期望返回 0 条记录，实际为 %d", len(rules))
	}
}

// TestDataQualityRuleRepository_Update 更新成功
func TestDataQualityRuleRepository_Update(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	created := createTestDataQualityRule(t, db, "tenant-001", "旧名称", "completeness", "active", "high", "asset-001")

	created.Name = "新名称"
	created.Status = "inactive"
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新数据质量规则失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的数据质量规则失败: %v", err)
	}
	if updated.Name != "新名称" {
		t.Errorf("期望 Name 为 '新名称'，实际为 '%s'", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", updated.Status)
	}
}

// TestDataQualityRuleRepository_Delete 软删除成功
func TestDataQualityRuleRepository_Delete(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityRuleRepository(db)

	created := createTestDataQualityRule(t, db, "tenant-001", "待删除", "completeness", "active", "high", "asset-001")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除数据质量规则失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// ==================== DataQualityCheckRepository 测试 ====================

// TestDataQualityCheckRepository_Create 创建数据质量检查成功
func TestDataQualityCheckRepository_Create(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	check := &datagov.DataQualityCheck{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		RuleID:      "rule-001",
		AssetID:     "asset-001",
		Status:      "passed",
		TriggeredBy: "manual",
		Result:      base.JSON{"message": "检查通过"},
	}

	err := repo.Create(context.Background(), check)
	if err != nil {
		t.Fatalf("创建数据质量检查失败: %v", err)
	}
	if check.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if check.RuleID != "rule-001" {
		t.Errorf("期望 RuleID 为 'rule-001'，实际为 '%s'", check.RuleID)
	}
}

// TestDataQualityCheckRepository_GetByID_Success 查询存在的数据质量检查
func TestDataQualityCheckRepository_GetByID_Success(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	created := createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")

	found, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询数据质量检查失败: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("期望 ID 为 '%s'，实际为 '%s'", created.ID, found.ID)
	}
	if found.Status != "passed" {
		t.Errorf("期望 Status 为 'passed'，实际为 '%s'", found.Status)
	}
}

// TestDataQualityCheckRepository_GetByID_NotFound 查询不存在的数据质量检查
func TestDataQualityCheckRepository_GetByID_NotFound(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	_, err := repo.GetByID(context.Background(), "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的数据质量检查返回错误")
	}
}

// TestDataQualityCheckRepository_List_All 无过滤返回全部
func TestDataQualityCheckRepository_List_All(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-002", "asset-002", "failed", "schedule")

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-001"})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	if len(checks) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(checks))
	}
}

// TestDataQualityCheckRepository_List_FilterByRuleID 按规则ID过滤
func TestDataQualityCheckRepository_List_FilterByRuleID(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-002", "asset-002", "failed", "schedule")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-003", "passed", "manual")

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-001", RuleID: "rule-001"})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, c := range checks {
		if c.RuleID != "rule-001" {
			t.Errorf("期望 RuleID 为 rule-001，实际为 '%s'", c.RuleID)
		}
	}
}

// TestDataQualityCheckRepository_List_FilterByAssetID 按资产ID过滤
func TestDataQualityCheckRepository_List_FilterByAssetID(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-002", "asset-001", "failed", "schedule")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-003", "asset-002", "passed", "manual")

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-001", AssetID: "asset-001"})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望总数为 2，实际为 %d", total)
	}
	for _, c := range checks {
		if c.AssetID != "asset-001" {
			t.Errorf("期望 AssetID 为 asset-001，实际为 '%s'", c.AssetID)
		}
	}
}

// TestDataQualityCheckRepository_List_FilterByStatus 按状态过滤
func TestDataQualityCheckRepository_List_FilterByStatus(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-002", "asset-002", "failed", "schedule")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-003", "asset-003", "running", "manual")

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-001", Status: "passed"})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(checks) != 1 || checks[0].Status != "passed" {
		t.Error("期望返回状态为 passed 的数据质量检查")
	}
}

// TestDataQualityCheckRepository_List_FilterByTriggeredBy 按触发方式过滤
func TestDataQualityCheckRepository_List_FilterByTriggeredBy(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-002", "asset-002", "failed", "schedule")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-003", "asset-003", "passed", "manual")

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-001", TriggeredBy: "schedule"})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", total)
	}
	if len(checks) != 1 || checks[0].TriggeredBy != "schedule" {
		t.Error("期望返回触发方式为 schedule 的数据质量检查")
	}
}

// TestDataQualityCheckRepository_List_WithPagination 分页正确
func TestDataQualityCheckRepository_List_WithPagination(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	for i := 1; i <= 5; i++ {
		createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	}

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-001", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", total)
	}
	if len(checks) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(checks))
	}
}

// TestDataQualityCheckRepository_List_EmptyResult 空结果
func TestDataQualityCheckRepository_List_EmptyResult(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	checks, total, err := repo.List(context.Background(), datagovapp.DataQualityCheckFilter{TenantID: "tenant-empty"})
	if err != nil {
		t.Fatalf("查询数据质量检查列表失败: %v", err)
	}
	if total != 0 {
		t.Errorf("期望总数为 0，实际为 %d", total)
	}
	if len(checks) != 0 {
		t.Errorf("期望返回 0 条记录，实际为 %d", len(checks))
	}
}

// TestDataQualityCheckRepository_Update 更新成功
func TestDataQualityCheckRepository_Update(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	created := createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "running", "manual")

	created.Status = "passed"
	created.Result = base.JSON{"message": "检查通过"}
	err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("更新数据质量检查失败: %v", err)
	}

	updated, err := repo.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("查询更新后的数据质量检查失败: %v", err)
	}
	if updated.Status != "passed" {
		t.Errorf("期望 Status 为 'passed'，实际为 '%s'", updated.Status)
	}
	if updated.Result["message"] != "检查通过" {
		t.Errorf("期望 Result 包含 '检查通过'，实际为 '%v'", updated.Result)
	}
}

// TestDataQualityCheckRepository_Delete 软删除成功
func TestDataQualityCheckRepository_Delete(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	created := createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")

	err := repo.Delete(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("删除数据质量检查失败: %v", err)
	}

	_, err = repo.GetByID(context.Background(), created.ID)
	if err == nil {
		t.Error("期望软删除后查询返回错误")
	}
}

// TestDataQualityCheckRepository_GetByRuleAndAsset_Success 根据规则和资产查询最新检查
func TestDataQualityCheckRepository_GetByRuleAndAsset_Success(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")
	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "failed", "schedule")

	found, err := repo.GetByRuleAndAsset(context.Background(), "rule-001", "asset-001")
	if err != nil {
		t.Fatalf("查询数据质量检查失败: %v", err)
	}
	if found.RuleID != "rule-001" {
		t.Errorf("期望 RuleID 为 'rule-001'，实际为 '%s'", found.RuleID)
	}
	if found.AssetID != "asset-001" {
		t.Errorf("期望 AssetID 为 'asset-001'，实际为 '%s'", found.AssetID)
	}
	// 应该返回最新创建的记录
	if found.Status != "failed" {
		t.Errorf("期望返回最新的检查记录，Status 为 'failed'，实际为 '%s'", found.Status)
	}
}

// TestDataQualityCheckRepository_GetByRuleAndAsset_NotFound 查询不存在的规则和资产组合
func TestDataQualityCheckRepository_GetByRuleAndAsset_NotFound(t *testing.T) {
	db := setupDatagovTestDB(t)
	repo := NewDataQualityCheckRepository(db)

	createTestDataQualityCheck(t, db, "tenant-001", "rule-001", "asset-001", "passed", "manual")

	_, err := repo.GetByRuleAndAsset(context.Background(), "rule-999", "asset-999")
	if err == nil {
		t.Fatal("期望查询不存在的规则和资产组合返回错误")
	}
}
