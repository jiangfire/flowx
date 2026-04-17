package datagov_test

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试用内存 SQLite 数据库
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 自动迁移所有表
	err = db.AutoMigrate(
		&datagov.DataPolicy{},
		&datagov.DataAsset{},
		&datagov.DataQualityRule{},
		&datagov.DataQualityCheck{},
	)
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// setupTestService 创建测试用 DataGovService
func setupTestService(t *testing.T) (*datagovapp.DataGovService, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)

	policyRepo := persistence.NewDataPolicyRepository(db)
	assetRepo := persistence.NewDataAssetRepository(db)
	ruleRepo := persistence.NewDataQualityRuleRepository(db)
	checkRepo := persistence.NewDataQualityCheckRepository(db)

	svc := datagovapp.NewDataGovService(policyRepo, assetRepo, ruleRepo, checkRepo)
	return svc, db
}

const testTenantID = "tenant_test_001"
const otherTenantID = "tenant_test_002"

// ==================== DataPolicy Tests ====================

func TestCreatePolicy_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreatePolicyRequest{
		Name:        "数据保留策略",
		Type:        "retention",
		Description: "测试保留策略",
		Status:      "active",
	}

	policy, err := svc.CreatePolicy(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}
	if policy.ID == "" {
		t.Fatal("策略ID不应为空")
	}
	if policy.Name != "数据保留策略" {
		t.Fatalf("策略名称不匹配: got %s", policy.Name)
	}
	if policy.Type != "retention" {
		t.Fatalf("策略类型不匹配: got %s", policy.Type)
	}
	if policy.Status != "active" {
		t.Fatalf("策略状态不匹配: got %s", policy.Status)
	}
	if policy.Version != 1 {
		t.Fatalf("策略版本不匹配: got %d", policy.Version)
	}
}

func TestCreatePolicy_EmptyName(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreatePolicyRequest{
		Name: "",
		Type: "retention",
	}

	_, err := svc.CreatePolicy(ctx, testTenantID, req)
	if err != datagovapp.ErrPolicyNameRequired {
		t.Fatalf("期望错误 ErrPolicyNameRequired, got: %v", err)
	}
}

func TestCreatePolicy_EmptyType(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreatePolicyRequest{
		Name: "测试策略",
		Type: "",
	}

	_, err := svc.CreatePolicy(ctx, testTenantID, req)
	if err != datagovapp.ErrPolicyTypeRequired {
		t.Fatalf("期望错误 ErrPolicyTypeRequired, got: %v", err)
	}
}

func TestCreatePolicy_DefaultStatus(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreatePolicyRequest{
		Name: "默认状态策略",
		Type: "classification",
	}

	policy, err := svc.CreatePolicy(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}
	if policy.Status != "active" {
		t.Fatalf("默认状态应为 active, got: %s", policy.Status)
	}
}

func TestGetPolicy_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreatePolicyRequest{
		Name: "获取测试策略",
		Type: "quality",
	}
	created, err := svc.CreatePolicy(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}

	policy, err := svc.GetPolicy(ctx, testTenantID, created.ID)
	if err != nil {
		t.Fatalf("获取策略失败: %v", err)
	}
	if policy.ID != created.ID {
		t.Fatal("策略ID不匹配")
	}
	if policy.Name != "获取测试策略" {
		t.Fatalf("策略名称不匹配: got %s", policy.Name)
	}
}

func TestGetPolicy_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.GetPolicy(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrPolicyNotFound {
		t.Fatalf("期望错误 ErrPolicyNotFound, got: %v", err)
	}
}

func TestGetPolicy_TenantMismatch(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreatePolicyRequest{
		Name: "租户A策略",
		Type: "access",
	}
	created, err := svc.CreatePolicy(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}

	_, err = svc.GetPolicy(ctx, otherTenantID, created.ID)
	if err != datagovapp.ErrTenantMismatch {
		t.Fatalf("期望错误 ErrTenantMismatch, got: %v", err)
	}
}

