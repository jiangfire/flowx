package persistence

import (
	"context"
	"fmt"

	"git.neolidy.top/neo/flowx/internal/domain/base"
	"git.neolidy.top/neo/flowx/internal/domain/bpmn"
	bpmnapp "git.neolidy.top/neo/flowx/internal/application/bpmn"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// processDefinitionPO 流程定义持久化对象
type processDefinitionPO struct {
	base.BaseModel
	Name           string `gorm:"size:200;not null"`
	Version        int    `gorm:"default:1"`
	Status         string `gorm:"size:20;default:draft;index"`
	DefinitionYAML string `gorm:"type:text;not null"`
}

// TableName 表名
func (processDefinitionPO) TableName() string { return "process_definitions" }

// toDomain 将 PO 转换为领域模型
func (po *processDefinitionPO) toDomain() (*bpmn.ProcessDefinition, error) {
	def := &bpmn.ProcessDefinition{}
	if err := yaml.Unmarshal([]byte(po.DefinitionYAML), def); err != nil {
		return nil, fmt.Errorf("解析流程定义 YAML 失败: %w", err)
	}
	def.ID = po.ID
	def.Name = po.Name
	def.Version = po.Version
	def.Status = po.Status
	return def, nil
}

// toPO 将领域模型转换为 PO
func toProcessDefinitionPO(tenantID string, def *bpmn.ProcessDefinition) (*processDefinitionPO, error) {
	yamlData, err := yaml.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("序列化流程定义 YAML 失败: %w", err)
	}
	return &processDefinitionPO{
		BaseModel: base.BaseModel{
			ID:       def.ID,
			TenantID: tenantID,
		},
		Name:           def.Name,
		Version:        def.Version,
		Status:         def.Status,
		DefinitionYAML: string(yamlData),
	}, nil
}

// ==================== ProcessDefinitionRepository ====================

// processDefinitionRepository 流程定义仓储实现
type processDefinitionRepository struct {
	db *gorm.DB
}

// NewProcessDefinitionRepository 创建流程定义仓储实例
func NewProcessDefinitionRepository(db *gorm.DB) bpmnapp.ProcessDefinitionRepository {
	return &processDefinitionRepository{db: db}
}

// Create 创建流程定义
func (r *processDefinitionRepository) Create(ctx context.Context, def *bpmn.ProcessDefinition) error {
	if def.ID == "" {
		def.ID = base.GenerateUUID()
	}
	po, err := toProcessDefinitionPO("", def)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(po).Error
}

// GetByID 根据 ID 查询流程定义
func (r *processDefinitionRepository) GetByID(ctx context.Context, id string) (*bpmn.ProcessDefinition, error) {
	var po processDefinitionPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("流程定义不存在: %s", id)
		}
		return nil, fmt.Errorf("查询流程定义失败: %w", err)
	}
	return po.toDomain()
}

// List 查询流程定义列表（支持过滤和分页）
func (r *processDefinitionRepository) List(ctx context.Context, filter bpmnapp.ProcessDefinitionFilter) ([]*bpmn.ProcessDefinition, int64, error) {
	var pos []processDefinitionPO
	var total int64

	query := r.db.WithContext(ctx).Model(&processDefinitionPO{}).Where("tenant_id = ?", filter.TenantID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+filter.Keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计流程定义数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, fmt.Errorf("查询流程定义列表失败: %w", err)
	}

	result := make([]*bpmn.ProcessDefinition, 0, len(pos))
	for i := range pos {
		def, err := pos[i].toDomain()
		if err != nil {
			return nil, 0, err
		}
		result = append(result, def)
	}

	return result, total, nil
}

// Update 更新流程定义
func (r *processDefinitionRepository) Update(ctx context.Context, def *bpmn.ProcessDefinition) error {
	po, err := toProcessDefinitionPO("", def)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&processDefinitionPO{}).Where("id = ?", def.ID).Updates(map[string]any{
		"name":            po.Name,
		"version":         po.Version,
		"status":          po.Status,
		"definition_yaml": po.DefinitionYAML,
	}).Error
}

// Delete 软删除流程定义
func (r *processDefinitionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&processDefinitionPO{}, "id = ?", id).Error
}

// ==================== ProcessInstanceRepository ====================

// processInstanceRepository 流程实例仓储实现
type processInstanceRepository struct {
	db *gorm.DB
}

// NewProcessInstanceRepository 创建流程实例仓储实例
func NewProcessInstanceRepository(db *gorm.DB) bpmnapp.ProcessInstanceRepository {
	return &processInstanceRepository{db: db}
}

