package bpmn

import (
	"fmt"
	"sync"
	"time"

	"git.neolidy.top/neo/flowx/internal/application/datagov/expression"
	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
	"git.neolidy.top/neo/flowx/internal/domain/base"
)

// taskInfo extends ProcessTask with internal tracking fields.
type taskInfo struct {
	task         *bpmn.ProcessTask
	subProcessID string // non-empty if inside a subprocess
}

// instanceState holds the runtime state of a process instance.
type instanceState struct {
	instance        *bpmn.ProcessInstance
	definition      *bpmn.ProcessDefinition
	tasks           []*taskInfo
	history         []*bpmn.ExecutionHistory
	joinReceived    map[string]int // gateway ID -> number of tokens received
	inclusiveTokens map[string]int // inclusive gateway ID -> number of tokens dispatched at fork
}

// Engine is an in-memory BPMN process execution engine.
type Engine struct {
	mu        sync.RWMutex
	instances map[string]*instanceState
	taskIndex map[string]string // taskID -> instanceID
}

// NewEngine creates a new Engine.
func NewEngine() *Engine {
	return &Engine{
		instances: make(map[string]*instanceState),
		taskIndex: make(map[string]string),
	}
}

// Start creates and starts a new process instance, executing until the first wait state.
func (e *Engine) Start(def *bpmn.ProcessDefinition, tenantID, startedBy string, variables map[string]any) *bpmn.ProcessInstance {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	inst := &bpmn.ProcessInstance{
		BaseModel: base.BaseModel{
			ID:        base.GenerateUUID(),
			TenantID:  tenantID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		DefinitionID:    def.ID,
		Status:          "running",
		Variables:       variables,
		CurrentElements: base.JSON{},
		StartedBy:       startedBy,
	}

	state := &instanceState{
		instance:        inst,
		definition:      def,
		tasks:           nil,
		history:         nil,
		joinReceived:    make(map[string]int),
		inclusiveTokens: make(map[string]int),
	}

	e.instances[inst.ID] = state

	// Find start event and begin execution
	startElem := findElement(def.Elements, func(el bpmn.Element) bool {
		return el.Type == "startEvent"
	})
	if startElem == nil {
		return inst
	}

	// Enter start event
	state.addHistory(inst.ID, startElem.ID, startElem.Type, "enter", variables)
	state.addHistory(inst.ID, startElem.ID, startElem.Type, "leave", variables)

	// Advance from start event
	e.advanceFromElement(state, *startElem, "")

	return inst
}

// CompleteTask completes a user task and advances the process.
func (e *Engine) CompleteTask(instanceID, taskID, completedBy string, submittedData map[string]any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.instances[instanceID]
	if !ok {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	if state.instance.Status != "running" {
		return fmt.Errorf("instance %s is not running (status: %s)", instanceID, state.instance.Status)
	}

	// Find the task
	var ti *taskInfo
	var taskIdx int
	for i, t := range state.tasks {
		if t.task.ID == taskID && t.task.Status == "pending" {
			ti = t
			taskIdx = i
			break
		}
	}
	if ti == nil {
		return fmt.Errorf("task %s not found or not pending", taskID)
	}

	task := ti.task

	// Mark task as completed
	now := time.Now()
	task.Status = "completed"
	task.CompletedBy = completedBy
	task.CompletedAt = &now
	if submittedData != nil {
		task.SubmittedData = submittedData
	}

	// Merge submitted data into instance variables
	if state.instance.Variables == nil {
		state.instance.Variables = make(map[string]any)
	}
	for k, v := range submittedData {
		state.instance.Variables[k] = v
	}

	// Remove task from pending
	state.tasks = append(state.tasks[:taskIdx], state.tasks[taskIdx+1:]...)
	delete(e.taskIndex, taskID)

	// Add history for task completion
	state.addHistory(instanceID, task.ElementID, "userTask", "complete", state.instance.Variables)

	// Find the element definition (search in subprocess if applicable)
	var elem *bpmn.Element
	if ti.subProcessID != "" {
		subElem := findElement(state.definition.Elements, func(el bpmn.Element) bool {
			return el.ID == ti.subProcessID
		})
		if subElem != nil {
			elem = findElement(subElem.Elements, func(el bpmn.Element) bool {
				return el.ID == task.ElementID
			})
		}
	} else {
		elem = findElement(state.definition.Elements, func(el bpmn.Element) bool {
			return el.ID == task.ElementID
		})
	}
	if elem == nil {
		return nil
	}

	// Advance from the completed task
	if ti.subProcessID != "" {
		subElem := findElement(state.definition.Elements, func(el bpmn.Element) bool {
			return el.ID == ti.subProcessID
		})
		if subElem != nil {
			e.advanceFromSubProcessElement(state, *subElem, *elem)
		}
	} else {
		e.advanceFromElement(state, *elem, ti.subProcessID)
	}

	return nil
}

// GetInstance returns the current state of a process instance.
func (e *Engine) GetInstance(instanceID string) *bpmn.ProcessInstance {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.instances[instanceID]
	if !ok {
		return nil
	}
	return state.instance
}

// GetPendingTasks returns all pending tasks for a process instance.
func (e *Engine) GetPendingTasks(instanceID string) []*bpmn.ProcessTask {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.instances[instanceID]
	if !ok {
		return nil
	}

	result := make([]*bpmn.ProcessTask, len(state.tasks))
	for i, ti := range state.tasks {
		result[i] = ti.task
	}
	return result
}

// GetHistory returns execution history for a process instance.
func (e *Engine) GetHistory(instanceID string) []*bpmn.ExecutionHistory {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.instances[instanceID]
	if !ok {
		return nil
	}

	result := make([]*bpmn.ExecutionHistory, len(state.history))
	copy(result, state.history)
	return result
}

// Suspend suspends a running process instance.
func (e *Engine) Suspend(instanceID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.instances[instanceID]
	if !ok || state.instance.Status != "running" {
		return
	}
	state.instance.Status = "suspended"
	state.instance.UpdatedAt = time.Now()
}

// Resume resumes a suspended process instance.
func (e *Engine) Resume(instanceID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.instances[instanceID]
	if !ok || state.instance.Status != "suspended" {
		return
	}
	state.instance.Status = "running"
	state.instance.UpdatedAt = time.Now()
}

// Cancel cancels a running or suspended process instance.
func (e *Engine) Cancel(instanceID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	state, ok := e.instances[instanceID]
	if !ok {
		return
	}
	if state.instance.Status != "running" && state.instance.Status != "suspended" {
		return
	}

	// Cancel all pending tasks
	for _, ti := range state.tasks {
		ti.task.Status = "cancelled"
		delete(e.taskIndex, ti.task.ID)
	}
	state.tasks = nil

	state.instance.Status = "cancelled"
	state.instance.UpdatedAt = time.Now()
}

// advanceFromElement advances execution from a given element.
func (e *Engine) advanceFromElement(state *instanceState, elem bpmn.Element, subprocessID string) {
	if len(elem.Outgoing) == 0 {
		return
	}

	for _, outgoingID := range elem.Outgoing {
		outgoingElem := findElement(state.definition.Elements, func(el bpmn.Element) bool {
			return el.ID == outgoingID
		})
		if outgoingElem == nil {
			continue
		}

		if outgoingElem.Type == "sequenceFlow" {
			for _, targetID := range outgoingElem.Outgoing {
				targetElem := findElement(state.definition.Elements, func(el bpmn.Element) bool {
					return el.ID == targetID
				})
				if targetElem != nil {
					e.executeElement(state, *targetElem, subprocessID)
				}
			}
		} else {
			e.executeElement(state, *outgoingElem, subprocessID)
		}
	}
}

// executeElement executes a single element in the process flow.
func (e *Engine) executeElement(state *instanceState, elem bpmn.Element, subprocessID string) {
	switch elem.Type {
	case "endEvent":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		if subprocessID != "" {
			// End event inside subprocess: continue with subprocess outgoing in main flow
			subElem := findElement(state.definition.Elements, func(el bpmn.Element) bool {
				return el.ID == subprocessID
			})
			if subElem != nil {
				e.advanceFromElement(state, *subElem, "")
			}
		} else {
			// End event in main flow: complete the instance
			now := time.Now()
			state.instance.Status = "completed"
			state.instance.CompletedAt = &now
			state.instance.UpdatedAt = now
		}

	case "userTask":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.createPendingTask(state, elem, subprocessID)

	case "serviceTask":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		if elem.Params != nil {
			if state.instance.Variables == nil {
				state.instance.Variables = make(map[string]any)
			}
			for k, v := range elem.Params {
				state.instance.Variables[k] = v
			}
		}
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "leave", state.instance.Variables)
		e.advanceFromElement(state, elem, subprocessID)

	case "scriptTask":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		// scriptTask 自动执行：将脚本结果合并到变量中（第一版仅记录执行，不实际运行脚本）
		if elem.Script != "" {
			if state.instance.Variables == nil {
				state.instance.Variables = make(map[string]any)
			}
			state.instance.Variables["_script_"+elem.ID] = "executed"
		}
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "leave", state.instance.Variables)
		e.advanceFromElement(state, elem, subprocessID)

	case "exclusiveGateway":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "leave", state.instance.Variables)
		e.executeExclusiveGateway(state, elem, subprocessID)

	case "parallelGateway":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.executeParallelGateway(state, elem, subprocessID)

	case "inclusiveGateway":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.executeInclusiveGateway(state, elem, subprocessID)

	case "subProcess", "embeddedSubProcess":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.executeSubProcess(state, elem)

	case "startEvent":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "leave", state.instance.Variables)
		e.advanceFromElement(state, elem, subprocessID)
	}
}