func TestListPolicies_All(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建多条策略
	for i := 0; i < 3; i++ {
		req := &datagovapp.CreatePolicyRequest{
			Name: "策略_" + string(rune('A'+i)),
			Type: "retention",
		}
		_, err := svc.CreatePolicy(ctx, testTenantID, req)
		if err != nil {
			t.Fatalf("创建策略失败: %v", err)
		}
	}

	policies, paginated, err := svc.ListPolicies(ctx, testTenantID, datagovapp.ListPoliciesFilter{})
	if err != nil {
		t.Fatalf("查询策略列表失败: %v", err)
	}
	if len(policies) != 3 {
		t.Fatalf("期望3条策略, got %d", len(policies))
	}
	if paginated.Total != 3 {
		t.Fatalf("期望总数3, got %d", paginated.Total)
	}
}

func TestListPolicies_WithTypeFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建不同类型的策略
	svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{Name: "保留策略", Type: "retention"})
	svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{Name: "分类策略", Type: "classification"})
	svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{Name: "保留策略2", Type: "retention"})

	policies, _, err := svc.ListPolicies(ctx, testTenantID, datagovapp.ListPoliciesFilter{Type: "retention"})
	if err != nil {
		t.Fatalf("查询策略列表失败: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("期望2条retention策略, got %d", len(policies))
	}
}

func TestListPolicies_WithStatusFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{Name: "活跃策略", Type: "retention", Status: "active"})
	svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{Name: "草稿策略", Type: "retention", Status: "draft"})

	policies, _, err := svc.ListPolicies(ctx, testTenantID, datagovapp.ListPoliciesFilter{Status: "draft"})
	if err != nil {
		t.Fatalf("查询策略列表失败: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("期望1条draft策略, got %d", len(policies))
	}
}

func TestListPolicies_WithPagination(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 创建5条策略
	for i := 0; i < 5; i++ {
		svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{
			Name: "分页策略_" + string(rune('A'+i)),
			Type: "retention",
		})
	}

	// 第1页，每页2条
	policies, paginated, err := svc.ListPolicies(ctx, testTenantID, datagovapp.ListPoliciesFilter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("查询策略列表失败: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("期望2条策略, got %d", len(policies))
	}
	if paginated.Total != 5 {
		t.Fatalf("期望总数5, got %d", paginated.Total)
	}
	if paginated.TotalPages != 3 {
		t.Fatalf("期望总页数3, got %d", paginated.TotalPages)
	}
	if paginated.Page != 1 {
		t.Fatalf("期望当前页1, got %d", paginated.Page)
	}
}

func TestUpdatePolicy_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{
		Name: "原始策略",
		Type: "retention",
	})
	if err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}

	newName := "更新后策略"
	newStatus := "inactive"
	updated, err := svc.UpdatePolicy(ctx, testTenantID, created.ID, &datagovapp.UpdatePolicyRequest{
		Name:   &newName,
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("更新策略失败: %v", err)
	}
	if updated.Name != "更新后策略" {
		t.Fatalf("策略名称不匹配: got %s", updated.Name)
	}
	if updated.Status != "inactive" {
		t.Fatalf("策略状态不匹配: got %s", updated.Status)
	}
}

func TestUpdatePolicy_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	newName := "不存在策略"
	_, err := svc.UpdatePolicy(ctx, testTenantID, "nonexistent_id", &datagovapp.UpdatePolicyRequest{
		Name: &newName,
	})
	if err != datagovapp.ErrPolicyNotFound {
		t.Fatalf("期望错误 ErrPolicyNotFound, got: %v", err)
	}
}

func TestDeletePolicy_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreatePolicy(ctx, testTenantID, &datagovapp.CreatePolicyRequest{
		Name: "待删除策略",
		Type: "retention",
	})
	if err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}

	err = svc.DeletePolicy(ctx, testTenantID, created.ID)
	if err != nil {
		t.Fatalf("删除策略失败: %v", err)
	}

	// 验证删除后无法获取
	_, err = svc.GetPolicy(ctx, testTenantID, created.ID)
	if err != datagovapp.ErrPolicyNotFound {
		t.Fatalf("删除后应返回 ErrPolicyNotFound, got: %v", err)
	}
}

