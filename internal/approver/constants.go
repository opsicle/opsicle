package approver

type Action string

const (
	approvalRequestCachePrefix        = "approvreq"
	ActionApprove              Action = "approve"
	ActionReject               Action = "reject"
)
