# FlowX API 接口文档

> Base URL: `/api/v1`
> 认证方式: Bearer Token（JWT）
> 所有认证接口需携带 `Authorization: Bearer <token>` 请求头
> 多租户隔离: 通过 JWT 中的 `tenant_id` 自动实现

---

## 通用响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

分页响应:
```json
{
  "code": 0,
  "message": "success",
  "data": [ ... ],
  "pagination": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

错误响应:
```json
{
  "code": -1,
  "message": "错误描述",
  "data": null
}
```

---

## 1. 健康检查

### GET /health
无需认证。

**响应 200:**
```json
{ "code": 0, "message": "success", "data": { "status": "ok", "version": "dev" } }
```

---

## 2. 认证管理

### POST /auth/register
用户注册。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | ✅ | 用户名，3-100 字符 |
| email | string | ✅ | 邮箱 |
| password | string | ✅ | 密码，6-100 字符 |
| tenant_id | string | ✅ | 租户 ID |

**响应 201:**
```json
{ "code": 0, "data": { "user": { ... }, "token": "jwt_token" } }
```

### POST /auth/login
用户登录。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| username | string | ✅ |
| password | string | ✅ |

**响应 200:**
```json
{ "code": 0, "data": { "token": "jwt_token" } }
```

### GET /auth/profile
获取当前用户信息。需认证。

**响应 200:**
```json
{ "code": 0, "data": { "id": "...", "username": "...", "email": "...", "role": "..." } }
```

---

## 3. 工具管理

### POST /tools
创建工具。创建时自动执行数据策略校验。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 工具名称 |
| type | string | ✅ | 工具类型（eda/cae/data_pipeline 等） |
| description | string | | 描述 |
| connector_id | string | | 关联连接器 ID |
| config | object | | 工具配置（JSON） |
| status | string | | 状态，默认 active |
| endpoint | string | | 工具端点 |
| icon | string | | 图标 |
| category | string | | 分类 |

**响应 201:** 工具对象

**错误 422 — 策略校验失败:**
```json
{
  "code": -1,
  "message": "策略校验失败",
  "data": [
    { "policy_name": "质量标准", "policy_type": "quality", "rule_key": "required_fields", "message": "工具缺少必填字段: endpoint" }
  ]
}
```

### GET /tools
工具列表。

**Query 参数:** `type`, `status`, `category`, `keyword`, `page`, `page_size`

### GET /tools/:id
获取工具详情。

### PUT /tools/:id
更新工具。更新时同样执行策略校验。

**请求体:** 同创建，所有字段可选（部分更新）。

### DELETE /tools/:id
删除工具。

### POST /tools/export
导出工具为 Excel。

**请求体:** `multipart/form-data`，字段 `file`。

### GET /tools/export/:task_id
查询导出任务状态。

### POST /tools/import
从 Excel 导入工具。导入前对所有工具批量执行策略校验，任一失败则全部不创建。

**请求体:** `multipart/form-data`，字段 `file`。

---

## 4. 连接器管理

### POST /connectors
创建连接器。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 连接器名称 |
| type | string | ✅ | 连接器类型 |
| endpoint | string | ✅ | 连接器端点 |
| config | object | | 连接器配置 |
| status | string | | 状态，默认 active |
| auth_type | string | | 认证类型 |
| auth_config | object | | 认证配置 |

### GET /connectors
连接器列表。**Query:** `type`, `status`, `keyword`, `page`, `page_size`

### GET /connectors/:id
连接器详情。

### PUT /connectors/:id
更新连接器。

### DELETE /connectors/:id
删除连接器。

---

## 5. 工作流管理

### POST /workflows
创建工作流（状态为 draft）。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 工作流名称 |
| type | string | ✅ | 类型（tool_deploy/data_review/change_request/custom） |
| description | string | | 描述 |
| definition | object | ✅ | 步骤定义（JSON） |

**definition 结构示例:**
```json
{
  "steps": [
    { "name": "技术评审", "approvers": ["user-id-1"] },
    { "name": "安全审批", "approvers": ["user-id-2"] }
  ]
}
```

### GET /workflows
工作流列表。**Query:** `page`, `page_size`

### GET /workflows/:id
工作流详情。

### POST /workflows/:id/activate
激活工作流（draft → active）。只有 draft 状态可激活。

**响应 200:** 更新后的工作流对象

**错误 400:** `只有草稿状态的工作流可以激活` / `工作流已归档`

### POST /workflows/:id/archive
归档工作流（draft/active → archived）。

**错误 400:** `工作流已归档`

---

## 6. 审批管理

### POST /approvals
发起审批。**只有 active 状态的工作流可以发起审批。**

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workflow_id | string | ✅ | 工作流 ID |
| title | string | ✅ | 审批标题 |
| context | object | | 审批上下文数据 |

**错误 400:** `工作流未激活，无法发起审批` / `工作流已归档，无法发起审批`

### GET /approvals
审批实例列表。**Query:** `page`, `page_size`

### GET /approvals/pending
获取当前用户的待审批列表。

### GET /approvals/:id
审批实例详情。

### POST /approvals/:id/approve
审批通过。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| instance_id | string | ✅ |
| comment | string | |

### POST /approvals/:id/reject
审批驳回。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| instance_id | string | ✅ |
| comment | string | ✅ |

### POST /approvals/:id/forward
转审。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| instance_id | string | ✅ |
| to_approver_id | string | ✅ |
| comment | string | |

### POST /approvals/:id/cancel
取消审批实例。

### GET /approvals/:id/suggestion
获取 AI 审批建议。

**响应 200:**
```json
{ "code": 0, "data": "建议审批通过，工具配置符合安全规范" }
```

---

## 7. Agent 智能体

### GET /agent/tools
获取可用工具列表（MCP Tool 定义）。

### POST /agent/tasks
创建并执行 Agent 任务。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | ✅ | 任务类型 |
| description | string | ✅ | 任务描述 |
| context | object | | 任务上下文 |
| steps | array | ✅ | 任务步骤（至少 1 个） |
| require_approval | boolean | | 是否需要人工审批 |

**steps 元素:**
| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 步骤类型（tool_execute/approval_review/data_check） |
| description | string | 步骤描述 |
| params | object | 步骤参数 |

**响应 201:** 任务结果

### GET /agent/tasks
任务列表。**Query:** `status`, `page`, `page_size`

### GET /agent/tasks/:id
任务详情。

### POST /agent/tasks/:id/approve
审批通过 Agent 任务。如果任务关联了工作流实例，会自动推进工作流。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| comment | string | |

### POST /agent/tasks/:id/reject
拒绝 Agent 任务。如果任务关联了工作流实例，会自动驳回工作流。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| comment | string | ✅ |

---

## 8. 数据治理 — 策略

### POST /data-policies
创建数据策略。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 策略名称 |
| type | string | ✅ | 类型（quality/classification/access/retention） |
| description | string | | 描述 |
| scope | string | | 适用范围（global/tool_type/category） |
| scope_value | string | | 范围值 |
| rules | object | | 策略规则（JSON） |
| priority | int | | 优先级（数值越大越高） |
| status | string | | 状态，默认 active |

**rules 示例（quality 类型）:**
```json
{
  "required_fields": ["endpoint", "config.auth_type"],
  "description_required": true
}
```

**rules 示例（access 类型 + 表达式）:**
```json
{
  "allowed_roles": ["admin", "tool_manager"],
  "expression": "tool.type == \"eda\" && user.role == \"admin\""
}
```

### GET /data-policies
策略列表。**Query:** `type`, `status`, `scope`, `keyword`, `page`, `page_size`

### GET /data-policies/:id
策略详情。

### PUT /data-policies/:id
更新策略。所有字段可选（部分更新）。

### DELETE /data-policies/:id
删除策略。

### POST /data-policies/export
导出策略为 Excel。

### POST /data-policies/import
从 Excel 导入策略。

---

## 9. 数据治理 — 资产

### POST /data-assets
创建数据资产。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 资产名称 |
| type | string | ✅ | 类型（dataset/model/report/config） |
| source | string | | 数据来源 |
| source_id | string | | 来源实体 ID |
| description | string | | 描述 |
| format | string | | 数据格式（csv/json/parquet/excel） |
| schema | object | | 数据结构定义 |
| size | int64 | | 数据大小（字节） |
| record_count | int64 | | 记录数 |
| location | string | | 存储位置 |
| tags | object | | 标签 |
| classification | string | | 分类（public/internal/confidential/restricted），默认 internal |
| owner_id | string | | 负责人 ID |
| status | string | | 状态，默认 active |

### GET /data-assets
资产列表。**Query:** `type`, `status`, `classification`, `source`, `keyword`, `page`, `page_size`

### GET /data-assets/:id
资产详情。

### PUT /data-assets/:id
更新资产。

### DELETE /data-assets/:id
删除资产。

### POST /data-assets/export
导出资产为 Excel。

### POST /data-assets/import
从 Excel 导入资产。

---

## 10. 数据治理 — 质量规则

### POST /data-quality/rules
创建数据质量规则。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 规则名称 |
| type | string | ✅ | 类型（not_null/unique/range/format/custom） |
| target_asset | string | | 目标数据资产 ID |
| target_field | string | | 目标字段名 |
| description | string | | 描述 |
| config | object | | 规则配置（范围值、正则等） |
| severity | string | | 严重级别（critical/warning/info），默认 warning |
| status | string | | 状态，默认 active |

### GET /data-quality/rules
规则列表。**Query:** `type`, `status`, `severity`, `target_asset`, `keyword`, `page`, `page_size`

### GET /data-quality/rules/:id
规则详情。

### PUT /data-quality/rules/:id
更新规则。

### DELETE /data-quality/rules/:id
删除规则。

### POST /data-quality/rules/export
导出规则为 Excel。

### POST /data-quality/rules/import
从 Excel 导入规则。

---

## 11. 数据治理 — 质量检查

### POST /data-quality/checks/run
执行数据质量检查。

**请求体:**
| 字段 | 类型 | 必填 |
|------|------|------|
| rule_id | string | ✅ |
| asset_id | string | ✅ |

**响应 200:** 检查记录（含 pass_rate、duration、result）

### GET /data-quality/checks
检查记录列表。**Query:** `rule_id`, `asset_id`, `status`, `triggered_by`, `page`, `page_size`

### GET /data-quality/checks/:id
检查记录详情。

---

## 12. 通知

### POST /notifications
创建通知。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | ✅ | 通知标题 |
| content | string | ✅ | 通知内容 |
| type | string | ✅ | 类型（system/approval/task/alert） |
| category | string | | 分类 |
| channel | string | | 渠道（in_app/email/sms/webhook） |
| ref_type | string | | 关联类型 |
| ref_id | string | | 关联 ID |
| extra | object | | 扩展数据 |

### GET /notifications
通知列表。**Query:** `type`, `channel`, `is_read`, `page`, `page_size`

### GET /notifications/unread-count
获取未读通知数量。

### GET /notifications/:id
通知详情。

### PUT /notifications/:id
更新通知。所有字段可选（部分更新）。

### DELETE /notifications/:id
删除通知。

### PUT /notifications/:id/read
标记为已读。

### PUT /notifications/read-all
全部标记为已读。

### POST /notifications/send
发送通知（使用模板）。

---

## 13. 通知模板

### POST /notification-templates
创建通知模板。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 模板名称 |
| type | string | ✅ | 模板类型 |
| channel | string | ✅ | 渠道 |
| subject | string | | 主题（邮件） |
| content | string | ✅ | 模板内容（支持 `{{.变量名}}` 占位符） |
| variables | object | | 变量定义 |

### GET /notification-templates
模板列表。**Query:** `type`, `channel`, `page`, `page_size`

### GET /notification-templates/:id
模板详情。

### PUT /notification-templates/:id
更新模板。

### DELETE /notification-templates/:id
删除模板。

---

## 14. 通知偏好

### POST /notification-preferences
创建通知偏好。

### GET /notification-preferences
偏好列表。**Query:** `user_id`, `type`, `channel`, `page`, `page_size`

### PUT /notification-preferences/:id
更新偏好。

### DELETE /notification-preferences/:id
删除偏好。

---

## 15. BPMN 流程管理

### POST /process-definitions
部署流程定义。请求体为 YAML 格式的流程定义字符串。

**请求体:** raw YAML（Content-Type: text/plain 或 application/json）

**YAML 格式示例:**
```yaml
id: leave-process
name: 请假流程
version: 1
status: active
elements:
  - id: start
    type: startEvent
    name: 开始
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
    name: 结束
    incoming: [flow2]
