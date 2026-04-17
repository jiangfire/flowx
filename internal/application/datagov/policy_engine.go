package datagov

import (
	"encoding/json"
	"fmt"
	"sort"

	"git.neolidy.top/neo/flowx/internal/application/datagov/expression"
	"git.neolidy.top/neo/flowx/internal/domain/datagov"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
)

// PolicyViolation 策略违规记录
type PolicyViolation struct {
	PolicyName string `json:"policy_name"`
	PolicyType string `json:"policy_type"`
	RuleKey    string `json:"rule_key"`
	Message    string `json:"message"`
}

// PolicyResult 单个工具的策略校验结果
type PolicyResult struct {
	Passed     bool
	Violations []PolicyViolation
}

// BatchResult 批量工具校验结果
type BatchResult struct {
	Passed bool
	Errors []BatchItemError
}

// BatchItemError 批量校验中单个工具的错误
type BatchItemError struct {
	Index      int
	ToolName   string
	Violations []PolicyViolation
}

// matchesScope 检查策略是否适用于给定工具
func matchesScope(p *datagov.DataPolicy, t *tool.Tool) bool {
	switch p.Scope {
	case "global":
		return true
	case "tool_type":
		return p.ScopeValue == t.Type
	case "category":
		return p.ScopeValue == t.Category
	}
	return false
}

// ValidateTool 校验单个工具是否满足策略要求
func ValidateTool(policies []*datagov.DataPolicy, t *tool.Tool, userRole string, action string) *PolicyResult {
	// 筛选活跃且匹配范围的策略，按优先级降序排列
	var applicable []*datagov.DataPolicy
	for _, p := range policies {
		if p.Status != "active" {
			continue
		}
		if !matchesScope(p, t) {
			continue
		}
		applicable = append(applicable, p)
	}
	sort.Slice(applicable, func(i, j int) bool {
		return applicable[i].Priority > applicable[j].Priority
	})

	result := &PolicyResult{Passed: true}
	for _, p := range applicable {
		violations := evaluatePolicy(p, t, userRole, action)
		if len(violations) > 0 {
			result.Passed = false
			result.Violations = append(result.Violations, violations...)
			// 硬阻断：首个失败策略即停止
			return result
		}
	}
	return result
}

// ValidateTools 批量校验多个工具（用于导入场景）
func ValidateTools(policies []*datagov.DataPolicy, tools []*tool.Tool, userRole string) *BatchResult {
	result := &BatchResult{Passed: true}
	for i, t := range tools {
		r := ValidateTool(policies, t, userRole, "import")
		if !r.Passed {
			result.Passed = false
			result.Errors = append(result.Errors, BatchItemError{
				Index:      i,
				ToolName:   t.Name,
				Violations: r.Violations,
			})
		}
	}
	return result
}

// evaluatePolicy 评估单个策略的规则
func evaluatePolicy(p *datagov.DataPolicy, t *tool.Tool, userRole string, action string) []PolicyViolation {
	ctx := &expression.Context{
		Tool: expression.ToolContext{
			Name: t.Name, Type: t.Type, Category: t.Category,
			Description: t.Description, Endpoint: t.Endpoint,
			Status: t.Status, ConnectorID: t.ConnectorID,
			Config: t.Config,
		},
		User: expression.UserContext{Role: userRole},
		Ctx:  expression.RequestContext{Action: action},
	}

	var violations []PolicyViolation

	if p.Rules == nil {
		return violations
	}

	// 检查表达式规则
	if expr, ok := p.Rules["expression"].(string); ok && expr != "" {
		passed, err := expression.Evaluate(ctx, expr)
		if err != nil {
			violations = append(violations, PolicyViolation{
				PolicyName: p.Name, PolicyType: p.Type,
				RuleKey: "expression", Message: fmt.Sprintf("表达式错误: %v", err),
			})
			return violations
		}
		if !passed {
			violations = append(violations, PolicyViolation{
				PolicyName: p.Name, PolicyType: p.Type,
				RuleKey: "expression", Message: fmt.Sprintf("不满足策略 '%s' 的表达式约束", p.Name),
			})
			return violations
		}
	}

	// 根据策略类型检查结构化规则
	switch p.Type {
	case "quality":
		violations = append(violations, evaluateQualityRules(p, t)...)
	case "classification":
		violations = append(violations, evaluateClassificationRules(p, t)...)
	case "access":
		violations = append(violations, evaluateAccessRules(p, userRole)...)
	case "retention":
		violations = append(violations, evaluateRetentionRules(p, t)...)
	}

	return violations
}