// executeExclusiveGateway evaluates conditions and takes the first matching branch.
func (e *Engine) executeExclusiveGateway(state *instanceState, gw bpmn.Element, subprocessID string) {
	var defaultFlow *bpmn.Element

	for _, outgoingID := range gw.Outgoing {
		flow := findElement(state.definition.Elements, func(el bpmn.Element) bool {
			return el.ID == outgoingID
		})
		if flow == nil {
			continue
		}

		if flow.Condition != "" {
			if e.evaluateCondition(flow.Condition, state.instance.Variables) {
				for _, targetID := range flow.Outgoing {
					target := findElement(state.definition.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeElement(state, *target, subprocessID)
					}
				}
				return
			}
		} else {
			defaultFlow = flow
		}
	}

	if defaultFlow != nil {
		for _, targetID := range defaultFlow.Outgoing {
			target := findElement(state.definition.Elements, func(el bpmn.Element) bool {
				return el.ID == targetID
			})
			if target != nil {
				e.executeElement(state, *target, subprocessID)
			}
		}
	}
}

// executeParallelGateway handles both fork and join behavior.
func (e *Engine) executeParallelGateway(state *instanceState, gw bpmn.Element, subprocessID string) {
	if isFork(gw) {
		for _, outgoingID := range gw.Outgoing {
			flow := findElement(state.definition.Elements, func(el bpmn.Element) bool {
				return el.ID == outgoingID
			})
			if flow == nil {
				continue
			}
			for _, targetID := range flow.Outgoing {
				target := findElement(state.definition.Elements, func(el bpmn.Element) bool {
					return el.ID == targetID
				})
				if target != nil {
					e.executeElement(state, *target, subprocessID)
				}
			}
		}
	} else {
		state.joinReceived[gw.ID]++
		expected := len(gw.Incoming)

		if state.joinReceived[gw.ID] >= expected {
			state.addHistory(state.instance.ID, gw.ID, gw.Type, "leave", state.instance.Variables)
			for _, outgoingID := range gw.Outgoing {
				flow := findElement(state.definition.Elements, func(el bpmn.Element) bool {
					return el.ID == outgoingID
				})
				if flow == nil {
					continue
				}
				for _, targetID := range flow.Outgoing {
					target := findElement(state.definition.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeElement(state, *target, subprocessID)
					}
				}
			}
		}
	}
}

