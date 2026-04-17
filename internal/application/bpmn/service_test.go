package bpmn

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
)

// ==================== 内存模拟仓储实现 ====================

// mockDefRepo 流程定义仓储的内存模拟实现
type mockDefRepo struct {
	mu    sync.RWMutex
	store map[string]*bpmn.ProcessDefinition
}

func newMockDefRepo() *mockDefRepo {
	return &mockDefRepo{store: make(map[string]*bpmn.ProcessDefinition)}
}

func (r *mockDefRepo) Create(_ context.Context, def *bpmn.ProcessDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[def.ID] = def
	return nil
}

func (r *mockDefRepo) GetByID(_ context.Context, id string) (*bpmn.ProcessDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.store[id]
	if !ok {
		return nil, fmt.Errorf("流程定义 %s 不存在", id)
	}
	return def, nil
}

func (r *mockDefRepo) List(_ context.Context, filter ProcessDefinitionFilter) ([]*bpmn.ProcessDefinition, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*bpmn.ProcessDefinition
	for _, def := range r.store {
		if filter.Status != "" && def.Status != filter.Status {
			continue
		}
		result = append(result, def)
	}
	return result, int64(len(result)), nil
}

func (r *mockDefRepo) Update(_ context.Context, def *bpmn.ProcessDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[def.ID] = def
	return nil
}

func (r *mockDefRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, id)
	return nil
}

// mockInstRepo 流程实例仓储的内存模拟实现
type mockInstRepo struct {
	mu    sync.RWMutex
	store map[string]*bpmn.ProcessInstance
	err   error // 可注入的错误，用于测试失败场景
}

func newMockInstRepo() *mockInstRepo {
	return &mockInstRepo{store: make(map[string]*bpmn.ProcessInstance)}
}

func (r *mockInstRepo) Create(_ context.Context, inst *bpmn.ProcessInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.store[inst.ID] = inst
	return nil
}

func (r *mockInstRepo) GetByID(_ context.Context, id string) (*bpmn.ProcessInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.store[id]
	if !ok {
		return nil, fmt.Errorf("流程实例 %s 不存在", id)
	}
	return inst, nil
}

func (r *mockInstRepo) List(_ context.Context, filter ProcessInstanceFilter) ([]*bpmn.ProcessInstance, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*bpmn.ProcessInstance
	for _, inst := range r.store {
		if filter.Status != "" && inst.Status != filter.Status {
			continue
		}
		result = append(result, inst)
	}
	return result, int64(len(result)), nil
}

func (r *mockInstRepo) Update(_ context.Context, inst *bpmn.ProcessInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.store[inst.ID] = inst
	return nil
}

// mockTaskRepo 流程任务仓储的内存模拟实现
type mockTaskRepo struct {
	mu    sync.RWMutex
	store map[string]*bpmn.ProcessTask
	err   error // 可注入的错误，用于测试失败场景
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{store: make(map[string]*bpmn.ProcessTask)}
}

func (r *mockTaskRepo) Create(_ context.Context, task *bpmn.ProcessTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.store[task.ID] = task
	return nil
}

func (r *mockTaskRepo) GetByID(_ context.Context, id string) (*bpmn.ProcessTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.store[id]
	if !ok {
		return nil, fmt.Errorf("任务 %s 不存在", id)
	}
	return task, nil
}

func (r *mockTaskRepo) ListByInstance(_ context.Context, instanceID string) ([]*bpmn.ProcessTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*bpmn.ProcessTask
	for _, task := range r.store {
		if task.InstanceID == instanceID {
			result = append(result, task)
		}
	}
	return result, nil
}

func (r *mockTaskRepo) ListPending(_ context.Context, tenantID, assignee string) ([]*bpmn.ProcessTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*bpmn.ProcessTask
	for _, task := range r.store {
		if task.Status == "pending" && task.Assignee == assignee {
			result = append(result, task)
		}
	}
	return result, nil
}

func (r *mockTaskRepo) Update(_ context.Context, task *bpmn.ProcessTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.store[task.ID] = task
	return nil
}

// mockHistoryRepo 执行历史仓储的内存模拟实现
type mockHistoryRepo struct {
	mu    sync.RWMutex
	store []*bpmn.ExecutionHistory
	err   error // 可注入的错误，用于测试失败场景
}

