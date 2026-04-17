package expression

import (
	"strings"
	"testing"
)

func TestEvaluate_SimpleComparison(t *testing.T) {
	ctx := &Context{
		Tool: ToolContext{Type: "eda"},
	}
	ok, err := Evaluate(ctx, `tool.type == "eda"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_NotEqual(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	ok, err := Evaluate(ctx, `tool.type != "cae"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_InOperator(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	ok, err := Evaluate(ctx, `tool.type in ["eda", "cae"]`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_InOperator_NotMatch(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "report"}}
	ok, err := Evaluate(ctx, `tool.type in ["eda", "cae"]`)
	if err != nil || ok {
		t.Fatalf("expected false, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_LogicalAnd(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}, User: UserContext{Role: "admin"}}
	ok, err := Evaluate(ctx, `tool.type == "eda" && user.role == "admin"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_LogicalAnd_Fail(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}, User: UserContext{Role: "viewer"}}
	ok, err := Evaluate(ctx, `tool.type == "eda" && user.role == "admin"`)
	if err != nil || ok {
		t.Fatalf("expected false, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_LogicalOr(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "report"}, User: UserContext{Role: "admin"}}
	ok, err := Evaluate(ctx, `tool.type == "eda" || user.role == "admin"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_NumericComparison(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"max_size": 50.0}}}
	ok, err := Evaluate(ctx, `tool.config.max_size <= 100`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_NumericComparison_Fail(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"max_size": 200.0}}}
	ok, err := Evaluate(ctx, `tool.config.max_size <= 100`)
	if err != nil || ok {
		t.Fatalf("expected false, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_Matches(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Endpoint: "https://api.example.com"}}
	ok, err := Evaluate(ctx, `tool.endpoint matches "^https://.*"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_Matches_Fail(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Endpoint: "http://api.example.com"}}
	ok, err := Evaluate(ctx, `tool.endpoint matches "^https://.*"`)
	if err != nil || ok {
		t.Fatalf("expected false, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_EmptyCheck(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Description: ""}}
	ok, err := Evaluate(ctx, `tool.description == empty`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_NotEmpty(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Description: "hello"}}
	ok, err := Evaluate(ctx, `tool.description != empty`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_NestedField(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"auth": map[string]any{"type": "oauth"}}}}
	ok, err := Evaluate(ctx, `tool.config.auth.type == "oauth"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_InvalidSyntax(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	_, err := Evaluate(ctx, `tool.type == `)
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
}

func TestEvaluate_ComplexExpression(t *testing.T) {
	ctx := &Context{
		Tool: ToolContext{Type: "eda", Category: "visualization", Description: "test"},
		User: UserContext{Role: "admin"},
	}
	ok, err := Evaluate(ctx, `(tool.type == "eda" || tool.type == "cae") && user.role == "admin" && tool.description != empty`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

// ========== Boundary tests ==========

func TestEvaluate_NilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil context")
		}
	}()
	Evaluate(nil, `tool.type == "eda"`)
}

func TestEvaluate_EmptyExpression(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	_, err := Evaluate(ctx, ``)
	if err == nil {
		t.Fatal("expected error for empty expression")
	}
}

func TestEvaluate_NilValueInCondition(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"key": nil}}}
	ok, err := Evaluate(ctx, `tool.config.key == null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for null comparison")
	}
}

func TestEvaluate_StringContains(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Description: "hello world"}}
	ok, err := Evaluate(ctx, `tool.description contains "hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for contains")
	}
}

func TestEvaluate_NestedMapDeepAccess(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"level1": map[string]any{"level2": map[string]any{"value": 42.0}}}}}
	ok, err := Evaluate(ctx, `tool.config.level1.level2.value == 42`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for deep nested access")
	}
}

