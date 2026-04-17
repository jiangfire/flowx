# FlowX 功能增强需求设计文档

> 版本: v1.0 | 日期: 2026-04-16 | 状态: 待确认

---

## 1. 概述

本文档描述 FlowX 平台三个核心功能增强的详细设计，基于架构审查中发现的设计缺口。

### 1.1 三大功能

| 功能 | 目标 | 优先级 |
|------|------|--------|
| 工作流发布机制 | 堵住"draft 工作流被直接使用"的漏洞 | P0 |
| 策略执行引擎 | 让 DataPolicy 从"存储"变为"执行" | P0 |
| Agent-审批打通 | 消除两套审批系统的隔离 | P1 |

### 1.2 设计决策记录

| 决策项 | 选择 | 原因 |
|--------|------|------|
| 工作流激活方式 | 管理员直接激活（单步） | 当前阶段无需两步审核 |
| 策略校验粒度 | 字段级校验（自定义表达式） | 比存在性拦截更有实际价值 |
| 表达式语法 | 类 CEL 简化语法 | 灵活且不需要引入第三方依赖 |
| 表达式可访问数据 | 工具属性 + 用户信息 + 请求上下文 | 覆盖所有校验场景 |
| 策略违规处理 | 硬拦截（返回 400） | 安全优先 |
| Agent 审批人 | 使用 Task 创建者 ID | 语义上是"发起人确认系统建议" |
| ImportTools 策略校验 | 先校验全部再批量创建 | 保证原子性 |
| 文档格式 | Markdown | 便于版本管理和协作 |

---

## 2. 功能一：工作流发布机制

### 2.1 状态机

```
                  ┌──────────────┐
          创建     │              │  归档
    ─────────────→│    draft     │←───────────
                  │              │             │
                  └──────┬───────┘             │
                         │ 激活                │
                         ↓                     │
                  ┌──────────────┐             │
                  │              │─────────────→│
                  │    active    │  归档        │
                  │              │             │
                  └──────────────┘             │
                                               │
                  ┌──────────────┐             │
                  │              │             │
                  │   archived   │             │
                  │              │─────────────→│ (无出口)
                  └──────────────┘
```

**状态转换规则：**

| 当前状态 | 目标状态 | 允许? | 说明 |
|---------|---------|-------|------|
| draft | active | ✅ | 激活 |
| draft | archived | ✅ | 直接归档（跳过激活） |
| active | archived | ✅ | 归档 |
| active | draft | ❌ | 不允许回退到草稿 |
| archived | active | ❌ | 不允许重新激活（需新建版本） |
| archived | draft | ❌ | 不允许 |

### 2.2 API 设计

#### ActivateWorkflow

```
POST /api/v1/workflows/:id/activate
Authorization: Bearer <token>
```

**响应：**
- `200` — 激活成功，返回更新后的 Workflow
- `400` — 状态不允许（如已经是 active 或 archived）
- `404` — 工作流不存在
- `403` — 租户不匹配

#### ArchiveWorkflow

```
POST /api/v1/workflows/:id/archive
Authorization: Bearer <token>
```

**响应：** 同上

#### StartApproval 状态校验

在现有的 `POST /api/v1/approvals` 中增加校验：
- 工作流 `status != active` → 返回 `400 "工作流未激活，无法发起审批"`
- 工作流 `status == archived` → 返回 `400 "工作流已归档，无法发起审批"`

### 2.3 代码变更

| 文件 | 变更 |
|------|------|
| `application/approval/service.go` | 新增 `ActivateWorkflow`、`ArchiveWorkflow` 方法；`StartApproval` 增加状态校验 |
| `application/approval/port.go` | `ApprovalRepository` 新增 `UpdateWorkflowStatus` 方法 |
| `infrastructure/persistence/approval_repo.go` | 实现 `UpdateWorkflowStatus` |
| `interfaces/http/handler/approval.go` | 新增 `ActivateWorkflow`、`ArchiveWorkflow` handler |
| `interfaces/http/router.go` | 新增 2 个路由 |
| `application/approval/service_test.go` | 新增测试用例 |

### 2.4 测试用例

