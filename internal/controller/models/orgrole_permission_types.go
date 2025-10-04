package models

type Action uint
type Resource string

var (
	ActionCreate  Action = 0b1
	ActionView    Action = 0b10
	ActionUpdate  Action = 0b100
	ActionDelete  Action = 0b1000
	ActionExecute Action = 0b10000
	ActionManage  Action = 0b100000

	ActionSetReporter Action = ActionView
	ActionSetUser     Action = ActionSetReporter | ActionExecute
	ActionSetOperator Action = ActionSetUser | ActionCreate | ActionUpdate | ActionDelete
	ActionSetAdmin    Action = ActionSetOperator | ActionManage

	ResourceAutomationLogs Resource = "automation_logs"
	ResourceAutomations    Resource = "automations"
	ResourceTemplates      Resource = "templates"
	ResourceOrg            Resource = "org"
	ResourceOrgBilling     Resource = "org_billing"
	ResourceOrgConfig      Resource = "org_config"
	ResourceOrgUser        Resource = "org_user"
)