func newMockHistoryRepo() *mockHistoryRepo {
	return &mockHistoryRepo{store: make([]*bpmn.ExecutionHistory, 0)}
}

func (r *mockHistoryRepo) Create(_ context.Context, h *bpmn.ExecutionHistory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.store = append(r.store, h)
	return nil
}

func (r *mockHistoryRepo) ListByInstance(_ context.Context, instanceID string) ([]*bpmn.ExecutionHistory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*bpmn.ExecutionHistory
	for _, h := range r.store {
		if h.InstanceID == instanceID {
			result = append(result, h)
		}
	}
	return result, nil
}

// ==================== 测试辅助函数 ====================

// newTestService 创建一个带有内存模拟仓储的 ProcessService
func newTestService() (*ProcessService, *mockDefRepo, *mockInstRepo, *mockTaskRepo, *mockHistoryRepo) {
	defRepo := newMockDefRepo()
	instRepo := newMockInstRepo()
	taskRepo := newMockTaskRepo()
	historyRepo := newMockHistoryRepo()
	engine := NewEngine()
	svc := NewProcessService(engine, defRepo, instRepo, taskRepo, historyRepo)
	return svc, defRepo, instRepo, taskRepo, historyRepo
}

// simpleDefYAML 返回一个简单的 startEvent -> userTask -> endEvent 流程定义 YAML
func simpleDefYAML() []byte {
	return []byte(`
id: simple-leave
name: 简单请假流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    outgoing: flow1
  - id: flow1
    type: sequenceFlow
    incoming: start
    outgoing: task1
  - id: task1
    type: userTask
    name: 审批任务
    assignee: admin
    incoming: flow1
    outgoing: flow2
  - id: flow2
    type: sequenceFlow
    incoming: task1
    outgoing: end
  - id: end
    type: endEvent
    incoming: flow2
`)
}

// deploySimpleDef 辅助函数：部署简单流程定义并返回定义
func deploySimpleDef(t *testing.T, svc *ProcessService) *bpmn.ProcessDefinition {
	t.Helper()
	def, err := svc.DeployDefinition(context.Background(), "tenant1", simpleDefYAML())
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}
	return def
}

// ==================== 1. DeployDefinition 测试 ====================

func TestProcessService_DeployDefinition_Success(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	def, err := svc.DeployDefinition(context.Background(), "tenant1", simpleDefYAML())
	if err != nil {
		t.Fatalf("期望部署成功，但出错: %v", err)
	}
	if def == nil {
		t.Fatal("期望返回非空的流程定义")
	}
	if def.ID != "simple-leave" {
		t.Fatalf("期望 ID 为 simple-leave，实际为 %s", def.ID)
	}
	if def.Name != "简单请假流程" {
		t.Fatalf("期望 Name 为 简单请假流程，实际为 %s", def.Name)
	}
	if def.Version != 1 {
		t.Fatalf("期望 Version 为 1，实际为 %d", def.Version)
	}
	if def.Status != "active" {
		t.Fatalf("期望 Status 为 active，实际为 %s", def.Status)
	}
	if len(def.Elements) != 5 {
		t.Fatalf("期望 5 个元素，实际为 %d", len(def.Elements))
	}
}

func TestProcessService_DeployDefinition_InvalidYAML(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	invalidYAML := []byte(`{invalid yaml content: [[[`)

	_, err := svc.DeployDefinition(context.Background(), "tenant1", invalidYAML)
	if err == nil {
		t.Fatal("期望返回 YAML 解析错误")
	}
}

func TestProcessService_DeployDefinition_EmptyID(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	yamlData := []byte(`
name: 无ID流程
version: 1
elements: []
`)

	_, err := svc.DeployDefinition(context.Background(), "tenant1", yamlData)
	if err == nil {
		t.Fatal("期望返回 ID 为空的错误")
	}
}

func TestProcessService_DeployDefinition_EmptyName(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	yamlData := []byte(`
id: no-name-def
version: 1
elements: []
`)

	_, err := svc.DeployDefinition(context.Background(), "tenant1", yamlData)
	if err == nil {
		t.Fatal("期望返回 Name 为空的错误")
	}
}

