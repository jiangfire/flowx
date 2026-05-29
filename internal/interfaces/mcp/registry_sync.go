package mcp

import (
	"context"
	"fmt"

	toolapp "github.com/jiangfire/flowx/internal/application/tool"
)

// SyncToolsFromDB loads all active tools from repository and registers them
func SyncToolsFromDB(ctx context.Context, registry ToolRegistry, toolRepo toolapp.ToolRepository) error {
	tools, _, err := toolRepo.List(ctx, toolapp.ToolFilter{Status: "active", PageSize: 10000})
	if err != nil {
		return err
	}
	for i := range tools {
		if err := registry.RegisterTool(&tools[i]); err != nil {
			// 单个工具注册失败不应阻塞整体同步
			fmt.Printf("注册工具 '%s' 到 MCP Registry 失败: %v\n", tools[i].Name, err)
		}
	}
	return nil
}