// executeInclusiveGateway handles both fork and join behavior for inclusive gateways.
func (e *Engine) executeInclusiveGateway(state *instanceState, gw bpmn.Element, subprocessID string) {
	if isFork(gw) {
		dispatched := 0
		for _, outgoingID := range gw.Outgoing {
			flow := findElement(state.definition.Elements, func(el bpmn.Element) bool {
				return el.ID == outgoingID
			})
			if flow == nil {
				continue
			}

			if flow.Condition == "" || e.evaluateCondition(flow.Condition, state.instance.Variables) {
				dispatched++
				for _, targetID := range flow.Outgoing {
					target := findElement(state.definition.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeElement(state, *target, subprocessID)
					}
				}
			}
		}
		// 记录 fork 派发的 token 数，供 join 使用
		state.inclusiveTokens[gw.ID] = dispatched
	} else {
		state.joinReceived[gw.ID]++
		// inclusive join: 只等待实际激活的分支到达
		expected := state.inclusiveTokens[gw.ID]
		if expected == 0 {
			expected = len(gw.Incoming) // 兜底：如果没有 fork 记录，使用所有入边
		}

		if state.joinReceived[gw.ID] >= expected {
			state.addHistory(state.instance.ID, gw.ID, gw.Type, "leave", state.instance.Variables)
			for _, outgoingID := range gw.Outgoing {
				flow := findElement(state.definition.Elements, func(el bpmn.Element) bool {
					return el.ID == outgoingID
				})
				if flow == nil {
					continue
				}
				for _, targetID := range flow.Outgoing {
					target := findElement(state.definition.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeElement(state, *target, subprocessID)
					}
				}
			}
		}
	}
}