func TestProcessService_DeployDefinition_DefaultStatus(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	// YAML 中不指定 status，应默认为 "active"
	yamlData := []byte(`
id: default-status-def
name: 默认状态流程
version: 1
elements: []
`)

	def, err := svc.DeployDefinition(context.Background(), "tenant1", yamlData)
	if err != nil {
		t.Fatalf("期望部署成功，但出错: %v", err)
	}
	if def.Status != "active" {
		t.Fatalf("期望默认状态为 active，实际为 %s", def.Status)
	}
}

// ==================== 2. GetDefinition 测试 ====================

func TestProcessService_GetDefinition_Success(t *testing.T) {
	svc, defRepo, _, _, _ := newTestService()

	// 直接在仓储中放入一个定义
	def := &bpmn.ProcessDefinition{
		ID:       "def-001",
		Name:     "测试流程",
		Version:  1,
		Status:   "active",
		Elements: []bpmn.Element{},
	}
	defRepo.store["def-001"] = def

	got, err := svc.GetDefinition(context.Background(), "tenant1", "def-001")
	if err != nil {
		t.Fatalf("期望获取成功，但出错: %v", err)
	}
	if got.ID != "def-001" {
		t.Fatalf("期望 ID 为 def-001，实际为 %s", got.ID)
	}
	if got.Name != "测试流程" {
		t.Fatalf("期望 Name 为 测试流程，实际为 %s", got.Name)
	}
}

func TestProcessService_GetDefinition_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.GetDefinition(context.Background(), "tenant1", "nonexistent")
	if err == nil {
		t.Fatal("期望返回未找到的错误")
	}
}

// ==================== 3. ListDefinitions 测试 ====================

func TestProcessService_ListDefinitions_Success(t *testing.T) {
	svc, defRepo, _, _, _ := newTestService()

	// 放入两个定义
	defRepo.store["def-1"] = &bpmn.ProcessDefinition{ID: "def-1", Name: "流程1", Status: "active"}
	defRepo.store["def-2"] = &bpmn.ProcessDefinition{ID: "def-2", Name: "流程2", Status: "draft"}

	list, total, err := svc.ListDefinitions(context.Background(), "tenant1", ProcessDefinitionFilter{})
	if err != nil {
		t.Fatalf("期望查询成功，但出错: %v", err)
	}
	if total != 2 {
		t.Fatalf("期望总数为 2，实际为 %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("期望返回 2 条记录，实际为 %d", len(list))
	}
}

func TestProcessService_ListDefinitions_WithFilter(t *testing.T) {
	svc, defRepo, _, _, _ := newTestService()

	defRepo.store["def-1"] = &bpmn.ProcessDefinition{ID: "def-1", Name: "流程1", Status: "active"}
	defRepo.store["def-2"] = &bpmn.ProcessDefinition{ID: "def-2", Name: "流程2", Status: "draft"}
	defRepo.store["def-3"] = &bpmn.ProcessDefinition{ID: "def-3", Name: "流程3", Status: "active"}

	// 按状态过滤
	list, total, err := svc.ListDefinitions(context.Background(), "tenant1", ProcessDefinitionFilter{Status: "active"})
	if err != nil {
		t.Fatalf("期望查询成功，但出错: %v", err)
	}
	if total != 2 {
		t.Fatalf("期望过滤后总数为 2，实际为 %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("期望返回 2 条记录，实际为 %d", len(list))
	}
	for _, def := range list {
		if def.Status != "active" {
			t.Fatalf("期望所有记录状态为 active，实际有 %s", def.Status)
		}
	}
}

// ==================== 4. StartProcess 测试 ====================

func TestProcessService_StartProcess_Success(t *testing.T) {
	svc, _, instRepo, taskRepo, historyRepo := newTestService()

	def := deploySimpleDef(t, svc)

	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", map[string]any{"reason": "年假"})
	if err != nil {
		t.Fatalf("期望启动流程成功，但出错: %v", err)
	}
	if inst == nil {
		t.Fatal("期望返回非空的流程实例")
	}
	if inst.DefinitionID != def.ID {
		t.Fatalf("期望 DefinitionID 为 %s，实际为 %s", def.ID, inst.DefinitionID)
	}
	if inst.Status != "running" {
		t.Fatalf("期望状态为 running，实际为 %s", inst.Status)
	}
	if inst.StartedBy != "user1" {
		t.Fatalf("期望 StartedBy 为 user1，实际为 %s", inst.StartedBy)
	}

	// 验证实例已持久化
	persisted, err := instRepo.GetByID(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("期望实例已持久化，但查询出错: %v", err)
	}
	if persisted.ID != inst.ID {
		t.Fatal("持久化的实例 ID 不匹配")
	}

	// 验证待办任务已持久化
	tasks, err := taskRepo.ListByInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("期望查询任务成功，但出错: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(tasks))
	}
	if tasks[0].Status != "pending" {
		t.Fatalf("期望任务状态为 pending，实际为 %s", tasks[0].Status)
	}
	if tasks[0].Assignee != "admin" {
		t.Fatalf("期望任务审批人为 admin，实际为 %s", tasks[0].Assignee)
	}

	// 验证执行历史已持久化
	histories, err := historyRepo.ListByInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("期望查询历史成功，但出错: %v", err)
	}
	if len(histories) == 0 {
		t.Fatal("期望至少有 1 条执行历史记录")
	}
}

func TestProcessService_StartProcess_DefinitionNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.StartProcess(context.Background(), "tenant1", "nonexistent-def", "user1", nil)
	if err == nil {
		t.Fatal("期望返回流程定义不存在的错误")
	}
}

