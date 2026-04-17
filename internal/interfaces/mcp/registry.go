package mcp

import (
	"context"
	"fmt"
	"sync"

	"git.neolidy.top/neo/flowx/internal/domain/tool"
)

// MCPToolDefinition MCP 工具定义
type MCPToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     ToolHandler    `json:"-"`
}

// ToolHandler 工具处理函数类型
type ToolHandler func(ctx context.Context, args map[string]any) (any, error)

// ToolCaller 工具调用者接口（Agent 消费者）
type ToolCaller interface {
	// ListTools 返回所有已注册的 MCP 工具定义
	ListTools() []MCPToolDefinition

	// GetTool 获取指定名称的 MCP 工具定义
	GetTool(name string) (*MCPToolDefinition, error)

	// CallTool 调用指定工具
	CallTool(ctx context.Context, name string, args map[string]any) (any, error)
}

// ToolRegistry 工具注册器接口，将 FlowX 内部工具/连接器转换为 MCP tool 定义
type ToolRegistry interface {
	ToolCaller

	// RegisterTool 注册领域工具（使用默认 handler）
	RegisterTool(t *tool.Tool) error

	// RegisterToolWithHandler 注册领域工具（使用自定义 handler）
	RegisterToolWithHandler(t *tool.Tool, handler ToolHandler) error

	// RegisterConnector 注册连接器
	RegisterConnector(c *tool.Connector) error
}

// toolRegistry 工具注册器实现
type toolRegistry struct {
	tools map[string]*MCPToolDefinition
	mu    sync.RWMutex
}

// NewToolRegistry 创建工具注册器实例
func NewToolRegistry() ToolRegistry {
	return &toolRegistry{
		tools: make(map[string]*MCPToolDefinition),
	}
}

// RegisterTool 注册领域工具（使用默认 handler）
func (r *toolRegistry) RegisterTool(t *tool.Tool) error {
	return r.RegisterToolWithHandler(t, func(ctx context.Context, args map[string]any) (any, error) {
		return map[string]any{
			"tool_name": t.Name,
			"tool_type": t.Type,
			"status":    "executed",
		}, nil
	})
}

// RegisterToolWithHandler 注册领域工具（使用自定义 handler）
func (r *toolRegistry) RegisterToolWithHandler(t *tool.Tool, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' 已注册", name)
	}

	r.tools[name] = &MCPToolDefinition{
		Name:        name,
		Description: t.Description,
		InputSchema: buildInputSchema(t),
		Handler:     handler,
	}
	return nil
}

// RegisterConnector 注册连接器（转换为 MCP tool）
func (r *toolRegistry) RegisterConnector(c *tool.Connector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := fmt.Sprintf("connector_%s", c.Name)
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("connector tool '%s' 已注册", name)
	}

	r.tools[name] = &MCPToolDefinition{
		Name:        name,
		Description: fmt.Sprintf("连接器: %s (%s)", c.Name, c.Type),
		InputSchema: buildConnectorInputSchema(c),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			return map[string]any{
				"connector_name": c.Name,
				"connector_type": c.Type,
				"endpoint":       c.Endpoint,
				"status":         "connected",
			}, nil
		},
	}
	return nil
}

// ListTools 返回所有已注册的 MCP 工具定义
func (r *toolRegistry) ListTools() []MCPToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]MCPToolDefinition, 0, len(r.tools))
	for _, def := range r.tools {
		defs = append(defs, *def)
	}
	return defs
}

// GetTool 获取指定名称的 MCP 工具定义
func (r *toolRegistry) GetTool(name string) (*MCPToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' 未注册", name)
	}
	return def, nil
}

// CallTool 调用指定工具
func (r *toolRegistry) CallTool(ctx context.Context, name string, args map[string]any) (any, error) {
	r.mu.RLock()
	def, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool '%s' 未注册", name)
	}

	return def.Handler(ctx, args)
}

// buildInputSchema 根据工具配置构建 JSON Schema
func buildInputSchema(t *tool.Tool) map[string]any {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tool_id": map[string]any{
				"type":        "string",
				"description": "工具 ID",
			},
			"action": map[string]any{
				"type":        "string",
				"description": "执行动作",
				"enum":        []string{"execute", "query", "configure"},
			},
		},
		"required": []string{"action"},
	}

	// 如果工具配置中有额外参数，合并到 schema
	if t.Config != nil {
		if params, ok := t.Config["parameters"]; ok {
			if paramMap, ok := params.(map[string]any); ok {
				props := schema["properties"].(map[string]any)
				for k, v := range paramMap {
					props[k] = v
				}
			}
		}
	}

	return schema
}

// buildConnectorInputSchema 根据连接器配置构建 JSON Schema
func buildConnectorInputSchema(c *tool.Connector) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connector_id": map[string]any{
				"type":        "string",
				"description": "连接器 ID",
			},
			"action": map[string]any{
				"type":        "string",
				"description": "执行动作",
				"enum":        []string{"connect", "disconnect", "query", "execute"},
			},
			"endpoint": map[string]any{
				"type":        "string",
				"description": "连接器端点",
			},
		},
		"required": []string{"action"},
	}
}
