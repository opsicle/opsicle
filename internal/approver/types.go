package approver

type ApprovalRequest struct {
	Id             string `json:"id"`
	Requester      string `json:"requester"`
	Approver       string `json:"approver"`
	AutomationName string `json:"automationName"`
	Chat           string `json:"chatId"`
}

type Cache interface {
	Get(key string) (value string)
	Set(key string, value string)
	Unset(key string)
	Ping() error
}