func TestProcessService_StartProcess_InstancePersistFails(t *testing.T) {
	svc, _, instRepo, _, _ := newTestService()

	def := deploySimpleDef(t, svc)

	// 注入实例持久化错误
	instRepo.err = fmt.Errorf("数据库连接失败")

	_, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err == nil {
		t.Fatal("期望返回实例持久化失败的错误")
	}
}

func TestProcessService_StartProcess_TaskPersistFails(t *testing.T) {
	svc, _, _, taskRepo, _ := newTestService()

	def := deploySimpleDef(t, svc)

	// 注入任务持久化错误（实例创建成功后任务创建失败）
	taskRepo.err = fmt.Errorf("任务保存失败")

	_, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err == nil {
		t.Fatal("期望返回任务持久化失败的错误")
	}
}

// ==================== 5. GetProcessInstance 测试 ====================

func TestProcessService_GetProcessInstance_Success(t *testing.T) {
	svc, _, instRepo, _, _ := newTestService()

	// 先部署并启动一个流程
	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 通过 GetProcessInstance 获取
	got, err := svc.GetProcessInstance(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("期望获取成功，但出错: %v", err)
	}
	if got.ID != inst.ID {
		t.Fatalf("期望 ID 为 %s，实际为 %s", inst.ID, got.ID)
	}
	if got.Status != "running" {
		t.Fatalf("期望状态为 running，实际为 %s", got.Status)
	}

	// 确认实例确实在仓储中
	_, err = instRepo.GetByID(context.Background(), inst.ID)
	if err != nil {
		t.Fatal("实例应已持久化到仓储中")
	}
}

func TestProcessService_GetProcessInstance_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	_, err := svc.GetProcessInstance(context.Background(), "tenant1", "nonexistent-inst")
	if err == nil {
		t.Fatal("期望返回实例不存在的错误")
	}
}

// ==================== 6. ListProcessInstances 测试 ====================

func TestProcessService_ListProcessInstances_Success(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	def := deploySimpleDef(t, svc)

	// 启动两个流程实例
	_, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动第一个流程失败: %v", err)
	}
	_, err = svc.StartProcess(context.Background(), "tenant1", def.ID, "user2", nil)
	if err != nil {
		t.Fatalf("启动第二个流程失败: %v", err)
	}

	list, total, err := svc.ListProcessInstances(context.Background(), "tenant1", ProcessInstanceFilter{})
	if err != nil {
		t.Fatalf("期望查询成功，但出错: %v", err)
	}
	if total != 2 {
		t.Fatalf("期望总数为 2，实际为 %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("期望返回 2 条记录，实际为 %d", len(list))
	}
}

