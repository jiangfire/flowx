package bpmn

import (
	"time"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"gopkg.in/yaml.v3"
)

// StringSlice 支持 YAML 中单个字符串或字符串数组
type StringSlice []string

func (s *StringSlice) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.SequenceNode {
		return value.Decode((*[]string)(s))
	}
	var single string
	if err := value.Decode(&single); err != nil {
		return err
	}
	*s = []string{single}
	return nil
}

// ProcessDefinition 流程定义（YAML 解析用，纯数据结构）
type ProcessDefinition struct {
	ID       string    `yaml:"id" json:"id"`
	Name     string    `yaml:"name" json:"name"`
	Version  int       `yaml:"version" json:"version"`
	Status   string    `yaml:"status" json:"status"` // draft/active/archived
	Elements []Element `yaml:"elements" json:"elements"`
}

// Element 流程元素
type Element struct {
	ID         string            `yaml:"id" json:"id"`
	Type       string            `yaml:"type" json:"type"` // startEvent, endEvent, userTask, serviceTask, scriptTask, exclusiveGateway, parallelGateway, inclusiveGateway, subProcess
	Name       string            `yaml:"name" json:"name"`
	Assignee   string            `yaml:"assignee" json:"assignee"`         // userTask: 审批人
	Condition  string            `yaml:"condition" json:"condition"`       // sequence flow condition
	Tool       string            `yaml:"tool" json:"tool"`                 // serviceTask: tool name
	Params     map[string]any    `yaml:"params" json:"params"`             // serviceTask: tool params
	Script     string            `yaml:"script" json:"script"`             // scriptTask: script expression
	FormFields []FormField       `yaml:"form_fields" json:"form_fields"`   // userTask: form fields
	Incoming   StringSlice       `yaml:"incoming" json:"incoming"`         // incoming flow IDs
	Outgoing   StringSlice       `yaml:"outgoing" json:"outgoing"`         // outgoing flow IDs
	Elements   []Element         `yaml:"elements" json:"elements"`         // subProcess: child elements
}

// FormField 表单字段定义
type FormField struct {
	ID       string   `yaml:"id" json:"id"`
	Label    string   `yaml:"label" json:"label"`
	Type     string   `yaml:"type" json:"type"`      // text/number/select/textarea
	Required bool     `yaml:"required" json:"required"`
	Options  []string `yaml:"options" json:"options"` // for select type
}

// ProcessInstance 流程实例
type ProcessInstance struct {
	base.BaseModel
	DefinitionID    string     `gorm:"size:26;index" json:"definition_id"`
	DefinitionYAML  string     `gorm:"type:text" json:"-"`
	Status          string     `gorm:"size:20;index" json:"status"` // running/suspended/completed/cancelled
	Variables       base.JSON  `gorm:"type:jsonb" json:"variables"`
	CurrentElements base.JSON  `gorm:"type:jsonb" json:"current_elements"`
	StartedBy       string     `gorm:"size:26" json:"started_by"`
	CompletedAt     *time.Time `json:"completed_at"`
}

// ProcessTask 流程任务（UserTask 产生的待办）
type ProcessTask struct {
	base.BaseModel
	InstanceID    string    `gorm:"size:26;index" json:"instance_id"`
	ElementID     string    `gorm:"size:26;index" json:"element_id"`
	Name          string    `gorm:"size:200" json:"name"`
	Assignee      string    `gorm:"size:26;index" json:"assignee"`
	Status        string    `gorm:"size:20;index" json:"status"` // pending/completed/cancelled
	FormFields    base.JSON `gorm:"type:jsonb" json:"form_fields"`
	SubmittedData base.JSON `gorm:"type:jsonb" json:"submitted_data"`
	CompletedBy   string    `gorm:"size:26" json:"completed_by"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// ExecutionHistory 执行历史
type ExecutionHistory struct {
	base.BaseModel
	InstanceID   string    `gorm:"size:26;index" json:"instance_id"`
	ElementID    string    `gorm:"size:26;index" json:"element_id"`
	ElementType  string    `gorm:"size:50" json:"element_type"`
	Action       string    `gorm:"size:20" json:"action"` // enter/leave/complete
	Variables    base.JSON `gorm:"type:jsonb" json:"variables"`
	Duration     int64     `json:"duration_ms"`
}

// TableName 表名
func (ProcessInstance) TableName() string   { return "process_instances" }
func (ProcessTask) TableName() string       { return "process_tasks" }
func (ExecutionHistory) TableName() string  { return "execution_histories" }
