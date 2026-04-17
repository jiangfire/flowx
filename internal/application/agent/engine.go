package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"git.neolidy.top/neo/flowx/internal/domain/tool"
	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
)

// Task 任务定义
type Task struct {
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	Description     string         `json:"description"`
	Context         map[string]any `json:"context"`
	Steps           []TaskStep     `json:"steps"`
	RequireApproval bool           `json:"require_approval"`
	WorkflowID      string         `json:"workflow_id,omitempty"` // 关联的工作流定义 ID（可选）
}

// TaskStep 任务步骤
type TaskStep struct {
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID string         `json:"task_id"`
	Status string         `json:"status"` // completed/pending_approval/failed
	Steps  []StepResult   `json:"steps"`
	Output map[string]any `json:"output"`
}

// StepResult 步骤执行结果
type StepResult struct {
	AgentName string `json:"agent_name"`
	Status    string `json:"status"`
	Output    any    `json:"output"`
	Error     string `json:"error,omitempty"`
}

// Agent Agent 接口
type Agent interface {
	// Name 返回 Agent 名称
	Name() string
	// Description 返回 Agent 描述
	Description() string
	// HandledTypes 返回该 Agent 能处理的任务类型列表
	HandledTypes() []string
	// Execute 执行任务步骤
	Execute(ctx context.Context, task *Task, tools mcpif.ToolCaller) (*StepResult, error)
}

// AgentEngine Agent 编排引擎接口
type AgentEngine interface {
	// RegisterAgent 注册 Agent（可选指定优先级，数值越大优先级越高）
	RegisterAgent(agent Agent, priority ...int)

	// Execute 执行任务
	Execute(ctx context.Context, task *Task) (*TaskResult, error)

	// ListAvailableTools 获取可用工具列表
	ListAvailableTools(ctx context.Context) ([]mcpif.MCPToolDefinition, error)
}

// routeEntry 路由条目，关联 Agent 及其优先级
type routeEntry struct {
	agent    Agent
	priority int
}

// agentEngine Agent 编排引擎实现
type agentEngine struct {
	routes  map[string][]routeEntry // taskType -> agents that can handle it
	tools   mcpif.ToolCaller
	mu      sync.RWMutex
	taskLog []TaskResult // 任务执行日志
	taskMu  sync.Mutex
}

// NewAgentEngine 创建 Agent 编排引擎实例
func NewAgentEngine(tools mcpif.ToolCaller) AgentEngine {
	return &agentEngine{
		routes:  make(map[string][]routeEntry),
		tools:   tools,
		taskLog: make([]TaskResult, 0),
	}
}

// RegisterAgent 注册 Agent（可选指定优先级，数值越大优先级越高）
func (e *agentEngine) RegisterAgent(a Agent, priority ...int) {
	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, t := range a.HandledTypes() {
		e.routes[t] = append(e.routes[t], routeEntry{agent: a, priority: p})
	}

	slog.Info("Agent 注册成功", "name", a.Name(), "description", a.Description(), "handled_types", a.HandledTypes())
}

// Execute 执行任务
func (e *agentEngine) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	if len(task.Steps) == 0 {
		return nil, fmt.Errorf("任务 '%s' 没有步骤", task.ID)
	}

	result := &TaskResult{
		TaskID: task.ID,
		Steps:  make([]StepResult, 0, len(task.Steps)),
		Output: make(map[string]any),
	}

	for _, step := range task.Steps {
		// 查找能处理该步骤类型的 Agent
		agent := e.findAgent(step.Type)
		if agent == nil {
			result.Status = "failed"
			stepResult := StepResult{
				AgentName: "",
				Status:    "failed",
				Error:     fmt.Sprintf("没有 Agent 能处理任务类型 '%s'", step.Type),
			}
			result.Steps = append(result.Steps, stepResult)
			e.recordTask(result)
			return result, fmt.Errorf("%s", stepResult.Error)
		}

		// 执行步骤
		stepResult, err := agent.Execute(ctx, task, e.tools)
		if err != nil {
			result.Status = "failed"
			stepResult = &StepResult{
				AgentName: agent.Name(),
				Status:    "failed",
				Error:     err.Error(),
			}
			result.Steps = append(result.Steps, *stepResult)
			e.recordTask(result)
			return result, nil
		}

		result.Steps = append(result.Steps, *stepResult)

		// 如果步骤需要审批，标记整个任务为 pending_approval
		if stepResult.Status == "pending_approval" {
			result.Status = "pending_approval"
			result.Output["approval_required"] = true
			e.recordTask(result)
			return result, nil
		}
	}

	// 所有步骤完成
	result.Status = "completed"
	result.Output["message"] = "任务执行完成"
	e.recordTask(result)
	return result, nil
}

// ListAvailableTools 获取可用工具列表
func (e *agentEngine) ListAvailableTools(ctx context.Context) ([]mcpif.MCPToolDefinition, error) {
	return e.tools.ListTools(), nil
}

// GetTaskLog 获取任务执行日志（返回副本，避免并发数据竞争）
func (e *agentEngine) GetTaskLog() []TaskResult {
	e.taskMu.Lock()
	defer e.taskMu.Unlock()
	cp := make([]TaskResult, len(e.taskLog))
	copy(cp, e.taskLog)
	return cp
}

// findAgent 查找能处理指定任务类型的最高优先级 Agent
func (e *agentEngine) findAgent(taskType string) Agent {
	e.mu.RLock()
	entries := e.routes[taskType]
	e.mu.RUnlock()

	if len(entries) == 0 {
		return nil
	}

	// 复制切片以避免修改原始数据
	sorted := make([]routeEntry, len(entries))
	copy(sorted, entries)

	// 按优先级降序排序
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].priority > sorted[j].priority
	})

	return sorted[0].agent
}

// maxTaskLogSize 任务日志最大容量，防止内存无限增长
const maxTaskLogSize = 1000

// recordTask 记录任务执行结果（用于审计和回溯）
func (e *agentEngine) recordTask(result *TaskResult) {
	e.taskMu.Lock()
	defer e.taskMu.Unlock()
	e.taskLog = append(e.taskLog, *result)
	// 防止内存无限增长
	if len(e.taskLog) > maxTaskLogSize {
		e.taskLog = e.taskLog[len(e.taskLog)-maxTaskLogSize:]
	}
}

// domainTool 辅助函数：创建领域工具（测试用）
func domainTool(name, typ, desc string) *tool.Tool {
	return &tool.Tool{
		Name:        name,
		Type:        typ,
		Description: desc,
		Status:      "active",
	}
}
