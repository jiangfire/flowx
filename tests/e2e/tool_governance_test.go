package e2e

import (
	"context"
	"errors"
	"strings"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	domaingov "git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	bizerrors "git.neolidy.top/neo/flowx/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupE2EDB 创建 E2E 测试用的 SQLite 内存数据库，并自动迁移所有相关表
func setupE2EDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建 E2E 测试数据库失败: %v", err)
	}
	// 迁移所有需要的领域模型表
	if err := db.AutoMigrate(
		&tool.Tool{},
		&tool.Connector{},
		&domaingov.DataPolicy{},
		&domaingov.DataAsset{},
		&domaingov.DataQualityRule{},
		&domaingov.DataQualityCheck{},
	); err != nil {
		t.Fatalf("E2E 数据库迁移失败: %v", err)
	}
	return db
}

// setupE2EService 创建完整的 E2E 测试环境，包含所有真实仓储和服务
func setupE2EService(t *testing.T) (*toolapp.ToolService, *gorm.DB) {
	t.Helper()
	db := setupE2EDB(t)

	// 创建所有真实持久化仓储
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	policyRepo := persistence.NewDataPolicyRepository(db)
	assetRepo := persistence.NewDataAssetRepository(db)
	ruleRepo := persistence.NewDataQualityRuleRepository(db)
	checkRepo := persistence.NewDataQualityCheckRepository(db)

	// 创建工具服务（注入所有仓储，不使用任何 mock）
	svc := toolapp.NewToolService(toolRepo, connectorRepo, policyRepo, assetRepo, ruleRepo, checkRepo)
	return svc, db
}

// createTestPolicy 辅助函数：通过 GORM 直接创建数据策略
func createTestPolicy(t *testing.T, db *gorm.DB, policy *domaingov.DataPolicy) {
	t.Helper()
	if policy.ID == "" {
		policy.ID = base.GenerateUUID()
	}
	if err := db.Create(policy).Error; err != nil {
		t.Fatalf("创建测试策略失败: %v", err)
	}
}

// createTestQualityRule 辅助函数：通过 GORM 直接创建数据质量规则
func createTestQualityRule(t *testing.T, db *gorm.DB, rule *domaingov.DataQualityRule) {
	t.Helper()
	if rule.ID == "" {
		rule.ID = base.GenerateUUID()
	}
	if err := db.Create(rule).Error; err != nil {
		t.Fatalf("创建测试质量规则失败: %v", err)
	}
}

// findAssetBySourceID 辅助函数：从数据资产列表中按 SourceID 查找
func findAssetBySourceID(assets []domaingov.DataAsset, sourceID string) *domaingov.DataAsset {
	for i := range assets {
		if assets[i].SourceID == sourceID {
			return &assets[i]
		}
	}
	return nil
}

// ==================== TestE2E_ToolGovernance_FullLifecycle ====================

