# FlowX

企业级智能工具治理与流程编排平台，集成策略引擎、BPMN 流程、Agent 智能体和审批工作流。

## 特性

- 🔧 **工具治理** — 工具全生命周期管理，策略校验自动拦截违规操作
- 📋 **BPMN 流程引擎** — 支持 9 种元素类型、并行/排他/包容网关、子流程
- 🤖 **Agent 智能体** — 工具编排、审批分析、数据质量检查三种内置 Agent
- ✅ **审批工作流** — 多步审批、转审、AI 智能建议、工作流发布机制
- 🛡️ **策略执行引擎** — 自定义表达式 + 四种策略类型（quality/classification/access/retention）
- 📊 **数据治理** — 数据资产注册、质量规则、质量检查、Excel 导入导出
- 🔔 **通知系统** — 多渠道通知、模板管理、偏好设置
- 🏢 **多租户** — 基于 JWT 的租户隔离
- 🔌 **MCP 协议** — 支持 Model Context Protocol 工具集成

## 技术栈

| 层级 | 技术 |
|------|------|
| 语言 | Go 1.24 |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | PostgreSQL / SQLite（测试） |
| 缓存 | Redis |
| 认证 | JWT (golang-jwt/v5) |
| 配置 | Viper |
| AI/LLM | Ollama (OpenAI 兼容接口) |
| 协议 | MCP (Model Context Protocol) |
| 容器 | Docker + Docker Compose |

## 项目结构

```
flowx/
├── cmd/server/              # 入口
├── internal/
│   ├── app/container.go     # 依赖注入容器
│   ├── application/         # 应用服务层
│   │   ├── agent/           # Agent 智能体（引擎 + 编排 + 服务）
│   │   ├── approval/        # 审批工作流
│   │   ├── auth/            # JWT 认证
│   │   ├── bpmn/            # BPMN 流程引擎
│   │   ├── datagov/         # 数据治理（策略引擎 + 表达式引擎）
│   │   ├── notification/    # 通知服务
│   │   └── tool/            # 工具管理
│   ├── domain/              # 领域模型
│   ├── infrastructure/      # 基础设施
│   │   ├── persistence/     # GORM 持久化
│   │   ├── cache/           # Redis 缓存
│   │   └── server/          # HTTP 服务
│   └── interfaces/          # 对外接口
│       ├── http/            # REST API (Gin)
│       │   ├── handler/     # HTTP 处理器
│       │   └── middleware/   # 中间件（认证/租户/超时/上传）
│       └── mcp/             # MCP 协议服务
├── pkg/                     # 公共包
│   ├── errors/              # 错误类型
│   ├── pagination/          # 分页
│   ├── response/            # 统一响应
│   └── tenant/              # 租户上下文
├── tests/e2e/               # E2E 场景测试
├── docs/                    # 设计文档
├── config.example.yaml      # 配置示例
├── Dockerfile               # 多阶段构建
├── docker-compose.yml       # 编排（PostgreSQL + Redis + App）
└── Makefile                 # 构建命令
```

## 快速开始

### 环境要求

- Go 1.24+
- PostgreSQL 14+（生产）/ SQLite（开发测试）
- Redis 7+（可选，用于缓存）

### 本地开发

```bash
# 1. 克隆
git clone https://github.com/jiangfire/flowx.git
cd flowx

# 2. 安装依赖
go mod tidy

# 3. 复制配置
cp config.example.yaml config.yaml
# 编辑 config.yaml 修改数据库连接和 JWT 密钥

# 4. 运行
go run ./cmd/server
```

服务默认监听 `http://localhost:8080`。

### Docker 部署

```bash
# 一键启动（PostgreSQL + Redis + FlowX）
docker compose up -d

# 查看日志
docker compose logs -f flowx-server

# 停止
docker compose down
```

### Makefile 命令

```bash
make build          # 编译
make run            # 运行
make test           # 全量测试
make test-short     # 快速测试
make docker-build   # Docker 构建
make docker-up      # Docker 启动
make docker-down    # Docker 停止
```

## 配置说明

参考 `config.example.yaml`，主要配置项：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.port` | 服务端口 | `8080` |
| `server.mode` | 运行模式 (debug/release/test) | `debug` |
| `database.host` | 数据库地址 | `localhost` |
| `database.port` | 数据库端口 | `5432` |
| `database.dbname` | 数据库名 | `flowx` |
| `redis.host` | Redis 地址 | `localhost` |
| `jwt.secret` | JWT 签名密钥 | **必须修改** |
| `jwt.expire_hours` | Token 过期时间（小时） | `24` |
| `llm.endpoint` | LLM 服务地址（OpenAI 兼容） | `http://localhost:11434/v1` |
| `llm.model` | LLM 模型名称 | `qwen2.5:7b` |

