package datagov

import (
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	domaingov "git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
)

// helper: create a policy
func makePolicy(name, typ, scope, scopeValue string, priority int, status string, rules base.JSON) *domaingov.DataPolicy {
	return &domaingov.DataPolicy{
		BaseModel: base.BaseModel{ID: name},
		Name:      name,
		Type:      typ,
		Scope:     scope,
		ScopeValue: scopeValue,
		Priority:  priority,
		Status:    status,
		Rules:     rules,
	}
}

// helper: create a tool
func makeTool(name, typ, category, description, endpoint, status, connectorID string, config base.JSON) *tool.Tool {
	return &tool.Tool{
		BaseModel:   base.BaseModel{ID: name, TenantID: "tenant-001"},
		Name:        name,
		Type:        typ,
		Category:    category,
		Description: description,
		Endpoint:    endpoint,
		Status:      status,
		ConnectorID: connectorID,
		Config:      config,
	}
}

// TestValidateTool_NoMatchingPolicy: no active policies -> pass
func TestValidateTool_NoMatchingPolicy(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("p1", "quality", "global", "", 1, "inactive", nil),
	}
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)

	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass when no active policies, got violations: %v", result.Violations)
	}
}

// TestValidateTool_NoPoliciesAtAll: empty policy list -> pass
func TestValidateTool_NoPoliciesAtAll(t *testing.T) {
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(nil, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass with nil policies, got violations: %v", result.Violations)
	}
}

// TestValidateTool_GlobalScope: global policy with expression -> evaluated
func TestValidateTool_GlobalScope(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("global-expr", "quality", "global", "", 1, "active", base.JSON{
			"expression": `tool.type == "eda"`,
		}),
	}

	// Should pass: tool.type == "eda"
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass for eda tool, got violations: %v", result.Violations)
	}

	// Should fail: tool.type != "eda"
	tl2 := makeTool("t2", "cae", "", "", "", "active", "", nil)
	result2 := ValidateTool(policies, tl2, "admin", "create")
	if result2.Passed {
		t.Error("expected fail for cae tool with eda-only policy")
	}
}

// TestValidateTool_ToolTypeScope: scope=tool_type, ScopeValue matches -> evaluated
func TestValidateTool_ToolTypeScope(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("eda-policy", "quality", "tool_type", "eda", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
	}

	// Should fail: eda tool missing endpoint
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail for eda tool missing endpoint")
	}
	if len(result.Violations) == 0 {
		t.Error("expected violations")
	}
	if result.Violations[0].RuleKey != "required_fields" {
		t.Errorf("expected rule_key 'required_fields', got '%s'", result.Violations[0].RuleKey)
	}
}

// TestValidateTool_ToolTypeScope_NoMatch: scope=tool_type, ScopeValue doesn't match -> skipped
func TestValidateTool_ToolTypeScope_NoMatch(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("eda-policy", "quality", "tool_type", "eda", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
	}

	// Should pass: cae tool, policy only applies to eda
	tl := makeTool("t1", "cae", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass for cae tool, got violations: %v", result.Violations)
	}
}

// TestValidateTool_QualityRules_RequiredFields: missing field -> fail
func TestValidateTool_QualityRules_RequiredFields(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("quality-1", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"endpoint", "description"},
		}),
	}

	// Missing both endpoint and description
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when required fields missing")
	}
	if len(result.Violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(result.Violations))
	}
}

// TestValidateTool_QualityRules_Pass: all required fields present -> pass
func TestValidateTool_QualityRules_Pass(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("quality-1", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"endpoint", "description"},
		}),
	}

	tl := makeTool("t1", "eda", "", "A tool", "http://example.com", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass, got violations: %v", result.Violations)
	}
}

// TestValidateTool_QualityRules_DescriptionRequired: description_required=true, empty desc -> fail
func TestValidateTool_QualityRules_DescriptionRequired(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("quality-2", "quality", "global", "", 1, "active", base.JSON{
			"description_required": true,
		}),
	}

	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when description_required and description is empty")
	}
	if result.Violations[0].RuleKey != "description_required" {
		t.Errorf("expected rule_key 'description_required', got '%s'", result.Violations[0].RuleKey)
	}
}

