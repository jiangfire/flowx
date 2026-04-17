package http

import (
	"time"

	"git.neolidy.top/neo/flowx/internal/app"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/handler"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 注册所有路由
func SetupRouter(r *gin.Engine, container *app.Container) {
	// Swagger 文档（无需认证）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", handler.HealthCheck)

		// 认证路由（无需认证）
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", container.AuthHandler.Register)
			authGroup.POST("/login", container.AuthHandler.Login)
		}

		// 需要认证的路由
		authRequired := v1.Group("")
		authRequired.Use(middleware.AuthMiddleware(container.JWTService))
		authRequired.Use(middleware.RequestTimeout(30 * time.Second))
		authRequired.Use(middleware.TenantMiddleware())
		{
			authRequired.GET("/auth/profile", container.AuthHandler.Profile)

			// 工具路由
			authRequired.POST("/tools", container.ToolHandler.CreateTool)
			authRequired.GET("/tools", container.ToolHandler.ListTools)
			authRequired.GET("/tools/export/:task_id", container.ToolHandler.GetExportStatus)
			authRequired.POST("/tools/export", container.ToolHandler.ExportTools)
			authRequired.POST("/tools/import", container.ToolHandler.ImportTools)
			authRequired.GET("/tools/:id", container.ToolHandler.GetTool)
			authRequired.PUT("/tools/:id", container.ToolHandler.UpdateTool)
			authRequired.DELETE("/tools/:id", container.ToolHandler.DeleteTool)

			// 连接器路由
			authRequired.POST("/connectors", container.ToolHandler.CreateConnector)
			authRequired.GET("/connectors", container.ToolHandler.ListConnectors)
			authRequired.GET("/connectors/:id", container.ToolHandler.GetConnector)
			authRequired.PUT("/connectors/:id", container.ToolHandler.UpdateConnector)
			authRequired.DELETE("/connectors/:id", container.ToolHandler.DeleteConnector)

			// 工作流路由
			authRequired.POST("/workflows", container.ApprovalHandler.CreateWorkflow)
			authRequired.GET("/workflows", container.ApprovalHandler.ListWorkflows)
			authRequired.GET("/workflows/:id", container.ApprovalHandler.GetWorkflow)
			authRequired.POST("/workflows/:id/activate", container.ApprovalHandler.ActivateWorkflow)
			authRequired.POST("/workflows/:id/archive", container.ApprovalHandler.ArchiveWorkflow)

			// 审批路由
			authRequired.POST("/approvals", container.ApprovalHandler.StartApproval)
			authRequired.GET("/approvals", container.ApprovalHandler.ListInstances)
			authRequired.GET("/approvals/pending", container.ApprovalHandler.GetMyPendingApprovals)
			authRequired.GET("/approvals/:id", container.ApprovalHandler.GetInstance)
			authRequired.POST("/approvals/:id/approve", container.ApprovalHandler.Approve)
			authRequired.POST("/approvals/:id/reject", container.ApprovalHandler.Reject)
			authRequired.POST("/approvals/:id/forward", container.ApprovalHandler.Forward)
			authRequired.POST("/approvals/:id/cancel", container.ApprovalHandler.CancelInstance)
			authRequired.GET("/approvals/:id/suggestion", container.ApprovalHandler.GetSuggestion)

			// Agent 路由
			authRequired.GET("/agent/tools", container.AgentHandler.ListTools)
			authRequired.POST("/agent/tasks", container.AgentHandler.CreateTask)
			authRequired.GET("/agent/tasks", container.AgentHandler.ListTasks)
			authRequired.GET("/agent/tasks/:id", container.AgentHandler.GetTask)
			authRequired.POST("/agent/tasks/:id/approve", container.AgentHandler.ApproveTask)
			authRequired.POST("/agent/tasks/:id/reject", container.AgentHandler.RejectTask)

			// 数据策略路由
			authRequired.POST("/data-policies", container.DataGovHandler.CreatePolicy)
			authRequired.GET("/data-policies", container.DataGovHandler.ListPolicies)
			authRequired.GET("/data-policies/:id", container.DataGovHandler.GetPolicy)
			authRequired.PUT("/data-policies/:id", container.DataGovHandler.UpdatePolicy)
			authRequired.DELETE("/data-policies/:id", container.DataGovHandler.DeletePolicy)
			authRequired.POST("/data-policies/export", container.DataGovHandler.ExportPolicies)
			authRequired.POST("/data-policies/import", container.DataGovHandler.ImportPolicies)

			// 数据资产路由
			authRequired.POST("/data-assets", container.DataGovHandler.CreateAsset)
			authRequired.GET("/data-assets", container.DataGovHandler.ListAssets)
			authRequired.GET("/data-assets/:id", container.DataGovHandler.GetAsset)
			authRequired.PUT("/data-assets/:id", container.DataGovHandler.UpdateAsset)
			authRequired.DELETE("/data-assets/:id", container.DataGovHandler.DeleteAsset)
			authRequired.POST("/data-assets/export", container.DataGovHandler.ExportAssets)
			authRequired.POST("/data-assets/import", container.DataGovHandler.ImportAssets)

			// 数据质量规则路由
			authRequired.POST("/data-quality/rules", container.DataGovHandler.CreateRule)
			authRequired.GET("/data-quality/rules", container.DataGovHandler.ListRules)
			authRequired.GET("/data-quality/rules/:id", container.DataGovHandler.GetRule)
			authRequired.PUT("/data-quality/rules/:id", container.DataGovHandler.UpdateRule)
			authRequired.DELETE("/data-quality/rules/:id", container.DataGovHandler.DeleteRule)
			authRequired.POST("/data-quality/rules/export", container.DataGovHandler.ExportRules)
			authRequired.POST("/data-quality/rules/import", container.DataGovHandler.ImportRules)

			// 数据质量检查路由
			authRequired.GET("/data-quality/checks", container.DataGovHandler.ListChecks)
			authRequired.GET("/data-quality/checks/:id", container.DataGovHandler.GetCheck)
			authRequired.POST("/data-quality/checks/run", container.DataGovHandler.RunQualityCheck)

			// 通知路由
			authRequired.POST("/notifications", container.NotifHandler.CreateNotification)
			authRequired.GET("/notifications", container.NotifHandler.ListNotifications)
			authRequired.GET("/notifications/unread-count", container.NotifHandler.GetUnreadCount)
			authRequired.PUT("/notifications/read-all", container.NotifHandler.MarkAllAsRead)
			authRequired.GET("/notifications/:id", container.NotifHandler.GetNotification)
			authRequired.PUT("/notifications/:id", container.NotifHandler.UpdateNotification)
			authRequired.DELETE("/notifications/:id", container.NotifHandler.DeleteNotification)
			authRequired.PUT("/notifications/:id/read", container.NotifHandler.MarkAsRead)
			authRequired.POST("/notifications/send", container.NotifHandler.SendNotification)

			// 通知模板路由
			authRequired.POST("/notification-templates", container.NotifHandler.CreateTemplate)
			authRequired.GET("/notification-templates", container.NotifHandler.ListTemplates)
			authRequired.GET("/notification-templates/:id", container.NotifHandler.GetTemplate)
			authRequired.PUT("/notification-templates/:id", container.NotifHandler.UpdateTemplate)
			authRequired.DELETE("/notification-templates/:id", container.NotifHandler.DeleteTemplate)

			// 通知偏好路由
			authRequired.POST("/notification-preferences", container.NotifHandler.CreatePreference)
			authRequired.GET("/notification-preferences", container.NotifHandler.ListPreferences)
			authRequired.PUT("/notification-preferences/:id", container.NotifHandler.UpdatePreference)
			authRequired.DELETE("/notification-preferences/:id", container.NotifHandler.DeletePreference)

			// 流程定义
			authRequired.POST("/process-definitions", container.BPMNHandler.DeployDefinition)
			authRequired.GET("/process-definitions", container.BPMNHandler.ListDefinitions)
			authRequired.GET("/process-definitions/:id", container.BPMNHandler.GetDefinition)

			// 流程实例
			authRequired.POST("/process-instances", container.BPMNHandler.StartProcess)
			authRequired.GET("/process-instances", container.BPMNHandler.ListProcessInstances)
			authRequired.GET("/process-instances/:id", container.BPMNHandler.GetProcessInstance)
			authRequired.POST("/process-instances/:id/suspend", container.BPMNHandler.SuspendProcess)
			authRequired.POST("/process-instances/:id/resume", container.BPMNHandler.ResumeProcess)
			authRequired.POST("/process-instances/:id/cancel", container.BPMNHandler.CancelProcess)
			authRequired.GET("/process-instances/:id/tasks", container.BPMNHandler.GetProcessTasks)
			authRequired.POST("/process-instances/:id/tasks/:taskId/complete", container.BPMNHandler.CompleteTask)

			// 流程任务
			authRequired.GET("/process-tasks/pending", container.BPMNHandler.GetPendingTasks)
		}
	}
}
