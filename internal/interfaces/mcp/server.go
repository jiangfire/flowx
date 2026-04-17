package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolHandlerFunc MCP tool 处理函数类型（与 server.ToolHandlerFunc 签名一致）
type ToolHandlerFunc = server.ToolHandlerFunc

// MCPServer MCP Server 封装，提供 FlowX 工具注册和调用能力
type MCPServer struct {
	name      string
	version   string
	inner     *server.MCPServer
	tools     map[string]ToolHandlerFunc
	resources []resourceEntry
	mu        sync.RWMutex
}

// ResourceInfo 资源信息（导出给外部使用）
type ResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mime_type"`
}

// resourceEntry 内部资源条目
type resourceEntry struct {
	uri         string
	name        string
	description string
	mimeType    string
	data        []byte
}

// NewMCPServer 创建 MCP Server 实例
func NewMCPServer(name, version string) *MCPServer {
	return &MCPServer{
		name:    name,
		version: version,
		inner:   server.NewMCPServer(name, version, server.WithToolCapabilities(false)),
		tools:   make(map[string]ToolHandlerFunc),
	}
}

// Name 返回 server 名称
func (s *MCPServer) Name() string {
	return s.name
}

// InnerServer 返回底层 MCPServer 实例
func (s *MCPServer) InnerServer() *server.MCPServer {
	return s.inner
}

// RegisterTool 注册 MCP tool
func (s *MCPServer) RegisterTool(tool mcp.Tool, handler ToolHandlerFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := tool.Name
	if _, exists := s.tools[name]; exists {
		return fmt.Errorf("tool '%s' 已注册", name)
	}

	s.tools[name] = handler
	s.inner.AddTool(tool, handler)
	return nil
}

// ListTools 返回所有已注册的 tool 名称列表
func (s *MCPServer) ListTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	toolMap := s.inner.ListTools()
	names := make([]string, 0, len(toolMap))
	for name := range toolMap {
		names = append(names, name)
	}
	return names
}

// RegisterResource 注册 MCP resource
func (s *MCPServer) RegisterResource(uri, name, description, mimeType string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource := mcp.NewResource(
		uri,
		name,
		mcp.WithResourceDescription(description),
		mcp.WithMIMEType(mimeType),
	)

	// 注册资源处理器
	s.inner.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: mimeType,
				Text:     string(data),
			},
		}, nil
	})

	s.resources = append(s.resources, resourceEntry{
		uri:         uri,
		name:        name,
		description: description,
		mimeType:    mimeType,
		data:        data,
	})
	return nil
}

// ListResources 返回所有已注册的 resource 信息
func (s *MCPServer) ListResources() []ResourceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ResourceInfo, len(s.resources))
	for i, r := range s.resources {
		result[i] = ResourceInfo{
			URI:         r.uri,
			Name:        r.name,
			Description: r.description,
			MIMEType:    r.mimeType,
		}
	}
	return result
}

// CallTool 调用指定 tool
func (s *MCPServer) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	handler, exists := s.tools[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool '%s' 未注册", name)
	}

	// 构建 CallToolRequest
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	return handler(ctx, req)
}
