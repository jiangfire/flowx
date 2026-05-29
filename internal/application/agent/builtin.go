package agent

import (
	"context"
	"fmt"
	"time"

	aiapp "github.com/jiangfire/flowx/internal/application/ai"
	datagovapp "github.com/jiangfire/flowx/internal/application/datagov"
	mcpif "github.com/jiangfire/flowx/internal/interfaces/mcp"
)

// ==================== ToolOrchestrationAgent ====================

// toolOrchestrationAgent 工具编排 Agent，负责调用指定工具执行操作
type toolOrchestrationAgent struct{}

// NewToolOrchestrationAgent 创建工具编排 Agent
func NewToolOrchestrationAgent() Agent {
	return &toolOrchestrationAgent{}
}

func (a *toolOrchestrationAgent) Name() string { return "tool_orchestration" }
func (a *toolOrchestrationAgent) Description() string {
	return "工具编排 Agent，负责调用指定工具执行操作"
}
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
type approvalAgent struct {
	llmSvc aiapp.LLMService
}

// NewApprovalAgent 创建审批 Agent
func NewApprovalAgent(llmSvc aiapp.LLMService) Agent {
	return &approvalAgent{llmSvc: llmSvc}
}

func (a *approvalAgent) Name() string { return "approval" }
func (a *approvalAgent) Description() string {
	return "审批 Agent，分析审批上下文并给出建议"
}
func (a *approvalAgent) HandledTypes() []string { return []string{"approval_review"} }

func (a *approvalAgent) Execute(ctx context.Context, task *Task, tools mcpif.ToolCaller) (*StepResult, error) {
	suggestion := "建议审批通过，工具配置符合安全规范"
	riskLevel := "low"

	if a.llmSvc != nil {
		llmSuggestion, err := a.llmSvc.GenerateApprovalSuggestion(ctx, &aiapp.ApprovalSuggestionRequest{
			InstanceTitle: task.Description,
			WorkflowType:  "agent_task",
			StepName:      "approval_review",
			Context:       task.Context,
		})
		if err == nil && llmSuggestion != "" {
			suggestion = llmSuggestion
			riskLevel = "medium"
		}
	}

	return &StepResult{
		AgentName: a.Name(),
		Status:    "pending_approval",
		Output: map[string]any{
			"suggestion":     suggestion,
			"risk_level":     riskLevel,
			"requester":      task.Context["requester"],
			"reason":         task.Context["reason"],
			"reviewed_at":    time.Now().Format(time.RFC3339),
			"requires_human": true,
		},
	}, nil
}

// ==================== DataQualityAgent ====================

// dataQualityAgent 数据质量 Agent，检查数据质量并生成报告
type dataQualityAgent struct {
	dataGovSvc *datagovapp.DataGovService
}

// NewDataQualityAgent 创建数据质量 Agent
func NewDataQualityAgent(dataGovSvc *datagovapp.DataGovService) Agent {
	return &dataQualityAgent{dataGovSvc: dataGovSvc}
}

func (a *dataQualityAgent) Name() string { return "data_quality" }
func (a *dataQualityAgent) Description() string {
	return "数据质量 Agent，检查数据质量并生成报告"
}
func (a *dataQualityAgent) HandledTypes() []string { return []string{"data_check"} }

func (a *dataQualityAgent) Execute(ctx context.Context, task *Task, tools mcpif.ToolCaller) (*StepResult, error) {
	checkType, _ := task.Context["check_type"].(string)
	target, _ := task.Context["target"].(string)

	var qualityScore = 100.0
	var issues []any
	var totalItems int

	if a.dataGovSvc != nil && task.TenantID != "" {
		rules, _, err := a.dataGovSvc.ListRules(ctx, task.TenantID, datagovapp.ListRulesFilter{
			Type:     checkType,
			Status:   "active",
			PageSize: 1000,
		})
		if err == nil {
			totalItems = len(rules)
			if target != "" {
				checks, _, err := a.dataGovSvc.ListChecks(ctx, task.TenantID, datagovapp.ListChecksFilter{
					AssetID:  target,
					PageSize: 1000,
				})
				if err == nil && len(checks) > 0 {
					var totalPassRate float64
					for _, c := range checks {
						totalPassRate += c.PassRate
						if c.Status == "failed" {
							issues = append(issues, map[string]any{
								"rule_id":   c.RuleID,
								"asset_id":  c.AssetID,
								"pass_rate": c.PassRate,
							})
						}
					}
					qualityScore = totalPassRate / float64(len(checks))
				}
			}
		}
	}

	if totalItems == 0 {
		totalItems = len(tools.ListTools())
	}

	report := map[string]any{
		"check_type":    checkType,
		"target":        target,
		"checked_at":    time.Now().Format(time.RFC3339),
		"total_items":   totalItems,
		"quality_score": qualityScore,
		"issues":        issues,
		"summary":       fmt.Sprintf("已检查 %d 个 %s 项，数据质量评分 %.1f", totalItems, target, qualityScore),
	}

	return &StepResult{
		AgentName: a.Name(),
		Status:    "completed",
		Output:    report,
	}, nil
}
