package tool_test

import (
	"context"
	"errors"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	domaingov "git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	bizerrors "git.neolidy.top/neo/flowx/pkg/errors"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockDataPolicyRepo 模拟数据策略仓储
type mockDataPolicyRepo struct {
	policies []domaingov.DataPolicy
}

func (m *mockDataPolicyRepo) Create(ctx context.Context, policy *domaingov.DataPolicy) error {
	return nil
}
func (m *mockDataPolicyRepo) GetByID(ctx context.Context, id string) (*domaingov.DataPolicy, error) {
	return nil, nil
}
func (m *mockDataPolicyRepo) List(ctx context.Context, filter datagovapp.DataPolicyFilter) ([]domaingov.DataPolicy, int64, error) {
	return m.policies, int64(len(m.policies)), nil
}
func (m *mockDataPolicyRepo) Update(ctx context.Context, policy *domaingov.DataPolicy) error {
	return nil
}
func (m *mockDataPolicyRepo) Delete(ctx context.Context, id string) error {
	return nil
}

// setupServiceTestDB 创建服务测试数据库
func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&tool.Tool{}, &tool.Connector{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// setupToolService 创建工具服务测试环境
func setupToolService(t *testing.T) (*toolapp.ToolService, *gorm.DB) {
	t.Helper()
	db := setupServiceTestDB(t)
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	svc := toolapp.NewToolService(toolRepo, connectorRepo, nil, nil, nil, nil)
	return svc, db
}

// TestCreateTool_Success 创建工具成功
func TestCreateTool_Success(t *testing.T) {
	svc, _ := setupToolService(t)

	req := &toolapp.CreateToolRequest{
		Name:   "Altium Designer",
		Type:   "eda",
		Status: "active",
	}

	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}
	if result.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if result.Name != "Altium Designer" {
		t.Errorf("期望 Name 为 'Altium Designer'，实际为 '%s'", result.Name)
	}
	if result.Type != "eda" {
		t.Errorf("期望 Type 为 'eda'，实际为 '%s'", result.Type)
	}
	if result.TenantID != "tenant-001" {
		t.Errorf("期望 TenantID 为 'tenant-001'，实际为 '%s'", result.TenantID)
	}
}

// TestCreateTool_MissingRequired 缺少必填字段返回错误
func TestCreateTool_MissingRequired(t *testing.T) {
	svc, _ := setupToolService(t)

	// 缺少 Name
	req := &toolapp.CreateToolRequest{
		Type: "eda",
	}
	_, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err == nil {
		t.Fatal("期望缺少 Name 时返回错误")
	}

	// 缺少 Type
	req2 := &toolapp.CreateToolRequest{
		Name: "TestTool",
	}
	_, err = svc.CreateTool(context.Background(), "tenant-001", req2, "admin")
	if err == nil {
		t.Fatal("期望缺少 Type 时返回错误")
	}
}

// TestGetTool_Exists 查询存在的工具
func TestGetTool_Exists(t *testing.T) {
	svc, db := setupToolService(t)

	// 先创建一个工具
	toolRepo := persistence.NewToolRepository(db)
	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "TestTool",
		Type:      "eda",
		Status:    "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	result, err := svc.GetTool(context.Background(), "tenant-001", tl.ID)
	if err != nil {
		t.Fatalf("查询工具失败: %v", err)
	}
	if result.Name != "TestTool" {
		t.Errorf("期望 Name 为 'TestTool'，实际为 '%s'", result.Name)
	}
}

// TestGetTool_NotExists 查询不存在的工具
func TestGetTool_NotExists(t *testing.T) {
	svc, _ := setupToolService(t)

	_, err := svc.GetTool(context.Background(), "tenant-001", "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的工具返回错误")
	}
}