| 用例 | 描述 |
|------|------|
| TestActivateWorkflow_Success | draft → active 成功 |
| TestActivateWorkflow_AlreadyActive | active → active 返回错误 |
| TestActivateWorkflow_Archived | archived → active 返回错误 |
| TestArchiveWorkflow_Success | active → archived 成功 |
| TestArchiveWorkflow_FromDraft | draft → archived 成功 |
| TestStartApproval_DraftWorkflow | 使用 draft 工作流发起审批返回错误 |
| TestStartApproval_ActiveWorkflow | 使用 active 工作流发起审批成功 |
| TestStartApproval_ArchivedWorkflow | 使用 archived 工作流发起审批返回错误 |

---

## 3. 功能二：策略执行引擎

### 3.1 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      策略执行引擎                             │
│                                                             │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────────┐   │
│  │ 工具创建  │───→│ PolicyEngine │───→│ 校验结果          │   │
│  │ 更新/导入 │    │              │    │ pass / reject     │   │
│  └──────────┘    └──────┬───────┘    └──────────────────┘   │
│                         │                                   │
│                         ↓                                   │
│                  ┌──────────────┐                           │
│                  │ Expression   │                           │
│                  │ Evaluator    │                           │
│                  │ (类CEL语法)   │                           │
│                  └──────────────┘                           │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 策略匹配逻辑

策略通过 `Scope` + `ScopeValue` 字段匹配工具：

| Scope | ScopeValue | 匹配规则 |
|-------|-----------|---------|
| `global` | （任意） | 匹配所有工具 |
| `tool_type` | `eda` / `cae` / ... | 匹配 `Tool.Type == ScopeValue` |
| `category` | `visualization` / ... | 匹配 `Tool.Category == ScopeValue` |

**匹配流程：**
1. 查询所有 `status=active` 且 `tenant_id=当前租户` 的 DataPolicy
2. 按 `priority DESC` 排序（高优先级先校验）
3. 对每条策略，检查是否匹配当前工具
4. 匹配的策略按优先级依次执行表达式校验
5. 任一策略校验失败 → 硬拦截，返回 400

### 3.3 四种策略类型的规则字段

#### quality（质量策略）

**校验对象：** 工具配置属性

**Rules JSON 结构：**
```json
{
  "required_fields": ["endpoint", "config.auth_type"],
  "max_config_size": 10240,
  "allowed_types": ["eda", "cae", "data_pipeline"],
  "description_required": true
}
```

**校验逻辑：**
- `required_fields`: 工具的对应字段必须非空
- `max_config_size`: `Tool.Config` JSON 序列化后不超过指定字节数
- `allowed_types`: `Tool.Type` 必须在列表中
- `description_required`: `Tool.Description` 必须非空

#### classification（分类策略）

**校验对象：** 工具的 category 字段

**Rules JSON 结构：**
```json
{
  "allowed_categories": ["visualization", "data_processing", "ai_model"],
  "default_category": "data_processing",
  "category_required": true
}
```

**校验逻辑：**
- `allowed_categories`: `Tool.Category` 必须在列表中
- `category_required`: `Tool.Category` 不能为空
- `default_category`: 当工具未指定 category 时自动设置（仅在 category_required=false 时生效）

#### access（访问控制策略）

**校验对象：** 创建者的角色

**Rules JSON 结构：**
```json
{
  "allowed_roles": ["admin", "tool_manager"],
  "blocked_roles": ["viewer"],
  "require_role": "admin"
}
```

**校验逻辑：**
- `allowed_roles`: 创建者的角色必须在列表中（白名单）
- `blocked_roles`: 创建者的角色不能在列表中（黑名单）
- `require_role`: 创建者必须拥有指定角色

#### retention（保留策略）

**校验对象：** 工具的配置属性（声明式，不执行清理）

**Rules JSON 结构：**
```json
{
  "max_retention_days": 90,
  "require_expiry_date": true,
  "auto_archive": true
}
```

**校验逻辑（第一版）：**
- `max_retention_days`: 检查工具 config 中是否配置了 `retention_days` 且不超过限制
- `require_expiry_date`: 检查工具 config 中是否配置了 `expiry_date`
- `auto_archive`: 仅做记录，不执行（标记为需要自动归档）