// executeSubProcess enters a subprocess and starts executing from its start event.
func (e *Engine) executeSubProcess(state *instanceState, subElem bpmn.Element) {
	startElem := findElement(subElem.Elements, func(el bpmn.Element) bool {
		return el.Type == "startEvent"
	})
	if startElem == nil {
		return
	}

	state.addHistory(state.instance.ID, startElem.ID, startElem.Type, "enter", state.instance.Variables)
	state.addHistory(state.instance.ID, startElem.ID, startElem.Type, "leave", state.instance.Variables)

	e.advanceFromSubProcessElement(state, subElem, *startElem)
}

// advanceFromSubProcessElement advances execution within a subprocess.
func (e *Engine) advanceFromSubProcessElement(state *instanceState, subElem bpmn.Element, elem bpmn.Element) {
	if len(elem.Outgoing) == 0 {
		return
	}

	for _, outgoingID := range elem.Outgoing {
		outgoingElem := findElement(subElem.Elements, func(el bpmn.Element) bool {
			return el.ID == outgoingID
		})
		if outgoingElem == nil {
			continue
		}

		if outgoingElem.Type == "sequenceFlow" {
			for _, targetID := range outgoingElem.Outgoing {
				targetElem := findElement(subElem.Elements, func(el bpmn.Element) bool {
					return el.ID == targetID
				})
				if targetElem != nil {
					e.executeSubProcessElement(state, subElem, *targetElem)
				}
			}
		} else {
			e.executeSubProcessElement(state, subElem, *outgoingElem)
		}
	}
}

// executeSubProcessElement executes an element within a subprocess.
func (e *Engine) executeSubProcessElement(state *instanceState, subElem bpmn.Element, elem bpmn.Element) {
	switch elem.Type {
	case "endEvent":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		state.addHistory(state.instance.ID, subElem.ID, subElem.Type, "leave", state.instance.Variables)
		e.advanceFromElement(state, subElem, "")

	case "userTask":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.createPendingTask(state, elem, subElem.ID)

	case "serviceTask":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		if elem.Params != nil {
			if state.instance.Variables == nil {
				state.instance.Variables = make(map[string]any)
			}
			for k, v := range elem.Params {
				state.instance.Variables[k] = v
			}
		}
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "leave", state.instance.Variables)
		e.advanceFromSubProcessElement(state, subElem, elem)

	case "exclusiveGateway":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "leave", state.instance.Variables)
		e.executeExclusiveGatewaySubProcess(state, subElem, elem)

	case "parallelGateway":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.executeParallelGatewaySubProcess(state, subElem, elem)

	case "inclusiveGateway":
		state.addHistory(state.instance.ID, elem.ID, elem.Type, "enter", state.instance.Variables)
		e.executeInclusiveGatewaySubProcess(state, subElem, elem)
	}
}

// executeExclusiveGatewaySubProcess handles exclusive gateway within subprocess.
func (e *Engine) executeExclusiveGatewaySubProcess(state *instanceState, subElem bpmn.Element, gw bpmn.Element) {
	var defaultFlow *bpmn.Element

	for _, outgoingID := range gw.Outgoing {
		flow := findElement(subElem.Elements, func(el bpmn.Element) bool {
			return el.ID == outgoingID
		})
		if flow == nil {
			continue
		}

		if flow.Condition != "" {
			if e.evaluateCondition(flow.Condition, state.instance.Variables) {
				for _, targetID := range flow.Outgoing {
					target := findElement(subElem.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeSubProcessElement(state, subElem, *target)
					}
				}
				return
			}
		} else {
			defaultFlow = flow
		}
	}

	if defaultFlow != nil {
		for _, targetID := range defaultFlow.Outgoing {
			target := findElement(subElem.Elements, func(el bpmn.Element) bool {
				return el.ID == targetID
			})
			if target != nil {
				e.executeSubProcessElement(state, subElem, *target)
			}
		}
	}
}