// TestListTools_Pagination 分页正确
func TestListTools_Pagination(t *testing.T) {
	svc, db := setupToolService(t)
	toolRepo := persistence.NewToolRepository(db)

	// 创建 5 条记录
	for i := 1; i <= 5; i++ {
		tl := &tool.Tool{
			BaseModel: base.BaseModel{TenantID: "tenant-001"},
			Name:      "Tool" + string(rune('0'+i)),
			Type:      "eda",
			Status:    "active",
		}
		if err := toolRepo.Create(context.Background(), tl); err != nil {
			t.Fatalf("创建工具失败: %v", err)
		}
	}

	tools, paginated, err := svc.ListTools(context.Background(), "tenant-001", toolapp.ListToolsFilter{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("查询工具列表失败: %v", err)
	}
	if paginated.Total != 5 {
		t.Errorf("期望总数为 5，实际为 %d", paginated.Total)
	}
	if len(tools) != 3 {
		t.Errorf("期望返回 3 条记录，实际为 %d", len(tools))
	}
	if paginated.TotalPages != 2 {
		t.Errorf("期望总页数为 2，实际为 %d", paginated.TotalPages)
	}
}

// TestUpdateTool_Success 更新工具成功
func TestUpdateTool_Success(t *testing.T) {
	svc, db := setupToolService(t)
	toolRepo := persistence.NewToolRepository(db)

	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "OldName",
		Type:      "eda",
		Status:    "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	newName := "NewName"
	inactive := "inactive"
	req := &toolapp.UpdateToolRequest{
		Name:   &newName,
		Status: &inactive,
	}

	result, err := svc.UpdateTool(context.Background(), "tenant-001", tl.ID, req, "admin")
	if err != nil {
		t.Fatalf("更新工具失败: %v", err)
	}
	if result.Name != "NewName" {
		t.Errorf("期望 Name 为 'NewName'，实际为 '%s'", result.Name)
	}
	if result.Status != "inactive" {
		t.Errorf("期望 Status 为 'inactive'，实际为 '%s'", result.Status)
	}
}

// TestDeleteTool_Success 删除工具成功
func TestDeleteTool_Success(t *testing.T) {
	svc, db := setupToolService(t)
	toolRepo := persistence.NewToolRepository(db)

	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "ToDelete",
		Type:      "eda",
		Status:    "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	err := svc.DeleteTool(context.Background(), "tenant-001", tl.ID)
	if err != nil {
		t.Fatalf("删除工具失败: %v", err)
	}

	// 确认已删除
	_, err = svc.GetTool(context.Background(), "tenant-001", tl.ID)
	if err == nil {
		t.Error("期望删除后查询返回错误")
	}
}

// TestDeleteTool_NotExists 删除不存在的工具返回错误
func TestDeleteTool_NotExists(t *testing.T) {
	svc, _ := setupToolService(t)

	err := svc.DeleteTool(context.Background(), "tenant-001", "non-existent-id")
	if err == nil {
		t.Fatal("期望删除不存在的工具返回错误")
	}
}

// TestCrossTenantOperation 跨租户操作返回错误
func TestCrossTenantOperation(t *testing.T) {
	svc, db := setupToolService(t)
	toolRepo := persistence.NewToolRepository(db)

	// 租户 A 创建工具
	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-a"},
		Name:      "ToolA",
		Type:      "eda",
		Status:    "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// 租户 B 尝试查询租户 A 的工具
	_, err := svc.GetTool(context.Background(), "tenant-b", tl.ID)
	if err == nil {
		t.Fatal("期望跨租户查询返回错误")
	}

	// 租户 B 尝试更新租户 A 的工具
	hacked := "Hacked"
	_, err = svc.UpdateTool(context.Background(), "tenant-b", tl.ID, &toolapp.UpdateToolRequest{Name: &hacked}, "admin")
	if err == nil {
		t.Fatal("期望跨租户更新返回错误")
	}

	// 租户 B 尝试删除租户 A 的工具
	err = svc.DeleteTool(context.Background(), "tenant-b", tl.ID)
	if err == nil {
		t.Fatal("期望跨租户删除返回错误")
	}
}

// TestCreateConnector_Success 创建连接器成功
func TestCreateConnector_Success(t *testing.T) {
	svc, _ := setupToolService(t)

	req := &toolapp.CreateConnectorRequest{
		Name:     "Windchill",
		Type:     "plm",
		Endpoint: "https://plm.example.com",
	}

	result, err := svc.CreateConnector(context.Background(), "tenant-001", req)
	if err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}
	if result.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if result.Name != "Windchill" {
		t.Errorf("期望 Name 为 'Windchill'，实际为 '%s'", result.Name)
	}
}

// TestListConnectors_Success 列出连接器成功
func TestListConnectors_Success(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "Windchill",
		Type:      "plm",
		Endpoint:  "https://plm.example.com",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	connectors, paginated, err := svc.ListConnectors(context.Background(), "tenant-001", toolapp.ListConnectorsFilter{})
	if err != nil {
		t.Fatalf("查询连接器列表失败: %v", err)
	}
	if paginated.Total != 1 {
		t.Errorf("期望总数为 1，实际为 %d", paginated.Total)
	}
	if len(connectors) != 1 {
		t.Errorf("期望返回 1 条记录，实际为 %d", len(connectors))
	}
}

// TestUpdateConnector_Success 更新连接器成功
func TestUpdateConnector_Success(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "OldName",
		Type:      "plm",
		Endpoint:  "https://old.example.com",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	newName := "NewName"
	req := &toolapp.UpdateConnectorRequest{
		Name: &newName,
	}

	result, err := svc.UpdateConnector(context.Background(), "tenant-001", conn.ID, req)
	if err != nil {
		t.Fatalf("更新连接器失败: %v", err)
	}
	if result.Name != "NewName" {
		t.Errorf("期望 Name 为 'NewName'，实际为 '%s'", result.Name)
	}
}

// TestDeleteConnector_Success 删除连接器成功
func TestDeleteConnector_Success(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "ToDelete",
		Type:      "plm",
		Endpoint:  "https://delete.example.com",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	err := svc.DeleteConnector(context.Background(), "tenant-001", conn.ID)
	if err != nil {
		t.Fatalf("删除连接器失败: %v", err)
	}
}

// ==================== 策略引擎集成测试 ====================

// setupToolServiceWithPolicy 创建带策略仓储的工具服务
func setupToolServiceWithPolicy(t *testing.T, policies []domaingov.DataPolicy) (*toolapp.ToolService, *gorm.DB) {
	t.Helper()
	db := setupServiceTestDB(t)
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	policyRepo := &mockDataPolicyRepo{policies: policies}
	svc := toolapp.NewToolService(toolRepo, connectorRepo, policyRepo, nil, nil, nil)
	return svc, db
}

