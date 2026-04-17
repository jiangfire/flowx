package bpmn

import (
	"testing"
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
)

// helper: create a simple process definition
func newDef(id string, elements []bpmn.Element) *bpmn.ProcessDefinition {
	return &bpmn.ProcessDefinition{
		ID:       id,
		Name:     id,
		Version:  1,
		Status:   "active",
		Elements: elements,
	}
}

// helper: start an instance
func startInstance(t *testing.T, engine *Engine, def *bpmn.ProcessDefinition, variables map[string]any) *bpmn.ProcessInstance {
	t.Helper()
	inst := engine.Start(def, "tenant1", "user1", variables)
	if inst == nil {
		t.Fatal("expected non-nil instance")
	}
	return inst
}

// ========== Test 1: Simple Sequence ==========

func TestEngine_SimpleSequence(t *testing.T) {
	def := newDef("simple", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	if inst.Status != "running" {
		t.Fatalf("expected running, got %s", inst.Status)
	}

	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(tasks))
	}
	if tasks[0].ElementID != "task1" {
		t.Fatalf("expected elementID task1, got %s", tasks[0].ElementID)
	}

	// Complete the task
	err := engine.CompleteTask(inst.ID, tasks[0].ID, "user1", map[string]any{"approved": true})
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}

	tasks = engine.GetPendingTasks(inst.ID)
	if len(tasks) != 0 {
		t.Fatalf("expected 0 pending tasks, got %d", len(tasks))
	}
}

// ========== Test 2: Exclusive Gateway ==========

func TestEngine_ExclusiveGateway(t *testing.T) {
	def := newDef("exclusive", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"gw1"}},
		{ID: "gw1", Type: "exclusiveGateway", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2", "flow3"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw1"}, Outgoing: bpmn.StringSlice{"taskA"}, Condition: "tool.config.amount > 1000"},
		{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw1"}, Outgoing: bpmn.StringSlice{"taskB"}},
		{ID: "taskA", Type: "userTask", Name: "TaskA", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow4"}},
		{ID: "taskB", Type: "userTask", Name: "TaskB", Incoming: bpmn.StringSlice{"flow3"}, Outgoing: bpmn.StringSlice{"flow5"}},
		{ID: "flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskA"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "flow5", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskB"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow4", "flow5"}},
	})

	// Branch A: amount > 1000
	t.Run("branchA", func(t *testing.T) {
		engine := NewEngine()
		inst := startInstance(t, engine, def, map[string]any{"amount": 2000})
		tasks := engine.GetPendingTasks(inst.ID)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 pending task, got %d", len(tasks))
		}
		if tasks[0].ElementID != "taskA" {
			t.Fatalf("expected taskA, got %s", tasks[0].ElementID)
		}
	})

	// Branch B: default (no condition matches)
	t.Run("branchB", func(t *testing.T) {
		engine := NewEngine()
		inst := startInstance(t, engine, def, map[string]any{"amount": 500})
		tasks := engine.GetPendingTasks(inst.ID)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 pending task, got %d", len(tasks))
		}
		if tasks[0].ElementID != "taskB" {
			t.Fatalf("expected taskB, got %s", tasks[0].ElementID)
		}
	})
}

// ========== Test 3: Parallel Gateway ==========

