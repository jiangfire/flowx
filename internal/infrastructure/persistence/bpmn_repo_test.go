package persistence

import (
	"context"
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
	bpmnapp "git.neolidy.top/neo/flowx/internal/application/bpmn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBPMNTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&processDefinitionPO{},
		&bpmn.ProcessInstance{},
		&bpmn.ProcessTask{},
		&bpmn.ExecutionHistory{},
	)
	require.NoError(t, err)

	return db
}

// ==================== ProcessDefinitionRepository Tests ====================

func TestCreateAndGetDefinition(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewProcessDefinitionRepository(db)
	ctx := context.Background()

	def := &bpmn.ProcessDefinition{
		ID:      "def-001",
		Name:    "请假流程",
		Version: 1,
		Status:  "active",
		Elements: []bpmn.Element{
			{ID: "start", Type: "startEvent", Name: "开始", Outgoing: []string{"flow1"}},
			{ID: "flow1", Type: "sequenceFlow", Outgoing: []string{"task1"}},
			{ID: "task1", Type: "userTask", Name: "审批", Assignee: "admin", Incoming: []string{"flow1"}, Outgoing: []string{"flow2"}},
			{ID: "flow2", Type: "sequenceFlow", Outgoing: []string{"end"}},
			{ID: "end", Type: "endEvent", Name: "结束", Incoming: []string{"flow2"}},
		},
	}

	err := repo.Create(ctx, def)
	require.NoError(t, err)
	assert.NotEmpty(t, def.ID)

	got, err := repo.GetByID(ctx, def.ID)
	require.NoError(t, err)
	assert.Equal(t, def.Name, got.Name)
	assert.Equal(t, def.Version, got.Version)
	assert.Equal(t, def.Status, got.Status)
	assert.Len(t, got.Elements, 5)

	// 查询不存在的定义
	_, err = repo.GetByID(ctx, "not-exist")
	assert.Error(t, err)
}