func TestProcessService_ListProcessInstances_WithStatusFilter(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	def := deploySimpleDef(t, svc)

	// 启动一个流程
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 挂起该流程
	err = svc.SuspendProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("挂起流程失败: %v", err)
	}

	// 按状态过滤：只查询 running 的实例
	list, total, err := svc.ListProcessInstances(context.Background(), "tenant1", ProcessInstanceFilter{Status: "running"})
	if err != nil {
		t.Fatalf("期望查询成功，但出错: %v", err)
	}
	if total != 0 {
		t.Fatalf("期望没有 running 的实例，实际为 %d", total)
	}
	if len(list) != 0 {
		t.Fatalf("期望返回 0 条记录，实际为 %d", len(list))
	}
}

// ==================== 7. SuspendProcess 测试 ====================

func TestProcessService_SuspendProcess_Success(t *testing.T) {
	svc, _, instRepo, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	err = svc.SuspendProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("期望挂起成功，但出错: %v", err)
	}

	// 验证持久化状态已更新
	persisted, err := instRepo.GetByID(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询持久化实例出错: %v", err)
	}
	if persisted.Status != "suspended" {
		t.Fatalf("期望持久化状态为 suspended，实际为 %s", persisted.Status)
	}

	// 验证引擎内部状态也已更新
	engineInst := svc.engine.GetInstance(inst.ID)
	if engineInst == nil {
		t.Fatal("引擎中应存在该实例")
	}
	if engineInst.Status != "suspended" {
		t.Fatalf("期望引擎中状态为 suspended，实际为 %s", engineInst.Status)
	}
}

func TestProcessService_SuspendProcess_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.SuspendProcess(context.Background(), "tenant1", "nonexistent-inst")
	if err == nil {
		t.Fatal("期望返回实例不存在的错误")
	}
}

// ==================== 8. ResumeProcess 测试 ====================

func TestProcessService_ResumeProcess_Success(t *testing.T) {
	svc, _, instRepo, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 先挂起
	err = svc.SuspendProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("挂起流程失败: %v", err)
	}

	// 再恢复
	err = svc.ResumeProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("期望恢复成功，但出错: %v", err)
	}

	// 验证持久化状态已更新
	persisted, err := instRepo.GetByID(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询持久化实例出错: %v", err)
	}
	if persisted.Status != "running" {
		t.Fatalf("期望持久化状态为 running，实际为 %s", persisted.Status)
	}

	// 验证引擎内部状态也已更新
	engineInst := svc.engine.GetInstance(inst.ID)
	if engineInst == nil {
		t.Fatal("引擎中应存在该实例")
	}
	if engineInst.Status != "running" {
		t.Fatalf("期望引擎中状态为 running，实际为 %s", engineInst.Status)
	}
}

func TestProcessService_ResumeProcess_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.ResumeProcess(context.Background(), "tenant1", "nonexistent-inst")
	if err == nil {
		t.Fatal("期望返回实例不存在的错误")
	}
}

// ==================== 9. CancelProcess 测试 ====================

func TestProcessService_CancelProcess_Success(t *testing.T) {
	svc, _, instRepo, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	err = svc.CancelProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("期望取消成功，但出错: %v", err)
	}

	// 验证持久化状态已更新
	persisted, err := instRepo.GetByID(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询持久化实例出错: %v", err)
	}
	if persisted.Status != "cancelled" {
		t.Fatalf("期望持久化状态为 cancelled，实际为 %s", persisted.Status)
	}

	// 验证引擎内部状态也已更新
	engineInst := svc.engine.GetInstance(inst.ID)
	if engineInst == nil {
		t.Fatal("引擎中应存在该实例")
	}
	if engineInst.Status != "cancelled" {
		t.Fatalf("期望引擎中状态为 cancelled，实际为 %s", engineInst.Status)
	}
}

func TestProcessService_CancelProcess_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.CancelProcess(context.Background(), "tenant1", "nonexistent-inst")
	if err == nil {
		t.Fatal("期望返回实例不存在的错误")
	}
}

// ==================== 10. GetPendingTasks 测试 ====================