func TestEngine_ParallelGateway(t *testing.T) {
	def := newDef("parallel", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"gw_fork"}},
		{ID: "gw_fork", Type: "parallelGateway", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2", "flow3"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_fork"}, Outgoing: bpmn.StringSlice{"taskA"}},
		{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_fork"}, Outgoing: bpmn.StringSlice{"taskB"}},
		{ID: "taskA", Type: "userTask", Name: "TaskA", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow4"}},
		{ID: "taskB", Type: "userTask", Name: "TaskB", Incoming: bpmn.StringSlice{"flow3"}, Outgoing: bpmn.StringSlice{"flow5"}},
		{ID: "flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskA"}, Outgoing: bpmn.StringSlice{"gw_join"}},
		{ID: "flow5", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskB"}, Outgoing: bpmn.StringSlice{"gw_join"}},
		{ID: "gw_join", Type: "parallelGateway", Incoming: bpmn.StringSlice{"flow4", "flow5"}, Outgoing: bpmn.StringSlice{"flow6"}},
		{ID: "flow6", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_join"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow6"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 pending tasks, got %d", len(tasks))
	}

	// Complete first task
	err := engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	inst = engine.GetInstance(inst.ID)
	if inst.Status != "running" {
		t.Fatalf("expected still running, got %s", inst.Status)
	}

	// Complete second task
	err = engine.CompleteTask(inst.ID, tasks[1].ID, "user1", nil)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

// ========== Test 4: Inclusive Gateway ==========

func TestEngine_InclusiveGateway(t *testing.T) {
	def := newDef("inclusive", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"gw1"}},
		{ID: "gw1", Type: "inclusiveGateway", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2", "flow3", "flow4"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw1"}, Outgoing: bpmn.StringSlice{"taskA"}, Condition: "tool.config.amount > 5000"},
		{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw1"}, Outgoing: bpmn.StringSlice{"taskB"}, Condition: "tool.config.role == \"manager\""},
		{ID: "flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw1"}, Outgoing: bpmn.StringSlice{"taskC"}},
		{ID: "taskA", Type: "userTask", Name: "TaskA", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow5"}},
		{ID: "taskB", Type: "userTask", Name: "TaskB", Incoming: bpmn.StringSlice{"flow3"}, Outgoing: bpmn.StringSlice{"flow6"}},
		{ID: "taskC", Type: "userTask", Name: "TaskC", Incoming: bpmn.StringSlice{"flow4"}, Outgoing: bpmn.StringSlice{"flow7"}},
		{ID: "flow5", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskA"}, Outgoing: bpmn.StringSlice{"gw2"}},
		{ID: "flow6", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskB"}, Outgoing: bpmn.StringSlice{"gw2"}},
		{ID: "flow7", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskC"}, Outgoing: bpmn.StringSlice{"gw2"}},
		{ID: "gw2", Type: "inclusiveGateway", Incoming: bpmn.StringSlice{"flow5", "flow6", "flow7"}, Outgoing: bpmn.StringSlice{"flow8"}},
		{ID: "flow8", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw2"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow8"}},
	})

	// amount=6000, role=manager -> TaskA + TaskB + TaskC (no condition = always)
	t.Run("allBranches", func(t *testing.T) {
		engine := NewEngine()
		inst := startInstance(t, engine, def, map[string]any{"amount": 6000, "role": "manager"})
		tasks := engine.GetPendingTasks(inst.ID)
		if len(tasks) != 3 {
			t.Fatalf("expected 3 pending tasks, got %d", len(tasks))
		}
		// Complete all
		for _, task := range tasks {
			engine.CompleteTask(inst.ID, task.ID, "user1", nil)
		}
		inst = engine.GetInstance(inst.ID)
		if inst.Status != "completed" {
			t.Fatalf("expected completed, got %s", inst.Status)
		}
	})

	// amount=1000, role=user -> only TaskC (no condition)
	t.Run("onlyDefault", func(t *testing.T) {
		engine := NewEngine()
		inst := startInstance(t, engine, def, map[string]any{"amount": 1000, "role": "user"})
		tasks := engine.GetPendingTasks(inst.ID)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 pending task, got %d", len(tasks))
		}
		if tasks[0].ElementID != "taskC" {
			t.Fatalf("expected taskC, got %s", tasks[0].ElementID)
		}
	})
}

// ========== Test 5: Service Task ==========

func TestEngine_ServiceTask(t *testing.T) {
	def := newDef("service", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"svc1"}},
		{ID: "svc1", Type: "serviceTask", Name: "AutoTask", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}, Params: map[string]any{"key": "value"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"svc1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	// Service task auto-completes, should reach end
	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}

	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 0 {
		t.Fatalf("expected 0 pending tasks, got %d", len(tasks))
	}
}

// ========== Test 6: Suspend / Resume ==========

func TestEngine_SuspendResume(t *testing.T) {
	def := newDef("suspend", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)
	tasks := engine.GetPendingTasks(inst.ID)

	// Suspend
	engine.Suspend(inst.ID)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "suspended" {
		t.Fatalf("expected suspended, got %s", inst.Status)
	}

	// Try complete while suspended -> error
	err := engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)
	if err == nil {
		t.Fatal("expected error when completing suspended instance")
	}

	// Resume
	engine.Resume(inst.ID)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "running" {
		t.Fatalf("expected running, got %s", inst.Status)
	}

	// Complete after resume -> success
	err = engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)
	if err != nil {
		t.Fatalf("CompleteTask after resume failed: %v", err)
	}

	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

// ========== Test 7: Cancel ==========