func TestDeletePolicy_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	err := svc.DeletePolicy(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrPolicyNotFound {
		t.Fatalf("期望错误 ErrPolicyNotFound, got: %v", err)
	}
}

// ==================== DataAsset Tests ====================

func TestCreateAsset_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateAssetRequest{
		Name:        "测试数据集",
		Type:        "dataset",
		Source:      "eda_system",
		Description: "测试数据集描述",
		Format:      "csv",
	}

	asset, err := svc.CreateAsset(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}
	if asset.ID == "" {
		t.Fatal("资产ID不应为空")
	}
	if asset.Name != "测试数据集" {
		t.Fatalf("资产名称不匹配: got %s", asset.Name)
	}
	if asset.Type != "dataset" {
		t.Fatalf("资产类型不匹配: got %s", asset.Type)
	}
	if asset.Classification != "internal" {
		t.Fatalf("默认分类应为 internal, got: %s", asset.Classification)
	}
	if asset.Status != "active" {
		t.Fatalf("默认状态应为 active, got: %s", asset.Status)
	}
}

func TestCreateAsset_EmptyName(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateAssetRequest{
		Name: "",
		Type: "dataset",
	}

	_, err := svc.CreateAsset(ctx, testTenantID, req)
	if err != datagovapp.ErrAssetNameRequired {
		t.Fatalf("期望错误 ErrAssetNameRequired, got: %v", err)
	}
}

func TestCreateAsset_EmptyType(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateAssetRequest{
		Name: "测试资产",
		Type: "",
	}

	_, err := svc.CreateAsset(ctx, testTenantID, req)
	if err != datagovapp.ErrAssetTypeRequired {
		t.Fatalf("期望错误 ErrAssetTypeRequired, got: %v", err)
	}
}

func TestGetAsset_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "获取测试资产",
		Type: "model",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	asset, err := svc.GetAsset(ctx, testTenantID, created.ID)
	if err != nil {
		t.Fatalf("获取资产失败: %v", err)
	}
	if asset.ID != created.ID {
		t.Fatal("资产ID不匹配")
	}
}

func TestGetAsset_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.GetAsset(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrAssetNotFound {
		t.Fatalf("期望错误 ErrAssetNotFound, got: %v", err)
	}
}

func TestGetAsset_TenantMismatch(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "租户A资产",
		Type: "dataset",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	_, err = svc.GetAsset(ctx, otherTenantID, created.ID)
	if err != datagovapp.ErrTenantMismatch {
		t.Fatalf("期望错误 ErrTenantMismatch, got: %v", err)
	}
}

func TestListAssets_All(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
			Name: "资产_" + string(rune('A'+i)),
			Type: "dataset",
		})
		if err != nil {
			t.Fatalf("创建资产失败: %v", err)
		}
	}

	assets, paginated, err := svc.ListAssets(ctx, testTenantID, datagovapp.ListAssetsFilter{})
	if err != nil {
		t.Fatalf("查询资产列表失败: %v", err)
	}
	if len(assets) != 3 {
		t.Fatalf("期望3条资产, got %d", len(assets))
	}
	if paginated.Total != 3 {
		t.Fatalf("期望总数3, got %d", paginated.Total)
	}
}

func TestListAssets_WithTypeFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{Name: "数据集1", Type: "dataset"})
	svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{Name: "模型1", Type: "model"})
	svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{Name: "数据集2", Type: "dataset"})

	assets, _, err := svc.ListAssets(ctx, testTenantID, datagovapp.ListAssetsFilter{Type: "dataset"})
	if err != nil {
		t.Fatalf("查询资产列表失败: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("期望2条dataset资产, got %d", len(assets))
	}
}

