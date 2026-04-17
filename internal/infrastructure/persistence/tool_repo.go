package persistence

import (
	"context"
	"fmt"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/tool"
	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	"gorm.io/gorm"
)

// ==================== ToolRepository ====================

// toolRepository 工具仓储实现
type toolRepository struct {
	db *gorm.DB
}

// NewToolRepository 创建工具仓储实例
func NewToolRepository(db *gorm.DB) toolapp.ToolRepository {
	return &toolRepository{db: db}
}

// Create 创建工具
func (r *toolRepository) Create(ctx context.Context, tl *tool.Tool) error {
	if tl.ID == "" {
		tl.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(tl).Error
}

// GetByID 根据 ID 查询工具
func (r *toolRepository) GetByID(ctx context.Context, id string) (*tool.Tool, error) {
	var tl tool.Tool
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tl).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工具不存在: %s", id)
		}
		return nil, fmt.Errorf("查询工具失败: %w", err)
	}
	return &tl, nil
}

// List 查询工具列表（支持过滤和分页）
func (r *toolRepository) List(ctx context.Context, filter toolapp.ToolFilter) ([]tool.Tool, int64, error) {
	var tools []tool.Tool
	var total int64

	query := r.db.WithContext(ctx).Model(&tool.Tool{}).Where("tenant_id = ?", filter.TenantID)

	// 按类型过滤
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	// 按状态过滤
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// 按分类过滤
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	// 按关键词搜索（名称模糊匹配）
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计工具数量失败: %w", err)
	}

	// 分页参数默认值
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tools).Error; err != nil {
		return nil, 0, fmt.Errorf("查询工具列表失败: %w", err)
	}

	return tools, total, nil
}

// Update 更新工具
func (r *toolRepository) Update(ctx context.Context, tl *tool.Tool) error {
	return r.db.WithContext(ctx).Save(tl).Error
}

// Delete 软删除工具
func (r *toolRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&tool.Tool{}, "id = ?", id).Error
}

// ==================== ConnectorRepository ====================

// connectorRepository 连接器仓储实现
type connectorRepository struct {
	db *gorm.DB
}

// NewConnectorRepository 创建连接器仓储实例
func NewConnectorRepository(db *gorm.DB) toolapp.ConnectorRepository {
	return &connectorRepository{db: db}
}

// Create 创建连接器
func (r *connectorRepository) Create(ctx context.Context, conn *tool.Connector) error {
	if conn.ID == "" {
		conn.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(conn).Error
}

// GetByID 根据 ID 查询连接器
func (r *connectorRepository) GetByID(ctx context.Context, id string) (*tool.Connector, error) {
	var conn tool.Connector
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&conn).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("连接器不存在: %s", id)
		}
		return nil, fmt.Errorf("查询连接器失败: %w", err)
	}
	return &conn, nil
}

// List 查询连接器列表（支持过滤和分页）
func (r *connectorRepository) List(ctx context.Context, filter toolapp.ConnectorFilter) ([]tool.Connector, int64, error) {
	var connectors []tool.Connector
	var total int64

	query := r.db.WithContext(ctx).Model(&tool.Connector{}).Where("tenant_id = ?", filter.TenantID)

	// 按类型过滤
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	// 按状态过滤
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// 按关键词搜索
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计连接器数量失败: %w", err)
	}

	// 分页参数默认值
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&connectors).Error; err != nil {
		return nil, 0, fmt.Errorf("查询连接器列表失败: %w", err)
	}

	return connectors, total, nil
}

// Update 更新连接器
func (r *connectorRepository) Update(ctx context.Context, conn *tool.Connector) error {
	return r.db.WithContext(ctx).Save(conn).Error
}

// Delete 软删除连接器
func (r *connectorRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&tool.Connector{}, "id = ?", id).Error
}
