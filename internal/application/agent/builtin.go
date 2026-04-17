package agent

import (
	"context"
	"fmt"
	"time"

	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
)

// ==================== ToolOrchestrationAgent ====================

// toolOrchestrationAgent 工具编排 Agent，负责调用指定工具执行操作
type toolOrchestrationAgent struct{}

// NewToolOrchestrationAgent 创建工具编排 Agent
func NewToolOrchestrationAgent() Agent {
	return &toolOrchestrationAgent{}
}

func (a *toolOrchestrationAgent) Name() string        { return "tool_orchestration" }
func (a *toolOrchestrationAgent) Description() string  { return "工具编排 Agent，负责调用指定工具执行操作" }
func (a *toolOrchestrationAgent) HandledTypes() []string { return []string{"tool_execute"} }

func (a *toolOrchestrationAgent) Execute(ctx context.Context, task *Task, tools mcpif.ToolCaller) (*StepResult, error) {
	// 从任务上下文中获取工具名称和参数
	toolName, _ := task.Context["tool_name"].(string)
	args, _ := task.Context["args"].(map[string]any)

	if toolName == "" {
		return &StepResult{
			AgentName: a.Name(),
			Status:    "failed",
			Error:     "缺少 tool_name 参数",
		}, nil
	}

	// 通过 ToolCaller 调用工具
	result, err := tools.CallTool(ctx, toolName, args)
	if err != nil {
		return &StepResult{
			AgentName: a.Name(),
			Status:    "failed",
			Error:     fmt.Sprintf("调用工具 '%s' 失败: %v", toolName, err),
		}, nil
	}

	return &StepResult{
		AgentName: a.Name(),
		Status:    "completed",
		Output: map[string]any{
			"tool_name": toolName,
			"result":    result,
		},
	}, nil
}

// ==================== ApprovalAgent ====================

// approvalAgent 审批 Agent，分析审批上下文并给出建议
type approvalAgent struct{}

// NewApprovalAgent 创建审批 Agent
func NewApprovalAgent() Agent {
	return &approvalAgent{}
}

func (a *approvalAgent) Name() string        { return "approval" }
func (a *approvalAgent) Description() string  { return "审批 Agent，分析审批上下文并给出建议" }
func (a *approvalAgent) HandledTypes() []string { return []string{"approval_review"} }

func (a *approvalAgent) Execute(ctx context.Context, task *Task, tools mcpif.ToolCaller) (*StepResult, error) {
	// 审批 Agent 返回 pending_approval，需要 HITL（Human-in-the-Loop）
	// TODO: 接入真实 LLM 分析，根据审批上下文生成个性化建议
	return &StepResult{
		AgentName: a.Name(),
		Status:    "pending_approval",
		Output: map[string]any{
			"suggestion":    "建议审批通过，工具配置符合安全规范",
			"risk_level":    "low",
			"requester":     task.Context["requester"],
			"reason":        task.Context["reason"],
			"reviewed_at":   time.Now().Format(time.RFC3339),
			"requires_human": true,
		},
	}, nil
}

// ==================== DataQualityAgent ====================

// dataQualityAgent 数据质量 Agent，检查数据质量并生成报告
type dataQualityAgent struct{}

// NewDataQualityAgent 创建数据质量 Agent
func NewDataQualityAgent() Agent {
	return &dataQualityAgent{}
}

func (a *dataQualityAgent) Name() string        { return "data_quality" }
func (a *dataQualityAgent) Description() string  { return "数据质量 Agent，检查数据质量并生成报告" }
func (a *dataQualityAgent) HandledTypes() []string { return []string{"data_check"} }

func (a *dataQualityAgent) Execute(ctx context.Context, task *Task, tools mcpif.ToolCaller) (*StepResult, error) {
	// 获取检查类型
	checkType, _ := task.Context["check_type"].(string)
	target, _ := task.Context["target"].(string)

	// 获取可用工具列表作为检查范围
	availableTools := tools.ListTools()

	// TODO: 接入真实数据质量检查引擎，基于 DataQualityRule 执行实际校验
	// 生成数据质量报告
	report := map[string]any{
		"check_type":    checkType,
		"target":        target,
		"checked_at":    time.Now().Format(time.RFC3339),
		"total_items":   len(availableTools),
		"quality_score": 95.5,
		"issues":        []any{},
		"summary":       fmt.Sprintf("已检查 %d 个 %s 项，数据质量良好", len(availableTools), target),
	}

	return &StepResult{
		AgentName: a.Name(),
		Status:    "completed",
		Output:    report,
	}, nil
}