func TestListAssets_WithStatusFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{Name: "活跃资产", Type: "dataset", Status: "active"})
	svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{Name: "归档资产", Type: "dataset", Status: "archived"})

	assets, _, err := svc.ListAssets(ctx, testTenantID, datagovapp.ListAssetsFilter{Status: "archived"})
	if err != nil {
		t.Fatalf("查询资产列表失败: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("期望1条archived资产, got %d", len(assets))
	}
}

func TestListAssets_WithPagination(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
			Name: "分页资产_" + string(rune('A'+i)),
			Type: "dataset",
		})
	}

	assets, paginated, err := svc.ListAssets(ctx, testTenantID, datagovapp.ListAssetsFilter{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("查询资产列表失败: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("期望2条资产, got %d", len(assets))
	}
	if paginated.Total != 5 {
		t.Fatalf("期望总数5, got %d", paginated.Total)
	}
	if paginated.Page != 2 {
		t.Fatalf("期望当前页2, got %d", paginated.Page)
	}
}

func TestUpdateAsset_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "原始资产",
		Type: "dataset",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	newName := "更新后资产"
	newStatus := "archived"
	updated, err := svc.UpdateAsset(ctx, testTenantID, created.ID, &datagovapp.UpdateAssetRequest{
		Name:   &newName,
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("更新资产失败: %v", err)
	}
	if updated.Name != "更新后资产" {
		t.Fatalf("资产名称不匹配: got %s", updated.Name)
	}
	if updated.Status != "archived" {
		t.Fatalf("资产状态不匹配: got %s", updated.Status)
	}
}

func TestDeleteAsset_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "待删除资产",
		Type: "dataset",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	err = svc.DeleteAsset(ctx, testTenantID, created.ID)
	if err != nil {
		t.Fatalf("删除资产失败: %v", err)
	}

	_, err = svc.GetAsset(ctx, testTenantID, created.ID)
	if err != datagovapp.ErrAssetNotFound {
		t.Fatalf("删除后应返回 ErrAssetNotFound, got: %v", err)
	}
}

func TestDeleteAsset_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	err := svc.DeleteAsset(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrAssetNotFound {
		t.Fatalf("期望错误 ErrAssetNotFound, got: %v", err)
	}
}

// ==================== DataQualityRule Tests ====================

func TestCreateRule_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateRuleRequest{
		Name:        "非空检查规则",
		Type:        "not_null",
		Description: "检查字段不为空",
		Severity:    "critical",
	}

	rule, err := svc.CreateRule(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	if rule.ID == "" {
		t.Fatal("规则ID不应为空")
	}
	if rule.Name != "非空检查规则" {
		t.Fatalf("规则名称不匹配: got %s", rule.Name)
	}
	if rule.Type != "not_null" {
		t.Fatalf("规则类型不匹配: got %s", rule.Type)
	}
	if rule.Severity != "critical" {
		t.Fatalf("规则严重级别不匹配: got %s", rule.Severity)
	}
	if rule.Status != "active" {
		t.Fatalf("默认状态应为 active, got: %s", rule.Status)
	}
}

func TestCreateRule_EmptyName(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateRuleRequest{
		Name: "",
		Type: "not_null",
	}

	_, err := svc.CreateRule(ctx, testTenantID, req)
	if err != datagovapp.ErrRuleNameRequired {
		t.Fatalf("期望错误 ErrRuleNameRequired, got: %v", err)
	}
}

func TestCreateRule_EmptyType(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateRuleRequest{
		Name: "测试规则",
		Type: "",
	}

	_, err := svc.CreateRule(ctx, testTenantID, req)
	if err != datagovapp.ErrRuleTypeRequired {
		t.Fatalf("期望错误 ErrRuleTypeRequired, got: %v", err)
	}
}

func TestCreateRule_DefaultSeverity(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	req := &datagovapp.CreateRuleRequest{
		Name: "默认严重级别规则",
		Type: "unique",
	}

	rule, err := svc.CreateRule(ctx, testTenantID, req)
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}
	if rule.Severity != "warning" {
		t.Fatalf("默认严重级别应为 warning, got: %s", rule.Severity)
	}
}