// TestValidateTool_ClassificationRules: allowed_categories, wrong category -> fail
func TestValidateTool_ClassificationRules(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("class-1", "classification", "global", "", 1, "active", base.JSON{
			"allowed_categories": []any{"eda", "cae"},
		}),
	}

	// Tool with category not in allowed list
	tl := makeTool("t1", "eda", "plm", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail for category not in allowed list")
	}
	if result.Violations[0].RuleKey != "allowed_categories" {
		t.Errorf("expected rule_key 'allowed_categories', got '%s'", result.Violations[0].RuleKey)
	}

	// Tool with category in allowed list -> pass
	tl2 := makeTool("t2", "eda", "eda", "", "", "active", "", nil)
	result2 := ValidateTool(policies, tl2, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass for allowed category, got violations: %v", result2.Violations)
	}

	// Tool with empty category -> pass (no restriction if no category set)
	tl3 := makeTool("t3", "eda", "", "", "", "active", "", nil)
	result3 := ValidateTool(policies, tl3, "admin", "create")
	if !result3.Passed {
		t.Errorf("expected pass for empty category, got violations: %v", result3.Violations)
	}
}

// TestValidateTool_ClassificationRules_CategoryRequired: category_required=true, empty category -> fail
func TestValidateTool_ClassificationRules_CategoryRequired(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("class-2", "classification", "global", "", 1, "active", base.JSON{
			"category_required": true,
		}),
	}

	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when category_required and category is empty")
	}
}

// TestValidateTool_AccessRules: allowed_roles, wrong role -> fail
func TestValidateTool_AccessRules(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("access-1", "access", "global", "", 1, "active", base.JSON{
			"allowed_roles": []any{"admin", "editor"},
		}),
	}

	// viewer role not in allowed list
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "viewer", "create")
	if result.Passed {
		t.Error("expected fail for viewer role with admin-only policy")
	}

	// admin role -> pass
	result2 := ValidateTool(policies, tl, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass for admin role, got violations: %v", result2.Violations)
	}
}

// TestValidateTool_AccessRules_BlockedRoles: blocked_roles -> fail
func TestValidateTool_AccessRules_BlockedRoles(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("access-2", "access", "global", "", 1, "active", base.JSON{
			"blocked_roles": []any{"viewer"},
		}),
	}

	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "viewer", "create")
	if result.Passed {
		t.Error("expected fail for blocked role")
	}

	result2 := ValidateTool(policies, tl, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass for non-blocked role, got violations: %v", result2.Violations)
	}
}

// TestValidateTool_AccessRules_RequireRole: require_role -> fail
func TestValidateTool_AccessRules_RequireRole(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("access-3", "access", "global", "", 1, "active", base.JSON{
			"require_role": "admin",
		}),
	}

	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "viewer", "create")
	if result.Passed {
		t.Error("expected fail when require_role is admin but user is viewer")
	}

	result2 := ValidateTool(policies, tl, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass for admin role, got violations: %v", result2.Violations)
	}
}

// TestValidateTool_RetentionRules: config missing retention_days -> fail
func TestValidateTool_RetentionRules(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("retention-1", "retention", "global", "", 1, "active", base.JSON{
			"require_expiry_date": true,
		}),
	}

	// Tool without expiry_date in config
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when require_expiry_date and config has no expiry_date")
	}

	// Tool with expiry_date in config -> pass
	tl2 := makeTool("t2", "eda", "", "", "", "active", "", base.JSON{
		"expiry_date": "2025-12-31",
	})
	result2 := ValidateTool(policies, tl2, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass with expiry_date, got violations: %v", result2.Violations)
	}
}

// TestValidateTool_RetentionRules_MaxRetentionDays: retention_days exceeds max -> fail
func TestValidateTool_RetentionRules_MaxRetentionDays(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("retention-2", "retention", "global", "", 1, "active", base.JSON{
			"max_retention_days": float64(30),
		}),
	}

	// retention_days = 60 > 30
	tl := makeTool("t1", "eda", "", "", "", "active", "", base.JSON{
		"retention_days": float64(60),
	})
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when retention_days exceeds max")
	}

	// retention_days = 20 <= 30 -> pass
	tl2 := makeTool("t2", "eda", "", "", "", "active", "", base.JSON{
		"retention_days": float64(20),
	})
	result2 := ValidateTool(policies, tl2, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass with retention_days within limit, got violations: %v", result2.Violations)
	}
}