// TestCreateTool_PolicyViolation 创建工具时策略校验失败
func TestCreateTool_PolicyViolation(t *testing.T) {
	policies := []domaingov.DataPolicy{
		{
			BaseModel: base.BaseModel{ID: "p1", TenantID: "tenant-001"},
			Name:      "require-endpoint", Type: "quality", Scope: "global",
			Priority: 1, Status: "active",
			Rules: base.JSON{"required_fields": []any{"endpoint"}},
		},
	}
	svc, _ := setupToolServiceWithPolicy(t, policies)

	req := &toolapp.CreateToolRequest{
		Name: "NoEndpoint",
		Type: "eda",
	}
	_, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err == nil {
		t.Fatal("期望策略校验失败返回错误")
	}

	var policyErr *bizerrors.PolicyViolationError
	if !isPolicyViolationError(err, &policyErr) {
		t.Fatalf("期望 PolicyViolationError，实际为 %T: %v", err, err)
	}
	if len(policyErr.Violations) == 0 {
		t.Error("期望有违规记录")
	}
	if policyErr.Violations[0].RuleKey != "required_fields" {
		t.Errorf("期望 rule_key 为 'required_fields'，实际为 '%s'", policyErr.Violations[0].RuleKey)
	}
}

// TestCreateTool_PolicyPass 创建工具时策略校验通过
func TestCreateTool_PolicyPass(t *testing.T) {
	policies := []domaingov.DataPolicy{
		{
			BaseModel: base.BaseModel{ID: "p1", TenantID: "tenant-001"},
			Name:      "require-endpoint", Type: "quality", Scope: "global",
			Priority: 1, Status: "active",
			Rules: base.JSON{"required_fields": []any{"endpoint"}},
		},
	}
	svc, _ := setupToolServiceWithPolicy(t, policies)

	req := &toolapp.CreateToolRequest{
		Name:     "WithEndpoint",
		Type:     "eda",
		Endpoint: "http://example.com",
	}
	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("期望策略校验通过，实际错误: %v", err)
	}
	if result.Name != "WithEndpoint" {
		t.Errorf("期望 Name 为 'WithEndpoint'，实际为 '%s'", result.Name)
	}
}

// TestCreateTool_NoPolicyRepo 无策略仓储时正常创建
func TestCreateTool_NoPolicyRepo(t *testing.T) {
	db := setupServiceTestDB(t)
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	svc := toolapp.NewToolService(toolRepo, connectorRepo, nil, nil, nil, nil)

	req := &toolapp.CreateToolRequest{
		Name: "NoPolicyRepo",
		Type: "eda",
	}
	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("期望无策略仓储时正常创建，实际错误: %v", err)
	}
	if result.Name != "NoPolicyRepo" {
		t.Errorf("期望 Name 为 'NoPolicyRepo'，实际为 '%s'", result.Name)
	}
}