// TestE2E_ToolGovernance_FullLifecycle 完整的工具治理生命周期 E2E 测试
// 覆盖：创建（策略校验通过 -> 资产注册 -> 质量检查触发）、策略违规、更新同步、删除归档
func TestE2E_ToolGovernance_FullLifecycle(t *testing.T) {
	svc, db := setupE2EService(t)
	ctx := context.Background()
	tenantID := "tenant-e2e-full"

	// ========== 前置准备：创建策略和质量规则 ==========
	// 创建全局质量策略：要求 endpoint 字段必填
	createTestPolicy(t, db, &domaingov.DataPolicy{
		BaseModel: base.BaseModel{TenantID: tenantID},
		Name:      "全局质量策略-必填endpoint", Type: "quality", Scope: "global",
		Priority: 10, Status: "active",
		Rules: base.JSON{"required_fields": []any{"endpoint"}},
	})

	// 创建质量规则：匹配 tool_type=eda 的工具
	ruleEDA := &domaingov.DataQualityRule{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        "EDA工具质量检查", Type: "completeness",
		Config:      base.JSON{"tool_type": "eda"},
		Severity:    "warning", Status: "active",
	}
	createTestQualityRule(t, db, ruleEDA)

	// ========== 步骤1：创建带 endpoint 的工具 -> 策略校验通过 ==========
	createdTool, err := svc.CreateTool(ctx, tenantID, &toolapp.CreateToolRequest{
		Name:     "Altium Designer",
		Type:     "eda",
		Endpoint: "http://altium.example.com",
	}, "admin")
	if err != nil {
		t.Fatalf("步骤1失败 - 创建工具应成功: %v", err)
	}
	if createdTool.ID == "" {
		t.Fatal("步骤1失败 - 创建后工具 ID 不应为空")
	}
	t.Logf("步骤1通过 - 工具创建成功, ID=%s", createdTool.ID)

	// ========== 步骤2：验证 DataAsset 自动注册 ==========
	assetRepo := persistence.NewDataAssetRepository(db)
	assets, _, err := assetRepo.List(ctx, datagovapp.DataAssetFilter{
		TenantID: tenantID,
		Source:   "tool",
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("步骤2失败 - 查询数据资产失败: %v", err)
	}
	asset := findAssetBySourceID(assets, createdTool.ID)
	if asset == nil {
		t.Fatalf("步骤2失败 - 未找到工具 %s 对应的数据资产", createdTool.ID)
	}
	if !strings.Contains(asset.Name, createdTool.Name) {
		t.Errorf("步骤2失败 - 资产名称 '%s' 应包含工具名称 '%s'", asset.Name, createdTool.Name)
	}
	if asset.Status != "active" {
		t.Errorf("步骤2失败 - 资产状态应为 'active'，实际为 '%s'", asset.Status)
	}
	if asset.Source != "tool" {
		t.Errorf("步骤2失败 - 资产来源应为 'tool'，实际为 '%s'", asset.Source)
	}
	t.Logf("步骤2通过 - 数据资产自动注册成功, 资产名=%s", asset.Name)

	// ========== 步骤3：验证 DataQualityCheck 自动触发 ==========
	checkRepo := persistence.NewDataQualityCheckRepository(db)
	checks, _, err := checkRepo.List(ctx, datagovapp.DataQualityCheckFilter{
		TenantID: tenantID,
		AssetID:  createdTool.ID,
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("步骤3失败 - 查询质量检查记录失败: %v", err)
	}
	if len(checks) == 0 {
		t.Fatalf("步骤3失败 - 未找到工具 %s 对应的质量检查记录", createdTool.ID)
	}
	foundCheck := false
	for _, c := range checks {
		if c.RuleID == ruleEDA.ID && c.AssetID == createdTool.ID {
			foundCheck = true
			if c.Status != "passed" {
				t.Errorf("步骤3失败 - 质量检查状态应为 'passed'，实际为 '%s'", c.Status)
			}
			if c.TriggeredBy != "auto" {
				t.Errorf("步骤3失败 - 触发方式应为 'auto'，实际为 '%s'", c.TriggeredBy)
			}
			break
		}
	}
	if !foundCheck {
		t.Fatal("步骤3失败 - 未找到匹配规则ID和质量检查记录")
	}
	t.Logf("步骤3通过 - 质量检查自动触发成功, 共 %d 条检查记录", len(checks))

	// ========== 步骤4：创建不带 endpoint 的工具 -> 策略违规 ==========
	_, err = svc.CreateTool(ctx, tenantID, &toolapp.CreateToolRequest{
		Name: "NoEndpointTool",
		Type: "eda",
		// 故意不填 Endpoint
	}, "admin")
	if err == nil {
		t.Fatal("步骤4失败 - 缺少 endpoint 时应返回策略违规错误")
	}
	var policyErr *bizerrors.PolicyViolationError
	if !errors.As(err, &policyErr) {
		t.Fatalf("步骤4失败 - 期望 PolicyViolationError，实际为 %T: %v", err, err)
	}
	if len(policyErr.Violations) == 0 {
		t.Fatal("步骤4失败 - 违规记录不应为空")
	}
	t.Logf("步骤4通过 - 策略违规正确拦截, 违规信息: %s", policyErr.Violations[0].Message)

	// ========== 步骤5：更新工具 -> 验证 DataAsset 同步更新 ==========
	updatedName := "Altium Designer Pro"
	_, err = svc.UpdateTool(ctx, tenantID, createdTool.ID, &toolapp.UpdateToolRequest{
		Name: &updatedName,
	}, "admin")
	if err != nil {
		t.Fatalf("步骤5失败 - 更新工具失败: %v", err)
	}
	// 重新查询数据资产，验证名称已同步
	assets, _, err = assetRepo.List(ctx, datagovapp.DataAssetFilter{
		TenantID: tenantID,
		Source:   "tool",
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("步骤5失败 - 查询数据资产失败: %v", err)
	}
	asset = findAssetBySourceID(assets, createdTool.ID)
	if asset == nil {
		t.Fatalf("步骤5失败 - 更新后未找到工具对应的数据资产")
	}
	if !strings.Contains(asset.Name, updatedName) {
		t.Errorf("步骤5失败 - 资产名称 '%s' 应包含更新后的工具名称 '%s'", asset.Name, updatedName)
	}
	t.Logf("步骤5通过 - 工具更新后数据资产同步成功, 新资产名=%s", asset.Name)

	// ========== 步骤6：删除工具 -> 验证 DataAsset 归档 ==========
	err = svc.DeleteTool(ctx, tenantID, createdTool.ID)
	if err != nil {
		t.Fatalf("步骤6失败 - 删除工具失败: %v", err)
	}
	// 重新查询数据资产，验证状态已归档
	assets, _, err = assetRepo.List(ctx, datagovapp.DataAssetFilter{
		TenantID: tenantID,
		Source:   "tool",
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("步骤6失败 - 查询数据资产失败: %v", err)
	}
	asset = findAssetBySourceID(assets, createdTool.ID)
	if asset == nil {
		t.Fatalf("步骤6失败 - 删除后未找到工具对应的数据资产")
	}
	if asset.Status != "archived" {
		t.Errorf("步骤6失败 - 资产状态应为 'archived'，实际为 '%s'", asset.Status)
	}
	t.Logf("步骤6通过 - 工具删除后数据资产已归档")
}

// ==================== TestE2E_ToolGovernance_PolicyBlocksCreate ====================

// TestE2E_ToolGovernance_PolicyBlocksCreate 策略按 tool_type 范围阻断创建
// 验证：匹配 tool_type 的策略会阻断创建，不匹配的 tool_type 不受影响
func TestE2E_ToolGovernance_PolicyBlocksCreate(t *testing.T) {
	svc, db := setupE2EService(t)
	ctx := context.Background()
	tenantID := "tenant-e2e-block"

	// 创建仅适用于 eda 类型工具的策略：要求 description 必填
	createTestPolicy(t, db, &domaingov.DataPolicy{
		BaseModel: base.BaseModel{TenantID: tenantID},
		Name:      "EDA工具描述必填策略", Type: "quality", Scope: "tool_type",
		ScopeValue: "eda",
		Priority:  10, Status: "active",
		Rules: base.JSON{"required_fields": []any{"description"}},
	})

	// ========== 创建 eda 工具且不带 description -> 应被策略阻断 ==========
	_, err := svc.CreateTool(ctx, tenantID, &toolapp.CreateToolRequest{
		Name: "EDA-No-Desc",
		Type: "eda",
	}, "admin")
	if err == nil {
		t.Fatal("期望 eda 工具缺少 description 时策略校验失败")
	}
	var policyErr *bizerrors.PolicyViolationError
	if !errors.As(err, &policyErr) {
		t.Fatalf("期望 PolicyViolationError，实际为 %T: %v", err, err)
	}
	t.Logf("策略正确阻断 eda 工具创建: %s", policyErr.Violations[0].Message)

	// ========== 验证没有为被阻断的工具创建 DataAsset ==========
	assetRepo := persistence.NewDataAssetRepository(db)
	assets, _, err := assetRepo.List(ctx, datagovapp.DataAssetFilter{
		TenantID: tenantID,
		Source:   "tool",
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("查询数据资产失败: %v", err)
	}
	// 被阻断的工具不应该有任何关联资产
	for _, a := range assets {
		if strings.Contains(a.Name, "EDA-No-Desc") {
			t.Errorf("被策略阻断的工具不应有数据资产，但找到了: %s", a.Name)
		}
	}
	t.Logf("验证通过 - 被阻断的工具没有创建数据资产")

	// ========== 创建 cae 工具且不带 description -> 应成功（策略不适用） ==========
	caeTool, err := svc.CreateTool(ctx, tenantID, &toolapp.CreateToolRequest{
		Name: "CAE-No-Desc",
		Type: "cae",
	}, "admin")
	if err != nil {
		t.Fatalf("cae 工具不受策略约束，应创建成功，实际错误: %v", err)
	}
	if caeTool == nil || caeTool.ID == "" {
		t.Fatal("cae 工具创建后 ID 不应为空")
	}
	t.Logf("cae 工具不受策略影响，创建成功, ID=%s", caeTool.ID)
}

// ==================== TestE2E_ToolGovernance_MultipleQualityRules ====================

// TestE2E_ToolGovernance_MultipleQualityRules 多条质量规则匹配时触发多次质量检查
// 验证：当多个质量规则都匹配同一工具时，每个规则都会触发独立的质量检查记录
func TestE2E_ToolGovernance_MultipleQualityRules(t *testing.T) {
	svc, db := setupE2EService(t)
	ctx := context.Background()
	tenantID := "tenant-e2e-multi-rule"

	// 创建两条都匹配 tool_type=eda 的质量规则
	rule1 := &domaingov.DataQualityRule{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        "EDA完整性检查", Type: "completeness",
		Config:      base.JSON{"tool_type": "eda"},
		Severity:    "warning", Status: "active",
	}
	rule2 := &domaingov.DataQualityRule{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        "EDA配置检查", Type: "format",
		Config:      base.JSON{"tool_type": "eda"},
		Severity:    "critical", Status: "active",
	}
	createTestQualityRule(t, db, rule1)
	createTestQualityRule(t, db, rule2)

	// ========== 创建 eda 工具 ==========
	createdTool, err := svc.CreateTool(ctx, tenantID, &toolapp.CreateToolRequest{
		Name: "MultiRuleTool",
		Type: "eda",
	}, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// ========== 验证产生了 2 条质量检查记录 ==========
	checkRepo := persistence.NewDataQualityCheckRepository(db)
	checks, _, err := checkRepo.List(ctx, datagovapp.DataQualityCheckFilter{
		TenantID: tenantID,
		AssetID:  createdTool.ID,
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("查询质量检查记录失败: %v", err)
	}
	if len(checks) != 2 {
		t.Errorf("期望产生 2 条质量检查记录，实际为 %d", len(checks))
	}

	// 验证两条记录分别对应不同的规则
	ruleIDs := make(map[string]bool)
	for _, c := range checks {
		ruleIDs[c.RuleID] = true
		if c.Status != "passed" {
			t.Errorf("质量检查状态应为 'passed'，实际为 '%s'，规则ID=%s", c.Status, c.RuleID)
		}
		if c.TriggeredBy != "auto" {
			t.Errorf("触发方式应为 'auto'，实际为 '%s'，规则ID=%s", c.TriggeredBy, c.RuleID)
		}
	}
	if !ruleIDs[rule1.ID] {
		t.Errorf("缺少规则 '%s' 对应的质量检查记录", rule1.Name)
	}
	if !ruleIDs[rule2.ID] {
		t.Errorf("缺少规则 '%s' 对应的质量检查记录", rule2.Name)
	}
	t.Logf("验证通过 - 两条质量规则均触发了独立的质量检查")
}

// ==================== TestE2E_ToolGovernance_ImportWithPolicyValidation ====================

// TestE2E_ToolGovernance_ImportWithPolicyValidation 批量导入时策略校验的原子性
// 验证：批量导入中任何一个工具违反策略，整个批次都不会被创建
func TestE2E_ToolGovernance_ImportWithPolicyValidation(t *testing.T) {
	svc, db := setupE2EService(t)
	ctx := context.Background()
	tenantID := "tenant-e2e-import"

	// 创建全局质量策略：要求 endpoint 必填
	createTestPolicy(t, db, &domaingov.DataPolicy{
		BaseModel: base.BaseModel{TenantID: tenantID},
		Name:      "导入策略-必填endpoint", Type: "quality", Scope: "global",
		Priority: 10, Status: "active",
		Rules: base.JSON{"required_fields": []any{"endpoint"}},
	})

	// ========== 批量导入 3 个工具：2 个有 endpoint，1 个没有 ==========
	importTools := []*tool.Tool{
		{BaseModel: base.BaseModel{TenantID: tenantID}, Name: "Import-OK-1", Type: "eda", Endpoint: "http://a.com", Status: "active"},
		{BaseModel: base.BaseModel{TenantID: tenantID}, Name: "Import-OK-2", Type: "cae", Endpoint: "http://b.com", Status: "active"},
		{BaseModel: base.BaseModel{TenantID: tenantID}, Name: "Import-Bad", Type: "eda", Status: "active"}, // 缺少 endpoint
	}

	_, err := svc.ImportTools(ctx, tenantID, importTools, "admin")
	if err == nil {
		t.Fatal("期望批量导入时策略校验失败")
	}
	var policyErr *bizerrors.PolicyViolationError
	if !errors.As(err, &policyErr) {
		t.Fatalf("期望 PolicyViolationError，实际为 %T: %v", err, err)
	}
	t.Logf("批量导入策略校验失败: %s", policyErr.Error())

	// ========== 验证没有任何工具被创建（原子性） ==========
	toolRepo := persistence.NewToolRepository(db)
	tools, total, err := toolRepo.List(ctx, toolapp.ToolFilter{
		TenantID: tenantID,
		PageSize: 1000,
	})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if total != 0 {
		t.Errorf("策略校验失败后不应创建任何工具，实际创建了 %d 个", total)
	}
	// 也检查具体的工具名称
	for _, tl := range tools {
		if tl.Name == "Import-OK-1" || tl.Name == "Import-OK-2" || tl.Name == "Import-Bad" {
			t.Errorf("工具 '%s' 不应在策略校验失败后被创建", tl.Name)
		}
	}
	t.Logf("验证通过 - 批量导入策略校验失败，所有工具均未创建（原子性保证）")
}
