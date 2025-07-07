package approver

import (
	"fmt"
	"opsicle/internal/integrations/telegram"
)

func getApprovalRequestMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"⚠️ Approval request\nID: `%s`\nMessage: `%s`\nRequester: %s \\(`%s`\\)",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
	)
}

func getApprovedMessage(req ApprovalRequest) string {
	return fmt.Sprintf(
		"✅ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *APPROVED*\nApproval ID: `%s`",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.Approval.Id,
	)
}

func getRejectedMessage(req ApprovalRequest) string {
	return fmt.Sprintf(
		"❌ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *REJECTED*\nApproval ID: `%s`",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.Approval.Id,
	)
}