func TestEngine_Cancel(t *testing.T) {
	def := newDef("cancel", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	engine.Cancel(inst.ID)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "cancelled" {
		t.Fatalf("expected cancelled, got %s", inst.Status)
	}

	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 0 {
		t.Fatalf("expected 0 pending tasks after cancel, got %d", len(tasks))
	}
}

// ========== Test 8: Variables ==========

func TestEngine_Variables(t *testing.T) {
	def := newDef("vars", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, map[string]any{"amount": 500, "requester": "alice"})

	tasks := engine.GetPendingTasks(inst.ID)
	err := engine.CompleteTask(inst.ID, tasks[0].ID, "user1", map[string]any{"approved": true, "comment": "ok"})
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	inst = engine.GetInstance(inst.ID)
	vars := inst.Variables
	if vars["amount"] != 500 {
		t.Fatalf("expected amount=500, got %v", vars["amount"])
	}
	if vars["requester"] != "alice" {
		t.Fatalf("expected requester=alice, got %v", vars["requester"])
	}
	if vars["approved"] != true {
		t.Fatalf("expected approved=true, got %v", vars["approved"])
	}
	if vars["comment"] != "ok" {
		t.Fatalf("expected comment=ok, got %v", vars["comment"])
	}
}

// ========== Test 9: SubProcess ==========

func TestEngine_SubProcess(t *testing.T) {
	def := newDef("subprocess", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"sub1"}},
		{ID: "sub1", Type: "subProcess", Name: "SubProcess", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow4"},
			Elements: []bpmn.Element{
				{ID: "sub_start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow2"}},
				{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub_start"}, Outgoing: bpmn.StringSlice{"sub_task"}},
				{ID: "sub_task", Type: "userTask", Name: "SubTask", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow3"}},
				{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub_task"}, Outgoing: bpmn.StringSlice{"sub_end"}},
				{ID: "sub_end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow3"}},
			},
		},
		{ID: "flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow4"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 pending task from subprocess, got %d", len(tasks))
	}
	if tasks[0].ElementID != "sub_task" {
		t.Fatalf("expected sub_task, got %s", tasks[0].ElementID)
	}

	// Complete subprocess task -> should complete the whole instance
	err := engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

// ========== Test 10: Execution History ==========

func TestEngine_ExecutionHistory(t *testing.T) {
	def := newDef("history", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	history := engine.GetHistory(inst.ID)
	if len(history) == 0 {
		t.Fatal("expected history entries, got 0")
	}

	// Should have at least: start enter, start leave, task1 enter
	hasStartEnter := false
	hasTaskEnter := false
	for _, h := range history {
		if h.ElementID == "start" && h.Action == "enter" {
			hasStartEnter = true
		}
		if h.ElementID == "task1" && h.Action == "enter" {
			hasTaskEnter = true
		}
	}
	if !hasStartEnter {
		t.Fatal("expected start enter history")
	}
	if !hasTaskEnter {
		t.Fatal("expected task1 enter history")
	}

	// Complete task and check more history
	tasks := engine.GetPendingTasks(inst.ID)
	engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)

	history = engine.GetHistory(inst.ID)
	hasEndEnter := false
	for _, h := range history {
		if h.ElementID == "end" && h.Action == "enter" {
			hasEndEnter = true
		}
	}
	if !hasEndEnter {
		t.Fatal("expected end enter history")
	}
}

// ========== Additional edge case tests ==========

func TestEngine_CompleteTask_InvalidTask(t *testing.T) {
	def := newDef("simple", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)

	err := engine.CompleteTask(inst.ID, "nonexistent-task-id", "user1", nil)
	if err == nil {
		t.Fatal("expected error for invalid task ID")
	}
}

func TestEngine_Suspend_AlreadySuspended(t *testing.T) {
	def := newDef("simple", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)
	engine.Suspend(inst.ID)

	// Double suspend should be safe (no-op or error, just don't panic)
	engine.Suspend(inst.ID)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "suspended" {
		t.Fatalf("expected still suspended, got %s", inst.Status)
	}
}

func TestEngine_Cancel_AlreadyCompleted(t *testing.T) {
	def := newDef("simple", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Name: "Review", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"task1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)
	tasks := engine.GetPendingTasks(inst.ID)
	engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)

	// Cancel completed instance should be safe (no-op)
	engine.Cancel(inst.ID)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Fatalf("expected still completed, got %s", inst.Status)
	}
}

func TestEngine_ServiceTask_MergesParams(t *testing.T) {
	def := newDef("service_params", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"svc1"}},
		{ID: "svc1", Type: "serviceTask", Name: "AutoTask", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}, Params: map[string]any{"result": "done"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"svc1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow2"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, map[string]any{"input": "test"})

	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}

	// Verify params merged into variables
	if inst.Variables["result"] != "done" {
		t.Fatalf("expected result=done, got %v", inst.Variables["result"])
	}
	if inst.Variables["input"] != "test" {
		t.Fatalf("expected input=test, got %v", inst.Variables["input"])
	}
}