// TestUpdateTool_PolicyViolation 更新工具时策略校验失败
func TestUpdateTool_PolicyViolation(t *testing.T) {
	policies := []domaingov.DataPolicy{
		{
			BaseModel: base.BaseModel{ID: "p1", TenantID: "tenant-001"},
			Name:      "require-endpoint", Type: "quality", Scope: "global",
			Priority: 1, Status: "active",
			Rules: base.JSON{"required_fields": []any{"endpoint"}},
		},
	}
	svc, db := setupToolServiceWithPolicy(t, policies)

	// 先创建一个有 endpoint 的工具
	toolRepo := persistence.NewToolRepository(db)
	tl := &tool.Tool{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "TestTool", Type: "eda", Endpoint: "http://example.com", Status: "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// 更新：清空 endpoint
	emptyEndpoint := ""
	req := &toolapp.UpdateToolRequest{Endpoint: &emptyEndpoint}
	_, err := svc.UpdateTool(context.Background(), "tenant-001", tl.ID, req, "admin")
	if err == nil {
		t.Fatal("期望策略校验失败返回错误")
	}
	var policyErr *bizerrors.PolicyViolationError
	if !isPolicyViolationError(err, &policyErr) {
		t.Fatalf("期望 PolicyViolationError，实际为 %T: %v", err, err)
	}
}

// TestImportTools_BatchPolicyViolation 批量导入时策略校验失败
func TestImportTools_BatchPolicyViolation(t *testing.T) {
	policies := []domaingov.DataPolicy{
		{
			BaseModel: base.BaseModel{ID: "p1", TenantID: "tenant-001"},
			Name:      "require-endpoint", Type: "quality", Scope: "global",
			Priority: 1, Status: "active",
			Rules: base.JSON{"required_fields": []any{"endpoint"}},
		},
	}
	svc, _ := setupToolServiceWithPolicy(t, policies)

	tools := []*tool.Tool{
		{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Name: "T1", Type: "eda", Endpoint: "http://a.com", Status: "active"},
		{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Name: "T2", Type: "cae", Status: "active"}, // missing endpoint
		{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Name: "T3", Type: "eda", Endpoint: "http://c.com", Status: "active"},
	}

	_, err := svc.ImportTools(context.Background(), "tenant-001", tools, "admin")
	if err == nil {
		t.Fatal("期望批量导入策略校验失败返回错误")
	}
	var policyErr *bizerrors.PolicyViolationError
	if !isPolicyViolationError(err, &policyErr) {
		t.Fatalf("期望 PolicyViolationError，实际为 %T: %v", err, err)
	}
}

// TestImportTools_BatchAllPass 批量导入时策略校验全部通过
func TestImportTools_BatchAllPass(t *testing.T) {
	policies := []domaingov.DataPolicy{
		{
			BaseModel: base.BaseModel{ID: "p1", TenantID: "tenant-001"},
			Name:      "require-endpoint", Type: "quality", Scope: "global",
			Priority: 1, Status: "active",
			Rules: base.JSON{"required_fields": []any{"endpoint"}},
		},
	}
	svc, _ := setupToolServiceWithPolicy(t, policies)

	tools := []*tool.Tool{
		{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Name: "T1", Type: "eda", Endpoint: "http://a.com", Status: "active"},
		{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Name: "T2", Type: "cae", Endpoint: "http://b.com", Status: "active"},
	}

	results, err := svc.ImportTools(context.Background(), "tenant-001", tools, "admin")
	if err != nil {
		t.Fatalf("期望批量导入策略校验通过，实际错误: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("期望导入 2 个工具，实际为 %d", len(results))
	}
}

// isPolicyViolationError 辅助函数：判断是否为策略违规错误
func isPolicyViolationError(err error, target **bizerrors.PolicyViolationError) bool {
	if e, ok := err.(*bizerrors.PolicyViolationError); ok {
		*target = e
		return true
	}
	return false
}

// ==================== 数据资产自动注册与质量检查测试 ====================

// mockDataAssetRepo 模拟数据资产仓储
type mockDataAssetRepo struct {
	assets []domaingov.DataAsset
	createCalled bool
	createdAsset *domaingov.DataAsset
	updateCalled bool
	updatedAsset *domaingov.DataAsset
	listCalled   bool
}

func (m *mockDataAssetRepo) Create(ctx context.Context, asset *domaingov.DataAsset) error {
	m.createCalled = true
	m.createdAsset = asset
	m.assets = append(m.assets, *asset)
	return nil
}
func (m *mockDataAssetRepo) GetByID(ctx context.Context, id string) (*domaingov.DataAsset, error) {
	for i := range m.assets {
		if m.assets[i].ID == id {
			return &m.assets[i], nil
		}
	}
	return nil, nil
}
func (m *mockDataAssetRepo) List(ctx context.Context, filter datagovapp.DataAssetFilter) ([]domaingov.DataAsset, int64, error) {
	m.listCalled = true
	return m.assets, int64(len(m.assets)), nil
}
func (m *mockDataAssetRepo) Update(ctx context.Context, asset *domaingov.DataAsset) error {
	m.updateCalled = true
	m.updatedAsset = asset
	for i := range m.assets {
		if m.assets[i].ID == asset.ID {
			m.assets[i] = *asset
			break
		}
	}
	return nil
}
func (m *mockDataAssetRepo) Delete(ctx context.Context, id string) error {
	return nil
}

// mockDataQualityRuleRepo 模拟数据质量规则仓储
type mockDataQualityRuleRepo struct {
	rules []domaingov.DataQualityRule
}

func (m *mockDataQualityRuleRepo) Create(ctx context.Context, rule *domaingov.DataQualityRule) error {
	m.rules = append(m.rules, *rule)
	return nil
}
func (m *mockDataQualityRuleRepo) GetByID(ctx context.Context, id string) (*domaingov.DataQualityRule, error) {
	for i := range m.rules {
		if m.rules[i].ID == id {
			return &m.rules[i], nil
		}
	}
	return nil, nil
}
func (m *mockDataQualityRuleRepo) List(ctx context.Context, filter datagovapp.DataQualityRuleFilter) ([]domaingov.DataQualityRule, int64, error) {
	return m.rules, int64(len(m.rules)), nil
}
func (m *mockDataQualityRuleRepo) Update(ctx context.Context, rule *domaingov.DataQualityRule) error {
	return nil
}
func (m *mockDataQualityRuleRepo) Delete(ctx context.Context, id string) error {
	return nil
}

// mockDataQualityCheckRepo 模拟数据质量检查仓储
type mockDataQualityCheckRepo struct {
	checks       []domaingov.DataQualityCheck
	createCalled bool
	createdCheck *domaingov.DataQualityCheck
}

func (m *mockDataQualityCheckRepo) Create(ctx context.Context, check *domaingov.DataQualityCheck) error {
	m.createCalled = true
	m.createdCheck = check
	m.checks = append(m.checks, *check)
	return nil
}
func (m *mockDataQualityCheckRepo) GetByID(ctx context.Context, id string) (*domaingov.DataQualityCheck, error) {
	return nil, nil
}
func (m *mockDataQualityCheckRepo) List(ctx context.Context, filter datagovapp.DataQualityCheckFilter) ([]domaingov.DataQualityCheck, int64, error) {
	return m.checks, int64(len(m.checks)), nil
}
func (m *mockDataQualityCheckRepo) Update(ctx context.Context, check *domaingov.DataQualityCheck) error {
	return nil
}
func (m *mockDataQualityCheckRepo) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockDataQualityCheckRepo) GetByRuleAndAsset(ctx context.Context, ruleID, assetID string) (*domaingov.DataQualityCheck, error) {
	return nil, nil
}

// setupToolServiceWithAssetRepos 创建带资产/规则/检查仓储的工具服务
func setupToolServiceWithAssetRepos(t *testing.T) (*toolapp.ToolService, *mockDataAssetRepo, *mockDataQualityRuleRepo, *mockDataQualityCheckRepo, *gorm.DB) {
	t.Helper()
	db := setupServiceTestDB(t)
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	assetRepo := &mockDataAssetRepo{}
	ruleRepo := &mockDataQualityRuleRepo{}
	checkRepo := &mockDataQualityCheckRepo{}
	svc := toolapp.NewToolService(toolRepo, connectorRepo, nil, assetRepo, ruleRepo, checkRepo)
	return svc, assetRepo, ruleRepo, checkRepo, db
}

// TestCreateTool_AutoRegistersAsset 创建工具时自动注册数据资产
func TestCreateTool_AutoRegistersAsset(t *testing.T) {
	svc, assetRepo, _, _, _ := setupToolServiceWithAssetRepos(t)

	req := &toolapp.CreateToolRequest{
		Name:     "Altium Designer",
		Type:     "eda",
		Category: "pcb-design",
		Endpoint: "http://altium.example.com",
	}

	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	if !assetRepo.createCalled {
		t.Fatal("期望自动创建 DataAsset")
	}
	if assetRepo.createdAsset == nil {
		t.Fatal("期望 createdAsset 不为 nil")
	}
	if assetRepo.createdAsset.Source != "tool" {
		t.Errorf("期望 Source 为 'tool'，实际为 '%s'", assetRepo.createdAsset.Source)
	}
	if assetRepo.createdAsset.SourceID != result.ID {
		t.Errorf("期望 SourceID 为 '%s'，实际为 '%s'", result.ID, assetRepo.createdAsset.SourceID)
	}
	if assetRepo.createdAsset.Type != "config" {
		t.Errorf("期望 Type 为 'config'，实际为 '%s'", assetRepo.createdAsset.Type)
	}
	if assetRepo.createdAsset.Classification != "pcb-design" {
		t.Errorf("期望 Classification 为 'pcb-design'，实际为 '%s'", assetRepo.createdAsset.Classification)
	}
	if assetRepo.createdAsset.Status != "active" {
		t.Errorf("期望 Status 为 'active'，实际为 '%s'", assetRepo.createdAsset.Status)
	}
}

// TestUpdateTool_UpdatesAsset 更新工具时同步更新数据资产
func TestUpdateTool_UpdatesAsset(t *testing.T) {
	svc, assetRepo, _, _, db := setupToolServiceWithAssetRepos(t)

	// 先创建工具
	req := &toolapp.CreateToolRequest{
		Name:     "OldTool",
		Type:     "eda",
		Category: "old-category",
	}
	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// 确认资产已创建
	if len(assetRepo.assets) != 1 {
		t.Fatalf("期望 1 个资产，实际为 %d", len(assetRepo.assets))
	}

	// 更新工具
	newName := "NewTool"
	newCategory := "new-category"
	updateReq := &toolapp.UpdateToolRequest{
		Name:     &newName,
		Category: &newCategory,
	}
	_, err = svc.UpdateTool(context.Background(), "tenant-001", result.ID, updateReq, "admin")
	if err != nil {
		t.Fatalf("更新工具失败: %v", err)
	}

	_ = db // 确认编译通过

	if !assetRepo.updateCalled {
		t.Fatal("期望自动更新 DataAsset")
	}
	if assetRepo.updatedAsset == nil {
		t.Fatal("期望 updatedAsset 不为 nil")
	}
	expectedName := "NewTool (工具元数据)"
	if assetRepo.updatedAsset.Name != expectedName {
		t.Errorf("期望 Name 为 '%s'，实际为 '%s'", expectedName, assetRepo.updatedAsset.Name)
	}
	if assetRepo.updatedAsset.Classification != "new-category" {
		t.Errorf("期望 Classification 为 'new-category'，实际为 '%s'", assetRepo.updatedAsset.Classification)
	}
}

// TestDeleteTool_ArchivesAsset 删除工具时归档数据资产
func TestDeleteTool_ArchivesAsset(t *testing.T) {
	svc, assetRepo, _, _, _ := setupToolServiceWithAssetRepos(t)

	// 先创建工具
	req := &toolapp.CreateToolRequest{
		Name: "ToDelete",
		Type: "eda",
	}
	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// 确认资产已创建
	if len(assetRepo.assets) != 1 {
		t.Fatalf("期望 1 个资产，实际为 %d", len(assetRepo.assets))
	}

	// 删除工具
	err = svc.DeleteTool(context.Background(), "tenant-001", result.ID)
	if err != nil {
		t.Fatalf("删除工具失败: %v", err)
	}

	if !assetRepo.updateCalled {
		t.Fatal("期望自动归档 DataAsset（通过 Update 设置 status=archived）")
	}
	if assetRepo.updatedAsset == nil {
		t.Fatal("期望 updatedAsset 不为 nil")
	}
	if assetRepo.updatedAsset.Status != "archived" {
		t.Errorf("期望 Status 为 'archived'，实际为 '%s'", assetRepo.updatedAsset.Status)
	}
}

// TestCreateTool_TriggersQualityCheck 创建工具时自动触发质量检查
func TestCreateTool_TriggersQualityCheck(t *testing.T) {
	svc, _, ruleRepo, checkRepo, _ := setupToolServiceWithAssetRepos(t)

	// 预设一个匹配 tool_type=eda 的质量规则
	ruleRepo.rules = []domaingov.DataQualityRule{
		{
			BaseModel:   base.BaseModel{ID: "rule-1", TenantID: "tenant-001"},
			Name:        "EDA工具检查",
			Type:        "completeness",
			Config:      base.JSON{"tool_type": "eda"},
			Severity:    "warning",
			Status:      "active",
		},
	}

	req := &toolapp.CreateToolRequest{
		Name: "Altium Designer",
		Type: "eda",
	}

	_, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	if !checkRepo.createCalled {
		t.Fatal("期望自动创建 DataQualityCheck")
	}
	if checkRepo.createdCheck == nil {
		t.Fatal("期望 createdCheck 不为 nil")
	}
	if checkRepo.createdCheck.RuleID != "rule-1" {
		t.Errorf("期望 RuleID 为 'rule-1'，实际为 '%s'", checkRepo.createdCheck.RuleID)
	}
	if checkRepo.createdCheck.TriggeredBy != "auto" {
		t.Errorf("期望 TriggeredBy 为 'auto'，实际为 '%s'", checkRepo.createdCheck.TriggeredBy)
	}
}

// TestCreateTool_NoMatchingRule 不匹配规则时不触发检查
func TestCreateTool_NoMatchingRule(t *testing.T) {
	svc, _, ruleRepo, checkRepo, _ := setupToolServiceWithAssetRepos(t)

	// 预设一个匹配 tool_type=cae 的规则，但创建的是 eda 工具
	ruleRepo.rules = []domaingov.DataQualityRule{
		{
			BaseModel:   base.BaseModel{ID: "rule-1", TenantID: "tenant-001"},
			Name:        "CAE工具检查",
			Type:        "completeness",
			Config:      base.JSON{"tool_type": "cae"},
			Severity:    "warning",
			Status:      "active",
		},
	}

	req := &toolapp.CreateToolRequest{
		Name: "Altium Designer",
		Type: "eda",
	}

	_, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	if checkRepo.createCalled {
		t.Error("期望不触发质量检查（规则不匹配）")
	}
}

// TestCreateTool_NilRepos 无资产/规则仓储时正常创建
func TestCreateTool_NilRepos(t *testing.T) {
	db := setupServiceTestDB(t)
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	// 所有新仓储传 nil
	svc := toolapp.NewToolService(toolRepo, connectorRepo, nil, nil, nil, nil)

	req := &toolapp.CreateToolRequest{
		Name: "NilRepos",
		Type: "eda",
	}
	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("期望无资产仓储时正常创建，实际错误: %v", err)
	}
	if result.Name != "NilRepos" {
		t.Errorf("期望 Name 为 'NilRepos'，实际为 '%s'", result.Name)
	}
}

