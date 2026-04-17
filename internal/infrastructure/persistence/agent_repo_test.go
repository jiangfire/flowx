package persistence

import (
	"context"
	"testing"

	"git.neolidy.top/neo/flowx/internal/domain/agent"
	"git.neolidy.top/neo/flowx/internal/domain/base"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupAgentTestDB 创建 Agent 模块测试数据库
func setupAgentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&agent.AgentTask{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	return db
}

// TestCreateAgentTask_Success 创建任务成功
func TestCreateAgentTask_Success(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	task := &agent.AgentTask{
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "created",
	}

	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	if task.ID == "" {
		t.Error("期望创建后 ID 不为空")
	}
}

// TestCreateAgentTask_GeneratesUUID 创建任务时自动生成 UUID
func TestCreateAgentTask_GeneratesUUID(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	task := &agent.AgentTask{
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "created",
	}

	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	if task.ID == "" {
		t.Error("期望创建后自动生成 ID")
	}
}

// TestGetByID_Success 按 ID 查询任务成功
func TestGetByID_Success(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	task := &agent.AgentTask{
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "running",
	}
	repo.Create(ctx, task)

	found, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if found.Type != "tool_execute" {
		t.Errorf("期望 Type 为 'tool_execute'，实际为 '%s'", found.Type)
	}
	if found.Description != "测试任务" {
		t.Errorf("期望 Description 为 '测试任务'，实际为 '%s'", found.Description)
	}
}

// TestGetByID_NotFound 任务不存在返回错误
func TestGetByID_NotFound(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("期望返回错误，但返回 nil")
	}
}

// TestList_All 列出所有任务
func TestList_All(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		task := &agent.AgentTask{
			BaseModel:   base.BaseModel{TenantID: "tenant-001"},
			Type:        "tool_execute",
			Description: "测试任务",
			Status:      "completed",
		}
		repo.Create(ctx, task)
	}

	tasks, total, err := repo.List(ctx, "tenant-001", "", 1, 10)
	if err != nil {
		t.Fatalf("列出任务失败: %v", err)
	}
	if total != 3 {
		t.Errorf("期望 total 为 3，实际为 %d", total)
	}
	if len(tasks) != 3 {
		t.Errorf("期望返回 3 条记录，实际为 %d", len(tasks))
	}
}

// TestList_WithStatus 按状态过滤任务
func TestList_WithStatus(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	task1 := &agent.AgentTask{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Type: "tool_execute", Status: "completed"}
	task2 := &agent.AgentTask{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Type: "tool_execute", Status: "running"}
	task3 := &agent.AgentTask{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Type: "tool_execute", Status: "completed"}
	repo.Create(ctx, task1)
	repo.Create(ctx, task2)
	repo.Create(ctx, task3)

	tasks, total, err := repo.List(ctx, "tenant-001", "completed", 1, 10)
	if err != nil {
		t.Fatalf("列出任务失败: %v", err)
	}
	if total != 2 {
		t.Errorf("期望 total 为 2，实际为 %d", total)
	}
	if len(tasks) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(tasks))
	}
}

// TestList_WithPagination 分页查询
func TestList_WithPagination(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		task := &agent.AgentTask{BaseModel: base.BaseModel{TenantID: "tenant-001"}, Type: "tool_execute", Status: "completed"}
		repo.Create(ctx, task)
	}

	tasks, total, err := repo.List(ctx, "tenant-001", "", 1, 2)
	if err != nil {
		t.Fatalf("列出任务失败: %v", err)
	}
	if total != 5 {
		t.Errorf("期望 total 为 5，实际为 %d", total)
	}
	if len(tasks) != 2 {
		t.Errorf("期望返回 2 条记录，实际为 %d", len(tasks))
	}
}

// TestUpdate_Success 更新任务成功
func TestUpdate_Success(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentTaskRepository(db)
	ctx := context.Background()

	task := &agent.AgentTask{
		Type:        "tool_execute",
		Description: "测试任务",
		Status:      "running",
	}
	repo.Create(ctx, task)

	task.Status = "completed"
	task.Result = `{"output": "done"}`
	err := repo.Update(ctx, task)
	if err != nil {
		t.Fatalf("更新任务失败: %v", err)
	}

	found, _ := repo.GetByID(ctx, task.ID)
	if found.Status != "completed" {
		t.Errorf("期望 Status 为 'completed'，实际为 '%s'", found.Status)
	}
	if found.Result != `{"output": "done"}` {
		t.Errorf("期望 Result 为 '{\"output\": \"done\"}'，实际为 '%s'", found.Result)
	}
}