> **注意：** retention 策略第一版仅做"配置存在性校验"，不实现定时清理。定时清理需要引入 cron 框架，超出本次范围。

### 3.4 自定义表达式引擎

#### 语法设计（类 CEL 简化版）

```
表达式 ::= 比较表达式 (逻辑运算符 比较表达式)*
比较表达式 ::= 路径 比较运算符 值
路径 ::= 标识符 ('.' 标识符)*
比较运算符 ::= '==' | '!=' | '>' | '>=' | '<' | '<=' | 'in' | 'not_in' | 'contains' | 'matches'
逻辑运算符 ::= '&&' | '||'
值 ::= 字符串 | 数字 | 布尔值 | 数组 | 'null' | 'empty'
```

**示例：**
```
tool.type in ["eda", "cae"]
tool.config.max_size <= 100
tool.category != ""
user.role == "admin"
tool.description != empty
tool.endpoint matches "^https://.*"
```

#### 可访问的数据上下文

| 变量 | 类型 | 说明 |
|------|------|------|
| `tool.name` | string | 工具名称 |
| `tool.type` | string | 工具类型 |
| `tool.category` | string | 工具分类 |
| `tool.description` | string | 工具描述 |
| `tool.endpoint` | string | 工具端点 |
| `tool.status` | string | 工具状态 |
| `tool.config` | map | 工具配置（JSON） |
| `tool.connector_id` | string | 关联连接器 ID |
| `user.id` | string | 创建者 ID |
| `user.role` | string | 创建者角色 |
| `user.username` | string | 创建者用户名 |
| `ctx.tenant_id` | string | 租户 ID |
| `ctx.action` | string | 操作类型（create/update/import） |

#### 表达式引擎实现

新建 `internal/application/datagov/expression/` 包：

```
expression/
├── evaluator.go      # 表达式求值器
├── parser.go         # 词法分析 + 语法解析
├── tokenizer.go      # 词法单元定义
├── context.go        # 求值上下文（tool/user/ctx）
└── evaluator_test.go # 测试
```

**核心接口：**
```go
// Evaluator 表达式求值器
type Evaluator interface {
    // Evaluate 对给定上下文求值表达式
    Evaluate(ctx *Context, expr string) (bool, error)
}

// Context 求值上下文
type Context struct {
    Tool  ToolContext
    User  UserContext
    Ctx   RequestContext
}

type ToolContext struct {
    Name, Type, Category, Description, Endpoint, Status, ConnectorID string
    Config map[string]any
}

type UserContext struct {
    ID, Role, Username string
}

type RequestContext struct {
    TenantID string
    Action   string // create/update/import
}
```

### 3.5 PolicyEngine 服务

新建 `internal/application/datagov/policy_engine.go`：

```go
// PolicyEngine 策略执行引擎
type PolicyEngine interface {
    // ValidateTool 校验工具是否符合策略
    // 返回匹配的策略列表和校验结果
    ValidateTool(ctx context.Context, tenantID string, tool *domain.Tool, userRole string, action string) (*PolicyValidationResult, error)

    // ValidateTools 批量校验工具（用于 ImportTools）
    ValidateTools(ctx context.Context, tenantID string, tools []*domain.Tool, userRole string) (*BatchValidationResult, error)
}

type PolicyValidationResult struct {
    Passed  bool
    Errors  []PolicyViolation
}

type PolicyViolation struct {
    PolicyName string
    PolicyType string
    RuleKey    string
    Message    string
}

type BatchValidationResult struct {
    Passed  bool
    Errors  []BatchItemError
}

type BatchItemError struct {
    Index     int
    ToolName  string
    Violations []PolicyViolation
}
```

### 3.6 集成点

#### CreateTool

```go
func (s *ToolService) CreateTool(ctx context.Context, tenantID string, req *CreateToolRequest, userRole string) (*tool.Tool, error) {
    // 1. 基础校验（现有逻辑）
    // 2. 构建工具对象
    tl := &tool.Tool{...}

    // 3. 策略校验（新增）
    result, err := s.policyEngine.ValidateTool(ctx, tenantID, tl, userRole, "create")
    if err != nil {
        return nil, fmt.Errorf("策略校验失败: %w", err)
    }
    if !result.Passed {
        return nil, &PolicyViolationError{Violations: result.Errors}
    }

    // 4. 持久化
    return tl, s.toolRepo.Create(ctx, tl)
}
```

