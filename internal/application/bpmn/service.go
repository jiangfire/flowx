package bpmn

import (
	"context"
	"fmt"

	"github.com/jiangfire/flowx/internal/domain/base"
	"github.com/jiangfire/flowx/internal/domain/bpmn"
	"gopkg.in/yaml.v3"
)

// ProcessService 流程服务，桥接内存引擎与持久化存储
type ProcessService struct {
	engine      *Engine
	defRepo     ProcessDefinitionRepository
	instRepo    ProcessInstanceRepository
	taskRepo    ProcessTaskRepository
	historyRepo ExecutionHistoryRepository
}

// NewProcessService 创建流程服务实例
func NewProcessService(engine *Engine, defRepo ProcessDefinitionRepository, instRepo ProcessInstanceRepository, taskRepo ProcessTaskRepository, historyRepo ExecutionHistoryRepository) *ProcessService {
	return &ProcessService{
		engine:      engine,
		defRepo:     defRepo,
		instRepo:    instRepo,
		taskRepo:    taskRepo,
		historyRepo: historyRepo,
	}
}

// DeployDefinition 部署流程定义
func (s *ProcessService) DeployDefinition(ctx context.Context, tenantID string, yamlData []byte) (*bpmn.ProcessDefinition, error) {
	var def bpmn.ProcessDefinition
	if err := yaml.Unmarshal(yamlData, &def); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	if def.ID == "" {
		return nil, fmt.Errorf("流程定义 ID 不能为空")
	}
	if def.Name == "" {
		return nil, fmt.Errorf("流程定义名称不能为空")
	}

	if def.Status == "" {
		def.Status = "active"
	}
	def.TenantID = tenantID

	if err := s.defRepo.Create(ctx, &def); err != nil {
		return nil, fmt.Errorf("保存流程定义失败: %w", err)
	}

	return &def, nil
}

// GetDefinition 获取流程定义
func (s *ProcessService) GetDefinition(ctx context.Context, tenantID, id string) (*bpmn.ProcessDefinition, error) {
	return s.defRepo.GetByID(ctx, tenantID, id)
}

// ListDefinitions 查询流程定义列表
func (s *ProcessService) ListDefinitions(ctx context.Context, tenantID string, filter ProcessDefinitionFilter) ([]*bpmn.ProcessDefinition, int64, error) {
	filter.TenantID = tenantID
	return s.defRepo.List(ctx, filter)
}

// StartProcess 启动流程实例
func (s *ProcessService) StartProcess(ctx context.Context, tenantID, defID, startedBy string, variables map[string]any) (*bpmn.ProcessInstance, error) {
	def, err := s.defRepo.GetByID(ctx, tenantID, defID)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}
	if def.TenantID != tenantID {
		return nil, fmt.Errorf("流程定义不存在")
	}

	// 使用内存引擎启动流程
	inst := s.engine.Start(def, tenantID, startedBy, variables)
	// 初始化 gateway state 为空
	inst.JoinState = base.JSON{}
	inst.InclusiveGatewayState = base.JSON{}

	// 持久化流程实例
	if err := s.instRepo.Create(ctx, inst); err != nil {
		return nil, fmt.Errorf("保存流程实例失败: %w", err)
	}

	// 持久化引擎产生的待办任务
	pendingTasks := s.engine.GetPendingTasks(inst.ID)
	for _, task := range pendingTasks {
		if err := s.taskRepo.Create(ctx, task); err != nil {
			return nil, fmt.Errorf("保存待办任务失败: %w", err)
		}
	}

	// 持久化执行历史
	histories := s.engine.GetHistory(inst.ID)
	for _, h := range histories {
		h.TenantID = tenantID
		if err := s.historyRepo.Create(ctx, h); err != nil {
			return nil, fmt.Errorf("保存执行历史失败: %w", err)
		}
	}

	return inst, nil
}

// GetProcessInstance 获取流程实例
func (s *ProcessService) GetProcessInstance(ctx context.Context, tenantID, id string) (*bpmn.ProcessInstance, error) {
	return s.instRepo.GetByID(ctx, tenantID, id)
}