func TestEngine_ParallelGateway_WaitForAll(t *testing.T) {
	def := newDef("parallel_wait", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"gw_fork"}},
		{ID: "gw_fork", Type: "parallelGateway", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2", "flow3"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_fork"}, Outgoing: bpmn.StringSlice{"taskA"}},
		{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_fork"}, Outgoing: bpmn.StringSlice{"taskB"}},
		{ID: "taskA", Type: "userTask", Name: "TaskA", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow4"}},
		{ID: "taskB", Type: "userTask", Name: "TaskB", Incoming: bpmn.StringSlice{"flow3"}, Outgoing: bpmn.StringSlice{"flow5"}},
		{ID: "flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskA"}, Outgoing: bpmn.StringSlice{"gw_join"}},
		{ID: "flow5", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskB"}, Outgoing: bpmn.StringSlice{"gw_join"}},
		{ID: "gw_join", Type: "parallelGateway", Incoming: bpmn.StringSlice{"flow4", "flow5"}, Outgoing: bpmn.StringSlice{"flow6"}},
		{ID: "flow6", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_join"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow6"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)
	tasks := engine.GetPendingTasks(inst.ID)

	// Complete only one task
	engine.CompleteTask(inst.ID, tasks[0].ID, "user1", nil)

	inst = engine.GetInstance(inst.ID)
	if inst.Status == "completed" {
		t.Fatal("should NOT be completed after only one branch")
	}
	if inst.Status != "running" {
		t.Fatalf("expected running, got %s", inst.Status)
	}

	// Complete second task -> now completed
	engine.CompleteTask(inst.ID, tasks[1].ID, "user1", nil)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

// Verify that history entries have proper timestamps
func TestEngine_ExecutionHistory_Timestamps(t *testing.T) {
	def := newDef("hist_ts", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow1"}},
	})

	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)
	history := engine.GetHistory(inst.ID)

	for _, h := range history {
		if h.CreatedAt.IsZero() {
			t.Fatalf("history entry %s should have non-zero CreatedAt", h.ID)
		}
		if h.InstanceID != inst.ID {
			t.Fatalf("history entry should reference instance %s, got %s", inst.ID, h.InstanceID)
		}
	}

	_ = time.Now() // ensure time import is used
}

// ========== Boundary: ExclusiveGateway no matching condition ==========

func TestEngine_ExclusiveGateway_NoMatchingCondition(t *testing.T) {
	def := newDef("no-match-test", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"gw1"}},
		{ID: "gw1", Type: "exclusiveGateway", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw1"}, Outgoing: bpmn.StringSlice{"taskA"}, Condition: "impossible_condition == true"},
		{ID: "taskA", Type: "userTask", Name: "需要条件", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow3"}},
		{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskA"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow3"}},
	})
	engine := NewEngine()
	inst := startInstance(t, engine, def, map[string]any{"impossible_condition": false})
	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks when no condition matches, got %d", len(tasks))
	}
}

// ========== Boundary: ParallelGateway different completion order ==========