// evaluateQualityRules 评估质量规则
func evaluateQualityRules(p *datagov.DataPolicy, t *tool.Tool) []PolicyViolation {
	var violations []PolicyViolation
	rules := p.Rules

	// required_fields
	if fields, ok := rules["required_fields"].([]any); ok {
		for _, f := range fields {
			fieldName, _ := f.(string)
			if fieldName == "" {
				continue
			}
			if !hasToolField(t, fieldName) {
				violations = append(violations, PolicyViolation{
					PolicyName: p.Name, PolicyType: "quality",
					RuleKey:    "required_fields",
					Message:    fmt.Sprintf("工具 '%s' 缺少必填字段: %s", t.Name, fieldName),
				})
			}
		}
	}
	if len(violations) > 0 {
		return violations
	}

	// description_required
	if desc, ok := rules["description_required"].(bool); ok && desc && t.Description == "" {
		violations = append(violations, PolicyViolation{
			PolicyName: p.Name, PolicyType: "quality",
			RuleKey:    "description_required",
			Message:    fmt.Sprintf("工具 '%s' 描述不能为空", t.Name),
		})
	}
	if len(violations) > 0 {
		return violations
	}

	// max_config_size
	if maxSize, ok := rules["max_config_size"].(float64); ok && maxSize > 0 {
		if t.Config != nil {
			configBytes, _ := json.Marshal(t.Config)
			if int64(len(configBytes)) > int64(maxSize) {
				violations = append(violations, PolicyViolation{
					PolicyName: p.Name, PolicyType: "quality",
					RuleKey:    "max_config_size",
					Message:    fmt.Sprintf("工具 '%s' 的配置大小 (%d 字节) 超过最大限制 (%.0f 字节)", t.Name, len(configBytes), maxSize),
				})
			}
		}
	}
	if len(violations) > 0 {
		return violations
	}

	// allowed_types
	if types, ok := rules["allowed_types"].([]any); ok && len(types) > 0 {
		allowed := make(map[string]bool, len(types))
		for _, tp := range types {
			if s, ok := tp.(string); ok {
				allowed[s] = true
			}
		}
		if !allowed[t.Type] {
			violations = append(violations, PolicyViolation{
				PolicyName: p.Name, PolicyType: "quality",
				RuleKey:    "allowed_types",
				Message:    fmt.Sprintf("工具 '%s' 的类型 '%s' 不在允许列表中", t.Name, t.Type),
			})
		}
	}
	return violations
}

// evaluateClassificationRules 评估分类规则
func evaluateClassificationRules(p *datagov.DataPolicy, t *tool.Tool) []PolicyViolation {
	var violations []PolicyViolation
	rules := p.Rules

	// category_required
	if req, ok := rules["category_required"].(bool); ok && req && t.Category == "" {
		// 如果有 default_category，自动设置（不视为违规）
		if def, ok := rules["default_category"].(string); ok && def != "" {
			t.Category = def
			return violations
		}
		violations = append(violations, PolicyViolation{
			PolicyName: p.Name, PolicyType: "classification",
			RuleKey:    "category_required",
			Message:    fmt.Sprintf("工具 '%s' 必须指定分类", t.Name),
		})
	}
	if len(violations) > 0 {
		return violations
	}

	// allowed_categories
	if cats, ok := rules["allowed_categories"].([]any); ok && len(cats) > 0 {
		allowed := make(map[string]bool, len(cats))
		for _, c := range cats {
			if s, ok := c.(string); ok {
				allowed[s] = true
			}
		}
		if t.Category != "" && !allowed[t.Category] {
			violations = append(violations, PolicyViolation{
				PolicyName: p.Name, PolicyType: "classification",
				RuleKey:    "allowed_categories",
				Message:    fmt.Sprintf("工具 '%s' 的分类 '%s' 不在允许列表中", t.Name, t.Category),
			})
		}
	}
	return violations
}