func TestProcessService_GetPendingTasks_Success(t *testing.T) {
	svc, _, _, taskRepo, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 获取 admin 的待办任务
	tasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "admin")
	if err != nil {
		t.Fatalf("期望获取待办成功，但出错: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(tasks))
	}
	if tasks[0].Assignee != "admin" {
		t.Fatalf("期望审批人为 admin，实际为 %s", tasks[0].Assignee)
	}
	if tasks[0].InstanceID != inst.ID {
		t.Fatalf("期望 InstanceID 为 %s，实际为 %s", inst.ID, tasks[0].InstanceID)
	}

	// 获取不存在审批人的待办，应返回空
	emptyTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "nobody")
	if err != nil {
		t.Fatalf("期望查询成功，但出错: %v", err)
	}
	if len(emptyTasks) != 0 {
		t.Fatalf("期望 0 个待办任务，实际为 %d", len(emptyTasks))
	}

	// 确认任务已通过仓储持久化
	allTasks, err := taskRepo.ListByInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询实例任务出错: %v", err)
	}
	if len(allTasks) != 1 {
		t.Fatalf("仓储中期望 1 个任务，实际为 %d", len(allTasks))
	}
}

// ==================== 11. CompleteTask 测试 ====================

func TestProcessService_CompleteTask_Success(t *testing.T) {
	svc, _, instRepo, taskRepo, historyRepo := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 获取待办任务
	pendingTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "admin")
	if err != nil {
		t.Fatalf("获取待办失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(pendingTasks))
	}
	taskID := pendingTasks[0].ID

	// 完成任务
	submittedData := map[string]any{"approved": true, "comment": "同意"}
	err = svc.CompleteTask(context.Background(), "tenant1", taskID, "admin", submittedData)
	if err != nil {
		t.Fatalf("期望完成任务成功，但出错: %v", err)
	}

	// 验证任务状态已更新为 completed
	updatedTask, err := taskRepo.GetByID(context.Background(), taskID)
	if err != nil {
		t.Fatalf("查询任务出错: %v", err)
	}
	if updatedTask.Status != "completed" {
		t.Fatalf("期望任务状态为 completed，实际为 %s", updatedTask.Status)
	}
	if updatedTask.CompletedBy != "admin" {
		t.Fatalf("期望 CompletedBy 为 admin，实际为 %s", updatedTask.CompletedBy)
	}

	// 验证流程实例状态已更新为 completed（因为 userTask 后面是 endEvent）
	updatedInst, err := instRepo.GetByID(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询实例出错: %v", err)
	}
	if updatedInst.Status != "completed" {
		t.Fatalf("期望实例状态为 completed，实际为 %s", updatedInst.Status)
	}

	// 验证执行历史中有新记录（完成任务 + 到达 endEvent）
	histories, err := historyRepo.ListByInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询历史出错: %v", err)
	}
	if len(histories) == 0 {
		t.Fatal("期望有执行历史记录")
	}

	// 验证变量已合并
	engineInst := svc.engine.GetInstance(inst.ID)
	if engineInst == nil {
		t.Fatal("引擎中应存在该实例")
	}
	if engineInst.Variables["approved"] != true {
		t.Fatalf("期望变量 approved=true，实际为 %v", engineInst.Variables["approved"])
	}
}

func TestProcessService_CompleteTask_TaskNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	err := svc.CompleteTask(context.Background(), "tenant1", "nonexistent-task", "admin", nil)
	if err == nil {
		t.Fatal("期望返回任务不存在的错误")
	}
}

func TestProcessService_CompleteTask_EngineCompleteFails(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 获取待办任务
	pendingTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "admin")
	if err != nil {
		t.Fatalf("获取待办失败: %v", err)
	}
	taskID := pendingTasks[0].ID

	// 先挂起流程，使引擎的 CompleteTask 失败
	err = svc.SuspendProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("挂起流程失败: %v", err)
	}

	// 尝试在挂起状态下完成任务，引擎应返回错误
	err = svc.CompleteTask(context.Background(), "tenant1", taskID, "admin", map[string]any{"approved": true})
	if err == nil {
		t.Fatal("期望引擎完成任务失败（实例已挂起）")
	}
}

// ==================== 12. GetProcessTasks 测试 ====================