// ==================== Boundary Tests ====================

// TestCreateTool_EmptyConfig 创建工具时 Config 为 nil 应仍然成功
func TestCreateTool_EmptyConfig(t *testing.T) {
	svc, _ := setupToolService(t)

	req := &toolapp.CreateToolRequest{
		Name:   "EmptyConfigTool",
		Type:   "eda",
		Status: "active",
		Config: nil,
	}

	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("Config 为 nil 时创建工具失败: %v", err)
	}
	if result.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
	if result.Name != "EmptyConfigTool" {
		t.Errorf("期望 Name 为 'EmptyConfigTool'，实际为 '%s'", result.Name)
	}
}

// TestUpdateTool_PartialUpdate 仅更新名称，验证其他字段不变
func TestUpdateTool_PartialUpdate(t *testing.T) {
	svc, db := setupToolService(t)
	toolRepo := persistence.NewToolRepository(db)

	tl := &tool.Tool{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Name:        "OriginalName",
		Type:        "eda",
		Category:    "pcb-design",
		Endpoint:    "http://original.example.com",
		Description: "原始描述",
		Status:      "active",
	}
	if err := toolRepo.Create(context.Background(), tl); err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	newName := "UpdatedName"
	req := &toolapp.UpdateToolRequest{
		Name: &newName,
	}

	result, err := svc.UpdateTool(context.Background(), "tenant-001", tl.ID, req, "admin")
	if err != nil {
		t.Fatalf("更新工具失败: %v", err)
	}
	if result.Name != "UpdatedName" {
		t.Errorf("期望 Name 为 'UpdatedName'，实际为 '%s'", result.Name)
	}
	if result.Type != "eda" {
		t.Errorf("期望 Type 不变仍为 'eda'，实际为 '%s'", result.Type)
	}
	if result.Category != "pcb-design" {
		t.Errorf("期望 Category 不变仍为 'pcb-design'，实际为 '%s'", result.Category)
	}
	if result.Endpoint != "http://original.example.com" {
		t.Errorf("期望 Endpoint 不变，实际为 '%s'", result.Endpoint)
	}
	if result.Description != "原始描述" {
		t.Errorf("期望 Description 不变，实际为 '%s'", result.Description)
	}
}

