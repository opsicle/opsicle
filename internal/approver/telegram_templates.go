package approver

import (
	"fmt"
	"opsicle/internal/integrations/telegram"

	"github.com/go-telegram/bot/models"
)

// getTelegramApprovalKeyboard returns the approve/reject keyboard
// that users will use to reject/approve an approval request
func getTelegramApprovalKeyboard(approvalData, rejectionData string) *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{{
			models.InlineKeyboardButton{
				Text:         "Approve",
				CallbackData: approvalData,
			},
			models.InlineKeyboardButton{
				Text:         "Reject",
				CallbackData: rejectionData,
			},
		}},
	}
}

// getTelegramApprovalRequestMessage returns the message template for
// use when a new approval request is sent
func getTelegramApprovalRequestMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"*⚠️ Incoming Approval Request*\n"+
			"*Request ID*: `%s`\n\n"+
			"*Message*: ```\n%s\n```"+
			"*Requester ID*: `%s`\n"+
			"*Requester Name*: `%s`\n"+
			"*Request UUID*: `%s`\n"+
			"\nStatus: *PENDING*",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.GetUuid(),
	)
}

// getTelegramApprovedMessage returns the message template that
// replaces the approval request message once the approval request
// is approved
func getTelegramApprovedMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"*✅ Approval Request \\- Approved*\n"+
			"*Request ID*: `%s`\n\n"+
			"*Message*: ```\n%s\n```"+
			"*Requester ID*: `%s`\n"+
			"*Requester Name*: `%s`\n"+
			"*Responder Name*: @%s\n"+
			"*Responder ID*: `%s`\n"+
			"*Request UUID*: `%s`\n"+
			"*Approval ID*: `%s`\n"+
			"*Timestamp*: `%s`\n"+
			"\nStatus: *APPROVED*",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterId,
		req.Spec.RequesterName,
		req.Spec.Approval.ApproverName,
		req.Spec.Approval.ApproverId,
		req.Spec.GetUuid(),
		req.Spec.Approval.Id,
		req.Spec.Approval.StatusUpdatedAt.Format("2006-01-02T15:03:04-0700"),
	)
}

// getTelegramApproveMessage returns the message to be sent to the chat
// once an approval request has been approved
func getTelegramApproveMessage(req ApprovalRequest, senderId int64, senderName string) string {
	userReference := fmt.Sprintf("user with ID `%v`", senderId)
	if senderName != "" {
		userReference = fmt.Sprintf("@%s", senderName)
	}
	return telegram.FormatInputf(
		"✅ Request has been approved by "+userReference+"\n\n"+
			"*Request UUID*: `%s`",
		req.Spec.GetUuid(),
	)
}

// getTelegramInfoMessage returns the message to be sent to a user when
// the `/info` command is sent to the bot
func getTelegramInfoMessage(update *telegram.Update) string {
	return telegram.FormatInputf(
		"ℹ️ Information\nThis chat's ID: `%v`\nYour user ID: `%v`\nYour username: `%v`\nYour current message ID: `%v`",
		fmt.Sprintf("%v", update.ChatId),
		fmt.Sprintf("%v", update.SenderId),
		update.SenderUsername,
		fmt.Sprintf("%v", update.MessageId),
	)
}

// getTelegramMfaRequestMessage returns the message to be sent to a
// user to request for the mfa token authentication
func getTelegramMfaRequestMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"🔓 Reply to this message with your assigned MFA token\n\nRequest UUID: `%s`",
		req.Spec.GetUuid(),
	)
}

// getTelegramMfaRejectedMessage returns the message to be sent to the
// chat when an mfa token is wrong/invalid
func getTelegramMfaRejectedMessage(req ApprovalRequest, senderId int64, senderName string) string {
	userReference := fmt.Sprintf("user with ID `%v`", senderId)
	if senderName != "" {
		userReference = fmt.Sprintf("@%s", senderName)
	}
	return telegram.FormatInputf(
		"⛔️ Wrong MFA entered by "+userReference+"\n\n"+
			"*Request UUID*: `%s`",
		req.Spec.GetUuid(),
	)
}

// getTelegramPendingMfaMessage returns the message to be used to replace
// the approval request message when an mfa is pending from a responder
func getTelegramPendingMfaMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"*⏳ Approval Request \\- Pending MFA*\n"+
			"*Request ID*: `%s`\n\n"+
			"*Message*: ```\n%s\n```"+
			"*Requester ID*: `%s`\n"+
			"*Requester Name*: `%s`\n"+
			"*Request UUID*: `%s`\n"+
			"\nStatus: *PENDING MFA*",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterName,
		req.Spec.RequesterId,
		req.Spec.GetUuid(),
	)
}

// getTelegramRejectedMessage returns the message template for replacing
// the approval request message after the approval request has been rejected
func getTelegramRejectedMessage(req ApprovalRequest) string {
	return telegram.FormatInputf(
		"*❌ Approval Request \\- Rejected*\n"+
			"*Request ID*: `%s`\n\n"+
			"*Message*: ```\n%s\n```"+
			"*Requester ID*: `%s`\n"+
			"*Requester Name*: `%s`\n"+
			"*Responder Name*: @%s\n"+
			"*Responder ID*: `%s`\n"+
			"*Request UUID*: `%s`\n"+
			"*Approval ID*: `%s`\n"+
			"*Timestamp*: `%s`\n"+
			"\nStatus: *REJECTED*",
		req.Spec.Id,
		req.Spec.Message,
		req.Spec.RequesterId,
		req.Spec.RequesterName,
		req.Spec.Approval.ApproverName,
		req.Spec.Approval.ApproverId,
		req.Spec.GetUuid(),
		req.Spec.Approval.Id,
		req.Spec.Approval.StatusUpdatedAt.Format("2006-01-02T15:03:04-0700"),
	)
}

// getTelegramRejectMessage returns the message to be sent to the chat
// when the approval request has been rejected
func getTelegramRejectMessage(req ApprovalRequest, senderId int64, senderName string) string {
	userReference := fmt.Sprintf("user with ID `%v`", senderId)
	if senderName != "" {
		userReference = fmt.Sprintf("@%s", senderName)
	}
	return telegram.FormatInputf(
		"❌ Request has been rejected by "+userReference+"\n\n"+
			"Request UUID:```%s```",
		req.Spec.GetUuid(),
	)
}

// getTelegramSystemErrorMessage returns the generic error message for
// when we fuck up, check the logs when it happens
func getTelegramSystemErrorMessage() string {
	return "⚠️ Looks like we messed up, please try again later or contact support if you're on a paid plan"
}

// getTelegramUnauthorizedMessage returns a message for telling the
// end-user that they are not authorised
func getTelegramUnauthorizedMessage() string {
	return "⚠️ You are not authorised to perform this action"
}
