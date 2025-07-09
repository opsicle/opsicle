package approver

type Action string

type Status string

const (
	approvalRequestCachePrefix        = "approvreq"
	approvalCachePrefix               = "approval"
	pendingMfaCachePrefix             = "pendingmfa"
	ActionApprove              Action = "approve"
	ActionReject               Action = "reject"
)