func TestGetRule_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "获取测试规则",
		Type: "range",
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	rule, err := svc.GetRule(ctx, testTenantID, created.ID)
	if err != nil {
		t.Fatalf("获取规则失败: %v", err)
	}
	if rule.ID != created.ID {
		t.Fatal("规则ID不匹配")
	}
}

func TestGetRule_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.GetRule(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrQualityRuleNotFound {
		t.Fatalf("期望错误 ErrQualityRuleNotFound, got: %v", err)
	}
}

func TestGetRule_TenantMismatch(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "租户A规则",
		Type: "format",
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	_, err = svc.GetRule(ctx, otherTenantID, created.ID)
	if err != datagovapp.ErrTenantMismatch {
		t.Fatalf("期望错误 ErrTenantMismatch, got: %v", err)
	}
}

func TestListRules_All(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
			Name: "规则_" + string(rune('A'+i)),
			Type: "not_null",
		})
		if err != nil {
			t.Fatalf("创建规则失败: %v", err)
		}
	}

	rules, paginated, err := svc.ListRules(ctx, testTenantID, datagovapp.ListRulesFilter{})
	if err != nil {
		t.Fatalf("查询规则列表失败: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("期望3条规则, got %d", len(rules))
	}
	if paginated.Total != 3 {
		t.Fatalf("期望总数3, got %d", paginated.Total)
	}
}

func TestListRules_WithTypeFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{Name: "非空规则", Type: "not_null"})
	svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{Name: "唯一规则", Type: "unique"})
	svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{Name: "非空规则2", Type: "not_null"})

	rules, _, err := svc.ListRules(ctx, testTenantID, datagovapp.ListRulesFilter{Type: "not_null"})
	if err != nil {
		t.Fatalf("查询规则列表失败: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("期望2条not_null规则, got %d", len(rules))
	}
}

func TestListRules_WithStatusFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{Name: "活跃规则", Type: "not_null", Status: "active"})
	svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{Name: "停用规则", Type: "not_null", Status: "inactive"})

	rules, _, err := svc.ListRules(ctx, testTenantID, datagovapp.ListRulesFilter{Status: "inactive"})
	if err != nil {
		t.Fatalf("查询规则列表失败: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("期望1条inactive规则, got %d", len(rules))
	}
}

func TestListRules_WithPagination(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
			Name: "分页规则_" + string(rune('A'+i)),
			Type: "not_null",
		})
	}

	rules, paginated, err := svc.ListRules(ctx, testTenantID, datagovapp.ListRulesFilter{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("查询规则列表失败: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("期望3条规则, got %d", len(rules))
	}
	if paginated.Total != 5 {
		t.Fatalf("期望总数5, got %d", paginated.Total)
	}
	if paginated.TotalPages != 2 {
		t.Fatalf("期望总页数2, got %d", paginated.TotalPages)
	}
}

func TestUpdateRule_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "原始规则",
		Type: "not_null",
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	newName := "更新后规则"
	newSeverity := "critical"
	updated, err := svc.UpdateRule(ctx, testTenantID, created.ID, &datagovapp.UpdateRuleRequest{
		Name:     &newName,
		Severity: &newSeverity,
	})
	if err != nil {
		t.Fatalf("更新规则失败: %v", err)
	}
	if updated.Name != "更新后规则" {
		t.Fatalf("规则名称不匹配: got %s", updated.Name)
	}
	if updated.Severity != "critical" {
		t.Fatalf("规则严重级别不匹配: got %s", updated.Severity)
	}
}

func TestDeleteRule_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	created, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "待删除规则",
		Type: "not_null",
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	err = svc.DeleteRule(ctx, testTenantID, created.ID)
	if err != nil {
		t.Fatalf("删除规则失败: %v", err)
	}

	_, err = svc.GetRule(ctx, testTenantID, created.ID)
	if err != datagovapp.ErrQualityRuleNotFound {
		t.Fatalf("删除后应返回 ErrQualityRuleNotFound, got: %v", err)
	}
}