// TestImportTools_EmptyList 导入空列表应返回空结果
func TestImportTools_EmptyList(t *testing.T) {
	svc, _ := setupToolService(t)

	results, err := svc.ImportTools(context.Background(), "tenant-001", []*tool.Tool{}, "admin")
	if err != nil {
		t.Fatalf("导入空列表失败: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("期望返回空结果，实际为 %d 条", len(results))
	}
}

// ==================== 连接器查询与租户隔离测试 ====================

// TestGetConnector_Success 查询存在的连接器
func TestGetConnector_Success(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-001"},
		Name:      "Windchill",
		Type:      "plm",
		Endpoint:  "https://plm.example.com",
		Status:    "active",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	result, err := svc.GetConnector(context.Background(), "tenant-001", conn.ID)
	if err != nil {
		t.Fatalf("查询连接器失败: %v", err)
	}
	if result.Name != "Windchill" {
		t.Errorf("期望 Name 为 'Windchill'，实际为 '%s'", result.Name)
	}
	if result.Type != "plm" {
		t.Errorf("期望 Type 为 'plm'，实际为 '%s'", result.Type)
	}
	if result.Endpoint != "https://plm.example.com" {
		t.Errorf("期望 Endpoint 为 'https://plm.example.com'，实际为 '%s'", result.Endpoint)
	}
}

// TestGetConnector_NotFound 查询不存在的连接器返回错误
func TestGetConnector_NotFound(t *testing.T) {
	svc, _ := setupToolService(t)

	_, err := svc.GetConnector(context.Background(), "tenant-001", "non-existent-id")
	if err == nil {
		t.Fatal("期望查询不存在的连接器返回错误")
	}
	if !errors.Is(err, toolapp.ErrConnectorNotFound) {
		t.Errorf("期望 ErrConnectorNotFound，实际为 %v", err)
	}
}

// TestGetConnector_CrossTenant 跨租户查询连接器返回错误
func TestGetConnector_CrossTenant(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	// 租户 A 创建连接器
	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-a"},
		Name:      "Windchill",
		Type:      "plm",
		Endpoint:  "https://plm.example.com",
		Status:    "active",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	// 租户 B 尝试查询租户 A 的连接器
	_, err := svc.GetConnector(context.Background(), "tenant-b", conn.ID)
	if err == nil {
		t.Fatal("期望跨租户查询返回错误")
	}
	if !errors.Is(err, toolapp.ErrTenantMismatch) {
		t.Errorf("期望 ErrTenantMismatch，实际为 %v", err)
	}
}

// TestDeleteConnector_NotFound 删除不存在的连接器返回错误
func TestDeleteConnector_NotFound(t *testing.T) {
	svc, _ := setupToolService(t)

	err := svc.DeleteConnector(context.Background(), "tenant-001", "non-existent-id")
	if err == nil {
		t.Fatal("期望删除不存在的连接器返回错误")
	}
	if !errors.Is(err, toolapp.ErrConnectorNotFound) {
		t.Errorf("期望 ErrConnectorNotFound，实际为 %v", err)
	}
}

// TestDeleteConnector_CrossTenant 跨租户删除连接器返回错误
func TestDeleteConnector_CrossTenant(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	// 租户 A 创建连接器
	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-a"},
		Name:      "Windchill",
		Type:      "plm",
		Endpoint:  "https://plm.example.com",
		Status:    "active",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	// 租户 B 尝试删除租户 A 的连接器
	err := svc.DeleteConnector(context.Background(), "tenant-b", conn.ID)
	if err == nil {
		t.Fatal("期望跨租户删除返回错误")
	}
	if !errors.Is(err, toolapp.ErrTenantMismatch) {
		t.Errorf("期望 ErrTenantMismatch，实际为 %v", err)
	}

	// 确认连接器未被删除
	result, err := svc.GetConnector(context.Background(), "tenant-a", conn.ID)
	if err != nil {
		t.Fatalf("租户 A 的连接器应仍然存在，查询失败: %v", err)
	}
	if result.Name != "Windchill" {
		t.Errorf("租户 A 的连接器名称应不变，实际为 '%s'", result.Name)
	}
}

// TestUpdateConnector_CrossTenant 跨租户更新连接器返回错误
func TestUpdateConnector_CrossTenant(t *testing.T) {
	svc, db := setupToolService(t)
	connectorRepo := persistence.NewConnectorRepository(db)

	// 租户 A 创建连接器
	conn := &tool.Connector{
		BaseModel: base.BaseModel{TenantID: "tenant-a"},
		Name:      "Windchill",
		Type:      "plm",
		Endpoint:  "https://plm.example.com",
		Status:    "active",
	}
	if err := connectorRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("创建连接器失败: %v", err)
	}

	// 租户 B 尝试更新租户 A 的连接器
	hackedName := "Hacked"
	req := &toolapp.UpdateConnectorRequest{
		Name: &hackedName,
	}
	_, err := svc.UpdateConnector(context.Background(), "tenant-b", conn.ID, req)
	if err == nil {
		t.Fatal("期望跨租户更新返回错误")
	}
	if !errors.Is(err, toolapp.ErrTenantMismatch) {
		t.Errorf("期望 ErrTenantMismatch，实际为 %v", err)
	}

	// 确认连接器未被修改
	result, err := svc.GetConnector(context.Background(), "tenant-a", conn.ID)
	if err != nil {
		t.Fatalf("租户 A 的连接器应仍然存在，查询失败: %v", err)
	}
	if result.Name != "Windchill" {
		t.Errorf("租户 A 的连接器名称应不变，实际为 '%s'", result.Name)
	}
}