func TestListDefinitions(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewProcessDefinitionRepository(db)
	ctx := context.Background()

	// 创建多个定义
	for i := 0; i < 3; i++ {
		def := &bpmn.ProcessDefinition{
			ID:      base.GenerateUUID(),
			Name:    "流程-" + string(rune('A'+i)),
			Version: 1,
			Status:  "active",
			Elements: []bpmn.Element{
				{ID: "start", Type: "startEvent", Name: "开始"},
				{ID: "end", Type: "endEvent", Name: "结束"},
			},
		}
		err := repo.Create(ctx, def)
		require.NoError(t, err)
	}

	// 查询全部
	defs, total, err := repo.List(ctx, bpmnapp.ProcessDefinitionFilter{TenantID: "", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, defs, 3)

	// 分页查询
	defs, total, err = repo.List(ctx, bpmnapp.ProcessDefinitionFilter{TenantID: "", Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, defs, 2)
}

// ==================== ProcessInstanceRepository Tests ====================

func TestCreateAndGetProcessInstance(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewProcessInstanceRepository(db)
	ctx := context.Background()

	now := time.Now()
	inst := &bpmn.ProcessInstance{
		BaseModel:      base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
		DefinitionID:   "def-001",
		Status:         "running",
		Variables:      base.JSON{"amount": 100},
		CurrentElements: base.JSON{"task1": true},
		StartedBy:      "user-001",
	}

	err := repo.Create(ctx, inst)
	require.NoError(t, err)
	assert.NotEmpty(t, inst.ID)

	got, err := repo.GetByID(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, inst.DefinitionID, got.DefinitionID)
	assert.Equal(t, inst.Status, got.Status)
	assert.Equal(t, inst.StartedBy, got.StartedBy)
}

func TestListProcessInstances(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewProcessInstanceRepository(db)
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 3; i++ {
		inst := &bpmn.ProcessInstance{
			BaseModel:    base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
			DefinitionID: "def-001",
			Status:       "running",
			StartedBy:    "user-001",
		}
		err := repo.Create(ctx, inst)
		require.NoError(t, err)
	}

	// 查询全部
	instances, total, err := repo.List(ctx, bpmnapp.ProcessInstanceFilter{TenantID: "tenant-001", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, instances, 3)

	// 按状态过滤
	instances, total, err = repo.List(ctx, bpmnapp.ProcessInstanceFilter{TenantID: "tenant-001", Status: "completed", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, instances, 0)
}

// ==================== ProcessTaskRepository Tests ====================

func TestCreateAndGetProcessTask(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewProcessTaskRepository(db)
	ctx := context.Background()

	now := time.Now()
	task := &bpmn.ProcessTask{
		BaseModel:   base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
		InstanceID:  "inst-001",
		ElementID:   "task1",
		Name:        "审批任务",
		Assignee:    "admin",
		Status:      "pending",
		FormFields:  base.JSON{"field1": "value1"},
	}

	err := repo.Create(ctx, task)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)

	got, err := repo.GetByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.Name, got.Name)
	assert.Equal(t, task.Assignee, got.Assignee)
	assert.Equal(t, task.Status, got.Status)
}

func TestListPendingTasks(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewProcessTaskRepository(db)
	ctx := context.Background()

	now := time.Now()

	// 创建 pending 任务
	task1 := &bpmn.ProcessTask{
		BaseModel:  base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
		InstanceID: "inst-001", ElementID: "task1", Name: "审批", Assignee: "admin", Status: "pending",
	}
	err := repo.Create(ctx, task1)
	require.NoError(t, err)

	// 创建 completed 任务
	task2 := &bpmn.ProcessTask{
		BaseModel:  base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
		InstanceID: "inst-001", ElementID: "task2", Name: "确认", Assignee: "admin", Status: "completed",
	}
	err = repo.Create(ctx, task2)
	require.NoError(t, err)

	// 创建另一个 assignee 的 pending 任务
	task3 := &bpmn.ProcessTask{
		BaseModel:  base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
		InstanceID: "inst-002", ElementID: "task3", Name: "审核", Assignee: "manager", Status: "pending",
	}
	err = repo.Create(ctx, task3)
	require.NoError(t, err)

	// 查询所有 pending
	pending, err := repo.ListPending(ctx, "tenant-001", "")
	require.NoError(t, err)
	assert.Len(t, pending, 2)

	// 按 assignee 过滤
	pending, err = repo.ListPending(ctx, "tenant-001", "admin")
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, "审批", pending[0].Name)

	// 按实例查询
	tasks, err := repo.ListByInstance(ctx, "inst-001")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

// ==================== ExecutionHistoryRepository Tests ====================

func TestCreateAndListHistory(t *testing.T) {
	db := setupBPMNTestDB(t)
	repo := NewExecutionHistoryRepository(db)
	ctx := context.Background()

	now := time.Now()
	histories := []*bpmn.ExecutionHistory{
		{
			BaseModel:    base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
			InstanceID:   "inst-001",
			ElementID:    "start",
			ElementType:  "startEvent",
			Action:       "enter",
			Variables:    base.JSON{"key": "value"},
		},
		{
			BaseModel:    base.BaseModel{ID: base.GenerateUUID(), TenantID: "tenant-001", CreatedAt: now, UpdatedAt: now},
			InstanceID:   "inst-001",
			ElementID:    "start",
			ElementType:  "startEvent",
			Action:       "leave",
		},
	}

	for _, h := range histories {
		err := repo.Create(ctx, h)
		require.NoError(t, err)
	}

	got, err := repo.ListByInstance(ctx, "inst-001")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "enter", got[0].Action)
	assert.Equal(t, "leave", got[1].Action)

	// 查询不存在的实例
	got, err = repo.ListByInstance(ctx, "not-exist")
	require.NoError(t, err)
	assert.Len(t, got, 0)
}
