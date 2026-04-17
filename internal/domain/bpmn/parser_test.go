package bpmn

import "testing"

func TestParseProcess_SimpleSequence(t *testing.T) {
	yaml := `
id: simple-process
name: 简单流程
elements:
  - id: start
    type: startEvent
    outgoing: task1
  - id: task1
    type: userTask
    name: 审批任务
    assignee: admin
    outgoing: end
  - id: end
    type: endEvent
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if def.ID != "simple-process" {
		t.Errorf("expected id=simple-process, got %s", def.ID)
	}
	if len(def.Elements) != 3 {
		t.Errorf("expected 3 elements, got %d", len(def.Elements))
	}
	if def.Elements[1].Type != "userTask" {
		t.Errorf("expected userTask, got %s", def.Elements[1].Type)
	}
}

func TestParseProcess_ExclusiveGateway(t *testing.T) {
	yaml := `
id: gateway-process
name: 排他网关流程
elements:
  - id: start
    type: startEvent
    outgoing: gw1
  - id: gw1
    type: exclusiveGateway
    name: 金额判断
    outgoing: [task1, task2]
  - id: task1
    type: userTask
    name: 大额审批
    condition: amount > 10000
    incoming: gw1
    outgoing: end
  - id: task2
    type: userTask
    name: 小额审批
    condition: amount <= 10000
    incoming: gw1
    outgoing: end
  - id: end
    type: endEvent
    incoming: [task1, task2]
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	// Find gateway
	var gw *Element
	for _, e := range def.Elements {
		if e.ID == "gw1" {
			p := e
			gw = &p
		}
	}
	if gw == nil {
		t.Fatal("gateway not found")
	}
	if len(gw.Outgoing) != 2 {
		t.Errorf("expected 2 outgoing, got %d", len(gw.Outgoing))
	}
}

func TestParseProcess_ParallelGateway(t *testing.T) {
	yaml := `
id: parallel-process
elements:
  - id: start
    type: startEvent
    outgoing: gw1
  - id: gw1
    type: parallelGateway
    outgoing: [task1, task2]
  - id: task1
    type: serviceTask
    name: 发送邮件
    incoming: gw1
    outgoing: gw2
  - id: task2
    type: serviceTask
    name: 发送通知
    incoming: gw1
    outgoing: gw2
  - id: gw2
    type: parallelGateway
    incoming: [task1, task2]
    outgoing: end
  - id: end
    type: endEvent
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Elements) != 6 {
		t.Errorf("expected 6 elements, got %d", len(def.Elements))
	}
}

func TestParseProcess_SubProcess(t *testing.T) {
	yaml := `
id: sub-process-test
elements:
  - id: start
    type: startEvent
    outgoing: sub1
  - id: sub1
    type: subProcess
    name: 子流程
    outgoing: end
    elements:
      - id: sub-start
        type: startEvent
        outgoing: sub-task
      - id: sub-task
        type: userTask
        name: 子任务
        outgoing: sub-end
      - id: sub-end
        type: endEvent
  - id: end
    type: endEvent
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	var sub *Element
	for _, e := range def.Elements {
		if e.ID == "sub1" {
			p := e
			sub = &p
		}
	}
	if sub == nil {
		t.Fatal("subprocess not found")
	}
	if len(sub.Elements) != 3 {
		t.Errorf("expected 3 sub-elements, got %d", len(sub.Elements))
	}
}

func TestParseProcess_InvalidYAML(t *testing.T) {
	_, err := ParseProcess([]byte("invalid: [yaml: broken"))
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestParseProcess_EmptyElements(t *testing.T) {
	yaml := `
id: empty
name: 空流程
elements: []
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Elements) != 0 {
		t.Errorf("expected 0 elements")
	}
}

func TestParseProcess_ServiceTask(t *testing.T) {
	yaml := `
id: svc-test
elements:
  - id: start
    type: startEvent
    outgoing: svc1
  - id: svc1
    type: serviceTask
    name: 调用工具
    tool: data_export
    params:
      format: csv
      target: s3
    incoming: start
    outgoing: end
  - id: end
    type: endEvent
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	var svc *Element
	for _, e := range def.Elements {
		if e.ID == "svc1" {
			p := e
			svc = &p
		}
	}
	if svc == nil {
		t.Fatal("service task not found")
	}
	if svc.Tool != "data_export" {
		t.Errorf("expected tool=data_export, got %s", svc.Tool)
	}
}

func TestParseProcess_ScriptTask(t *testing.T) {
	yaml := `
id: script-test
elements:
  - id: start
    type: startEvent
    outgoing: sc1
  - id: sc1
    type: scriptTask
    name: 设置变量
    script: "result = amount * 0.9"
    incoming: start
    outgoing: end
  - id: end
    type: endEvent
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	var sc *Element
	for _, e := range def.Elements {
		if e.ID == "sc1" {
			p := e
			sc = &p
		}
	}
	if sc == nil {
		t.Fatal("script task not found")
	}
	if sc.Script != "result = amount * 0.9" {
		t.Errorf("unexpected script: %s", sc.Script)
	}
}

func TestParseProcess_InclusiveGateway(t *testing.T) {
	yaml := `
id: inclusive-test
elements:
  - id: start
    type: startEvent
    outgoing: gw1
  - id: gw1
    type: inclusiveGateway
    outgoing: [task1, task2, task3]
  - id: task1
    type: userTask
    condition: amount > 5000
    incoming: gw1
    outgoing: gw2
  - id: task2
    type: userTask
    condition: role == "manager"
    incoming: gw1
    outgoing: gw2
  - id: task3
    type: userTask
    incoming: gw1
    outgoing: gw2
  - id: gw2
    type: inclusiveGateway
    incoming: [task1, task2, task3]
    outgoing: end
  - id: end
    type: endEvent
`
	def, err := ParseProcess([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Elements) != 7 {
		t.Errorf("expected 7 elements, got %d", len(def.Elements))
	}
}
