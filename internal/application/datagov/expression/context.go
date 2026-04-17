package expression

// Context 求值上下文
type Context struct {
	Tool ToolContext
	User UserContext
	Ctx  RequestContext
}

type ToolContext struct {
	Name, Type, Category, Description, Endpoint, Status, ConnectorID string
	Config                                                         map[string]any
}

type UserContext struct {
	ID, Role, Username string
}

type RequestContext struct {
	TenantID string
	Action   string
}
