package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	approvalapp "git.neolidy.top/neo/flowx/internal/application/approval"
	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
	domainagent "git.neolidy.top/neo/flowx/internal/domain/agent"
	domainapproval "git.neolidy.top/neo/flowx/internal/domain/approval"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/pkg/pagination"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockAgentTaskRepository 基于 GORM 的测试用 Agent 任务仓储
type mockAgentTaskRepository struct {
	db *gorm.DB
}

func newMockAgentTaskRepository(db *gorm.DB) AgentTaskRepository {
	return &mockAgentTaskRepository{db: db}
}

func (r *mockAgentTaskRepository) Create(ctx context.Context, task *domainagent.AgentTask) error {
	if task.ID == "" {
		task.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *mockAgentTaskRepository) GetByID(ctx context.Context, id string) (*domainagent.AgentTask, error) {
	var task domainagent.AgentTask
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		return nil, fmt.Errorf("任务不存在")
	}
	return &task, nil
}

func (r *mockAgentTaskRepository) Update(ctx context.Context, task *domainagent.AgentTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *mockAgentTaskRepository) List(ctx context.Context, tenantID, status string, page, pageSize int) ([]domainagent.AgentTask, int64, error) {
	var tasks []domainagent.AgentTask
	var total int64
	q := r.db.WithContext(ctx).Model(&domainagent.AgentTask{}).Where("tenant_id = ?", tenantID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Find(&tasks).Error; err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// mockApprovalService 模拟审批服务
type mockApprovalService struct {
	approveCalled bool
	rejectCalled  bool
	instanceID    string
	comment       string
	approverID    string
}

func (m *mockApprovalService) CreateWorkflow(ctx context.Context, tenantID string, req *approvalapp.CreateWorkflowRequest) (*domainapproval.Workflow, error) {
	return nil, nil
}
func (m *mockApprovalService) GetWorkflow(ctx context.Context, tenantID string, id string) (*domainapproval.Workflow, error) {
	return nil, nil
}
func (m *mockApprovalService) ListWorkflows(ctx context.Context, tenantID string, filter approvalapp.WorkflowFilter) ([]domainapproval.Workflow, *pagination.PaginatedResult, error) {
	return nil, nil, nil
}
func (m *mockApprovalService) ActivateWorkflow(ctx context.Context, tenantID string, id string) (*domainapproval.Workflow, error) {
	return nil, nil
}
func (m *mockApprovalService) ArchiveWorkflow(ctx context.Context, tenantID string, id string) (*domainapproval.Workflow, error) {
	return nil, nil
}
func (m *mockApprovalService) StartApproval(ctx context.Context, tenantID string, initiatorID string, req *approvalapp.StartApprovalRequest) (*domainapproval.WorkflowInstance, error) {
	return nil, nil
}
func (m *mockApprovalService) GetInstance(ctx context.Context, tenantID string, id string) (*domainapproval.WorkflowInstance, error) {
	return nil, nil
}
func (m *mockApprovalService) ListInstances(ctx context.Context, tenantID string, filter approvalapp.InstanceFilter) ([]domainapproval.WorkflowInstance, *pagination.PaginatedResult, error) {
	return nil, nil, nil
}
func (m *mockApprovalService) CancelInstance(ctx context.Context, tenantID string, id string) error {
	return nil
}
func (m *mockApprovalService) Approve(ctx context.Context, tenantID string, approverID string, req *approvalapp.ApproveRequest) (*domainapproval.Approval, error) {
	m.approveCalled = true
	m.instanceID = req.InstanceID
	m.comment = req.Comment
	m.approverID = approverID
	return nil, nil
}
func (m *mockApprovalService) Reject(ctx context.Context, tenantID string, approverID string, req *approvalapp.RejectRequest) (*domainapproval.Approval, error) {
	m.rejectCalled = true
	m.instanceID = req.InstanceID
	m.comment = req.Comment
	m.approverID = approverID
	return nil, nil
}
func (m *mockApprovalService) Forward(ctx context.Context, tenantID string, approverID string, req *approvalapp.ForwardRequest) (*domainapproval.Approval, error) {
	return nil, nil
}
func (m *mockApprovalService) GetSuggestion(ctx context.Context, tenantID string, instanceID string) (string, error) {
	return "", nil
}
func (m *mockApprovalService) GetMyPendingApprovals(ctx context.Context, tenantID string, approverID string) ([]domainapproval.WorkflowInstance, error) {
	return nil, nil
}
func (m *mockApprovalService) UpdateInstance(ctx context.Context, inst *domainapproval.WorkflowInstance) error {
	return nil
}

// setupAgentServiceTest 创建 Agent 服务测试环境
func setupAgentServiceTest(t *testing.T) (*AgentService, *mockAgentTaskRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&domainagent.AgentTask{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	repo := newMockAgentTaskRepository(db)
	svc := NewAgentService(nil, repo, nil)
	return svc, repo.(*mockAgentTaskRepository), db
}

// createPendingApprovalTask 创建待审批的任务
func createPendingApprovalTask(t *testing.T, ctx context.Context, repo AgentTaskRepository, tenantID, createdBy, workflowInstanceID string) *domainagent.AgentTask {
	t.Helper()
	task := &domainagent.AgentTask{
		BaseModel:          base.BaseModel{TenantID: tenantID},
		Type:               "tool_execute",
		Description:        "测试任务",
		Status:             "pending_approval",
		RequireApproval:    true,
		CreatedBy:          createdBy,
		WorkflowInstanceID: workflowInstanceID,
	}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}
	return task
}

// ===================== Agent-Approval Linking Tests =====================

// TestApproveTask_WithWorkflow 审批通过带工作流关联的任务，应调用审批服务
func TestApproveTask_WithWorkflow(t *testing.T) {
	svc, repo, _ := setupAgentServiceTest(t)
	ctx := context.Background()

	mockApproval := &mockApprovalService{}
	svc.approvalSvc = mockApproval

	task := createPendingApprovalTask(t, ctx, repo, "tenant-001", "user-001", "workflow-inst-001")

	result, err := svc.ApproveTask(ctx, "tenant-001", "approver-001", task.ID, "同意部署")
	if err != nil {
		t.Fatalf("审批通过任务失败: %v", err)
	}
	if result.Status != "approved" {
		t.Errorf("期望任务状态为 'approved'，实际为 '%s'", result.Status)
	}
	if !mockApproval.approveCalled {
		t.Error("期望调用审批服务的 Approve 方法")
	}
	if mockApproval.instanceID != "workflow-inst-001" {
		t.Errorf("期望审批实例 ID 为 'workflow-inst-001'，实际为 '%s'", mockApproval.instanceID)
	}
	if mockApproval.approverID != "user-001" {
		t.Errorf("期望审批人为 'user-001'（任务创建者），实际为 '%s'", mockApproval.approverID)
	}
	if mockApproval.comment != "同意部署" {
		t.Errorf("期望审批意见为 '同意部署'，实际为 '%s'", mockApproval.comment)
	}
}

// TestApproveTask_WithoutWorkflow 审批通过不带工作流关联的任务，不应调用审批服务
func TestApproveTask_WithoutWorkflow(t *testing.T) {
	svc, repo, _ := setupAgentServiceTest(t)
	ctx := context.Background()

	mockApproval := &mockApprovalService{}
	svc.approvalSvc = mockApproval

	task := createPendingApprovalTask(t, ctx, repo, "tenant-001", "user-001", "")

	result, err := svc.ApproveTask(ctx, "tenant-001", "approver-001", task.ID, "同意")
	if err != nil {
		t.Fatalf("审批通过任务失败: %v", err)
	}
	if result.Status != "approved" {
		t.Errorf("期望任务状态为 'approved'，实际为 '%s'", result.Status)
	}
	if mockApproval.approveCalled {
		t.Error("期望不调用审批服务的 Approve 方法")
	}
}

// TestRejectTask_WithWorkflow 拒绝带工作流关联的任务，应调用审批服务
func TestRejectTask_WithWorkflow(t *testing.T) {
	svc, repo, _ := setupAgentServiceTest(t)
	ctx := context.Background()

	mockApproval := &mockApprovalService{}
	svc.approvalSvc = mockApproval

	task := createPendingApprovalTask(t, ctx, repo, "tenant-001", "user-001", "workflow-inst-002")

	result, err := svc.RejectTask(ctx, "tenant-001", "approver-001", task.ID, "工具版本不合规")
	if err != nil {
		t.Fatalf("拒绝任务失败: %v", err)
	}
	if result.Status != "rejected" {
		t.Errorf("期望任务状态为 'rejected'，实际为 '%s'", result.Status)
	}
	if !mockApproval.rejectCalled {
		t.Error("期望调用审批服务的 Reject 方法")
	}
	if mockApproval.instanceID != "workflow-inst-002" {
		t.Errorf("期望审批实例 ID 为 'workflow-inst-002'，实际为 '%s'", mockApproval.instanceID)
	}
	if mockApproval.approverID != "user-001" {
		t.Errorf("期望审批人为 'user-001'（任务创建者），实际为 '%s'", mockApproval.approverID)
	}
}

// TestRejectTask_WithoutWorkflow 拒绝不带工作流关联的任务，不应调用审批服务
func TestRejectTask_WithoutWorkflow(t *testing.T) {
	svc, repo, _ := setupAgentServiceTest(t)
	ctx := context.Background()

	mockApproval := &mockApprovalService{}
	svc.approvalSvc = mockApproval

	task := createPendingApprovalTask(t, ctx, repo, "tenant-001", "user-001", "")

	result, err := svc.RejectTask(ctx, "tenant-001", "approver-001", task.ID, "不同意")
	if err != nil {
		t.Fatalf("拒绝任务失败: %v", err)
	}
	if result.Status != "rejected" {
		t.Errorf("期望任务状态为 'rejected'，实际为 '%s'", result.Status)
	}
	if mockApproval.rejectCalled {
		t.Error("期望不调用审批服务的 Reject 方法")
	}
}

// ===================== Mock AgentEngine =====================

// mockAgentEngine 模拟 Agent 引擎，支持可配置行为
type mockAgentEngine struct {
	executeResult *TaskResult
	executeErr    error
	tools         []mcpif.MCPToolDefinition
	toolsErr      error
}

func (m *mockAgentEngine) RegisterAgent(agent Agent, priority ...int) {}

func (m *mockAgentEngine) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	return m.executeResult, m.executeErr
}

func (m *mockAgentEngine) ListAvailableTools(ctx context.Context) ([]mcpif.MCPToolDefinition, error) {
	return m.tools, m.toolsErr
}

// setupAgentServiceWithEngine 创建带模拟引擎的 Agent 服务测试环境
func setupAgentServiceWithEngine(t *testing.T) (*AgentService, *mockAgentTaskRepository, *mockAgentEngine, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&domainagent.AgentTask{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	repo := newMockAgentTaskRepository(db)
	engine := &mockAgentEngine{}
	svc := NewAgentService(engine, repo, nil)
	return svc, repo.(*mockAgentTaskRepository), engine, db
}

// ===================== CreateAndExecuteTask Tests =====================

// TestCreateAndExecuteTask_Success 创建并执行任务成功，状态应更新为 completed
func TestCreateAndExecuteTask_Success(t *testing.T) {
	svc, repo, engine, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	engine.executeResult = &TaskResult{
		Status: "completed",
		Output: map[string]any{"key": "value"},
	}

	task := &Task{
		Type:        "tool_execute",
		Description: "测试任务",
		Context:     map[string]any{"input": "data"},
		Steps: []TaskStep{
			{Type: "step1", Description: "步骤一", Params: map[string]any{"p1": "v1"}},
		},
	}

	result, err := svc.CreateAndExecuteTask(ctx, "tenant-001", "user-001", task)
	if err != nil {
		t.Fatalf("期望执行成功，实际返回错误: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("期望状态为 'completed'，实际为 '%s'", result.Status)
	}
	if result.TaskID == "" {
		t.Error("期望 TaskID 不为空")
	}

	// 验证任务已持久化且状态为 completed
	persisted, err := repo.GetByID(ctx, result.TaskID)
	if err != nil {
		t.Fatalf("查询持久化任务失败: %v", err)
	}
	if persisted.Status != "completed" {
		t.Errorf("期望持久化状态为 'completed'，实际为 '%s'", persisted.Status)
	}
	if persisted.Type != "tool_execute" {
		t.Errorf("期望类型为 'tool_execute'，实际为 '%s'", persisted.Type)
	}
}

// TestCreateAndExecuteTask_EngineFails 引擎执行失败时，任务状态应更新为 failed
func TestCreateAndExecuteTask_EngineFails(t *testing.T) {
	svc, repo, engine, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	engine.executeErr = errors.New("引擎执行超时")

	task := &Task{
		Type:        "tool_execute",
		Description: "会失败的任务",
	}

	_, err := svc.CreateAndExecuteTask(ctx, "tenant-001", "user-001", task)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}

	// 验证任务已持久化且状态为 failed
	var tasks []domainagent.AgentTask
	repo.db.Find(&tasks)
	if len(tasks) != 1 {
		t.Fatalf("期望有 1 条任务记录，实际为 %d", len(tasks))
	}
	if tasks[0].Status != "failed" {
		t.Errorf("期望状态为 'failed'，实际为 '%s'", tasks[0].Status)
	}
}

// TestCreateAndExecuteTask_ContextAndStepsJSON 验证 Context 和 Steps 被 JSON 序列化到 AgentTask
func TestCreateAndExecuteTask_ContextAndStepsJSON(t *testing.T) {
	svc, repo, engine, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	engine.executeResult = &TaskResult{Status: "completed"}

	task := &Task{
		Type:        "data_process",
		Description: "数据处理任务",
		Context:     map[string]any{"source": "db", "query": "SELECT 1"},
		Steps: []TaskStep{
			{Type: "extract", Description: "提取数据", Params: map[string]any{"table": "users"}},
			{Type: "transform", Description: "转换数据"},
		},
	}

	result, err := svc.CreateAndExecuteTask(ctx, "tenant-001", "user-001", task)
	if err != nil {
		t.Fatalf("期望执行成功，实际返回错误: %v", err)
	}

	persisted, err := repo.GetByID(ctx, result.TaskID)
	if err != nil {
		t.Fatalf("查询持久化任务失败: %v", err)
	}

	// 验证 Context JSON
	if persisted.Context == "" {
		t.Fatal("期望 Context 字段不为空")
	}
	var ctxMap map[string]any
	if err := json.Unmarshal([]byte(persisted.Context), &ctxMap); err != nil {
		t.Fatalf("解析 Context JSON 失败: %v", err)
	}
	if ctxMap["source"] != "db" {
		t.Errorf("期望 Context 中 source='db'，实际为 '%v'", ctxMap["source"])
	}
	if ctxMap["query"] != "SELECT 1" {
		t.Errorf("期望 Context 中 query='SELECT 1'，实际为 '%v'", ctxMap["query"])
	}

	// 验证 Steps JSON
	if persisted.Steps == "" {
		t.Fatal("期望 Steps 字段不为空")
	}
	var steps []map[string]any
	if err := json.Unmarshal([]byte(persisted.Steps), &steps); err != nil {
		t.Fatalf("解析 Steps JSON 失败: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("期望有 2 个步骤，实际为 %d", len(steps))
	}
	if steps[0]["type"] != "extract" {
		t.Errorf("期望第一个步骤 type='extract'，实际为 '%v'", steps[0]["type"])
	}
	if steps[1]["type"] != "transform" {
		t.Errorf("期望第二个步骤 type='transform'，实际为 '%v'", steps[1]["type"])
	}
}

// ===================== ListTasks Tests =====================

// TestListTasks_Success 查询任务列表成功，返回分页结果
func TestListTasks_Success(t *testing.T) {
	svc, repo, _, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	// 创建多个任务
	for i := 0; i < 3; i++ {
		task := &domainagent.AgentTask{
			BaseModel:   base.BaseModel{TenantID: "tenant-001"},
			Type:        "tool_execute",
			Description: fmt.Sprintf("任务-%d", i),
			Status:      "completed",
		}
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("创建测试任务失败: %v", err)
		}
	}

	tasks, pageResult, err := svc.ListTasks(ctx, "tenant-001", "", 1, 10)
	if err != nil {
		t.Fatalf("期望查询成功，实际返回错误: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("期望返回 3 条任务，实际为 %d", len(tasks))
	}
	if pageResult.Total != 3 {
		t.Errorf("期望总数为 3，实际为 %d", pageResult.Total)
	}
}

// TestListTasks_WithStatusFilter 按状态过滤任务列表
func TestListTasks_WithStatusFilter(t *testing.T) {
	svc, repo, _, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	// 创建不同状态的任务
	taskCompleted := &domainagent.AgentTask{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "已完成任务",
		Status:      "completed",
	}
	taskFailed := &domainagent.AgentTask{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "失败任务",
		Status:      "failed",
	}
	for _, task := range []*domainagent.AgentTask{taskCompleted, taskFailed} {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("创建测试任务失败: %v", err)
		}
	}

	tasks, pageResult, err := svc.ListTasks(ctx, "tenant-001", "completed", 1, 10)
	if err != nil {
		t.Fatalf("期望查询成功，实际返回错误: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("期望返回 1 条 completed 任务，实际为 %d", len(tasks))
	}
	if pageResult.Total != 1 {
		t.Errorf("期望过滤后总数为 1，实际为 %d", pageResult.Total)
	}
	if tasks[0].Status != "completed" {
		t.Errorf("期望任务状态为 'completed'，实际为 '%s'", tasks[0].Status)
	}
}

// ===================== GetTask Tests =====================

// TestGetTask_Success 获取任务详情成功
func TestGetTask_Success(t *testing.T) {
	svc, repo, _, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	task := &domainagent.AgentTask{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
	}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}

	result, err := svc.GetTask(ctx, "tenant-001", task.ID)
	if err != nil {
		t.Fatalf("期望获取成功，实际返回错误: %v", err)
	}
	if result.ID != task.ID {
		t.Errorf("期望任务 ID 为 '%s'，实际为 '%s'", task.ID, result.ID)
	}
	if result.Type != "tool_execute" {
		t.Errorf("期望类型为 'tool_execute'，实际为 '%s'", result.Type)
	}
}

// TestGetTask_NotFound 任务不存在时返回 ErrTaskNotFound
func TestGetTask_NotFound(t *testing.T) {
	svc, _, _, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	_, err := svc.GetTask(ctx, "tenant-001", "non-existent-id")
	if err != ErrTaskNotFound {
		t.Errorf("期望返回 ErrTaskNotFound，实际为 '%v'", err)
	}
}

// TestGetTask_CrossTenant 跨租户访问任务应返回 ErrTaskNotFound
func TestGetTask_CrossTenant(t *testing.T) {
	svc, repo, _, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	task := &domainagent.AgentTask{
		BaseModel:   base.BaseModel{TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
	}
	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("创建测试任务失败: %v", err)
	}

	// 使用不同租户 ID 查询
	_, err := svc.GetTask(ctx, "tenant-002", task.ID)
	if err != ErrTaskNotFound {
		t.Errorf("期望跨租户返回 ErrTaskNotFound，实际为 '%v'", err)
	}
}

// ===================== ListAvailableTools Tests =====================

// TestListAvailableTools_Success 获取可用工具列表成功
func TestListAvailableTools_Success(t *testing.T) {
	svc, _, engine, _ := setupAgentServiceWithEngine(t)
	ctx := context.Background()

	engine.tools = []mcpif.MCPToolDefinition{
		{Name: "tool1", Description: "工具一", InputSchema: map[string]any{"type": "object"}},
		{Name: "tool2", Description: "工具二", InputSchema: map[string]any{"type": "object"}},
	}

	tools, err := svc.ListAvailableTools(ctx)
	if err != nil {
		t.Fatalf("期望获取工具列表成功，实际返回错误: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("期望返回 2 个工具，实际为 %d", len(tools))
	}
	if tools[0].Name != "tool1" {
		t.Errorf("期望第一个工具名称为 'tool1'，实际为 '%s'", tools[0].Name)
	}
	if tools[1].Name != "tool2" {
		t.Errorf("期望第二个工具名称为 'tool2'，实际为 '%s'", tools[1].Name)
	}
}

// ===================== TaskToResponse Tests =====================

// TestTaskToResponse_ValidContext Context 为有效 JSON 时，响应中应包含 context 字段
func TestTaskToResponse_ValidContext(t *testing.T) {
	task := domainagent.AgentTask{
		BaseModel:   base.BaseModel{ID: "task-001", TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
		Context:     `{"key":"value","num":42}`,
	}

	resp := TaskToResponse(task)
	if _, ok := resp["context"]; !ok {
		t.Fatal("期望响应中包含 context 字段")
	}
	ctxMap, ok := resp["context"].(map[string]any)
	if !ok {
		t.Fatal("期望 context 为 map 类型")
	}
	if ctxMap["key"] != "value" {
		t.Errorf("期望 context.key='value'，实际为 '%v'", ctxMap["key"])
	}
	if ctxMap["num"] != float64(42) {
		t.Errorf("期望 context.num=42，实际为 '%v'", ctxMap["num"])
	}
}

// TestTaskToResponse_ValidResult Result 为有效 JSON 时，响应中应包含 result 字段
func TestTaskToResponse_ValidResult(t *testing.T) {
	task := domainagent.AgentTask{
		BaseModel:   base.BaseModel{ID: "task-001", TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
		Result:      `{"output":"done","count":10}`,
	}

	resp := TaskToResponse(task)
	if _, ok := resp["result"]; !ok {
		t.Fatal("期望响应中包含 result 字段")
	}
	resultMap, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("期望 result 为 map 类型")
	}
	if resultMap["output"] != "done" {
		t.Errorf("期望 result.output='done'，实际为 '%v'", resultMap["output"])
	}
	if resultMap["count"] != float64(10) {
		t.Errorf("期望 result.count=10，实际为 '%v'", resultMap["count"])
	}
}

// TestTaskToResponse_EmptyContext Context 为空时，响应中不应包含 context 字段
func TestTaskToResponse_EmptyContext(t *testing.T) {
	task := domainagent.AgentTask{
		BaseModel:   base.BaseModel{ID: "task-001", TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
		Context:     "",
	}

	resp := TaskToResponse(task)
	if _, ok := resp["context"]; ok {
		t.Error("期望响应中不包含 context 字段")
	}
}

// TestTaskToResponse_EmptyResult Result 为空时，响应中不应包含 result 字段
func TestTaskToResponse_EmptyResult(t *testing.T) {
	task := domainagent.AgentTask{
		BaseModel:   base.BaseModel{ID: "task-001", TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
		Result:      "",
	}

	resp := TaskToResponse(task)
	if _, ok := resp["result"]; ok {
		t.Error("期望响应中不包含 result 字段")
	}
}

// TestTaskToResponse_InvalidContext Context 为无效 JSON 时，响应中不应包含 context 字段（优雅处理）
func TestTaskToResponse_InvalidContext(t *testing.T) {
	task := domainagent.AgentTask{
		BaseModel:   base.BaseModel{ID: "task-001", TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
		Context:     `{invalid json}`,
	}

	resp := TaskToResponse(task)
	if _, ok := resp["context"]; ok {
		t.Error("期望无效 JSON 的 Context 不出现在响应中")
	}
}

// TestTaskToResponse_InvalidResult Result 为无效 JSON 时，响应中不应包含 result 字段（优雅处理）
func TestTaskToResponse_InvalidResult(t *testing.T) {
	task := domainagent.AgentTask{
		BaseModel:   base.BaseModel{ID: "task-001", TenantID: "tenant-001"},
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "completed",
		Result:      `{not valid}`,
	}

	resp := TaskToResponse(task)
	if _, ok := resp["result"]; ok {
		t.Error("期望无效 JSON 的 Result 不出现在响应中")
	}
}
