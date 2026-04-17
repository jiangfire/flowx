package agent

import (
	"context"
	"errors"
	"testing"

	"git.neolidy.top/neo/flowx/internal/interfaces/mcp"
)

// mockAgent 模拟 Agent 实现
type mockAgent struct {
	name        string
	description string
	taskTypes   []string
	handleFunc  func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error)
}

func (a *mockAgent) Name() string        { return a.name }
func (a *mockAgent) Description() string  { return a.description }
func (a *mockAgent) HandledTypes() []string { return a.taskTypes }
func (a *mockAgent) Execute(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
	if a.handleFunc != nil {
		return a.handleFunc(ctx, task, tools)
	}
	return &StepResult{AgentName: a.name, Status: "completed", Output: "mock result"}, nil
}

// TestRegisterAgent_Success 验证注册 Agent 成功
func TestRegisterAgent_Success(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	agent := &mockAgent{
		name:        "test_agent",
		description: "测试 Agent",
		taskTypes:   []string{"test_task"},
	}

	eng.RegisterAgent(agent)
}

// TestExecute_SimpleTask 验证简单任务（单步，无需审批）直接完成
func TestExecute_SimpleTask(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	agent := &mockAgent{
		name:      "simple_agent",
		taskTypes: []string{"simple_task"},
	}
	eng.RegisterAgent(agent)

	task := &Task{
		ID:          "task-001",
		Type:        "simple_task",
		Description: "简单测试任务",
		Steps:       []TaskStep{{Type: "simple_task"}},
	}

	result, err := eng.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("执行任务失败: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%s'", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("期望 1 个步骤结果，实际为 %d", len(result.Steps))
	}
	if result.Steps[0].AgentName != "simple_agent" {
		t.Errorf("期望 Agent 名称为 'simple_agent'，实际为 '%s'", result.Steps[0].AgentName)
	}
}

// TestExecute_MultiStepTask 验证多步任务按顺序执行
func TestExecute_MultiStepTask(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	callOrder := []string{}
	agent1 := &mockAgent{
		name:      "agent_step1",
		taskTypes: []string{"step1"},
		handleFunc: func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
			callOrder = append(callOrder, "step1")
			return &StepResult{AgentName: "agent_step1", Status: "completed", Output: "step1 done"}, nil
		},
	}
	agent2 := &mockAgent{
		name:      "agent_step2",
		taskTypes: []string{"step2"},
		handleFunc: func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
			callOrder = append(callOrder, "step2")
			return &StepResult{AgentName: "agent_step2", Status: "completed", Output: "step2 done"}, nil
		},
	}
	eng.RegisterAgent(agent1)
	eng.RegisterAgent(agent2)

	task := &Task{
		ID:          "task-002",
		Type:        "multi_step",
		Description: "多步测试任务",
		Steps: []TaskStep{
			{Type: "step1"},
			{Type: "step2"},
		},
	}

	result, err := eng.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("执行任务失败: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%s'", result.Status)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("期望 2 个步骤结果，实际为 %d", len(result.Steps))
	}

	// 验证执行顺序
	if len(callOrder) != 2 || callOrder[0] != "step1" || callOrder[1] != "step2" {
		t.Errorf("期望执行顺序为 [step1, step2]，实际为 %v", callOrder)
	}
}

// TestExecute_RequireApproval 验证需要审批的任务返回 pending_approval
func TestExecute_RequireApproval(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	agent := &mockAgent{
		name:      "approval_agent",
		taskTypes: []string{"approval_task"},
		handleFunc: func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
			return &StepResult{
				AgentName: "approval_agent",
				Status:    "pending_approval",
				Output:    "需要人工审批",
			}, nil
		},
	}
	eng.RegisterAgent(agent)

	task := &Task{
		ID:              "task-003",
		Type:            "approval_task",
		Description:     "需要审批的任务",
		RequireApproval: true,
		Steps:           []TaskStep{{Type: "approval_task"}},
	}

	result, err := eng.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("执行任务失败: %v", err)
	}
	if result.Status != "pending_approval" {
		t.Errorf("期望状态为 'pending_approval'，实际为 '%s'", result.Status)
	}
}

// TestExecute_NoMatchingAgent 验证没有匹配的 agent 返回错误
func TestExecute_NoMatchingAgent(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	task := &Task{
		ID:          "task-004",
		Type:        "unknown_type",
		Description: "未知类型任务",
		Steps:       []TaskStep{{Type: "unknown_type"}},
	}

	_, err := eng.Execute(context.Background(), task)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
}

// TestListAvailableTools 验证返回工具列表
func TestListAvailableTools(t *testing.T) {
	registry := mcp.NewToolRegistry()
	registry.RegisterToolWithHandler(
		domainTool("test_tool", "custom", "测试工具"),
		func(ctx context.Context, args map[string]any) (any, error) {
			return "ok", nil
		},
	)

	eng := NewAgentEngine(registry)

	tools, err := eng.ListAvailableTools(context.Background())
	if err != nil {
		t.Fatalf("获取工具列表失败: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("期望 1 个工具，实际为 %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Errorf("期望工具名称为 'test_tool'，实际为 '%s'", tools[0].Name)
	}
}

// TestExecute_AgentFailed 验证 Agent 执行失败，任务状态为 failed
func TestExecute_AgentFailed(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	agent := &mockAgent{
		name:      "failing_agent",
		taskTypes: []string{"fail_task"},
		handleFunc: func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
			return nil, errors.New("模拟执行失败")
		},
	}
	eng.RegisterAgent(agent)

	task := &Task{
		ID:          "task-005",
		Type:        "fail_task",
		Description: "会失败的任务",
		Steps:       []TaskStep{{Type: "fail_task"}},
	}

	result, err := eng.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("执行任务失败: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("期望状态为 'failed'，实际为 '%s'", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("期望 1 个步骤结果，实际为 %d", len(result.Steps))
	}
	if result.Steps[0].Status != "failed" {
		t.Errorf("期望步骤状态为 'failed'，实际为 '%s'", result.Steps[0].Status)
	}
}

// TestRegisterAgent_Priority 验证优先级注册，高优先级 Agent 优先执行
func TestRegisterAgent_Priority(t *testing.T) {
	eng := NewAgentEngine(mcp.NewToolRegistry())

	var executedAgent string
	lowPriorityAgent := &mockAgent{
		name:      "low_priority_agent",
		taskTypes: []string{"shared_task"},
		handleFunc: func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
			executedAgent = "low"
			return &StepResult{AgentName: "low_priority_agent", Status: "completed"}, nil
		},
	}
	highPriorityAgent := &mockAgent{
		name:      "high_priority_agent",
		taskTypes: []string{"shared_task"},
		handleFunc: func(ctx context.Context, task *Task, tools mcp.ToolCaller) (*StepResult, error) {
			executedAgent = "high"
			return &StepResult{AgentName: "high_priority_agent", Status: "completed"}, nil
		},
	}

	// 先注册低优先级，再注册高优先级
	eng.RegisterAgent(lowPriorityAgent, 1)
	eng.RegisterAgent(highPriorityAgent, 10)

	task := &Task{
		ID:          "task-priority-001",
		Type:        "shared_task",
		Description: "优先级测试任务",
		Steps:       []TaskStep{{Type: "shared_task"}},
	}

	result, err := eng.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("执行任务失败: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%s'", result.Status)
	}
	if executedAgent != "high" {
		t.Errorf("期望执行高优先级 Agent，实际执行了 '%s'", executedAgent)
	}
}