// ==================== 批量导入边界测试 ====================

// TestImportTools_PartialFailure 批量导入时部分失败，验证成功和失败结果
func TestImportTools_PartialFailure(t *testing.T) {
	svc, db := setupToolService(t)
	toolRepo := persistence.NewToolRepository(db)

	// 先创建一个工具，模拟 ID 冲突
	existing := &tool.Tool{
		BaseModel: base.BaseModel{ID: "duplicate-id", TenantID: "tenant-001"},
		Name:      "ExistingTool",
		Type:      "eda",
		Status:    "active",
	}
	if err := toolRepo.Create(context.Background(), existing); err != nil {
		t.Fatalf("创建已存在工具失败: %v", err)
	}

	// 导入 3 个工具，其中一个是重复 ID
	tools := []*tool.Tool{
		{BaseModel: base.BaseModel{ID: "tool-1", TenantID: "tenant-001"}, Name: "NewTool1", Type: "eda", Status: "active"},
		{BaseModel: base.BaseModel{ID: "duplicate-id", TenantID: "tenant-001"}, Name: "DuplicateTool", Type: "cae", Status: "active"},
		{BaseModel: base.BaseModel{ID: "tool-2", TenantID: "tenant-001"}, Name: "NewTool2", Type: "plm", Status: "active"},
	}

	results, err := svc.ImportTools(context.Background(), "tenant-001", tools, "admin")
	if err != nil {
		t.Fatalf("批量导入不应整体失败，实际错误: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("期望返回 3 条结果，实际为 %d", len(results))
	}

	// 第一个应该成功
	if results[0].Status != "success" {
		t.Errorf("期望第 1 个工具导入成功，实际状态为 '%s'", results[0].Status)
	}

	// 第二个应该失败（重复 ID）
	if results[1].Status != "error" {
		t.Errorf("期望第 2 个工具导入失败（重复 ID），实际状态为 '%s'", results[1].Status)
	}

	// 第三个应该成功
	if results[2].Status != "success" {
		t.Errorf("期望第 3 个工具导入成功，实际状态为 '%s'", results[2].Status)
	}
}

// ==================== 默认状态测试 ====================

// TestCreateTool_DefaultStatus 创建工具时不指定状态，默认为 active
func TestCreateTool_DefaultStatus(t *testing.T) {
	svc, _ := setupToolService(t)

	req := &toolapp.CreateToolRequest{
		Name: "DefaultStatusTool",
		Type: "eda",
		// 不指定 Status
	}

	result, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}
	if result.Status != "active" {
		t.Errorf("期望默认 Status 为 'active'，实际为 '%s'", result.Status)
	}
}

