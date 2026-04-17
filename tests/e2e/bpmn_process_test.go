package e2e

import (
	"context"
	"testing"

	bpmnapp "git.neolidy.top/neo/flowx/internal/application/bpmn"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// processDefinitionTable 用于 E2E 测试中创建 process_definitions 表的本地结构体
// 对应 persistence 包中未导出的 processDefinitionPO
type processDefinitionTable struct {
	base.BaseModel
	Name           string `gorm:"size:200;not null"`
	Version        int    `gorm:"default:1"`
	Status         string `gorm:"size:20;default:draft;index"`
	DefinitionYAML string `gorm:"type:text;not null"`
}

func (processDefinitionTable) TableName() string { return "process_definitions" }

// setupBPMNE2E 创建 BPMN E2E 测试环境：使用 SQLite 内存数据库和真实服务实例
func setupBPMNE2E(t *testing.T) (*bpmnapp.ProcessService, *gorm.DB) {
	t.Helper()

	// 创建 SQLite 内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 自动迁移所有相关模型（包括流程定义表）
	if err := db.AutoMigrate(
		&bpmn.ProcessInstance{},
		&bpmn.ProcessTask{},
		&bpmn.ExecutionHistory{},
		&processDefinitionTable{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建真实引擎和仓储实例
	engine := bpmnapp.NewEngine()
	defRepo := persistence.NewProcessDefinitionRepository(db)
	instRepo := persistence.NewProcessInstanceRepository(db)
	taskRepo := persistence.NewProcessTaskRepository(db)
	historyRepo := persistence.NewExecutionHistoryRepository(db)
	processSvc := bpmnapp.NewProcessService(engine, defRepo, instRepo, taskRepo, historyRepo)

	return processSvc, db
}

// ==================== 场景1: 简单顺序流完整生命周期 ====================

func TestE2E_BPMN_SimpleSequence(t *testing.T) {
	svc, _ := setupBPMNE2E(t)
	ctx := context.Background()
	const tenantID = "tenant-e2e-1"

	// 简单顺序流定义 YAML：start -> task1 -> end
	simpleSeqYAML := []byte(`
id: simple-seq
name: 简单顺序流
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
    name: 审核任务
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

	// 1. 部署流程定义
	def, err := svc.DeployDefinition(ctx, tenantID, simpleSeqYAML)
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}
	if def.ID != "simple-seq" {
		t.Fatalf("期望定义 ID 为 simple-seq，实际为 %s", def.ID)
	}

	// 2. 启动流程实例
	inst, err := svc.StartProcess(ctx, tenantID, def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程实例失败: %v", err)
	}
	if inst.Status != "running" {
		t.Fatalf("期望实例状态为 running，实际为 %s", inst.Status)
	}

	// 3. 获取待办任务，验证有 1 个且 element_id 为 task1
	pendingTasks, err := svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(pendingTasks))
	}
	if pendingTasks[0].ElementID != "task1" {
		t.Fatalf("期望任务 element_id 为 task1，实际为 %s", pendingTasks[0].ElementID)
	}
	if pendingTasks[0].Name != "审核任务" {
		t.Fatalf("期望任务名称为 审核任务，实际为 %s", pendingTasks[0].Name)
	}

	// 4. 完成任务
	err = svc.CompleteTask(ctx, tenantID, pendingTasks[0].ID, "user1", map[string]any{"approved": true})
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	// 5. 验证实例状态已变为 completed
	updatedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if updatedInst.Status != "completed" {
		t.Fatalf("期望实例状态为 completed，实际为 %s", updatedInst.Status)
	}

	// 6. 验证执行历史包含 start、task1、end 的记录
	histories, err := svc.GetProcessHistory(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取执行历史失败: %v", err)
	}

	// 收集历史中的元素 ID
	elementIDs := make(map[string]bool)
	for _, h := range histories {
		elementIDs[h.ElementID] = true
	}

	if !elementIDs["start"] {
		t.Fatal("期望执行历史中包含 start 事件记录")
	}
	if !elementIDs["task1"] {
		t.Fatal("期望执行历史中包含 task1 任务记录")
	}
	if !elementIDs["end"] {
		t.Fatal("期望执行历史中包含 end 事件记录")
	}
}

// ==================== 场景2: 并行网关 + 挂起/恢复 ====================

func TestE2E_BPMN_ParallelGateway_SuspendResume(t *testing.T) {
	svc, _ := setupBPMNE2E(t)
	ctx := context.Background()
	const tenantID = "tenant-e2e-2"

	// 并行网关定义 YAML：start -> fork -> taskA + taskB -> join -> end
	parallelYAML := []byte(`
id: parallel-gateway
name: 并行网关流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    outgoing: flow1
  - id: flow1
    type: sequenceFlow
    incoming: start
    outgoing: fork
  - id: fork
    type: parallelGateway
    incoming: flow1
    outgoing:
      - flow2
      - flow3
  - id: flow2
    type: sequenceFlow
    incoming: fork
    outgoing: taskA
  - id: flow3
    type: sequenceFlow
    incoming: fork
    outgoing: taskB
  - id: taskA
    type: userTask
    name: 任务A
    incoming: flow2
    outgoing: flow4
  - id: taskB
    type: userTask
    name: 任务B
    incoming: flow3
    outgoing: flow5
  - id: flow4
    type: sequenceFlow
    incoming: taskA
    outgoing: join
  - id: flow5
    type: sequenceFlow
    incoming: taskB
    outgoing: join
  - id: join
    type: parallelGateway
    incoming:
      - flow4
      - flow5
    outgoing: flow6
  - id: flow6
    type: sequenceFlow
    incoming: join
    outgoing: end
  - id: end
    type: endEvent
    incoming: flow6
`)

	// 1. 部署并行网关定义
	def, err := svc.DeployDefinition(ctx, tenantID, parallelYAML)
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}

	// 2. 启动流程实例
	inst, err := svc.StartProcess(ctx, tenantID, def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程实例失败: %v", err)
	}
	if inst.Status != "running" {
		t.Fatalf("期望实例状态为 running，实际为 %s", inst.Status)
	}

	// 3. 验证有 2 个待办任务
	pendingTasks, err := svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 2 {
		t.Fatalf("期望 2 个待办任务，实际为 %d", len(pendingTasks))
	}

	// 4. 挂起流程
	err = svc.SuspendProcess(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("挂起流程失败: %v", err)
	}
	suspendedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if suspendedInst.Status != "suspended" {
		t.Fatalf("期望实例状态为 suspended，实际为 %s", suspendedInst.Status)
	}

	// 5. 挂起状态下尝试完成任务，期望报错
	err = svc.CompleteTask(ctx, tenantID, pendingTasks[0].ID, "user1", nil)
	if err == nil {
		t.Fatal("期望挂起状态下完成任务失败，但实际成功了")
	}

	// 6. 恢复流程
	err = svc.ResumeProcess(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("恢复流程失败: %v", err)
	}
	resumedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if resumedInst.Status != "running" {
		t.Fatalf("期望实例状态为 running，实际为 %s", resumedInst.Status)
	}

	// 7. 完成两个任务
	// 重新获取待办任务（挂起/恢复后任务仍在）
	pendingTasks, err = svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 2 {
		t.Fatalf("期望恢复后仍有 2 个待办任务，实际为 %d", len(pendingTasks))
	}

	// 完成第一个任务
	err = svc.CompleteTask(ctx, tenantID, pendingTasks[0].ID, "user1", map[string]any{"taskA_result": "done"})
	if err != nil {
		t.Fatalf("完成第一个任务失败: %v", err)
	}

	// 完成第二个任务（并行汇聚后流程应继续到 endEvent）
	err = svc.CompleteTask(ctx, tenantID, pendingTasks[1].ID, "user1", map[string]any{"taskB_result": "done"})
	if err != nil {
		t.Fatalf("完成第二个任务失败: %v", err)
	}

	// 8. 验证流程已完成
	completedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if completedInst.Status != "completed" {
		t.Fatalf("期望实例状态为 completed，实际为 %s", completedInst.Status)
	}
}

// ==================== 场景3: 排他网关 + 基于变量的路由 ====================

func TestE2E_BPMN_ExclusiveGateway_Routing(t *testing.T) {
	svc, _ := setupBPMNE2E(t)
	ctx := context.Background()
	const tenantID = "tenant-e2e-3"

	// 排他网关定义 YAML：start -> gateway -> (amount>1000: taskA, default: taskB) -> end
	exclusiveYAML := []byte(`
id: exclusive-gateway
name: 排他网关流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    outgoing: flow1
  - id: flow1
    type: sequenceFlow
    incoming: start
    outgoing: gw1
  - id: gw1
    type: exclusiveGateway
    incoming: flow1
    outgoing:
      - flow2
      - flow3
  - id: flow2
    type: sequenceFlow
    incoming: gw1
    outgoing: taskA
    condition: "tool.config.amount > 1000"
  - id: flow3
    type: sequenceFlow
    incoming: gw1
    outgoing: taskB
  - id: taskA
    type: userTask
    name: 大额审批
    incoming: flow2
    outgoing: flow4
  - id: taskB
    type: userTask
    name: 小额审批
    incoming: flow3
    outgoing: flow5
  - id: flow4
    type: sequenceFlow
    incoming: taskA
    outgoing: end
  - id: flow5
    type: sequenceFlow
    incoming: taskB
    outgoing: end
  - id: end
    type: endEvent
    incoming:
      - flow4
      - flow5
`)

	// 1. 部署排他网关定义
	def, err := svc.DeployDefinition(ctx, tenantID, exclusiveYAML)
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}

	// 2. 启动流程实例，amount=5000 应走 taskA 分支（amount > 1000）
	inst, err := svc.StartProcess(ctx, tenantID, def.ID, "user1", map[string]any{"amount": 5000})
	if err != nil {
		t.Fatalf("启动流程实例失败: %v", err)
	}

	// 3. 验证只有 1 个待办任务，且为 taskA（大额审批）
	pendingTasks, err := svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(pendingTasks))
	}
	if pendingTasks[0].ElementID != "taskA" {
		t.Fatalf("期望走 taskA 分支（amount=5000 > 1000），实际为 %s", pendingTasks[0].ElementID)
	}
	if pendingTasks[0].Name != "大额审批" {
		t.Fatalf("期望任务名称为 大额审批，实际为 %s", pendingTasks[0].Name)
	}

	// 4. 完成 taskA，验证流程结束
	err = svc.CompleteTask(ctx, tenantID, pendingTasks[0].ID, "user1", map[string]any{"approved": true})
	if err != nil {
		t.Fatalf("完成任务失败: %v", err)
	}

	completedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if completedInst.Status != "completed" {
		t.Fatalf("期望实例状态为 completed，实际为 %s", completedInst.Status)
	}
}

// ==================== 场景4: 取消运行中的流程 ====================

func TestE2E_BPMN_CancelRunningProcess(t *testing.T) {
	svc, _ := setupBPMNE2E(t)
	ctx := context.Background()
	const tenantID = "tenant-e2e-4"

	// 简单流程定义
	simpleYAML := []byte(`
id: cancel-test
name: 取消测试流程
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
    name: 待办任务
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

	// 1. 部署流程定义
	def, err := svc.DeployDefinition(ctx, tenantID, simpleYAML)
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}

	// 2. 启动流程实例
	inst, err := svc.StartProcess(ctx, tenantID, def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程实例失败: %v", err)
	}

	// 3. 取消流程
	err = svc.CancelProcess(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("取消流程失败: %v", err)
	}

	// 4. 验证实例状态为 cancelled
	cancelledInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if cancelledInst.Status != "cancelled" {
		t.Fatalf("期望实例状态为 cancelled，实际为 %s", cancelledInst.Status)
	}

	// 5. 验证没有待办任务
	pendingTasks, err := svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 0 {
		t.Fatalf("期望取消后没有待办任务，实际为 %d", len(pendingTasks))
	}
}

// ==================== 场景5: 服务任务自动完成 ====================

func TestE2E_BPMN_ServiceTask_AutoComplete(t *testing.T) {
	svc, _ := setupBPMNE2E(t)
	ctx := context.Background()
	const tenantID = "tenant-e2e-5"

	// 包含服务任务的流程定义：start -> serviceTask -> end
	// 服务任务会自动完成，不需要人工干预
	serviceYAML := []byte(`
id: service-task-test
name: 服务任务自动完成流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    outgoing: flow1
  - id: flow1
    type: sequenceFlow
    incoming: start
    outgoing: svc1
  - id: svc1
    type: serviceTask
    name: 自动处理
    incoming: flow1
    outgoing: flow2
    params:
      auto_result: success
  - id: flow2
    type: sequenceFlow
    incoming: svc1
    outgoing: end
  - id: end
    type: endEvent
    incoming: flow2
`)

	// 1. 部署流程定义
	def, err := svc.DeployDefinition(ctx, tenantID, serviceYAML)
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}

	// 2. 启动流程实例
	inst, err := svc.StartProcess(ctx, tenantID, def.ID, "user1", nil)
	if err != nil {
		t.Fatalf("启动流程实例失败: %v", err)
	}

	// 3. 验证流程立即完成（服务任务自动完成）
	if inst.Status != "completed" {
		t.Fatalf("期望实例状态为 completed（服务任务自动完成），实际为 %s", inst.Status)
	}

	// 4. 验证没有待办任务
	pendingTasks, err := svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 0 {
		t.Fatalf("期望没有待办任务，实际为 %d", len(pendingTasks))
	}

	// 5. 验证服务任务的参数已合并到实例变量中
	updatedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if updatedInst.Variables == nil {
		t.Fatal("期望实例变量不为空")
	}
	if updatedInst.Variables["auto_result"] != "success" {
		t.Fatalf("期望变量 auto_result=success，实际为 %v", updatedInst.Variables["auto_result"])
	}
}

// ==================== 场景6: 变量跨步骤持久化 ====================

func TestE2E_BPMN_Variables_PersistAcrossSteps(t *testing.T) {
	svc, _ := setupBPMNE2E(t)
	ctx := context.Background()
	const tenantID = "tenant-e2e-6"

	// 包含两个顺序 userTask 的流程定义：start -> task1 -> task2 -> end
	twoTaskYAML := []byte(`
id: variable-persist-test
name: 变量持久化测试流程
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
    incoming: flow1
    outgoing: flow2
  - id: flow2
    type: sequenceFlow
    incoming: task1
    outgoing: task2
  - id: task2
    type: userTask
    name: 第二步审批
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

	// 1. 部署流程定义
	def, err := svc.DeployDefinition(ctx, tenantID, twoTaskYAML)
	if err != nil {
		t.Fatalf("部署流程定义失败: %v", err)
	}

	// 2. 启动流程实例，携带初始变量
	inst, err := svc.StartProcess(ctx, tenantID, def.ID, "user1", map[string]any{
		"requester": "alice",
		"amount":    100,
	})
	if err != nil {
		t.Fatalf("启动流程实例失败: %v", err)
	}

	// 3. 获取第一个待办任务并完成
	pendingTasks, err := svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(pendingTasks))
	}
	if pendingTasks[0].ElementID != "task1" {
		t.Fatalf("期望第一个任务为 task1，实际为 %s", pendingTasks[0].ElementID)
	}

	err = svc.CompleteTask(ctx, tenantID, pendingTasks[0].ID, "user1", map[string]any{
		"approved": true,
	})
	if err != nil {
		t.Fatalf("完成第一个任务失败: %v", err)
	}

	// 4. 获取第二个待办任务并完成
	pendingTasks, err = svc.GetPendingTasks(ctx, tenantID, "")
	if err != nil {
		t.Fatalf("获取待办任务失败: %v", err)
	}
	if len(pendingTasks) != 1 {
		t.Fatalf("期望 1 个待办任务，实际为 %d", len(pendingTasks))
	}
	if pendingTasks[0].ElementID != "task2" {
		t.Fatalf("期望第二个任务为 task2，实际为 %s", pendingTasks[0].ElementID)
	}

	err = svc.CompleteTask(ctx, tenantID, pendingTasks[0].ID, "user1", map[string]any{
		"comment": "ok",
	})
	if err != nil {
		t.Fatalf("完成第二个任务失败: %v", err)
	}

	// 5. 验证流程已完成
	completedInst, err := svc.GetProcessInstance(ctx, tenantID, inst.ID)
	if err != nil {
		t.Fatalf("获取流程实例失败: %v", err)
	}
	if completedInst.Status != "completed" {
		t.Fatalf("期望实例状态为 completed，实际为 %s", completedInst.Status)
	}

	// 6. 验证变量包含所有合并后的值
	if completedInst.Variables == nil {
		t.Fatal("期望实例变量不为空")
	}

	// 验证启动时的初始变量（JSON 反序列化后数字为 float64）
	if completedInst.Variables["requester"] != "alice" {
		t.Fatalf("期望变量 requester=alice，实际为 %v", completedInst.Variables["requester"])
	}
	amountVal, ok := completedInst.Variables["amount"].(float64)
	if !ok || amountVal != 100 {
		t.Fatalf("期望变量 amount=100，实际为 %v", completedInst.Variables["amount"])
	}

	// 验证第一个任务提交的变量
	if completedInst.Variables["approved"] != true {
		t.Fatalf("期望变量 approved=true，实际为 %v", completedInst.Variables["approved"])
	}

	// 验证第二个任务提交的变量
	if completedInst.Variables["comment"] != "ok" {
		t.Fatalf("期望变量 comment=ok，实际为 %v", completedInst.Variables["comment"])
	}
}
