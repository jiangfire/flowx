package mcp

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/tool"
)

// TestToolRegistry_RegisterTool_Success 验证注册工具成功
func TestToolRegistry_RegisterTool_Success(t *testing.T) {
	reg := NewToolRegistry()

	domainTool := &tool.Tool{
		Name:        "altium_designer",
		Type:        "eda",
		Description: "Altium Designer EDA 工具",
		Status:      "active",
	}

	err := reg.RegisterTool(domainTool)
	if err != nil {
		t.Fatalf("注册工具失败: %v", err)
	}

	tools := reg.ListTools()
	if len(tools) != 1 {
		t.Fatalf("期望 1 个工具，实际为 %d", len(tools))
	}
	if tools[0].Name != "altium_designer" {
		t.Errorf("期望工具名称为 'altium_designer'，实际为 '%s'", tools[0].Name)
	}
}

// TestToolRegistry_RegisterTool_Duplicate 验证重复注册同名工具返回错误
func TestToolRegistry_RegisterTool_Duplicate(t *testing.T) {
	reg := NewToolRegistry()

	domainTool := &tool.Tool{
		Name:        "dup_tool",
		Type:        "eda",
		Description: "重复工具",
		Status:      "active",
	}

	err := reg.RegisterTool(domainTool)
	if err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}

	err = reg.RegisterTool(domainTool)
	if err == nil {
		t.Fatal("期望重复注册返回错误，实际为 nil")
	}
}

// TestToolRegistry_RegisterConnector_Success 验证注册连接器成功
func TestToolRegistry_RegisterConnector_Success(t *testing.T) {
	reg := NewToolRegistry()

	connector := &tool.Connector{
		Name:        "windchill",
		Type:        "plm",
		Description: "Windchill PLM 连接器",
		Endpoint:    "https://plm.example.com",
		Status:      "active",
	}

	err := reg.RegisterConnector(connector)
	if err != nil {
		t.Fatalf("注册连接器失败: %v", err)
	}

	tools := reg.ListTools()
	if len(tools) != 1 {
		t.Fatalf("期望 1 个工具，实际为 %d", len(tools))
	}
	if tools[0].Name != "connector_windchill" {
		t.Errorf("期望工具名称为 'connector_windchill'，实际为 '%s'", tools[0].Name)
	}
}

// TestToolRegistry_ListTools 验证返回所有已注册工具
func TestToolRegistry_ListTools(t *testing.T) {
	reg := NewToolRegistry()

	// 注册多个工具
	reg.RegisterTool(&tool.Tool{Name: "tool1", Type: "eda", Status: "active"})
	reg.RegisterTool(&tool.Tool{Name: "tool2", Type: "cae", Status: "active"})
	reg.RegisterConnector(&tool.Connector{Name: "conn1", Type: "plm", Status: "active"})

	tools := reg.ListTools()
	if len(tools) != 3 {
		t.Fatalf("期望 3 个工具，实际为 %d", len(tools))
	}
}

// TestToolRegistry_GetTool_Exists 验证获取已存在的工具
func TestToolRegistry_GetTool_Exists(t *testing.T) {
	reg := NewToolRegistry()

	reg.RegisterTool(&tool.Tool{
		Name:        "my_tool",
		Type:        "eda",
		Description: "我的工具",
		Status:      "active",
	})

	def, err := reg.GetTool("my_tool")
	if err != nil {
		t.Fatalf("获取工具失败: %v", err)
	}
	if def.Name != "my_tool" {
		t.Errorf("期望名称为 'my_tool'，实际为 '%s'", def.Name)
	}
	if def.Description != "我的工具" {
		t.Errorf("期望描述为 '我的工具'，实际为 '%s'", def.Description)
	}
}

// TestToolRegistry_GetTool_NotExists 验证获取不存在的工具返回错误
func TestToolRegistry_GetTool_NotExists(t *testing.T) {
	reg := NewToolRegistry()

	_, err := reg.GetTool("nonexistent")
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
}

// TestToolRegistry_CallTool_Success 验证调用 handler 返回结果
func TestToolRegistry_CallTool_Success(t *testing.T) {
	reg := NewToolRegistry()

	// 注册带自定义 handler 的工具
	reg.RegisterTool(&tool.Tool{
		Name:        "echo",
		Type:        "custom",
		Description: "回显工具",
		Status:      "active",
	})

	// 设置自定义 handler
	def, _ := reg.GetTool("echo")
	def.Handler = func(ctx context.Context, args map[string]any) (any, error) {
		return args["message"], nil
	}

	result, err := reg.CallTool(context.Background(), "echo", map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("调用工具失败: %v", err)
	}
	if result != "hello" {
		t.Errorf("期望返回 'hello'，实际为 '%v'", result)
	}
}

// TestToolRegistry_CallTool_NotExists 验证调用不存在的工具返回错误
func TestToolRegistry_CallTool_NotExists(t *testing.T) {
	reg := NewToolRegistry()

	_, err := reg.CallTool(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
}

// TestToolRegistry_RegisterToolWithHandler 验证注册带自定义 handler 的工具
func TestToolRegistry_RegisterToolWithHandler(t *testing.T) {
	reg := NewToolRegistry()

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return "custom_result", nil
	}

	err := reg.RegisterToolWithHandler(&tool.Tool{
		Name:        "custom_tool",
		Type:        "custom",
		Description: "自定义工具",
		Status:      "active",
	}, handler)
	if err != nil {
		t.Fatalf("注册工具失败: %v", err)
	}

	result, err := reg.CallTool(context.Background(), "custom_tool", nil)
	if err != nil {
		t.Fatalf("调用工具失败: %v", err)
	}
	if result != "custom_result" {
		t.Errorf("期望返回 'custom_result'，实际为 '%v'", result)
	}
}