#### ImportTools

```go
func (s *ToolService) ImportTools(ctx context.Context, tenantID string, req *ImportRequest, userRole string) ([]tool.Tool, error) {
    // 1. 解析 Excel
    tools := parseExcel(req.File)

    // 2. 批量策略校验（先校验再创建）
    result, err := s.policyEngine.ValidateTools(ctx, tenantID, tools, userRole)
    if err != nil {
        return nil, fmt.Errorf("策略校验失败: %w", err)
    }
    if !result.Passed {
        return nil, &BatchPolicyViolationError{Errors: result.Errors}
    }

    // 3. 批量创建
    for _, tl := range tools {
        s.toolRepo.Create(ctx, tl)
    }
    return tools, nil
}
```

### 3.7 错误响应格式

策略校验失败返回 `400`，响应体：

```json
{
  "code": 42201,
  "message": "策略校验失败",
  "data": {
    "violations": [
      {
        "policy_name": "EDA工具质量标准",
        "policy_type": "quality",
        "rule_key": "required_fields",
        "message": "工具 '数据清洗' 缺少必填字段: endpoint"
      },
      {
        "policy_name": "访问控制策略",
        "policy_type": "access",
        "rule_key": "allowed_roles",
        "message": "角色 'viewer' 不允许创建 cae 类型的工具"
      }
    ]
  }
}
```

### 3.8 代码变更

| 文件 | 变更 |
|------|------|
| `application/datagov/expression/evaluator.go` | **新建** — 表达式求值器 |
| `application/datagov/expression/parser.go` | **新建** — 词法+语法解析 |
| `application/datagov/expression/tokenizer.go` | **新建** — 词法单元 |
| `application/datagov/expression/context.go` | **新建** — 求值上下文 |
| `application/datagov/policy_engine.go` | **新建** — 策略引擎服务 |
| `application/datagov/port.go` | 新增 `PolicyEngine` 依赖到 `DataGovService` |
| `application/datagov/service.go` | `DataGovService` 新增 `PolicyEngine` 字段 |
| `application/tool/port.go` | `ToolService` 构造函数新增 `PolicyEngine` 参数 |
| `application/tool/service.go` | `CreateTool`/`UpdateTool`/`ImportTools` 增加策略校验 |
| `interfaces/http/handler/tool.go` | handler 传递 `userRole` 给 service |
| `pkg/errors/policy.go` | **新建** — `PolicyViolationError` 错误类型 |
| `app/container.go` | 初始化 `PolicyEngine` 并注入 |
| `config.example.yaml` | 新增策略引擎相关配置 |

### 3.9 测试用例

| 用例 | 描述 |
|------|------|
| TestEvaluator_SimpleComparison | `tool.type == "eda"` 求值 |
| TestEvaluator_InOperator | `tool.type in ["eda", "cae"]` 求值 |
| TestEvaluator_LogicalAnd | `tool.type == "eda" && user.role == "admin"` |
| TestEvaluator_Matches | `tool.endpoint matches "^https://.*"` |
| TestEvaluator_EmptyCheck | `tool.description != empty` |
| TestEvaluator_NestedField | `tool.config.max_size <= 100` |
| TestEvaluator_InvalidSyntax | 语法错误返回明确错误信息 |
| TestPolicyEngine_QualityRules | quality 策略校验 required_fields |
| TestPolicyEngine_ClassificationRules | classification 策略校验 allowed_categories |
| TestPolicyEngine_AccessRules | access 策略校验 allowed_roles |
| TestPolicyEngine_RetentionRules | retention 策略校验配置存在性 |
| TestPolicyEngine_NoMatchingPolicy | 无匹配策略时通过 |
| TestPolicyEngine_PriorityOrder | 高优先级策略先执行 |
| TestPolicyEngine_GlobalScope | global 策略匹配所有工具 |
| TestCreateTool_PolicyViolation | 创建工具违反策略返回 400 |
| TestImportTools_BatchValidation | 导入时先校验再创建 |
| TestImportTools_PartialFailure | 导入时部分工具违反策略全部回滚 |

