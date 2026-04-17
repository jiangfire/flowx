package bpmn

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseProcess 解析 YAML 流程定义
func ParseProcess(data []byte) (*ProcessDefinition, error) {
	var def ProcessDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("解析流程定义失败: %w", err)
	}
	return &def, nil
}
