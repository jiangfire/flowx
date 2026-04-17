package tool

import (
	"context"

	domaintool "git.neolidy.top/neo/flowx/internal/domain/tool"
)

// ToolFilter 工具查询过滤条件
type ToolFilter struct {
	TenantID string
	Type     string
	Status   string
	Category string
	Keyword  string
	Page     int
	PageSize int
}

// ConnectorFilter 连接器查询过滤条件
type ConnectorFilter struct {
	TenantID string
	Type     string
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

// ToolRepository 工具仓储接口
type ToolRepository interface {
	Create(ctx context.Context, tool *domaintool.Tool) error
	GetByID(ctx context.Context, id string) (*domaintool.Tool, error)
	List(ctx context.Context, filter ToolFilter) ([]domaintool.Tool, int64, error)
	Update(ctx context.Context, tool *domaintool.Tool) error
	Delete(ctx context.Context, id string) error
}

// ConnectorRepository 连接器仓储接口
type ConnectorRepository interface {
	Create(ctx context.Context, connector *domaintool.Connector) error
	GetByID(ctx context.Context, id string) (*domaintool.Connector, error)
	List(ctx context.Context, filter ConnectorFilter) ([]domaintool.Connector, int64, error)
	Update(ctx context.Context, connector *domaintool.Connector) error
	Delete(ctx context.Context, id string) error
}
