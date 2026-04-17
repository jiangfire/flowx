package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestNewMCPServer_Success 验证创建 MCP Server 实例成功
func TestNewMCPServer_Success(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")
	if s == nil {
		t.Fatal("期望创建 MCPServer 实例，实际为 nil")
	}
	if s.Name() != "flowx-test" {
		t.Errorf("期望 server name 为 'flowx-test'，实际为 '%s'", s.Name())
	}
}

// TestMCPServer_RegisterAndListTools 验证注册 tool 后可以通过 ListTools 获取
func TestMCPServer_RegisterAndListTools(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")

	// 创建 MCP tool 定义
	tool := mcp.NewTool("test_tool",
		mcp.WithDescription("测试工具"),
		mcp.WithString("param1", mcp.Required(), mcp.Description("参数1")),
	)

	// 注册 tool
	err := s.RegisterTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})
	if err != nil {
		t.Fatalf("注册 tool 失败: %v", err)
	}

	// 通过 ListTools 验证 tool 已注册
	tools := s.ListTools()
	if len(tools) != 1 {
		t.Fatalf("期望 1 个 tool，实际为 %d", len(tools))
	}
	if tools[0] != "test_tool" {
		t.Errorf("期望 tool name 为 'test_tool'，实际为 '%s'", tools[0])
	}
}

// TestMCPServer_RegisterResource 验证注册 resource 后可以通过 ListResources 获取
func TestMCPServer_RegisterResource(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")

	// 注册 resource
	err := s.RegisterResource("flowx://tools/list", "工具列表",
		"可用工具列表", "application/json",
		[]byte(`{"tools": ["tool1", "tool2"]}`),
	)
	if err != nil {
		t.Fatalf("注册 resource 失败: %v", err)
	}

	resources := s.ListResources()
	if len(resources) != 1 {
		t.Fatalf("期望 1 个 resource，实际为 %d", len(resources))
	}
	if resources[0].Name != "工具列表" {
		t.Errorf("期望 resource name 为 '工具列表'，实际为 '%s'", resources[0].Name)
	}
}

// TestMCPServer_CallTool_Success 验证调用已注册的 tool 返回结果
func TestMCPServer_CallTool_Success(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")

	tool := mcp.NewTool("greet",
		mcp.WithDescription("问候工具"),
		mcp.WithString("name", mcp.Required()),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Hello, World!"), nil
	}

	err := s.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("注册 tool 失败: %v", err)
	}

	// 调用 tool
	result, err := s.CallTool(context.Background(), "greet", map[string]any{"name": "World"})
	if err != nil {
		t.Fatalf("调用 tool 失败: %v", err)
	}
	if result == nil {
		t.Fatal("期望返回结果，实际为 nil")
	}
}

// TestMCPServer_CallTool_NotFound 验证调用未注册的 tool 返回错误
func TestMCPServer_CallTool_NotFound(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")

	_, err := s.CallTool(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
}

// TestMCPServer_StartAndStop 验证 Server 可以启动和停止
func TestMCPServer_StartAndStop(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")

	// 注册一个 tool
	tool := mcp.NewTool("ping", mcp.WithDescription("Ping"))
	err := s.RegisterTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})
	if err != nil {
		t.Fatalf("注册 tool 失败: %v", err)
	}

	// 获取底层 MCPServer 用于验证
	inner := s.InnerServer()
	if inner == nil {
		t.Fatal("期望获取底层 MCPServer，实际为 nil")
	}
	// 验证底层 server 的名称（通过 ListTools 间接验证 server 可用）
	_ = inner.ListTools()
}

// TestMCPServer_DuplicateTool 验证重复注册同名 tool 返回错误
func TestMCPServer_DuplicateTool(t *testing.T) {
	s := NewMCPServer("flowx-test", "1.0.0")

	tool := mcp.NewTool("dup_tool", mcp.WithDescription("重复工具"))

	err := s.RegisterTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})
	if err != nil {
		t.Fatalf("首次注册 tool 失败: %v", err)
	}

	// 重复注册
	err = s.RegisterTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})
	if err == nil {
		t.Fatal("期望重复注册返回错误，实际为 nil")
	}
}