func TestEngine_ParallelGateway_DifferentCompletionOrder(t *testing.T) {
	def := newDef("order-test", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"gw_fork"}},
		{ID: "gw_fork", Type: "parallelGateway", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow2", "flow3"}},
		{ID: "flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_fork"}, Outgoing: bpmn.StringSlice{"taskA"}},
		{ID: "flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_fork"}, Outgoing: bpmn.StringSlice{"taskB"}},
		{ID: "taskA", Type: "userTask", Name: "A", Incoming: bpmn.StringSlice{"flow2"}, Outgoing: bpmn.StringSlice{"flow4"}},
		{ID: "taskB", Type: "userTask", Name: "B", Incoming: bpmn.StringSlice{"flow3"}, Outgoing: bpmn.StringSlice{"flow5"}},
		{ID: "flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskA"}, Outgoing: bpmn.StringSlice{"gw_join"}},
		{ID: "flow5", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"taskB"}, Outgoing: bpmn.StringSlice{"gw_join"}},
		{ID: "gw_join", Type: "parallelGateway", Incoming: bpmn.StringSlice{"flow4", "flow5"}, Outgoing: bpmn.StringSlice{"flow6"}},
		{ID: "flow6", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"gw_join"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow6"}},
	})
	engine := NewEngine()
	inst := startInstance(t, engine, def, nil)
	tasks := engine.GetPendingTasks(inst.ID)
	// Complete taskB first (reverse order)
	var taskBID string
	for _, task := range tasks {
		if task.ElementID == "taskB" {
			taskBID = task.ID
		}
	}
	engine.CompleteTask(inst.ID, taskBID, "admin", nil)
	inst = engine.GetInstance(inst.ID)
	if inst.Status == "completed" {
		t.Error("should not complete after only one of two parallel tasks")
	}
	// Complete taskA
	remaining := engine.GetPendingTasks(inst.ID)
	engine.CompleteTask(inst.ID, remaining[0].ID, "admin", nil)
	inst = engine.GetInstance(inst.ID)
	if inst.Status != "completed" {
		t.Errorf("expected completed, got %s", inst.Status)
	}
}

// ========== Boundary: CompleteTask with wrong instance ==========

func TestEngine_CompleteTask_WrongInstance(t *testing.T) {
	def := newDef("wrong-inst", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"task1"}},
		{ID: "task1", Type: "userTask", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"task1"}},
	})
	engine := NewEngine()
	inst := engine.Start(def, "t1", "u1", nil)
	tasks := engine.GetPendingTasks(inst.ID)
	err := engine.CompleteTask("nonexistent-instance", tasks[0].ID, "admin", nil)
	if err == nil {
		t.Error("expected error for nonexistent instance")
	}
}

// ========== Boundary: SubProcess with gateway ==========

func TestEngine_SubProcess_WithGateway(t *testing.T) {
	def := newDef("sub-gw-test", []bpmn.Element{
		{ID: "start", Type: "startEvent", Outgoing: bpmn.StringSlice{"flow1"}},
		{ID: "flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"start"}, Outgoing: bpmn.StringSlice{"sub1"}},
		{ID: "sub1", Type: "subProcess", Name: "子流程", Incoming: bpmn.StringSlice{"flow1"}, Outgoing: bpmn.StringSlice{"flow6"},
			Elements: []bpmn.Element{
				{ID: "sub-start", Type: "startEvent", Outgoing: bpmn.StringSlice{"sub-flow1"}},
				{ID: "sub-flow1", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub-start"}, Outgoing: bpmn.StringSlice{"sub-gw"}},
				{ID: "sub-gw", Type: "exclusiveGateway", Incoming: bpmn.StringSlice{"sub-flow1"}, Outgoing: bpmn.StringSlice{"sub-flow2", "sub-flow3"}},
				{ID: "sub-flow2", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub-gw"}, Outgoing: bpmn.StringSlice{"sub-taskA"}, Condition: "tool.config.branch == \"A\""},
				{ID: "sub-flow3", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub-gw"}, Outgoing: bpmn.StringSlice{"sub-taskB"}},
				{ID: "sub-taskA", Type: "userTask", Name: "A分支", Incoming: bpmn.StringSlice{"sub-flow2"}, Outgoing: bpmn.StringSlice{"sub-flow4"}},
				{ID: "sub-taskB", Type: "userTask", Name: "B分支", Incoming: bpmn.StringSlice{"sub-flow3"}, Outgoing: bpmn.StringSlice{"sub-flow5"}},
				{ID: "sub-flow4", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub-taskA"}, Outgoing: bpmn.StringSlice{"sub-end"}},
				{ID: "sub-flow5", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub-taskB"}, Outgoing: bpmn.StringSlice{"sub-end"}},
				{ID: "sub-end", Type: "endEvent", Incoming: bpmn.StringSlice{"sub-flow4", "sub-flow5"}},
			},
		},
		{ID: "flow6", Type: "sequenceFlow", Incoming: bpmn.StringSlice{"sub1"}, Outgoing: bpmn.StringSlice{"end"}},
		{ID: "end", Type: "endEvent", Incoming: bpmn.StringSlice{"flow6"}},
	})
	engine := NewEngine()
	inst := startInstance(t, engine, def, map[string]any{"branch": "A"})
	tasks := engine.GetPendingTasks(inst.ID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ElementID != "sub-taskA" {
		t.Errorf("expected sub-taskA, got %s", tasks[0].ElementID)
	}
}
