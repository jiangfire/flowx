package approval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"git.neolidy.top/neo/flowx/internal/application/ai"
	"git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/pagination"
	"git.neolidy.top/neo/flowx/pkg/tenant"
)

// 定义审批服务错误
var (
	ErrWorkflowNotFound  = errors.New("工作流不存在")
	ErrInstanceNotFound  = errors.New("工作流实例不存在")
	ErrApprovalNotFound  = errors.New("待审批记录不存在")
	ErrAlreadyApproved   = errors.New("该步骤已审批")
	ErrInstanceCancelled = errors.New("实例已取消")
	ErrInstanceFinished  = errors.New("实例已结束")
	ErrTenantMismatch    = errors.New("租户不匹配")
)

// CreateWorkflowRequest 创建工作流请求
type CreateWorkflowRequest struct {
	Name        string    `json:"name" binding:"required,max=200"`
	Type        string    `json:"type" binding:"required,max=50"`
	Description string    `json:"description"`
	Definition  base.JSON `json:"definition" binding:"required"`
}

// StartApprovalRequest 发起审批请求
type StartApprovalRequest struct {
	WorkflowID string    `json:"workflow_id" binding:"required"`
	Title      string    `json:"title" binding:"required,max=500"`
	Context    base.JSON `json:"context"`
}

// ApproveRequest 审批通过请求
type ApproveRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Comment    string `json:"comment"`
}

// RejectRequest 审批驳回请求
type RejectRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Comment    string `json:"comment" binding:"required"`
}

// ForwardRequest 转审请求
type ForwardRequest struct {
	InstanceID   string `json:"instance_id" binding:"required"`
	ToApproverID string `json:"to_approver_id" binding:"required"`
	Comment      string `json:"comment"`
}

// ApprovalService 审批服务接口
type ApprovalService interface {
	// Workflow
	CreateWorkflow(ctx context.Context, tenantID string, req *CreateWorkflowRequest) (*approval.Workflow, error)
	GetWorkflow(ctx context.Context, tenantID string, id string) (*approval.Workflow, error)
	ListWorkflows(ctx context.Context, tenantID string, filter WorkflowFilter) ([]approval.Workflow, *pagination.PaginatedResult, error)
	ActivateWorkflow(ctx context.Context, tenantID string, id string) (*approval.Workflow, error)
	ArchiveWorkflow(ctx context.Context, tenantID string, id string) (*approval.Workflow, error)

	// Instance
	StartApproval(ctx context.Context, tenantID string, initiatorID string, req *StartApprovalRequest) (*approval.WorkflowInstance, error)
	GetInstance(ctx context.Context, tenantID string, id string) (*approval.WorkflowInstance, error)
	ListInstances(ctx context.Context, tenantID string, filter InstanceFilter) ([]approval.WorkflowInstance, *pagination.PaginatedResult, error)
	CancelInstance(ctx context.Context, tenantID string, id string) error
	UpdateInstance(ctx context.Context, inst *approval.WorkflowInstance) error

	// Approval actions
	Approve(ctx context.Context, tenantID string, approverID string, req *ApproveRequest) (*approval.Approval, error)
	Reject(ctx context.Context, tenantID string, approverID string, req *RejectRequest) (*approval.Approval, error)
	Forward(ctx context.Context, tenantID string, approverID string, req *ForwardRequest) (*approval.Approval, error)

	// AI suggestion
	GetSuggestion(ctx context.Context, tenantID string, instanceID string) (string, error)

	// My pending approvals
	GetMyPendingApprovals(ctx context.Context, tenantID string, approverID string) ([]approval.WorkflowInstance, error)
}

// approvalService 审批服务实现
type approvalService struct {
	repo   ApprovalRepository
	llmSvc ai.LLMService
}

// NewApprovalService 创建审批服务实例
func NewApprovalService(repo ApprovalRepository, llmSvc ai.LLMService) ApprovalService {
	return &approvalService{
		repo:   repo,
		llmSvc: llmSvc,
	}
}

// ===================== Workflow =====================

// CreateWorkflow 创建工作流
func (s *approvalService) CreateWorkflow(ctx context.Context, tenantID string, req *CreateWorkflowRequest) (*approval.Workflow, error) {
	w := &approval.Workflow{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Definition:  req.Definition,
		Version:     1,
		Status:      "draft",
	}
	if err := s.repo.CreateWorkflow(ctx, w); err != nil {
		return nil, fmt.Errorf("创建工作流失败: %w", err)
	}
	return w, nil
}