// ListProcessInstances 查询流程实例列表
func (s *ProcessService) ListProcessInstances(ctx context.Context, tenantID string, filter ProcessInstanceFilter) ([]*bpmn.ProcessInstance, int64, error) {
	filter.TenantID = tenantID
	return s.instRepo.List(ctx, filter)
}

// SuspendProcess 挂起流程实例
func (s *ProcessService) SuspendProcess(ctx context.Context, tenantID, id string) error {
	inst, err := s.instRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// 确保引擎内存中存在该实例
	if !s.engine.HasInstance(id) {
		if err := s.ensureInstanceInEngine(ctx, tenantID, id); err != nil {
			return fmt.Errorf("流程实例运行态不可恢复: %w", err)
		}
	}

	s.engine.Suspend(id)
	inst.Status = "suspended"
	return s.persistInstanceState(ctx, inst, id)
}

// ResumeProcess 恢复流程实例
func (s *ProcessService) ResumeProcess(ctx context.Context, tenantID, id string) error {
	inst, err := s.instRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// 确保引擎内存中存在该实例
	if !s.engine.HasInstance(id) {
		if err := s.ensureInstanceInEngine(ctx, tenantID, id); err != nil {
			return fmt.Errorf("流程实例运行态不可恢复: %w", err)
		}
	}

	s.engine.Resume(id)
	inst.Status = "running"
	return s.persistInstanceState(ctx, inst, id)
}

// CancelProcess 取消流程实例
func (s *ProcessService) CancelProcess(ctx context.Context, tenantID, id string) error {
	inst, err := s.instRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// 确保引擎内存中存在该实例
	if !s.engine.HasInstance(id) {
		if err := s.ensureInstanceInEngine(ctx, tenantID, id); err != nil {
			return fmt.Errorf("流程实例运行态不可恢复: %w", err)
		}
	}

	s.engine.Cancel(id)
	inst.Status = "cancelled"

	// 更新所有未完成的任务状态为 cancelled
	pendingTasks, err := s.taskRepo.ListByInstance(ctx, tenantID, id)
	if err == nil {
		for _, t := range pendingTasks {
			if t.Status == "pending" {
				t.Status = "cancelled"
				_ = s.taskRepo.Update(ctx, t)
			}
		}
	}

	return s.persistInstanceState(ctx, inst, id)
}

// GetPendingTasks 获取待办任务
func (s *ProcessService) GetPendingTasks(ctx context.Context, tenantID, assignee string) ([]*bpmn.ProcessTask, error) {
	return s.taskRepo.ListPending(ctx, tenantID, assignee)
}

// CompleteTask 完成任务
func (s *ProcessService) CompleteTask(ctx context.Context, tenantID, taskID, completedBy string, submittedData map[string]any) error {
	task, err := s.taskRepo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return err
	}
	if task.TenantID != tenantID {
		return fmt.Errorf("流程任务不存在")
	}

	// 确保引擎内存中存在该实例，若不在则尝试从 DB 恢复
	if !s.engine.HasInstance(task.InstanceID) {
		if err := s.ensureInstanceInEngine(ctx, tenantID, task.InstanceID); err != nil {
			return fmt.Errorf("流程实例运行态不可恢复: %w", err)
		}
	}

	if err := s.engine.CompleteTask(task.InstanceID, taskID, completedBy, submittedData); err != nil {
		return err
	}

	// 更新任务状态
	task.Status = "completed"
	task.CompletedBy = completedBy
	if submittedData != nil {
		task.SubmittedData = submittedData
	}
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 持久化引擎新产生的待办任务（跳过已持久化的部分）
	allPending := s.engine.GetPendingTasks(task.InstanceID)
	// 获取已持久化的任务 ID 集合
	existingTasks, _ := s.taskRepo.ListByInstance(ctx, tenantID, task.InstanceID)
	existingTaskIDs := make(map[string]bool)
	for _, et := range existingTasks {
		existingTaskIDs[et.ID] = true
	}
	for _, t := range allPending {
		if !existingTaskIDs[t.ID] {
			if err := s.taskRepo.Create(ctx, t); err != nil {
				return fmt.Errorf("保存新待办任务失败: %w", err)
			}
		}
	}

	// 持久化新增的执行历史（跳过已持久化的部分）
	allHistories := s.engine.GetHistory(task.InstanceID)
	// 从 DB 获取已持久化的历史数量
	existingHistories, _ := s.historyRepo.ListByInstance(ctx, tenantID, task.InstanceID)
	existingCount := len(existingHistories)
	if len(allHistories) > existingCount {
		newHistories := allHistories[existingCount:]
		for _, h := range newHistories {
			h.TenantID = tenantID
			if err := s.historyRepo.Create(ctx, h); err != nil {
				return fmt.Errorf("保存执行历史失败: %w", err)
			}
		}
	}

	// 更新流程实例状态（包含 gateway state）
	engineInst := s.engine.GetInstance(task.InstanceID)
	if engineInst != nil {
		if err := s.persistInstanceState(ctx, engineInst, task.InstanceID); err != nil {
			return err
		}
	}

	return nil
}

