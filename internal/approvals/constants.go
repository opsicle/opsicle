package approvals

const (
	PlatformTelegram Platform = "telegram"
)

const (
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type Platform string
type Status string