// GetWorkflow 获取工作流详情
func (s *approvalService) GetWorkflow(ctx context.Context, tenantID string, id string) (*approval.Workflow, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	w, err := s.repo.GetWorkflowByID(tenantCtx, id)
	if err != nil {
		return nil, ErrWorkflowNotFound
	}
	return w, nil
}

// ListWorkflows 列出工作流
func (s *approvalService) ListWorkflows(ctx context.Context, tenantID string, filter WorkflowFilter) ([]approval.Workflow, *pagination.PaginatedResult, error) {
	filter.TenantID = tenantID
	workflows, total, err := s.repo.ListWorkflows(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("查询工作流列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return workflows, pagination.NewResult(total, page, pageSize), nil
}

// ActivateWorkflow 激活工作流
func (s *approvalService) ActivateWorkflow(ctx context.Context, tenantID string, id string) (*approval.Workflow, error) {
	w, err := s.GetWorkflow(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if w.Status != "draft" {
		if w.Status == "active" {
			return nil, errors.New("工作流已激活，无需重复激活")
		}
		return nil, errors.New("工作流已归档，无法激活")
	}
	if err := s.repo.UpdateWorkflowStatus(ctx, id, "active"); err != nil {
		return nil, fmt.Errorf("激活工作流失败: %w", err)
	}
	w.Status = "active"
	return w, nil
}

// ArchiveWorkflow 归档工作流
func (s *approvalService) ArchiveWorkflow(ctx context.Context, tenantID string, id string) (*approval.Workflow, error) {
	w, err := s.GetWorkflow(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if w.Status == "archived" {
		return nil, errors.New("工作流已归档")
	}
	if err := s.repo.UpdateWorkflowStatus(ctx, id, "archived"); err != nil {
		return nil, fmt.Errorf("归档工作流失败: %w", err)
	}
	w.Status = "archived"
	return w, nil
}

// ===================== Instance =====================

// StartApproval 发起审批
func (s *approvalService) StartApproval(ctx context.Context, tenantID string, initiatorID string, req *StartApprovalRequest) (*approval.WorkflowInstance, error) {
	// 获取工作流定义
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	w, err := s.repo.GetWorkflowByID(tenantCtx, req.WorkflowID)
	if err != nil {
		return nil, ErrWorkflowNotFound
	}

	// 检查工作流状态
	if w.Status != "active" {
		if w.Status == "archived" {
			return nil, errors.New("工作流已归档，无法发起审批")
		}
		return nil, errors.New("工作流未激活，无法发起审批")
	}

	// 创建实例
	inst := &approval.WorkflowInstance{
		BaseModel:   base.BaseModel{TenantID: tenantID},
		WorkflowID:  w.ID,
		Title:       req.Title,
		Status:      "approving",
		CurrentStep: 0,
		InitiatorID: initiatorID,
		Context:     req.Context,
	}
	if err := s.repo.CreateInstance(ctx, inst); err != nil {
		return nil, fmt.Errorf("创建实例失败: %w", err)
	}

	// 从工作流定义中提取第一步审批人，创建审批记录
	steps, ok := w.Definition["steps"].([]any)
	if ok && len(steps) > 0 {
		firstStep := steps[0]
		stepMap, ok := firstStep.(map[string]any)
		if ok {
			approvers, ok := stepMap["approvers"].([]any)
			if ok && len(approvers) > 0 {
				approverID, ok := approvers[0].(string)
				if ok {
					a := &approval.Approval{
						BaseModel:  base.BaseModel{TenantID: tenantID},
						InstanceID: inst.ID,
						Step:       1,
						ApproverID: approverID,
						Status:     "pending",
					}
					if err := s.repo.CreateApproval(ctx, a); err != nil {
						// 审批已成功创建，记录创建失败但不影响主流程
						slog.Error("创建审批记录失败", "error", err, "instance_id", inst.ID)
					}
				}
			}
		}
	}

	return inst, nil
}

// GetInstance 获取实例详情
func (s *approvalService) GetInstance(ctx context.Context, tenantID string, id string) (*approval.WorkflowInstance, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	inst, err := s.repo.GetInstanceByID(tenantCtx, id)
	if err != nil {
		return nil, ErrInstanceNotFound
	}
	return inst, nil
}

// ListInstances 列出实例
func (s *approvalService) ListInstances(ctx context.Context, tenantID string, filter InstanceFilter) ([]approval.WorkflowInstance, *pagination.PaginatedResult, error) {
	filter.TenantID = tenantID
	instances, total, err := s.repo.ListInstances(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("查询工作流实例列表失败: %w", err)
	}

	page, pageSize := pagination.NormalizePage(filter.Page, filter.PageSize)
	return instances, pagination.NewResult(total, page, pageSize), nil
}

// CancelInstance 取消实例
func (s *approvalService) CancelInstance(ctx context.Context, tenantID string, id string) error {
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	inst, err := s.repo.GetInstanceByID(tenantCtx, id)
	if err != nil {
		return ErrInstanceNotFound
	}

	if inst.Status == "approved" || inst.Status == "rejected" || inst.Status == "cancelled" {
		return ErrInstanceFinished
	}

	inst.Status = "cancelled"
	return s.repo.UpdateInstance(ctx, inst)
}

// UpdateInstance 更新工作流实例（内部使用，用于回写关联字段）
func (s *approvalService) UpdateInstance(ctx context.Context, inst *approval.WorkflowInstance) error {
	return s.repo.UpdateInstance(ctx, inst)
}

// ===================== Approval Actions =====================

// Approve 审批通过
func (s *approvalService) Approve(ctx context.Context, tenantID string, approverID string, req *ApproveRequest) (*approval.Approval, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	inst, err := s.repo.GetInstanceByID(tenantCtx, req.InstanceID)
	if err != nil {
		return nil, ErrInstanceNotFound
	}

	if inst.Status != "approving" {
		return nil, ErrInstanceFinished
	}

	// 获取当前步骤的待审批记录
	currentStep := inst.CurrentStep + 1
	pendingApproval, err := s.repo.GetPendingApproval(ctx, req.InstanceID, currentStep)
	if err != nil {
		return nil, ErrApprovalNotFound
	}

	// 验证审批人
	if pendingApproval.ApproverID != approverID {
		return nil, ErrApprovalNotFound
	}

	// 更新审批记录
	now := time.Now()
	pendingApproval.Status = "approved"
	pendingApproval.Comment = req.Comment
	pendingApproval.ReviewedAt = &now
	if err := s.repo.UpdateApproval(ctx, pendingApproval); err != nil {
		return nil, fmt.Errorf("更新审批记录失败: %w", err)
	}

	// 获取工作流定义，检查是否还有下一步
	tenantCtx2 := tenant.WithTenantID(ctx, tenantID)
	w, err := s.repo.GetWorkflowByID(tenantCtx2, inst.WorkflowID)
	if err != nil {
		return pendingApproval, nil // 审批已成功，工作流获取失败不影响
	}

	steps, ok := w.Definition["steps"].([]any)
	if ok {
		inst.CurrentStep = currentStep
		if currentStep >= len(steps) {
			// 最后一步审批通过
			inst.Status = "approved"
		} else {
			// 创建下一步审批记录
			nextStep := steps[currentStep]
			stepMap, ok := nextStep.(map[string]any)
			if ok {
				approvers, ok := stepMap["approvers"].([]any)
				if ok && len(approvers) > 0 {
					nextApproverID, ok := approvers[0].(string)
					if ok {
						a := &approval.Approval{
							BaseModel:  base.BaseModel{TenantID: tenantID},
							InstanceID: inst.ID,
							Step:       currentStep + 1,
							ApproverID: nextApproverID,
							Status:     "pending",
						}
						if err := s.repo.CreateApproval(ctx, a); err != nil {
							// 审批已成功，记录创建失败但不影响主流程
							slog.Error("创建下一步审批记录失败", "error", err, "instance_id", inst.ID)
						}
					}
				}
			}
		}
		if err := s.repo.UpdateInstance(ctx, inst); err != nil {
			return nil, fmt.Errorf("更新工作流实例失败: %w", err)
		}
	}

	return pendingApproval, nil
}

// Reject 审批驳回
func (s *approvalService) Reject(ctx context.Context, tenantID string, approverID string, req *RejectRequest) (*approval.Approval, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	inst, err := s.repo.GetInstanceByID(tenantCtx, req.InstanceID)
	if err != nil {
		return nil, ErrInstanceNotFound
	}

	if inst.Status != "approving" {
		return nil, ErrInstanceFinished
	}

	currentStep := inst.CurrentStep + 1
	pendingApproval, err := s.repo.GetPendingApproval(ctx, req.InstanceID, currentStep)
	if err != nil {
		return nil, ErrApprovalNotFound
	}

	if pendingApproval.ApproverID != approverID {
		return nil, ErrApprovalNotFound
	}

	now := time.Now()
	pendingApproval.Status = "rejected"
	pendingApproval.Comment = req.Comment
	pendingApproval.ReviewedAt = &now
	if err := s.repo.UpdateApproval(ctx, pendingApproval); err != nil {
		return nil, fmt.Errorf("更新审批记录失败: %w", err)
	}

	// 更新实例状态
	inst.Status = "rejected"
	if err := s.repo.UpdateInstance(ctx, inst); err != nil {
		return nil, fmt.Errorf("更新工作流实例失败: %w", err)
	}

	return pendingApproval, nil
}

// Forward 转审
func (s *approvalService) Forward(ctx context.Context, tenantID string, approverID string, req *ForwardRequest) (*approval.Approval, error) {
	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	inst, err := s.repo.GetInstanceByID(tenantCtx, req.InstanceID)
	if err != nil {
		return nil, ErrInstanceNotFound
	}

	if inst.Status != "approving" {
		return nil, ErrInstanceFinished
	}

	currentStep := inst.CurrentStep + 1
	pendingApproval, err := s.repo.GetPendingApproval(ctx, req.InstanceID, currentStep)
	if err != nil {
		return nil, ErrApprovalNotFound
	}

	if pendingApproval.ApproverID != approverID {
		return nil, ErrApprovalNotFound
	}

	// 更新原审批记录为已转审
	now := time.Now()
	pendingApproval.Status = "forwarded"
	pendingApproval.Comment = req.Comment
	pendingApproval.ReviewedAt = &now
	if err := s.repo.UpdateApproval(ctx, pendingApproval); err != nil {
		return nil, fmt.Errorf("更新审批记录失败: %w", err)
	}

	// 为转审人创建新的待审批记录
	newApproval := &approval.Approval{
		BaseModel:  base.BaseModel{TenantID: tenantID},
		InstanceID: inst.ID,
		Step:       currentStep,
		ApproverID: req.ToApproverID,
		Status:     "pending",
	}
	if err := s.repo.CreateApproval(ctx, newApproval); err != nil {
		return nil, fmt.Errorf("创建转审记录失败: %w", err)
	}

	return pendingApproval, nil
}

// ===================== AI Suggestion =====================

// GetSuggestion 获取 AI 审批建议
func (s *approvalService) GetSuggestion(ctx context.Context, tenantID string, instanceID string) (string, error) {
	if s.llmSvc == nil {
		return "", errors.New("LLM 服务未配置")
	}

	tenantCtx := tenant.WithTenantID(ctx, tenantID)
	inst, err := s.repo.GetInstanceByID(tenantCtx, instanceID)
	if err != nil {
		return "", ErrInstanceNotFound
	}

	// 获取工作流信息
	tenantCtx2 := tenant.WithTenantID(ctx, tenantID)
	w, err := s.repo.GetWorkflowByID(tenantCtx2, inst.WorkflowID)
	if err != nil {
		return "", ErrWorkflowNotFound
	}

	// 获取审批历史
	approvals, err := s.repo.ListApprovalsByInstance(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("获取审批历史失败: %w", err)
	}

	// 构建历史记录
	var history []ai.ApprovalHistory
	for _, a := range approvals {
		history = append(history, ai.ApprovalHistory{
			ApproverID: a.ApproverID,
			Status:     a.Status,
			Comment:    a.Comment,
		})
	}

	// 获取当前步骤名称
	stepName := ""
	steps, ok := w.Definition["steps"].([]any)
	if ok && inst.CurrentStep < len(steps) {
		stepMap, ok := steps[inst.CurrentStep].(map[string]any)
		if ok {
			if name, ok := stepMap["name"].(string); ok {
				stepName = name
			}
		}
	}

	// 调用 LLM 服务
	suggestion, err := s.llmSvc.GenerateApprovalSuggestion(ctx, &ai.ApprovalSuggestionRequest{
		InstanceTitle: inst.Title,
		WorkflowType:  w.Type,
		StepName:      stepName,
		Context:       inst.Context,
		History:       history,
	})
	if err != nil {
		return "", fmt.Errorf("生成审批建议失败: %w", err)
	}

	return suggestion, nil
}

// ===================== My Pending Approvals =====================

// GetMyPendingApprovals 获取我的待审批列表
func (s *approvalService) GetMyPendingApprovals(ctx context.Context, tenantID string, approverID string) ([]approval.WorkflowInstance, error) {
	return s.repo.GetPendingApprovalsByApprover(ctx, tenantID, approverID)
}