// Create 创建流程实例
func (r *processInstanceRepository) Create(ctx context.Context, inst *bpmn.ProcessInstance) error {
	if inst.ID == "" {
		inst.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(inst).Error
}

// GetByID 根据 ID 查询流程实例
func (r *processInstanceRepository) GetByID(ctx context.Context, id string) (*bpmn.ProcessInstance, error) {
	var inst bpmn.ProcessInstance
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&inst).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("流程实例不存在: %s", id)
		}
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}
	return &inst, nil
}

// List 查询流程实例列表（支持过滤和分页）
func (r *processInstanceRepository) List(ctx context.Context, filter bpmnapp.ProcessInstanceFilter) ([]*bpmn.ProcessInstance, int64, error) {
	var instances []bpmn.ProcessInstance
	var total int64

	query := r.db.WithContext(ctx).Model(&bpmn.ProcessInstance{}).Where("tenant_id = ?", filter.TenantID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计流程实例数量失败: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&instances).Error; err != nil {
		return nil, 0, fmt.Errorf("查询流程实例列表失败: %w", err)
	}

	result := make([]*bpmn.ProcessInstance, len(instances))
	for i := range instances {
		result[i] = &instances[i]
	}

	return result, total, nil
}

// Update 更新流程实例
func (r *processInstanceRepository) Update(ctx context.Context, inst *bpmn.ProcessInstance) error {
	return r.db.WithContext(ctx).Save(inst).Error
}

// ==================== ProcessTaskRepository ====================

// processTaskRepository 流程任务仓储实现
type processTaskRepository struct {
	db *gorm.DB
}

// NewProcessTaskRepository 创建流程任务仓储实例
func NewProcessTaskRepository(db *gorm.DB) bpmnapp.ProcessTaskRepository {
	return &processTaskRepository{db: db}
}

// Create 创建流程任务
func (r *processTaskRepository) Create(ctx context.Context, task *bpmn.ProcessTask) error {
	if task.ID == "" {
		task.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID 根据 ID 查询流程任务
func (r *processTaskRepository) GetByID(ctx context.Context, id string) (*bpmn.ProcessTask, error) {
	var task bpmn.ProcessTask
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("流程任务不存在: %s", id)
		}
		return nil, fmt.Errorf("查询流程任务失败: %w", err)
	}
	return &task, nil
}

// ListByInstance 根据实例 ID 查询任务列表
func (r *processTaskRepository) ListByInstance(ctx context.Context, instanceID string) ([]*bpmn.ProcessTask, error) {
	var tasks []bpmn.ProcessTask
	if err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at ASC").Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("查询流程任务列表失败: %w", err)
	}
	result := make([]*bpmn.ProcessTask, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}
	return result, nil
}

// ListPending 查询待办任务
func (r *processTaskRepository) ListPending(ctx context.Context, tenantID, assignee string) ([]*bpmn.ProcessTask, error) {
	var tasks []bpmn.ProcessTask
	query := r.db.WithContext(ctx).Where("tenant_id = ? AND status = ?", tenantID, "pending")
	if assignee != "" {
		query = query.Where("assignee = ?", assignee)
	}
	if err := query.Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("查询待办任务列表失败: %w", err)
	}
	result := make([]*bpmn.ProcessTask, len(tasks))
	for i := range tasks {
		result[i] = &tasks[i]
	}
	return result, nil
}

// Update 更新流程任务
func (r *processTaskRepository) Update(ctx context.Context, task *bpmn.ProcessTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// ==================== ExecutionHistoryRepository ====================

// executionHistoryRepository 执行历史仓储实现
type executionHistoryRepository struct {
	db *gorm.DB
}

// NewExecutionHistoryRepository 创建执行历史仓储实例
func NewExecutionHistoryRepository(db *gorm.DB) bpmnapp.ExecutionHistoryRepository {
	return &executionHistoryRepository{db: db}
}

// Create 创建执行历史
func (r *executionHistoryRepository) Create(ctx context.Context, h *bpmn.ExecutionHistory) error {
	if h.ID == "" {
		h.ID = base.GenerateUUID()
	}
	return r.db.WithContext(ctx).Create(h).Error
}

// ListByInstance 根据实例 ID 查询执行历史
func (r *executionHistoryRepository) ListByInstance(ctx context.Context, instanceID string) ([]*bpmn.ExecutionHistory, error) {
	var histories []bpmn.ExecutionHistory
	if err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("created_at ASC").Find(&histories).Error; err != nil {
		return nil, fmt.Errorf("查询执行历史列表失败: %w", err)
	}
	result := make([]*bpmn.ExecutionHistory, len(histories))
	for i := range histories {
		result[i] = &histories[i]
	}
	return result, nil
}