func TestEvaluate_NestedMapMissingKey(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"level1": map[string]any{}}}}
	ok, err := Evaluate(ctx, `tool.config.level1.missing == "test"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for missing nested key")
	}
}

// ========== 运算符与错误处理测试 ==========

// TestEvaluate_NotInOperator: not_in 运算符，值不在列表中 -> true
func TestEvaluate_NotInOperator(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "report"}}
	ok, err := Evaluate(ctx, `tool.type not_in ["eda", "cae"]`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

// TestEvaluate_NotInOperator_Match: not_in 运算符，值在列表中 -> false
func TestEvaluate_NotInOperator_Match(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	ok, err := Evaluate(ctx, `tool.type not_in ["eda", "cae"]`)
	if err != nil || ok {
		t.Fatalf("expected false, got ok=%v err=%v", ok, err)
	}
}

// TestEvaluate_GreaterThan: 大于比较，200 > 100 -> true
func TestEvaluate_GreaterThan(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"amount": 200.0}}}
	ok, err := Evaluate(ctx, `tool.config.amount > 100`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

// TestEvaluate_GreaterThanOrEqual: 大于等于比较，100 >= 100 -> true
func TestEvaluate_GreaterThanOrEqual(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"amount": 100.0}}}
	ok, err := Evaluate(ctx, `tool.config.amount >= 100`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

// TestEvaluate_NonNumericComparison: 非数值类型比较 -> 错误
func TestEvaluate_NonNumericComparison(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	_, err := Evaluate(ctx, `tool.type > 100`)
	if err == nil {
		t.Fatal("expected error for non-numeric comparison")
	}
	if !strings.Contains(err.Error(), "cannot compare non-numbers") {
		t.Errorf("expected 'cannot compare non-numbers' error, got: %v", err)
	}
}

// TestEvaluate_Contains_NonString: contains 操作数非字符串 -> 错误
func TestEvaluate_Contains_NonString(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"value": 42}}}
	_, err := Evaluate(ctx, `tool.config.value contains "test"`)
	if err == nil {
		t.Fatal("expected error for contains on non-string")
	}
	if !strings.Contains(err.Error(), "'contains' requires strings") {
		t.Errorf("expected 'contains' requires strings error, got: %v", err)
	}
}

// TestEvaluate_Matches_NonString: matches 操作数非字符串 -> 错误
func TestEvaluate_Matches_NonString(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Config: map[string]any{"value": 42}}}
	_, err := Evaluate(ctx, `tool.config.value matches ".*"`)
	if err == nil {
		t.Fatal("expected error for matches on non-string")
	}
	if !strings.Contains(err.Error(), "'matches' requires strings") {
		t.Errorf("expected 'matches' requires strings error, got: %v", err)
	}
}

// TestEvaluate_Matches_InvalidRegex: 无效正则表达式 -> 错误
func TestEvaluate_Matches_InvalidRegex(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Endpoint: "https://api.example.com"}}
	_, err := Evaluate(ctx, `tool.endpoint matches "[invalid"`)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

// TestEvaluate_In_NonArray: in 运算符右侧非数组 -> 错误
func TestEvaluate_In_NonArray(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	_, err := Evaluate(ctx, `tool.type in "eda"`)
	if err == nil {
		t.Fatal("expected error for in with non-array right side")
	}
	if !strings.Contains(err.Error(), "right side of 'in' must be array") {
		t.Errorf("expected 'right side of in must be array' error, got: %v", err)
	}
}

// TestEvaluate_UserContextFields: 测试 user.id 和 user.username 字段访问
func TestEvaluate_UserContextFields(t *testing.T) {
	ctx := &Context{
		User: UserContext{ID: "user-001", Username: "alice"},
	}
	ok, err := Evaluate(ctx, `user.id == "user-001" && user.username == "alice"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

// TestEvaluate_RequestContextFields: 测试 ctx.action 和 ctx.tenant_id 字段访问
func TestEvaluate_RequestContextFields(t *testing.T) {
	ctx := &Context{
		Ctx: RequestContext{Action: "create", TenantID: "tenant-001"},
	}
	ok, err := Evaluate(ctx, `ctx.action == "create" && ctx.tenant_id == "tenant-001"`)
	if err != nil || !ok {
		t.Fatalf("expected true, got ok=%v err=%v", ok, err)
	}
}

// TestEvaluate_UnknownRoot: 未知根字段 -> 错误
func TestEvaluate_UnknownRoot(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	_, err := Evaluate(ctx, `invalid.field == "test"`)
	if err == nil {
		t.Fatal("expected error for unknown root")
	}
	if !strings.Contains(err.Error(), "unknown root") {
		t.Errorf("expected 'unknown root' error, got: %v", err)
	}
}

// TestEvaluate_OrShortCircuit: || 短路求值，左侧为 true 时不评估右侧
func TestEvaluate_OrShortCircuit(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	// 左侧 tool.type == "eda" 为 true，右侧 invalid.field 不会触发 "unknown root" 错误
	ok, err := Evaluate(ctx, `tool.type == "eda" || invalid.field == "test"`)
	if err != nil {
		t.Fatalf("unexpected error (short-circuit should skip right side): %v", err)
	}
	if !ok {
		t.Fatal("expected true")
	}
}

// TestEvaluate_AndShortCircuit: && 短路求值，左侧为 false 时不评估右侧
func TestEvaluate_AndShortCircuit(t *testing.T) {
	ctx := &Context{Tool: ToolContext{Type: "eda"}}
	// 左侧 tool.type == "cae" 为 false，右侧 invalid.field 不会触发 "unknown root" 错误
	ok, err := Evaluate(ctx, `tool.type == "cae" && invalid.field == "test"`)
	if err != nil {
		t.Fatalf("unexpected error (short-circuit should skip right side): %v", err)
	}
	if ok {
		t.Fatal("expected false")
	}
}