// TestValidateTool_PriorityOrder: higher priority policy evaluated first, fails -> stop
func TestValidateTool_PriorityOrder(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		// Low priority: would pass
		makePolicy("low-priority", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
		// High priority: fails
		makePolicy("high-priority", "access", "global", "", 10, "active", base.JSON{
			"blocked_roles": []any{"viewer"},
		}),
	}

	tl := makeTool("t1", "eda", "http://example.com", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "viewer", "create")
	if result.Passed {
		t.Error("expected fail from high priority policy")
	}
	// Should only have 1 violation from the high-priority policy (hard block)
	if len(result.Violations) != 1 {
		t.Errorf("expected 1 violation (hard block), got %d", len(result.Violations))
	}
	if result.Violations[0].PolicyName != "high-priority" {
		t.Errorf("expected violation from 'high-priority', got '%s'", result.Violations[0].PolicyName)
	}
}

// TestValidateTool_ExpressionError: invalid expression -> violation
func TestValidateTool_ExpressionError(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("bad-expr", "quality", "global", "", 1, "active", base.JSON{
			"expression": "invalid!!!",
		}),
	}

	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail for invalid expression")
	}
	if result.Violations[0].RuleKey != "expression" {
		t.Errorf("expected rule_key 'expression', got '%s'", result.Violations[0].RuleKey)
	}
}

// TestValidateTool_CategoryScope: scope=category, ScopeValue matches tool.Category -> evaluated
func TestValidateTool_CategoryScope(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("cat-policy", "quality", "category", "eda", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
	}

	// Tool with category "eda" -> policy applies
	tl := makeTool("t1", "custom", "eda", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail for eda category tool missing endpoint")
	}

	// Tool with category "cae" -> policy does not apply
	tl2 := makeTool("t2", "custom", "cae", "", "", "active", "", nil)
	result2 := ValidateTool(policies, tl2, "admin", "create")
	if !result2.Passed {
		t.Errorf("expected pass for cae category tool, got violations: %v", result2.Violations)
	}
}

// TestValidateTools_BatchValidation: multiple tools, one fails -> all rejected
func TestValidateTools_BatchValidation(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("quality-1", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
	}

	tools := []*tool.Tool{
		makeTool("t1", "eda", "", "", "http://a.com", "active", "", nil), // has endpoint
		makeTool("t2", "cae", "", "", "", "active", "", nil),              // missing endpoint
		makeTool("t3", "eda", "", "", "http://c.com", "active", "", nil), // has endpoint
	}

	result := ValidateTools(policies, tools, "admin")
	if result.Passed {
		t.Error("expected batch to fail when one tool violates policy")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].ToolName != "t2" {
		t.Errorf("expected error for 't2', got '%s'", result.Errors[0].ToolName)
	}
	if result.Errors[0].Index != 1 {
		t.Errorf("expected index 1, got %d", result.Errors[0].Index)
	}
}

// TestValidateTools_BatchAllPass: multiple tools, all pass -> success
func TestValidateTools_BatchAllPass(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("quality-1", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
	}

	tools := []*tool.Tool{
		makeTool("t1", "eda", "", "", "http://a.com", "active", "", nil),
		makeTool("t2", "cae", "", "", "http://b.com", "active", "", nil),
	}

	result := ValidateTools(policies, tools, "admin")
	if !result.Passed {
		t.Errorf("expected batch pass, got errors: %v", result.Errors)
	}
}

// ========== 边界与扩展测试 ==========

// TestValidateTool_UnknownScope: 未知作用域的策略应被跳过 -> 通过
func TestValidateTool_UnknownScope(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("unknown-scope-policy", "quality", "unknown_scope", "some_value", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
	}
	// 工具缺少 endpoint，但由于策略作用域未知，策略应被跳过
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass for unknown scope (should be skipped), got violations: %v", result.Violations)
	}
}

