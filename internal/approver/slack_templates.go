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
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Approved by: <@%s> (`%s`)", req.Spec.Approval.ApproverId, req.Spec.Approval.ApproverId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", "Status: ✅", false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Status updated at: `%s`", req.Spec.Approval.StatusUpdatedAt.Format(time.RFC1123)), false, false)),
		},
	}
}

func getSlackApprovalDetailsMessage(userId string, respondedAt time.Time) string {
	return fmt.Sprintf(
		"✅ Approved by <@%s> (`%s`) at %s UTC",
		userId,
		userId,
		respondedAt.UTC().Format("2006-01-02 15:04:05"),
	)
}

func getSlackRejectionDetailsMessage(userId string, respondedAt time.Time) string {
	return fmt.Sprintf(
		"⛔️ Rejected by <@%s> (`%s`) at %s UTC",
		userId,
		userId,
		respondedAt.UTC().Format("2006-01-02 15:04:05"),
	)
}

func getSlackSystemErrorMessage() string {
	return "⚠️ Looks like we messed up, please try again later or contact support if you're on a paid plan"
}

func getSlackMfaRejectedMessage(userId string) string {
	return fmt.Sprintf(
		"⚠️ <@%s> (`%s`) provided an invalid MFA token",
		userId,
		userId,
	)
}

func getSlackPendingMfaMessage(userId string) string {
	return fmt.Sprintf(
		"⏳ Approval is now pending MFA token by <@%s> (`%s`)",
		userId,
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
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Rejected by: <@%s> (`%s`)", req.Spec.Approval.ApproverId, req.Spec.Approval.ApproverId), false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", "Status: ⛔️", false, false)),
			slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("Status updated at: `%s`", req.Spec.Approval.StatusUpdatedAt.Format(time.RFC1123)), false, false)),
		},
	}
}

func getSlackUnauthorizedMessage(userId, action string) string {
	return fmt.Sprintf(
		"⚠️ <@%s> tried to perform the `%s` action but is not authorised to do that",
		userId,
		action,
	)
}