// GetProcessTasks 获取流程实例的任务列表
func (s *ProcessService) GetProcessTasks(ctx context.Context, tenantID, instanceID string) ([]*bpmn.ProcessTask, error) {
	return s.taskRepo.ListByInstance(ctx, tenantID, instanceID)
}

// GetProcessHistory 获取流程实例的执行历史
func (s *ProcessService) GetProcessHistory(ctx context.Context, tenantID, instanceID string) ([]*bpmn.ExecutionHistory, error) {
	return s.historyRepo.ListByInstance(ctx, tenantID, instanceID)
}

// ensureInstanceInEngine 从持久化存储重建运行时状态到内存引擎
func (s *ProcessService) ensureInstanceInEngine(ctx context.Context, tenantID, instanceID string) error {
	inst, err := s.instRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}
	if inst.TenantID != tenantID {
		return fmt.Errorf("流程实例不存在")
	}

	def, err := s.defRepo.GetByID(ctx, tenantID, inst.DefinitionID)
	if err != nil {
		return err
	}

	tasks, err := s.taskRepo.ListByInstance(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}

	histories, err := s.historyRepo.ListByInstance(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}

	// 恢复 gateway state
	joinState := make(map[string]int)
	if inst.JoinState != nil {
		for k, v := range inst.JoinState {
			if count, ok := v.(float64); ok {
				joinState[k] = int(count)
			}
		}
	}
	inclusiveState := make(map[string]int)
	if inst.InclusiveGatewayState != nil {
		for k, v := range inst.InclusiveGatewayState {
			if count, ok := v.(float64); ok {
				inclusiveState[k] = int(count)
			}
		}
	}

	return s.engine.RestoreInstanceWithGatewayState(inst, def, tasks, histories, joinState, inclusiveState)
}

// persistInstanceState 持久化流程实例状态（包含 gateway state）
func (s *ProcessService) persistInstanceState(ctx context.Context, engineInst *bpmn.ProcessInstance, instanceID string) error {
	joinState, inclusiveState, err := s.engine.GetGatewayState(instanceID)
	if err != nil {
		return fmt.Errorf("获取 gateway state 失败: %w", err)
	}

	engineInst.JoinState = base.JSON{}
	for k, v := range joinState {
		engineInst.JoinState[k] = v
	}
	engineInst.InclusiveGatewayState = base.JSON{}
	for k, v := range inclusiveState {
		engineInst.InclusiveGatewayState[k] = v
	}

	if err := s.instRepo.Update(ctx, engineInst); err != nil {
		return fmt.Errorf("更新流程实例状态失败: %w", err)
	}
	return nil
}

// RestoreRunningInstances 启动时恢复所有运行中和挂起的流程实例到引擎
func (s *ProcessService) RestoreRunningInstances(ctx context.Context) error {
	instances, _, err := s.instRepo.List(ctx, ProcessInstanceFilter{
		Status:   "running",
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return fmt.Errorf("查询运行中流程实例失败: %w", err)
	}

	suspended, _, err := s.instRepo.List(ctx, ProcessInstanceFilter{
		Status:   "suspended",
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return fmt.Errorf("查询挂起流程实例失败: %w", err)
	}

	instances = append(instances, suspended...)

	for _, inst := range instances {
		if err := s.ensureInstanceInEngine(ctx, inst.TenantID, inst.ID); err != nil {
			// 记录错误但不中断，允许部分实例恢复
			fmt.Printf("恢复流程实例 %s 失败: %v\n", inst.ID, err)
		}
	}

	return nil
}