// executeParallelGatewaySubProcess handles parallel gateway within subprocess.
func (e *Engine) executeParallelGatewaySubProcess(state *instanceState, subElem bpmn.Element, gw bpmn.Element) {
	if isFork(gw) {
		for _, outgoingID := range gw.Outgoing {
			flow := findElement(subElem.Elements, func(el bpmn.Element) bool {
				return el.ID == outgoingID
			})
			if flow == nil {
				continue
			}
			for _, targetID := range flow.Outgoing {
				target := findElement(subElem.Elements, func(el bpmn.Element) bool {
					return el.ID == targetID
				})
				if target != nil {
					e.executeSubProcessElement(state, subElem, *target)
				}
			}
		}
	} else {
		state.joinReceived[gw.ID]++
		expected := len(gw.Incoming)

		if state.joinReceived[gw.ID] >= expected {
			state.addHistory(state.instance.ID, gw.ID, gw.Type, "leave", state.instance.Variables)
			for _, outgoingID := range gw.Outgoing {
				flow := findElement(subElem.Elements, func(el bpmn.Element) bool {
					return el.ID == outgoingID
				})
				if flow == nil {
					continue
				}
				for _, targetID := range flow.Outgoing {
					target := findElement(subElem.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeSubProcessElement(state, subElem, *target)
					}
				}
			}
		}
	}
}

// executeInclusiveGatewaySubProcess handles inclusive gateway within subprocess.
func (e *Engine) executeInclusiveGatewaySubProcess(state *instanceState, subElem bpmn.Element, gw bpmn.Element) {
	if isFork(gw) {
		dispatched := 0
		for _, outgoingID := range gw.Outgoing {
			flow := findElement(subElem.Elements, func(el bpmn.Element) bool {
				return el.ID == outgoingID
			})
			if flow == nil {
				continue
			}

			if flow.Condition == "" || e.evaluateCondition(flow.Condition, state.instance.Variables) {
				dispatched++
				for _, targetID := range flow.Outgoing {
					target := findElement(subElem.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeSubProcessElement(state, subElem, *target)
					}
				}
			}
		}
		state.inclusiveTokens[gw.ID] = dispatched
	} else {
		state.joinReceived[gw.ID]++
		expected := state.inclusiveTokens[gw.ID]
		if expected == 0 {
			expected = len(gw.Incoming)
		}

		if state.joinReceived[gw.ID] >= expected {
			state.addHistory(state.instance.ID, gw.ID, gw.Type, "leave", state.instance.Variables)
			for _, outgoingID := range gw.Outgoing {
				flow := findElement(subElem.Elements, func(el bpmn.Element) bool {
					return el.ID == outgoingID
				})
				if flow == nil {
					continue
				}
				for _, targetID := range flow.Outgoing {
					target := findElement(subElem.Elements, func(el bpmn.Element) bool {
						return el.ID == targetID
					})
					if target != nil {
						e.executeSubProcessElement(state, subElem, *target)
					}
				}
			}
		}
	}
}

// createPendingTask creates a pending user task.
func (e *Engine) createPendingTask(state *instanceState, elem bpmn.Element, subprocessID string) {
	now := time.Now()
	task := &bpmn.ProcessTask{
		BaseModel: base.BaseModel{
			ID:        base.GenerateUUID(),
			TenantID:  state.instance.TenantID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		InstanceID:    state.instance.ID,
		ElementID:     elem.ID,
		Name:          elem.Name,
		Assignee:      elem.Assignee,
		Status:        "pending",
		SubmittedData: nil,
	}

	state.tasks = append(state.tasks, &taskInfo{
		task:         task,
		subProcessID: subprocessID,
	})
	e.taskIndex[task.ID] = state.instance.ID
}

// evaluateCondition evaluates a condition expression against the given variables.
func (e *Engine) evaluateCondition(condition string, variables map[string]any) bool {
	ctx := &expression.Context{
		Tool: expression.ToolContext{
			Config: variables,
		},
	}
	ok, err := expression.Evaluate(ctx, condition)
	return err == nil && ok
}

// addHistory adds an execution history entry.
func (s *instanceState) addHistory(instanceID, elementID, elementType, action string, variables map[string]any) {
	now := time.Now()
	h := &bpmn.ExecutionHistory{
		BaseModel: base.BaseModel{
			ID:        base.GenerateUUID(),
			TenantID:  "",
			CreatedAt: now,
			UpdatedAt: now,
		},
		InstanceID:  instanceID,
		ElementID:   elementID,
		ElementType: elementType,
		Action:      action,
		Variables:   variables,
	}
	s.history = append(s.history, h)
}

// findElement searches for an element in a slice matching the predicate.
func findElement(elements []bpmn.Element, pred func(bpmn.Element) bool) *bpmn.Element {
	for i := range elements {
		if pred(elements[i]) {
			return &elements[i]
		}
	}
	return nil
}

// isFork determines if a gateway is a fork (more outgoing than incoming) or join.
// 当 Outgoing >= Incoming 时视为 fork（处理既是 fork 又是 join 的网关）
func isFork(gw bpmn.Element) bool {
	return len(gw.Outgoing) >= len(gw.Incoming)
}
