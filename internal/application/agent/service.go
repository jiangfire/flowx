package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	approvalapp "git.neolidy.top/neo/flowx/internal/application/approval"
	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
	domainagent "git.neolidy.top/neo/flowx/internal/domain/agent"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/pagination"
)

// Sentinel errors for Agent service
var (
	ErrTaskNotFound   = errors.New("任务不存在")
	ErrTaskNotPending = errors.New("任务不在待审批状态")
)

// AgentService Agent 应用服务
type AgentService struct {
	engine     AgentEngine
	repo       AgentTaskRepository
	approvalSvc approvalapp.ApprovalService // can be nil
}

// NewAgentService 创建 Agent 服务实例
func NewAgentService(engine AgentEngine, repo AgentTaskRepository, approvalSvc approvalapp.ApprovalService) *AgentService {
	return &AgentService{engine: engine, repo: repo, approvalSvc: approvalSvc}
}

// ListAvailableTools 获取可用工具列表
func (s *AgentService) ListAvailableTools(ctx context.Context) ([]mcpif.MCPToolDefinition, error) {
	return s.engine.ListAvailableTools(ctx)
}

// CreateAndExecuteTask 创建并执行任务
func (s *AgentService) CreateAndExecuteTask(ctx context.Context, tenantID, userID string, task *Task) (*TaskResult, error) {
	// 持久化任务记录
	agentTask := &domainagent.AgentTask{
		BaseModel:       base.BaseModel{TenantID: tenantID},
		Type:            task.Type,
		Description:     task.Description,
		Status:          "running",
		RequireApproval: task.RequireApproval,
		CreatedBy:       userID,
	}
	if contextJSON, err := json.Marshal(task.Context); err == nil {
		agentTask.Context = string(contextJSON)
	}
	if stepsJSON, err := json.Marshal(task.Steps); err == nil {
		agentTask.Steps = string(stepsJSON)
	}

	// 如果提供了 workflow_id，创建关联的工作流实例
	if task.WorkflowID != "" && s.approvalSvc != nil {
		var ctxJSON base.JSON
		if task.Context != nil {
			ctxJSON = task.Context
		}
		inst, err := s.approvalSvc.StartApproval(ctx, tenantID, userID, &approvalapp.StartApprovalRequest{
			WorkflowID: task.WorkflowID,
			Title:      fmt.Sprintf("Agent 任务审批: %s", task.Description),
			Context:    ctxJSON,
		})
		if err != nil {
			return nil, fmt.Errorf("创建关联工作流实例失败: %w", err)
		}
		// 设置双向关联
		agentTask.WorkflowInstanceID = inst.ID
		inst.AgentTaskID = agentTask.ID
		// 回写 agent_task_id 到工作流实例
		_ = s.approvalSvc.UpdateInstance(ctx, inst)
	}

	if err := s.repo.Create(ctx, agentTask); err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	// 执行任务
	result, err := s.engine.Execute(ctx, task)
	if err != nil {
		agentTask.Status = "failed"
		_ = s.repo.Update(ctx, agentTask)
		return nil, fmt.Errorf("执行任务失败: %w", err)
	}

	// 使用持久化的 AgentTask ID 作为 TaskID
	result.TaskID = agentTask.ID

	// 更新任务状态
	resultJSON, _ := json.Marshal(result)
	agentTask.Status = result.Status
	agentTask.Result = string(resultJSON)
	if updateErr := s.repo.Update(ctx, agentTask); updateErr != nil {
		slog.Error("更新任务状态失败", "error", updateErr, "task_id", task.ID)
	}

	return result, nil
}

// ListTasks 查询任务列表
func (s *AgentService) ListTasks(ctx context.Context, tenantID string, status string, page, pageSize int) ([]domainagent.AgentTask, *pagination.PaginatedResult, error) {
	tasks, total, err := s.repo.List(ctx, tenantID, status, page, pageSize)
	if err != nil {
		return nil, nil, fmt.Errorf("查询任务列表失败: %w", err)
	}

	page, pageSize = pagination.NormalizePage(page, pageSize)
	return tasks, pagination.NewResult(total, page, pageSize), nil
}

// GetTask 获取任务详情
func (s *AgentService) GetTask(ctx context.Context, tenantID, taskID string) (*domainagent.AgentTask, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	if task.TenantID != tenantID {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// ApproveTask 审批通过任务
func (s *AgentService) ApproveTask(ctx context.Context, tenantID, userID, taskID, comment string) (*domainagent.AgentTask, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	if task.TenantID != tenantID {
		return nil, ErrTaskNotFound
	}

	if task.Status != "pending_approval" {
		return nil, ErrTaskNotPending
	}

	task.Status = "approved"
	task.ApprovedBy = userID
	task.ApprovalComment = comment
	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("审批操作失败: %w", err)
	}

	// 如果任务关联了工作流实例，调用审批服务推进工作流
	if s.approvalSvc != nil && task.WorkflowInstanceID != "" {
		_, _ = s.approvalSvc.Approve(ctx, tenantID, task.CreatedBy, &approvalapp.ApproveRequest{
			InstanceID: task.WorkflowInstanceID,
			Comment:    comment,
		})
		// Ignore error — task approval succeeded, workflow advancement is best-effort
	}

	return task, nil
}

// RejectTask 拒绝任务
func (s *AgentService) RejectTask(ctx context.Context, tenantID, userID, taskID, comment string) (*domainagent.AgentTask, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	if task.TenantID != tenantID {
		return nil, ErrTaskNotFound
	}

	if task.Status != "pending_approval" {
		return nil, ErrTaskNotPending
	}

	task.Status = "rejected"
	task.ApprovedBy = userID
	task.ApprovalComment = comment
	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("拒绝操作失败: %w", err)
	}

	// 如果任务关联了工作流实例，调用审批服务驳回工作流
	if s.approvalSvc != nil && task.WorkflowInstanceID != "" {
		_, _ = s.approvalSvc.Reject(ctx, tenantID, task.CreatedBy, &approvalapp.RejectRequest{
			InstanceID: task.WorkflowInstanceID,
			Comment:    comment,
		})
		// Ignore error — task rejection succeeded, workflow advancement is best-effort
	}

	return task, nil
}

// TaskToResponse 将 AgentTask 转换为响应格式
func TaskToResponse(t domainagent.AgentTask) map[string]any {
	resp := map[string]any{
		"id":               t.ID,
		"type":             t.Type,
		"description":      t.Description,
		"status":           t.Status,
		"require_approval": t.RequireApproval,
		"created_by":       t.CreatedBy,
		"approved_by":      t.ApprovedBy,
		"approval_comment": t.ApprovalComment,
		"created_at":       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":       t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if t.Context != "" {
		var ctx map[string]any
		if json.Unmarshal([]byte(t.Context), &ctx) == nil {
			resp["context"] = ctx
		}
	}
	if t.Result != "" {
		var result map[string]any
		if json.Unmarshal([]byte(t.Result), &result) == nil {
			resp["result"] = result
		}
	}

	return resp
}