---

## 4. 功能三：Agent-审批打通

### 4.1 设计方案

在 Agent Task 和 WorkflowInstance 之间建立双向关联：

```
┌──────────────────┐         ┌──────────────────┐
│   Agent Task     │         │ WorkflowInstance  │
│                  │         │                  │
│ workflow_instance│←───────→│ agent_task_id    │
│ _id (新增字段)    │         │ (新增字段)        │
│                  │         │                  │
│ 审批通过 ──────────────────→│ 自动推进工作流     │
│ 审批拒绝 ──────────────────→│ 自动驳回工作流     │
└──────────────────┘         └──────────────────┘
```

### 4.2 数据模型变更

#### AgentTask 新增字段

```go
// domain/agent/model.go
type AgentTask struct {
    base.BaseModel
    // ... 现有字段 ...
    WorkflowInstanceID string `gorm:"size:26;index" json:"workflow_instance_id"` // 关联的工作流实例 ID
}
```

#### WorkflowInstance 新增字段

```go
// domain/approval/model.go
type WorkflowInstance struct {
    base.BaseModel
    // ... 现有字段 ...
    AgentTaskID string `gorm:"size:26;index" json:"agent_task_id"` // 关联的 Agent 任务 ID
}
```

### 4.3 联动流程

#### 创建关联

当 Agent Task 需要工作流审批时，创建 Task 的同时创建 WorkflowInstance：

```
1. 用户创建 Agent Task（携带 workflow_id）
2. AgentService 创建 AgentTask 记录
3. AgentService 调用 ApprovalService.StartApproval 创建 WorkflowInstance
4. 双向设置关联 ID：
   - AgentTask.WorkflowInstanceID = instance.ID
   - WorkflowInstance.AgentTaskID = task.ID
```

#### 审批通过联动

```
1. 用户审批通过 Agent Task
2. AgentService 更新 AgentTask.status = "approved"
3. 如果 AgentTask.WorkflowInstanceID 非空：
   a. 调用 ApprovalService.Approve 推进工作流
   b. approverID = AgentTask.CreatedBy（任务创建者）
   c. instanceID = AgentTask.WorkflowInstanceID
```

#### 审批拒绝联动

```
1. 用户拒绝 Agent Task
2. AgentService 更新 AgentTask.status = "rejected"
3. 如果 AgentTask.WorkflowInstanceID 非空：
   a. 调用 ApprovalService.Reject 驳回工作流
   b. approverID = AgentTask.CreatedBy
   c. instanceID = AgentTask.WorkflowInstanceID
```

### 4.4 API 变更

#### CreateTask 请求扩展

```json
{
  "type": "tool_execute",
  "description": "部署 EDA 工具到生产环境",
  "workflow_id": "workflow-001",
  "context": { ... },
  "steps": [ ... ]
}
```

新增 `workflow_id` 字段（可选）。如果提供，则自动创建关联的工作流实例。

#### 审批 API 无变更

现有的 `POST /agent/tasks/:id/approve` 和 `POST /agent/tasks/:id/reject` 行为不变，只是内部增加了联动逻辑。

### 4.5 代码变更

| 文件 | 变更 |
|------|------|
| `domain/agent/model.go` | 新增 `WorkflowInstanceID` 字段 |
| `domain/approval/model.go` | 新增 `AgentTaskID` 字段 |
| `application/agent/port.go` | `AgentTaskRepository` 新增 `Update` 方法（如没有） |
| `application/agent/service.go` | `CreateAndExecuteTask` 支持关联工作流；`ApproveTask`/`RejectTask` 增加联动 |
| `application/approval/port.go` | `ApprovalRepository` 新增 `UpdateInstanceAgentTaskID` 方法 |
| `infrastructure/persistence/agent_repo.go` | 实现新增方法 |
| `infrastructure/persistence/approval_repo.go` | 实现新增方法 |
| `infrastructure/persistence/database.go` | AutoMigrate 新增字段 |
| `interfaces/http/handler/agent.go` | CreateTask 解析 `workflow_id` 参数 |
| `app/container.go` | AgentService 注入 ApprovalService |

### 4.6 测试用例