// TestValidateTool_NilRules: 策略规则为 nil -> 通过（无违规）
func TestValidateTool_NilRules(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("nil-rules-policy", "quality", "global", "", 1, "active", nil),
	}
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass for nil rules, got violations: %v", result.Violations)
	}
}

// TestValidateTool_ExpressionAndStructuredRules: 同时包含表达式（通过）和质量规则（缺失字段）-> 应在表达式通过后因质量规则失败
func TestValidateTool_ExpressionAndStructuredRules(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("combined-policy", "quality", "global", "", 1, "active", base.JSON{
			"expression":      `tool.type == "eda"`,
			"required_fields": []any{"endpoint"},
		}),
	}
	// 工具类型为 eda（表达式通过），但缺少 endpoint（质量规则失败）
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail: expression passes but required_fields missing")
	}
	found := false
	for _, v := range result.Violations {
		if v.RuleKey == "required_fields" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected violation with rule_key 'required_fields', got: %v", result.Violations)
	}
}

// TestValidateTool_MultiplePoliciesAllPass: 两个活跃策略均通过 -> result.Passed=true
func TestValidateTool_MultiplePoliciesAllPass(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("quality-pass", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"endpoint"},
		}),
		makePolicy("access-pass", "access", "global", "", 2, "active", base.JSON{
			"allowed_roles": []any{"admin"},
		}),
	}
	tl := makeTool("t1", "eda", "", "", "http://example.com", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass when all policies pass, got violations: %v", result.Violations)
	}
}

// TestValidateTool_AccessRules_EmptyAllowedRoles: allowed_roles 为空数组 -> 跳过（通过）
func TestValidateTool_AccessRules_EmptyAllowedRoles(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("empty-roles-policy", "access", "global", "", 1, "active", base.JSON{
			"allowed_roles": []any{},
		}),
	}
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "viewer", "create")
	if !result.Passed {
		t.Errorf("expected pass for empty allowed_roles (should be skipped), got violations: %v", result.Violations)
	}
}

// TestValidateTool_RetentionRules_NilConfig: 保留规则要求过期日期，工具 Config 为 nil -> 失败
func TestValidateTool_RetentionRules_NilConfig(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("retention-nil-config", "retention", "global", "", 1, "active", base.JSON{
			"require_expiry_date": true,
		}),
	}
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when require_expiry_date=true and Config is nil")
	}
	if result.Violations[0].RuleKey != "require_expiry_date" {
		t.Errorf("expected rule_key 'require_expiry_date', got '%s'", result.Violations[0].RuleKey)
	}
}

// TestValidateTool_HasToolField_CustomConfigField: required_fields 包含自定义 config 字段，工具 Config 中存在该字段 -> 通过
func TestValidateTool_HasToolField_CustomConfigField(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("custom-field-policy", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"custom_key"},
		}),
	}
	tl := makeTool("t1", "eda", "", "", "", "active", "", base.JSON{
		"custom_key": "value",
	})
	result := ValidateTool(policies, tl, "admin", "create")
	if !result.Passed {
		t.Errorf("expected pass when custom_key exists in Config, got violations: %v", result.Violations)
	}
}

// TestValidateTool_HasToolField_CustomConfigField_NilConfig: required_fields 包含自定义 config 字段，工具 Config 为 nil -> 失败
func TestValidateTool_HasToolField_CustomConfigField_NilConfig(t *testing.T) {
	policies := []*domaingov.DataPolicy{
		makePolicy("custom-field-nil-config", "quality", "global", "", 1, "active", base.JSON{
			"required_fields": []any{"custom_key"},
		}),
	}
	tl := makeTool("t1", "eda", "", "", "", "active", "", nil)
	result := ValidateTool(policies, tl, "admin", "create")
	if result.Passed {
		t.Error("expected fail when custom_key required but Config is nil")
	}
	if result.Violations[0].RuleKey != "required_fields" {
		t.Errorf("expected rule_key 'required_fields', got '%s'", result.Violations[0].RuleKey)
	}
}
