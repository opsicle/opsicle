package approvals

const (
	PlatformSlack    Platform = "slack"
	PlatformTelegram Platform = "telegram"
)

const (
	StatusApproved   Status = "approved"
	StatusError      Status = "error"
	StatusNew        Status = "new"
	StatusMfaInvalid Status = "invalidmfa"
	StatusRejected   Status = "rejected"
)

const (
	CallbackWebhook CallbackType = "webhook"
)

type Platform string
type Status string
type CallbackType string
