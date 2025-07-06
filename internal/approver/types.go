package approver

import "opsicle/internal/approvals"

type ApprovalRequest struct {
	Spec approvals.RequestSpec
}

type Approval struct {
	Spec approvals.ApprovalSpec
}