也支持环境变量，格式为 `FLOWX_` 前缀 + 大写配置路径，例如 `FLOWX_DATABASE_HOST`、`FLOWX_JWT_SECRET`。

## API 概览

服务启动后访问 `http://localhost:8080/api/v1`，共 **89 个接口**：

| 模块 | 接口数 | 说明 |
|------|--------|------|
| 认证 | 3 | 注册 / 登录 / 个人信息 |
| 工具 | 8 | CRUD + 导入导出 |
| 连接器 | 5 | CRUD |
| 工作流 | 5 | CRUD + 激活 / 归档 |
| 审批 | 8 | 发起 / 列表 / 通过 / 驳回 / 转审 / 取消 / AI 建议 |
| Agent | 6 | 工具列表 / 任务 CRUD / 审批 |
| 数据策略 | 7 | CRUD + 导入导出 |
| 数据资产 | 7 | CRUD + 导入导出 |
| 质量规则 | 7 | CRUD + 导入导出 |
| 质量检查 | 3 | 执行 / 列表 / 详情 |
| 通知 | 8 | CRUD + 已读 / 发送 |
| 通知模板 | 5 | CRUD |
| 通知偏好 | 4 | CRUD |
| BPMN 流程 | 12 | 定义部署 / 实例管理 / 任务处理 |
| 健康检查 | 1 | 状态检查 |

### 认证

所有接口（除健康检查和注册登录外）需要 Bearer Token：

```bash
# 注册
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"123456","tenant_id":"tenant-001"}'

# 登录
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

# 带认证访问
curl http://localhost:8080/api/v1/tools \
  -H "Authorization: Bearer <your-token>"
```

### 策略引擎表达式语法

策略校验支持类 CEL 简化表达式：

```
# 比较运算
tool.type == "eda"
tool.config.max_size <= 100

# 逻辑运算
tool.type == "eda" && user.role == "admin"
tool.category == "visualization" || tool.category == "ai_model"

# 集合运算
tool.type in ["eda", "cae", "data_pipeline"]
tool.type not_in ["internal"]

# 字符串运算
tool.endpoint matches "^https://.*"
tool.description contains "数据分析"

# 空值检查
tool.description != empty
```

可访问的上下文变量：`tool.*`（工具属性）、`user.*`（用户信息）、`ctx.*`（请求上下文）

### BPMN 流程定义示例

```yaml
id: leave-process
name: 请假流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    outgoing: [flow1]
  - id: flow1
    type: sequenceFlow
    outgoing: [task1]
  - id: task1
    type: userTask
    name: 部门审批
    assignee: manager
    incoming: [flow1]
    outgoing: [flow2]
  - id: flow2
    type: sequenceFlow
    outgoing: [end]
  - id: end
    type: endEvent
    incoming: [flow2]
```

## 测试

```bash
# 全量测试（630+ 用例）
go test ./... -count=1

# 含 E2E 场景测试
go test ./tests/e2e/... -v

# 单个模块
go test ./internal/application/bpmn/... -v
```

测试覆盖：
- **单元测试** — 每个模块独立测试，使用内存 mock 或 SQLite 内存数据库
- **E2E 场景测试** — 工具治理链路、BPMN 流程生命周期、Agent-审批联动

## 架构设计

```
┌─────────────────────────────────────────────────┐
│                  HTTP / MCP 接口层                │
├─────────────────────────────────────────────────┤
│              应用服务层 (Application)              │
│  ┌────────┐ ┌────────┐ ┌──────┐ ┌────────────┐  │
│  │ 工具   │ │ BPMN  │ │ Agent│ │  审批工作流  │  │
│  │ 管理   │ │ 引擎  │ │ 智能体│ │            │  │
│  └───┬────┘ └────────┘ └──┬───┘ └─────┬──────┘  │
│      │                    │           │          │
│  ┌───┴────────────────────┴───────────┴──────┐   │
│  │           策略执行引擎 + 表达式引擎         │   │
│  └───────────────────────────────────────────┘   │
├─────────────────────────────────────────────────┤
│              领域层 (Domain)                      │
├─────────────────────────────────────────────────┤
│           基础设施层 (Infrastructure)             │
│     GORM 持久化 │ Redis 缓存 │ HTTP 服务        │
└─────────────────────────────────────────────────┘
```

## License

MIT