func TestProcessService_GetProcessTasks_Success(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	tasks, err := svc.GetProcessTasks(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("期望获取任务列表成功，但出错: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("期望 1 个任务，实际为 %d", len(tasks))
	}
	if tasks[0].InstanceID != inst.ID {
		t.Fatalf("期望任务 InstanceID 为 %s，实际为 %s", inst.ID, tasks[0].InstanceID)
	}
	if tasks[0].Name != "审批任务" {
		t.Fatalf("期望任务名称为 审批任务，实际为 %s", tasks[0].Name)
	}
}

// ==================== 13. GetProcessHistory 测试 ====================

func TestProcessService_GetProcessHistory_Success(t *testing.T) {
	svc, _, _, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	histories, err := svc.GetProcessHistory(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("期望获取历史成功，但出错: %v", err)
	}
	if len(histories) == 0 {
		t.Fatal("期望至少有 1 条历史记录")
	}

	// 验证历史记录关联正确的实例
	for _, h := range histories {
		if h.InstanceID != inst.ID {
			t.Fatalf("期望历史 InstanceID 为 %s，实际为 %s", inst.ID, h.InstanceID)
		}
	}

	// 验证历史中包含 start 事件
	hasStartEnter := false
	for _, h := range histories {
		if h.ElementID == "start" && h.Action == "enter" {
			hasStartEnter = true
		}
	}
	if !hasStartEnter {
		t.Fatal("期望历史中包含 start 事件的 enter 记录")
	}
}

// ==================== 集成场景测试 ====================

func TestProcessService_FullLifecycle(t *testing.T) {
	// 完整生命周期测试：部署 -> 启动 -> 获取待办 -> 完成 -> 验证历史
	svc, _, instRepo, taskRepo, historyRepo := newTestService()

	// 1. 部署流程定义
	def, err := svc.DeployDefinition(context.Background(), "tenant1", simpleDefYAML())
	if err != nil {
		t.Fatalf("部署失败: %v", err)
	}
	if def.ID != "simple-leave" {
		t.Fatalf("期望定义 ID 为 simple-leave，实际为 %s", def.ID)
	}

	// 2. 获取定义
	gotDef, err := svc.GetDefinition(context.Background(), "tenant1", def.ID)
	if err != nil {
		t.Fatalf("获取定义失败: %v", err)
	}
	if gotDef.ID != def.ID {
		t.Fatal("获取的定义 ID 不匹配")
	}

	// 3. 启动流程
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", map[string]any{"days": 3})
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 4. 获取实例
	gotInst, err := svc.GetProcessInstance(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("获取实例失败: %v", err)
	}
	if gotInst.Status != "running" {
		t.Fatalf("期望 running，实际为 %s", gotInst.Status)
	}

	// 5. 获取待办任务
	pendingTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "admin")
	if err != nil {
		t.Fatalf("获取待办失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办，实际为 %d", len(pendingTasks))
	}

	// 6. 挂起流程
	err = svc.SuspendProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("挂起失败: %v", err)
	}
	suspendedInst, _ := instRepo.GetByID(context.Background(), inst.ID)
	if suspendedInst.Status != "suspended" {
		t.Fatalf("期望 suspended，实际为 %s", suspendedInst.Status)
	}

	// 7. 恢复流程
	err = svc.ResumeProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	resumedInst, _ := instRepo.GetByID(context.Background(), inst.ID)
	if resumedInst.Status != "running" {
		t.Fatalf("期望 running，实际为 %s", resumedInst.Status)
	}

	// 8. 完成任务
	err = svc.CompleteTask(context.Background(), "tenant1", pendingTasks[0].ID, "admin", map[string]any{"approved": true})
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	// 9. 验证流程已完成
	completedInst, _ := instRepo.GetByID(context.Background(), inst.ID)
	if completedInst.Status != "completed" {
		t.Fatalf("期望 completed，实际为 %s", completedInst.Status)
	}

	// 10. 验证任务已标记完成
	completedTask, _ := taskRepo.GetByID(context.Background(), pendingTasks[0].ID)
	if completedTask.Status != "completed" {
		t.Fatalf("期望任务 completed，实际为 %s", completedTask.Status)
	}

	// 11. 验证历史记录
	histories, _ := historyRepo.ListByInstance(context.Background(), inst.ID)
	if len(histories) == 0 {
		t.Fatal("期望有历史记录")
	}

	// 12. 查询实例列表
	list, total, err := svc.ListProcessInstances(context.Background(), "tenant1", ProcessInstanceFilter{})
	if err != nil {
		t.Fatalf("查询实例列表失败: %v", err)
	}
	if total != 1 {
		t.Fatalf("期望 1 个实例，实际为 %d", total)
	}
	if len(list) != 1 {
		t.Fatalf("期望返回 1 条记录，实际为 %d", len(list))
	}

	// 13. 查询任务列表
	instTasks, err := svc.GetProcessTasks(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("查询任务列表失败: %v", err)
	}
	if len(instTasks) != 1 {
		t.Fatalf("期望 1 个任务，实际为 %d", len(instTasks))
	}

	// 14. 查询历史
	instHistory, err := svc.GetProcessHistory(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if len(instHistory) == 0 {
		t.Fatal("期望有历史记录")
	}
}