| 用例 | 描述 |
|------|------|
| TestCreateTask_WithWorkflowID | 创建 Task 时自动创建 WorkflowInstance 并双向关联 |
| TestCreateTask_WithoutWorkflowID | 不提供 workflow_id 时行为与现有逻辑一致 |
| TestApproveTask_WithWorkflow | 审批通过时自动推进关联的工作流实例 |
| TestApproveTask_WithoutWorkflow | 无关联工作流时仅更新任务状态 |
| TestRejectTask_WithWorkflow | 拒绝时自动驳回关联的工作流实例 |
| TestRejectTask_WithoutWorkflow | 无关联工作流时仅更新任务状态 |
| TestApproveTask_WorkflowAlreadyFinished | 工作流实例已结束时忽略联动 |

---

## 5. 范围边界

### 5.1 在范围内

- 上述三个功能的全部代码实现
- 新增的 API 路由和 handler
- 表达式引擎（简化版 CEL）
- 数据模型字段新增
- 单元测试和集成测试
- config.example.yaml 更新

### 5.2 超出范围

| 边界 | 说明 |
|------|------|
| 前端代码 | 不涉及任何 UI 改动 |
| 工作流版本管理 | Version 字段不动 |
| 工作流可视化编辑器 | 不涉及 |
| 条件分支/会签/或签 | Definition 结构不变 |
| 数据质量真实执行引擎 | RunQualityCheck 保持模拟 |
| 数据血缘 | 不涉及 |
| 事件总线/消息队列 | 不引入 |
| 定时任务框架 | 不实现 retention 定时清理 |
| 现有 API 签名变更 | 不修改已有 API 的请求/响应格式 |
| 数据库迁移工具 | 依赖 GORM AutoMigrate |

---

## 6. 成功标准

### 功能一：工作流发布机制
- [ ] `POST /api/v1/workflows/:id/activate` 可以将 draft 工作流变为 active
- [ ] `POST /api/v1/workflows/:id/archive` 可以将 active/draft 工作流变为 archived
- [ ] `POST /api/v1/approvals` 对 draft 工作流返回 400
- [ ] 对 archived 工作流返回 400
- [ ] 所有状态转换测试通过

### 功能二：策略执行引擎
- [ ] 表达式引擎支持 `==`, `!=`, `>`, `>=`, `<`, `<=`, `in`, `not_in`, `contains`, `matches`, `&&`, `||`
- [ ] quality 策略校验 required_fields、max_config_size、allowed_types
- [ ] classification 策略校验 allowed_categories、category_required
- [ ] access 策略校验 allowed_roles、blocked_roles、require_role
- [ ] retention 策略校验配置存在性
- [ ] CreateTool 策略违规返回 400 + 详细违规信息
- [ ] ImportTools 先校验再创建，任一失败全部回滚
- [ ] 无匹配策略时工具创建正常通过
- [ ] 所有表达式引擎测试通过

### 功能三：Agent-审批打通
- [ ] 创建 Agent Task 时可通过 workflow_id 关联工作流实例
- [ ] Agent Task 审批通过时自动推进关联的工作流
- [ ] Agent Task 审批拒绝时自动驳回关联的工作流
- [ ] 未关联工作流时行为与现有逻辑一致
- [ ] 所有联动测试通过

### 全局
- [ ] `go build ./cmd/server` 编译通过
- [ ] `go test ./... -count=1` 全部通过，零失败
- [ ] 不破坏任何现有 API 的行为

---

## 7. 实施计划

| 阶段 | 内容 | 预估改动量 | 依赖 |
|------|------|-----------|------|
| **Phase 1** | 表达式引擎（tokenizer + parser + evaluator） | ~400 行 | 无 |
| **Phase 2** | 工作流发布机制（状态机 + API） | ~150 行 | 无 |
| **Phase 3** | 策略执行引擎（PolicyEngine + 集成） | ~300 行 | Phase 1 |
| **Phase 4** | Agent-审批打通（关联 + 联动） | ~200 行 | 无 |
| **Phase 5** | 全量测试 + 修复 | ~200 行 | Phase 1-4 |

Phase 1 和 Phase 2 可以并行，Phase 3 依赖 Phase 1，Phase 4 独立。