```

**响应 201:** 流程定义对象

### GET /process-definitions
流程定义列表。**Query:** `status`, `keyword`, `page`, `page_size`

### GET /process-definitions/:id
流程定义详情。

### POST /process-instances
启动流程实例。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| definition_id | string | ✅ | 流程定义 ID |
| variables | object | | 流程变量 |

**响应 201:** 流程实例对象

### GET /process-instances
流程实例列表。**Query:** `status`, `page`, `page_size`

### GET /process-instances/:id
流程实例详情。

### POST /process-instances/:id/suspend
挂起流程实例。

### POST /process-instances/:id/resume
恢复流程实例。

### POST /process-instances/:id/cancel
取消流程实例。

### GET /process-instances/:id/tasks
获取流程实例的任务列表。

### POST /process-instances/:id/tasks/:taskId/complete
完成任务。

**请求体:**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| submitted_data | object | | 提交数据 |

### GET /process-tasks/pending
获取待办任务列表。**Query:** `assignee`

---

## 接口总览

| 模块 | 接口数 | 说明 |
|------|--------|------|
| 健康检查 | 1 | 无需认证 |
| 认证 | 3 | 注册/登录/个人信息 |
| 工具 | 8 | CRUD + 导入导出 |
| 连接器 | 5 | CRUD |
| 工作流 | 5 | CRUD + 激活/归档 |
| 审批 | 8 | 发起/列表/通过/驳回/转审/取消/AI建议 |
| Agent | 6 | 工具列表/任务CRUD/审批 |
| 数据策略 | 7 | CRUD + 导入导出 |
| 数据资产 | 7 | CRUD + 导入导出 |
| 质量规则 | 7 | CRUD + 导入导出 |
| 质量检查 | 3 | 执行/列表/详情 |
| 通知 | 8 | CRUD + 已读/发送 |
| 通知模板 | 5 | CRUD |
| 通知偏好 | 4 | CRUD |
| BPMN 流程 | 12 | 定义部署/实例管理/任务处理 |
| **合计** | **89** | |
