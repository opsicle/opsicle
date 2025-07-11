package approver

import (
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

func getSlackApprovalRequestBlocks(req *ApprovalRequest, callbackData string) slack.Blocks {
	return slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewHeaderBlock(
				slack.NewTextBlockObject("plain_text", "📥 New Approval Request", false, false),
			),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request Message\n```\n%s\n```", req.Spec.Message), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Requester: `%s`", req.Spec.RequesterName), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Requester ID: `%s`", req.Spec.RequesterId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request ID: `%s`", req.Spec.Id), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request UUID: `%s`", req.Spec.GetUuid()), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", "Status: `PENDING`", false, false)),
			slack.NewActionBlock("approval_actions_"+req.Spec.GetUuid(),
				slack.NewButtonBlockElement(string(ActionApprove), callbackData, slack.NewTextBlockObject("plain_text", "Approve", false, false)),
				slack.NewButtonBlockElement(string(ActionReject), callbackData, slack.NewTextBlockObject("plain_text", "Reject", false, false)),
			),
		},
	}
}

func getSlackApprovedBlocks(req *ApprovalRequest) slack.Blocks {
	return slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewHeaderBlock(
				slack.NewTextBlockObject("plain_text", "✅ Approved Approval Request", false, false),
			),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request Message\n```\n%s\n```", req.Spec.Message), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Requester: `%s`", req.Spec.RequesterName), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Requester ID: `%s`", req.Spec.RequesterId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request ID: `%s`", req.Spec.Id), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request UUID: `%s`", req.Spec.GetUuid()), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Approved by: <@%s>", req.Spec.Approval.ApproverId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", "Status: ✅", false, false)),
		},
	}
}

func getSlackApprovedMessage(req ApprovalRequest) string {
	return fmt.Sprintf(
		"✅ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s (`%s`)\n\nStatus: *APPROVED*\nApproval ID: `%s`\nRequest ID: `%s`",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.Approval.Id,
		req.Spec.GetUuid(),
	)
}

func getSlackApprovalDetailsMessage(userId string, respondedAt time.Time) string {
	return fmt.Sprintf(
		"✅ Approved by <@%s> at %s UTC",
		userId,
		respondedAt.UTC().Format("2006-01-02 15:04:05"),
	)
}

func getSlackRejectionDetailsMessage(userId string, respondedAt time.Time) string {
	return fmt.Sprintf(
		"⛔️ Rejected by <@%s> at %s UTC",
		userId,
		respondedAt.UTC().Format("2006-01-02 15:04:05"),
	)
}

func getSlackSystemErrorMessage() string {
	return "⚠️ Looks like we messed up, please try again later or contact support if you're on a paid plan"
}

func getSlackMfaRejectedMessage(userId string) string {
	return fmt.Sprintf(
		"⚠️ <@%s> provided an invalid MFA token",
		userId,
	)
}

func getSlackPendingMfaMessage(userId string) string {
	return fmt.Sprintf(
		"⏳ Approval is pending MFA token by <@%s>",
		userId,
	)
}

func getSlackRejectedBlocks(req *ApprovalRequest) slack.Blocks {
	return slack.Blocks{
		BlockSet: []slack.Block{
			slack.NewHeaderBlock(
				slack.NewTextBlockObject("plain_text", "⛔️ Rejected Approval Request", false, false),
			),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request Message\n```\n%s\n```", req.Spec.Message), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Requester: `%s`", req.Spec.RequesterName), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Requester ID: `%s`", req.Spec.RequesterId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request ID: `%s`", req.Spec.Id), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Request UUID: `%s`", req.Spec.GetUuid()), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Rejected by: <@%s>", req.Spec.Approval.ApproverId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", "Status: ⛔️", false, false)),
		},
	}
}

func getSlackRejectedMessage(req ApprovalRequest) string {
	return fmt.Sprintf(
		"❌ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s (`%s`)\n\nStatus: *REJECTED*\nApproval ID: `%s`",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.Approval.Id,
	)
}

func getSlackUnauthorizedMessage() string {
	return "⚠️ You are not authorised to perform this action"
}
