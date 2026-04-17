package app

import (
	"time"

	agentapp "git.neolidy.top/neo/flowx/internal/application/agent"
	"git.neolidy.top/neo/flowx/internal/application/approval"
	"git.neolidy.top/neo/flowx/internal/application/auth"
	bpmnapp "git.neolidy.top/neo/flowx/internal/application/bpmn"
	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
	notificationapp "git.neolidy.top/neo/flowx/internal/application/notification"
	toolapp "git.neolidy.top/neo/flowx/internal/application/tool"
	aiapp "git.neolidy.top/neo/flowx/internal/application/ai"
	"git.neolidy.top/neo/flowx/internal/config"
	"git.neolidy.top/neo/flowx/internal/infrastructure/persistence"
	mcpif "git.neolidy.top/neo/flowx/internal/interfaces/mcp"
	"git.neolidy.top/neo/flowx/internal/interfaces/http/handler"
	"gorm.io/gorm"
)

// Container 应用服务容器，负责初始化所有服务
type Container struct {
	JWTService      auth.JWTService
	AuthHandler     *handler.AuthHandler
	ToolHandler     *handler.ToolHandler
	ApprovalHandler *handler.ApprovalHandler
	AgentHandler    *handler.AgentHandler
	DataGovHandler  *handler.DataGovHandler
	NotifHandler    *handler.NotificationHandler
	BPMNHandler     *handler.BPMNHandler
	approvalSvc     approval.ApprovalService
}

// NewContainer 创建并初始化所有服务
func NewContainer(db *gorm.DB, cfg config.Config, llmSvc aiapp.LLMService) *Container {
	c := &Container{}
	c.initAuthServices(db, cfg)
	c.initToolServices(db)
	c.initDataGovServices(db)
	c.initApprovalServices(db, llmSvc)
	c.initAgentServices(db)
	c.initNotificationServices(db)
	c.initBPMNServices(db)
	return c
}

// initAuthServices 初始化认证相关服务
func (c *Container) initAuthServices(db *gorm.DB, cfg config.Config) {
	jwtService := auth.NewJWTService(cfg.JWT.Secret, time.Duration(cfg.JWT.ExpireHours)*time.Hour)
	userRepo := persistence.NewUserRepository(db)
	authService := auth.NewAuthService(userRepo, jwtService)
	c.JWTService = jwtService
	c.AuthHandler = handler.NewAuthHandler(authService)
}

// initToolServices 初始化工具相关服务
func (c *Container) initToolServices(db *gorm.DB) {
	toolRepo := persistence.NewToolRepository(db)
	connectorRepo := persistence.NewConnectorRepository(db)
	dataPolicyRepo := persistence.NewDataPolicyRepository(db)
	dataAssetRepo := persistence.NewDataAssetRepository(db)
	dataRuleRepo := persistence.NewDataQualityRuleRepository(db)
	dataCheckRepo := persistence.NewDataQualityCheckRepository(db)
	toolService := toolapp.NewToolService(toolRepo, connectorRepo, dataPolicyRepo, dataAssetRepo, dataRuleRepo, dataCheckRepo)
	excelService := toolapp.NewExcelService(toolRepo)
	c.ToolHandler = handler.NewToolHandler(toolService, excelService)
}

// initDataGovServices 初始化数据治理相关服务
func (c *Container) initDataGovServices(db *gorm.DB) {
	dataPolicyRepo := persistence.NewDataPolicyRepository(db)
	dataAssetRepo := persistence.NewDataAssetRepository(db)
	dataRuleRepo := persistence.NewDataQualityRuleRepository(db)
	dataCheckRepo := persistence.NewDataQualityCheckRepository(db)
	dataGovService := datagovapp.NewDataGovService(dataPolicyRepo, dataAssetRepo, dataRuleRepo, dataCheckRepo)
	dataGovExcelService := datagovapp.NewDataGovExcelService(dataPolicyRepo, dataAssetRepo, dataRuleRepo)
	c.DataGovHandler = handler.NewDataGovHandler(dataGovService, dataGovExcelService)
}

// initApprovalServices 初始化审批相关服务
func (c *Container) initApprovalServices(db *gorm.DB, llmSvc aiapp.LLMService) {
	approvalRepo := persistence.NewApprovalRepository(db)
	approvalSvc := approval.NewApprovalService(approvalRepo, llmSvc)
	c.ApprovalHandler = handler.NewApprovalHandler(approvalSvc)
	c.approvalSvc = approvalSvc
}

// initAgentServices 初始化 Agent 相关服务
func (c *Container) initAgentServices(db *gorm.DB) {
	registry := mcpif.NewToolRegistry()
	engine := agentapp.NewAgentEngine(registry)
	engine.RegisterAgent(agentapp.NewToolOrchestrationAgent(), 1)
	engine.RegisterAgent(agentapp.NewApprovalAgent(), 1)
	engine.RegisterAgent(agentapp.NewDataQualityAgent(), 1)
	agentTaskRepo := persistence.NewAgentTaskRepository(db)
	agentSvc := agentapp.NewAgentService(engine, agentTaskRepo, c.approvalSvc)
	c.AgentHandler = handler.NewAgentHandler(agentSvc)
}

// initNotificationServices 初始化通知相关服务
func (c *Container) initNotificationServices(db *gorm.DB) {
	notifRepo := persistence.NewNotificationRepository(db)
	templateRepo := persistence.NewNotificationTemplateRepository(db)
	preferenceRepo := persistence.NewNotificationPreferenceRepository(db)
	notifService := notificationapp.NewNotificationService(notifRepo, templateRepo, preferenceRepo)
	c.NotifHandler = handler.NewNotificationHandler(notifService)
}

// initBPMNServices 初始化 BPMN 流程相关服务
func (c *Container) initBPMNServices(db *gorm.DB) {
	engine := bpmnapp.NewEngine()
	defRepo := persistence.NewProcessDefinitionRepository(db)
	instRepo := persistence.NewProcessInstanceRepository(db)
	taskRepo := persistence.NewProcessTaskRepository(db)
	historyRepo := persistence.NewExecutionHistoryRepository(db)
	processSvc := bpmnapp.NewProcessService(engine, defRepo, instRepo, taskRepo, historyRepo)
	c.BPMNHandler = handler.NewBPMNHandler(processSvc)
}