func TestProcessService_CancelBeforeComplete(t *testing.T) {
	// 测试取消流程后不能再完成任务
	svc, _, _, _, _ := newTestService()

	def := deploySimpleDef(t, svc)
	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	pendingTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "admin")
	if err != nil {
		t.Fatalf("获取待办失败: %v", err)
	}
	taskID := pendingTasks[0].ID

	// 取消流程
	err = svc.CancelProcess(context.Background(), "tenant1", inst.ID)
	if err != nil {
		t.Fatalf("取消流程失败: %v", err)
	}

	// 尝试完成任务，引擎应返回错误（实例已取消）
	err = svc.CompleteTask(context.Background(), "tenant1", taskID, "admin", nil)
	if err == nil {
		t.Fatal("期望取消后完成任务失败")
	}
}

func TestProcessService_CompleteTask_PersistsNewTasksAndHistory(t *testing.T) {
	// 测试完成任务后，引擎产生的新任务和历史被正确持久化
	svc, _, _, taskRepo, historyRepo := newTestService()

	// 使用包含两个顺序 userTask 的流程定义
	twoTaskYAML := []byte(`
id: two-task-flow
name: 双任务流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    outgoing: flow1
  - id: flow1
    type: sequenceFlow
    incoming: start
    outgoing: task1
  - id: task1
    type: userTask
    name: 第一步审批
    assignee: manager
    incoming: flow1
    outgoing: flow2
  - id: flow2
    type: sequenceFlow
    incoming: task1
    outgoing: task2
  - id: task2
    type: userTask
    name: 第二步审批
    assignee: director
    incoming: flow2
    outgoing: flow3
  - id: flow3
    type: sequenceFlow
    incoming: task2
    outgoing: end
  - id: end
    type: endEvent
    incoming: flow3
`)

	def, err := svc.DeployDefinition(context.Background(), "tenant1", twoTaskYAML)
	if err != nil {
		t.Fatalf("部署失败: %v", err)
	}

	inst, err := svc.StartProcess(context.Background(), "tenant1", def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程失败: %v", err)
	}

	// 获取第一个待办
	pendingTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "manager")
	if err != nil {
		t.Fatalf("获取待办失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办，实际为 %d", len(pendingTasks))
	}

	// 完成第一个任务
	err = svc.CompleteTask(context.Background(), "tenant1", pendingTasks[0].ID, "manager", map[string]any{"step1_result": "pass"})
	if err != nil {
		t.Fatalf("完成第一个任务失败: %v", err)
	}

	// 验证第二个任务已持久化
	newPendingTasks, err := svc.GetPendingTasks(context.Background(), "tenant1", "director")
	if err != nil {
		t.Fatalf("获取新待办失败: %v", err)
	}
	if len(newPendingTasks) != 1 {
		t.Fatalf("期望 1 个新待办，实际为 %d", len(newPendingTasks))
	}
	if newPendingTasks[0].Name != "第二步审批" {
		t.Fatalf("期望任务名称为 第二步审批，实际为 %s", newPendingTasks[0].Name)
	}

	// 验证新任务已通过仓储持久化
	allTasks, err := taskRepo.ListByInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询实例任务出错: %v", err)
	}
	if len(allTasks) != 2 {
		t.Fatalf("仓储中期望 2 个任务（1个已完成 + 1个待办），实际为 %d", len(allTasks))
	}

	// 验证历史记录已持久化（包含完成任务和新任务进入的历史）
	allHistory, err := historyRepo.ListByInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("查询历史出错: %v", err)
	}
	if len(allHistory) == 0 {
		t.Fatal("期望有历史记录")
	}
}