// ==================== 质量规则按分类匹配测试 ====================

// TestShouldRunRule_ByCategory 通过 CreateTool 验证按 tool_category 匹配质量规则
func TestShouldRunRule_ByCategory(t *testing.T) {
	svc, _, ruleRepo, checkRepo, _ := setupToolServiceWithAssetRepos(t)

	// 预设一个按 tool_category 匹配的质量规则
	ruleRepo.rules = []domaingov.DataQualityRule{
		{
			BaseModel:   base.BaseModel{ID: "rule-category", TenantID: "tenant-001"},
			Name:        "PCB设计工具检查",
			Type:        "completeness",
			Config:      base.JSON{"tool_category": "pcb-design"},
			Severity:    "warning",
			Status:      "active",
		},
	}

	req := &toolapp.CreateToolRequest{
		Name:     "Altium Designer",
		Type:     "eda",
		Category: "pcb-design",
	}

	_, err := svc.CreateTool(context.Background(), "tenant-001", req, "admin")
	if err != nil {
		t.Fatalf("创建工具失败: %v", err)
	}

	// 验证按 category 匹配的规则触发了质量检查
	if !checkRepo.createCalled {
		t.Fatal("期望 tool_category 匹配时触发质量检查")
	}
	if checkRepo.createdCheck == nil {
		t.Fatal("期望 createdCheck 不为 nil")
	}
	if checkRepo.createdCheck.RuleID != "rule-category" {
		t.Errorf("期望 RuleID 为 'rule-category'，实际为 '%s'", checkRepo.createdCheck.RuleID)
	}
}