func TestDeleteRule_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	err := svc.DeleteRule(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrQualityRuleNotFound {
		t.Fatalf("期望错误 ErrQualityRuleNotFound, got: %v", err)
	}
}

// ==================== DataQualityCheck Tests ====================

func TestGetCheck_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 先创建规则和资产
	rule, _ := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	asset, _ := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "检查资产",
		Type: "dataset",
	})

	// 执行质量检查
	check, err := svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  rule.ID,
		AssetID: asset.ID,
	})
	if err != nil {
		t.Fatalf("执行质量检查失败: %v", err)
	}

	// 获取检查记录
	got, err := svc.GetCheck(ctx, testTenantID, check.ID)
	if err != nil {
		t.Fatalf("获取检查记录失败: %v", err)
	}
	if got.ID != check.ID {
		t.Fatal("检查记录ID不匹配")
	}
	if got.Status != "passed" {
		t.Fatalf("检查状态应为 passed, got: %s", got.Status)
	}
}

func TestGetCheck_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	_, err := svc.GetCheck(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrQualityCheckNotFound {
		t.Fatalf("期望错误 ErrQualityCheckNotFound, got: %v", err)
	}
}

func TestGetCheck_TenantMismatch(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	asset, _ := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "检查资产",
		Type: "dataset",
	})

	check, _ := svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  rule.ID,
		AssetID: asset.ID,
	})

	_, err := svc.GetCheck(ctx, otherTenantID, check.ID)
	if err != datagovapp.ErrTenantMismatch {
		t.Fatalf("期望错误 ErrTenantMismatch, got: %v", err)
	}
}

func TestListChecks_All(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	asset, _ := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "检查资产",
		Type: "dataset",
	})

	// 执行多次检查
	for i := 0; i < 3; i++ {
		_, err := svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
			RuleID:  rule.ID,
			AssetID: asset.ID,
		})
		if err != nil {
			t.Fatalf("执行质量检查失败: %v", err)
		}
	}

	checks, paginated, err := svc.ListChecks(ctx, testTenantID, datagovapp.ListChecksFilter{})
	if err != nil {
		t.Fatalf("查询检查列表失败: %v", err)
	}
	if len(checks) != 3 {
		t.Fatalf("期望3条检查记录, got %d", len(checks))
	}
	if paginated.Total != 3 {
		t.Fatalf("期望总数3, got %d", paginated.Total)
	}
}

func TestListChecks_WithStatusFilter(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	asset, _ := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "检查资产",
		Type: "dataset",
	})

	// 执行多次检查
	svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{RuleID: rule.ID, AssetID: asset.ID})
	svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{RuleID: rule.ID, AssetID: asset.ID})

	checks, _, err := svc.ListChecks(ctx, testTenantID, datagovapp.ListChecksFilter{Status: "passed"})
	if err != nil {
		t.Fatalf("查询检查列表失败: %v", err)
	}
	// 所有检查应该都是 passed
	for _, c := range checks {
		if c.Status != "passed" {
			t.Fatalf("期望状态为 passed, got: %s", c.Status)
		}
	}
}

func TestListChecks_WithPagination(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	asset, _ := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "检查资产",
		Type: "dataset",
	})

	for i := 0; i < 5; i++ {
		svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{RuleID: rule.ID, AssetID: asset.ID})
	}

	checks, paginated, err := svc.ListChecks(ctx, testTenantID, datagovapp.ListChecksFilter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("查询检查列表失败: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("期望2条检查记录, got %d", len(checks))
	}
	if paginated.Total != 5 {
		t.Fatalf("期望总数5, got %d", paginated.Total)
	}
	if paginated.TotalPages != 3 {
		t.Fatalf("期望总页数3, got %d", paginated.TotalPages)
	}
}

