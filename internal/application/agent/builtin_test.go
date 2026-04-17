package agent

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/interfaces/mcp"
)

// TestToolOrchestrationAgent_HandledTypes 验证 ToolOrchestrationAgent 正确声明任务类型
func TestToolOrchestrationAgent_HandledTypes(t *testing.T) {
	agent := NewToolOrchestrationAgent()

	types := agent.HandledTypes()
	if len(types) != 1 || types[0] != "tool_execute" {
		t.Errorf("期望 HandledTypes() 返回 ['tool_execute']，实际为 %v", types)
	}
}

// TestApprovalAgent_HandledTypes 验证 ApprovalAgent 正确声明任务类型
func TestApprovalAgent_HandledTypes(t *testing.T) {
	agent := NewApprovalAgent()

	types := agent.HandledTypes()
	if len(types) != 1 || types[0] != "approval_review" {
		t.Errorf("期望 HandledTypes() 返回 ['approval_review']，实际为 %v", types)
	}
}

// TestDataQualityAgent_HandledTypes 验证 DataQualityAgent 正确声明任务类型
func TestDataQualityAgent_HandledTypes(t *testing.T) {
	agent := NewDataQualityAgent()

	types := agent.HandledTypes()
	if len(types) != 1 || types[0] != "data_check" {
		t.Errorf("期望 HandledTypes() 返回 ['data_check']，实际为 %v", types)
	}
}

// TestToolOrchestrationAgent_Execute 验证 ToolOrchestrationAgent 调用工具返回结果
func TestToolOrchestrationAgent_Execute(t *testing.T) {
	registry := mcp.NewToolRegistry()
	registry.RegisterToolWithHandler(
		domainTool("echo_tool", "custom", "回显工具"),
		func(ctx context.Context, args map[string]any) (any, error) {
			return map[string]any{"echoed": args["message"]}, nil
		},
	)

	agent := NewToolOrchestrationAgent()

	task := &Task{
		ID:          "task-exec-001",
		Type:        "tool_execute",
		Description: "执行回显工具",
		Context: map[string]any{
			"tool_name": "echo_tool",
			"args":      map[string]any{"message": "hello agent"},
		},
		Steps: []TaskStep{{Type: "tool_execute"}},
	}

	result, err := agent.Execute(context.Background(), task, registry)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%s'", result.Status)
	}
}

// TestToolOrchestrationAgent_Execute_ToolNotFound 验证调用不存在的工具返回错误
func TestToolOrchestrationAgent_Execute_ToolNotFound(t *testing.T) {
	registry := mcp.NewToolRegistry()

	agent := NewToolOrchestrationAgent()

	task := &Task{
		ID:          "task-exec-002",
		Type:        "tool_execute",
		Description: "执行不存在的工具",
		Context: map[string]any{
			"tool_name": "nonexistent_tool",
			"args":      map[string]any{},
		},
		Steps: []TaskStep{{Type: "tool_execute"}},
	}

	result, err := agent.Execute(context.Background(), task, registry)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("期望状态为 'failed'，实际为 '%s'", result.Status)
	}
}

// TestApprovalAgent_Execute 验证 ApprovalAgent 返回 pending_approval
func TestApprovalAgent_Execute(t *testing.T) {
	registry := mcp.NewToolRegistry()

	agent := NewApprovalAgent()

	task := &Task{
		ID:          "task-approval-001",
		Type:        "approval_review",
		Description: "审批测试",
		Context: map[string]any{
			"requester": "user-001",
			"reason":    "工具部署审批",
		},
		Steps: []TaskStep{{Type: "approval_review"}},
	}

	result, err := agent.Execute(context.Background(), task, registry)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if result.Status != "pending_approval" {
		t.Errorf("期望状态为 'pending_approval'，实际为 '%s'", result.Status)
	}
}

// TestDataQualityAgent_Execute 验证 DataQualityAgent 返回检查结果
func TestDataQualityAgent_Execute(t *testing.T) {
	registry := mcp.NewToolRegistry()

	agent := NewDataQualityAgent()

	task := &Task{
		ID:          "task-dq-001",
		Type:        "data_check",
		Description: "数据质量检查",
		Context: map[string]any{
			"check_type": "completeness",
			"target":     "tools",
		},
		Steps: []TaskStep{{Type: "data_check"}},
	}

	result, err := agent.Execute(context.Background(), task, registry)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%s'", result.Status)
	}
}

// TestBuiltinAgents_NameAndDescription 验证内置 Agent 的名称和描述
func TestBuiltinAgents_NameAndDescription(t *testing.T) {
	agents := []Agent{
		NewToolOrchestrationAgent(),
		NewApprovalAgent(),
		NewDataQualityAgent(),
	}

	expectedNames := []string{"tool_orchestration", "approval", "data_quality"}
	for i, a := range agents {
		if a.Name() != expectedNames[i] {
			t.Errorf("期望 Agent %d 名称为 '%s'，实际为 '%s'", i, expectedNames[i], a.Name())
		}
		if a.Description() == "" {
			t.Errorf("Agent %d 描述不应为空", i)
		}
	}
}
