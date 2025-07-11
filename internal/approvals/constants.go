package approvals

const (
	PlatformSlack    Platform = "slack"
	PlatformTelegram Platform = "telegram"
)

const (
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type Platform string
type Status string