// evaluateAccessRules 评估访问控制规则
func evaluateAccessRules(p *datagov.DataPolicy, userRole string) []PolicyViolation {
	var violations []PolicyViolation
	rules := p.Rules

	// blocked_roles
	if blocked, ok := rules["blocked_roles"].([]any); ok {
		for _, r := range blocked {
			if role, ok := r.(string); ok && role == userRole {
				violations = append(violations, PolicyViolation{
					PolicyName: p.Name, PolicyType: "access",
					RuleKey:    "blocked_roles",
					Message:    fmt.Sprintf("角色 '%s' 被禁止执行此操作", userRole),
				})
				return violations
			}
		}
	}

	// require_role
	if req, ok := rules["require_role"].(string); ok && req != "" && userRole != req {
		violations = append(violations, PolicyViolation{
			PolicyName: p.Name, PolicyType: "access",
			RuleKey:    "require_role",
			Message:    fmt.Sprintf("需要 '%s' 角色，当前角色为 '%s'", req, userRole),
		})
		return violations
	}

	// allowed_roles
	if allowed, ok := rules["allowed_roles"].([]any); ok && len(allowed) > 0 {
		found := false
		for _, r := range allowed {
			if role, ok := r.(string); ok && role == userRole {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, PolicyViolation{
				PolicyName: p.Name, PolicyType: "access",
				RuleKey:    "allowed_roles",
				Message:    fmt.Sprintf("角色 '%s' 不在允许列表中", userRole),
			})
		}
	}
	return violations
}

// evaluateRetentionRules 评估数据保留规则
func evaluateRetentionRules(p *datagov.DataPolicy, t *tool.Tool) []PolicyViolation {
	var violations []PolicyViolation
	rules := p.Rules

	// require_expiry_date
	if req, ok := rules["require_expiry_date"].(bool); ok && req {
		if t.Config == nil || t.Config["expiry_date"] == nil {
			violations = append(violations, PolicyViolation{
				PolicyName: p.Name, PolicyType: "retention",
				RuleKey:    "require_expiry_date",
				Message:    fmt.Sprintf("工具 '%s' 必须配置过期日期 (config.expiry_date)", t.Name),
			})
		}
	}

	// max_retention_days
	if maxDays, ok := rules["max_retention_days"].(float64); ok {
		if t.Config != nil {
			if days, ok := t.Config["retention_days"].(float64); ok && days > maxDays {
				violations = append(violations, PolicyViolation{
					PolicyName: p.Name, PolicyType: "retention",
					RuleKey:    "max_retention_days",
					Message:    fmt.Sprintf("工具 '%s' 的保留天数 (%.0f) 超过最大限制 (%.0f)", t.Name, days, maxDays),
				})
			}
		}
	}

	// auto_archive: 仅记录标记，不执行（第一版）
	if auto, ok := rules["auto_archive"].(bool); ok && auto {
		if t.Config == nil {
			t.Config = make(map[string]any)
		}
		t.Config["_auto_archive"] = true
	}

	return violations
}

// hasToolField 检查工具是否具有非空字段
func hasToolField(t *tool.Tool, field string) bool {
	switch field {
	case "name":
		return t.Name != ""
	case "type":
		return t.Type != ""
	case "description":
		return t.Description != ""
	case "endpoint":
		return t.Endpoint != ""
	case "category":
		return t.Category != ""
	case "connector_id":
		return t.ConnectorID != ""
	case "status":
		return t.Status != ""
	case "config":
		return t.Config != nil
	default:
		// 在 config map 中查找
		if t.Config != nil {
			if v, ok := t.Config[field]; ok {
				switch val := v.(type) {
				case string:
					return val != ""
				case map[string]any:
					return len(val) > 0
				default:
					return v != nil
				}
			}
		}
		return false
	}
}
