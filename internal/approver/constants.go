package approver

type Action string

type Status string

const (
	approvalRequestCachePrefix        = "approvreq"
	approvalCachePrefix               = "approval"
	ActionApprove              Action = "approve"
	ActionReject               Action = "reject"
)