func TestDeleteCheck_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, _ := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	asset, _ := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "检查资产",
		Type: "dataset",
	})

	check, _ := svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  rule.ID,
		AssetID: asset.ID,
	})

	err := svc.DeleteCheck(ctx, testTenantID, check.ID)
	if err != nil {
		t.Fatalf("删除检查记录失败: %v", err)
	}

	_, err = svc.GetCheck(ctx, testTenantID, check.ID)
	if err != datagovapp.ErrQualityCheckNotFound {
		t.Fatalf("删除后应返回 ErrQualityCheckNotFound, got: %v", err)
	}
}

func TestDeleteCheck_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	err := svc.DeleteCheck(ctx, testTenantID, "nonexistent_id")
	if err != datagovapp.ErrQualityCheckNotFound {
		t.Fatalf("期望错误 ErrQualityCheckNotFound, got: %v", err)
	}
}

// ==================== RunQualityCheck Business Logic Tests ====================

func TestRunQualityCheck_Success(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "非空检查",
		Type: "not_null",
		Config: base.JSON{
			"fail_rate": 0.1,
		},
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	asset, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "测试数据集",
		Type: "dataset",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	check, err := svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  rule.ID,
		AssetID: asset.ID,
	})
	if err != nil {
		t.Fatalf("执行质量检查失败: %v", err)
	}
	if check.ID == "" {
		t.Fatal("检查记录ID不应为空")
	}
	if check.RuleID != rule.ID {
		t.Fatal("规则ID不匹配")
	}
	if check.AssetID != asset.ID {
		t.Fatal("资产ID不匹配")
	}
	if check.Status != "passed" {
		t.Fatalf("检查状态应为 passed, got status: %s", check.Status)
	}
	if check.PassRate != 100.0 {
		t.Fatalf("通过率应为100.0, got: %f", check.PassRate)
	}
	if check.TriggeredBy != "manual" {
		t.Fatalf("触发方式应为 manual, got: %s", check.TriggeredBy)
	}
}

func TestRunQualityCheck_RuleNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	asset, err := svc.CreateAsset(ctx, testTenantID, &datagovapp.CreateAssetRequest{
		Name: "测试数据集",
		Type: "dataset",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	_, err = svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  "nonexistent_rule_id",
		AssetID: asset.ID,
	})
	if err != datagovapp.ErrQualityRuleNotFound {
		t.Fatalf("期望错误 ErrQualityRuleNotFound, got: %v", err)
	}
}

func TestRunQualityCheck_AssetNotFound(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	rule, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "检查规则",
		Type: "not_null",
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	_, err = svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  rule.ID,
		AssetID: "nonexistent_asset_id",
	})
	if err != datagovapp.ErrAssetNotFound {
		t.Fatalf("期望错误 ErrAssetNotFound, got: %v", err)
	}
}

func TestRunQualityCheck_TenantMismatch(t *testing.T) {
	svc, _ := setupTestService(t)
	ctx := context.Background()

	// 在租户A下创建规则
	rule, err := svc.CreateRule(ctx, testTenantID, &datagovapp.CreateRuleRequest{
		Name: "租户A规则",
		Type: "not_null",
	})
	if err != nil {
		t.Fatalf("创建规则失败: %v", err)
	}

	// 在租户B下创建资产
	asset, err := svc.CreateAsset(ctx, otherTenantID, &datagovapp.CreateAssetRequest{
		Name: "租户B资产",
		Type: "dataset",
	})
	if err != nil {
		t.Fatalf("创建资产失败: %v", err)
	}

	// 用租户A尝试对租户B的资产执行检查
	_, err = svc.RunQualityCheck(ctx, testTenantID, &datagovapp.RunQualityCheckRequest{
		RuleID:  rule.ID,
		AssetID: asset.ID,
	})
	if err != datagovapp.ErrTenantMismatch {
		t.Fatalf("期望错误 ErrTenantMismatch, got: %v", err)
	}
}
