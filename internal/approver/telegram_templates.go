package approver

import (
	"opsicle/internal/integrations/telegram"
)

func getTelegramApprovalRequestMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"⚠️ Approval request\nID: `%s`\nMessage: `%s`\nRequester: %s \\(`%s`\\)",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
	)
}

func getTelegramApprovedMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"✅ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *APPROVED*\nApproval ID: `%s`\nRequest ID: `%s`",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.Approval.Id,
		req.Spec.GetUuid(),
	)
}

func getTelegramMfaRequestMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"🔓 Reply to this message with your assigned MFA token \\(expires in 60 seconds\\)\nRequest ID: `%s`",
		req.Spec.GetUuid(),
	)
}

func getTelegramPendingMfaMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"⏳ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *PENDING MFA*",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
	)
}

func getTelegramRejectedMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"❌ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *REJECTED*\nApproval ID: `%s`",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.Approval.Id,
	)
}

func getTelegramUnauthorizedMessage() string {
	return "⚠️ You are not authorised to perform this action"
}
