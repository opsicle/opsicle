package approver

import (
	"context"
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/common"
	"opsicle/internal/integrations/telegram"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func getDefaultHandler(
	serviceLogs chan<- common.ServiceLog,
) func(context.Context, *telegram.Bot, *telegram.Update) {
	return func(ctx context.Context, bot *telegram.Bot, update *telegram.Update) {
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "chat[%v] << %s", update.ChatId, update.Message)
		// if it's not an approve/reject button call, do nothing
		if update.CallbackData == "" {
			return
		}
		action, notificationId, requestId, err := ParseTelegramApprovalCallbackData(update.CallbackData)
		if err != nil {
			logrus.Errorf("failed to parse callback: %s", err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				logrus.Errorf("failed to send error response to user: %s", err)
			}
			return
		}
		logrus.Infof("received callback with action[%s] on request[%s:%s]", action, notificationId, requestId)
		currentApprovalRequest, err := LoadApprovalRequest(ApprovalRequest{
			Spec: approvals.RequestSpec{
				Id:   requestId,
				Uuid: &notificationId,
			},
		})
		if err != nil {
			logrus.Errorf("failed to fetch request from cache: %v", err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				logrus.Errorf("failed to send error response to user: %s", err)
			}
			return
		}
		o, _ := json.MarshalIndent(currentApprovalRequest, "", "  ")
		logrus.Infof("retrieved approval request:\n%s", string(o))

		var response = ""
		if currentApprovalRequest.Spec.Approval != nil {
			if currentApprovalRequest.Spec.Approval.Status == "approved" {
				response = "This request has already been approved"
			} else {
				response = "This request has already been rejected"
			}
		} else if action == ActionApprove {
			approval := Approval{
				Spec: approvals.ApprovalSpec{
					ApproverId:      strconv.FormatInt(update.SenderId, 10),
					ApproverName:    update.SenderUsername,
					Id:              uuid.New().String(),
					RequestId:       currentApprovalRequest.Spec.Id,
					RequesterId:     currentApprovalRequest.Spec.RequesterId,
					RequesterName:   currentApprovalRequest.Spec.RequesterName,
					Status:          string(StatusApproved),
					StatusUpdatedAt: time.Now(),
					Telegram: approvals.TelegramResponseSpec{
						ChatId:   update.ChatId,
						UserId:   update.SenderId,
						Username: update.SenderUsername,
					},
					Type: PlatformTelegram,
				},
			}
			if err := CreateApproval(approval); err != nil {
				logrus.Errorf("failed to create approval[%s]: %s", approval.Spec.Id, err)
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
					logrus.Errorf("failed to send error response to user: %s", err)
				}
			}
			currentApprovalRequest.Spec.Approval = &approval.Spec
			if err := UpdateApprovalRequest(*currentApprovalRequest); err != nil {
				logrus.Errorf("failed to update approvalRequest[%s]: %s", currentApprovalRequest.Spec.GetUuid(), err)
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
					logrus.Errorf("failed to send error response to user: %s", err)
				}
			}
			if err := bot.UpdateMessage(
				update.ChatId,
				update.MessageId,
				fmt.Sprintf(
					"✅ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *APPROVED*\nApproval ID: `%s`",
					bot.EscapeMarkdown(currentApprovalRequest.Spec.Id),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.Message),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.RequesterName),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.RequesterId),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.Approval.Id),
				),
				nil,
			); err != nil {
				logrus.Errorf("failed to update message[%v]: %s", update.MessageId, err)
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
					logrus.Errorf("failed to send error response to user: %s", err)
				}
				return
			}
		} else if action == ActionReject {
			approval := Approval{
				Spec: approvals.ApprovalSpec{
					ApproverId:      strconv.FormatInt(update.SenderId, 10),
					ApproverName:    update.SenderUsername,
					Id:              uuid.New().String(),
					RequestId:       currentApprovalRequest.Spec.Id,
					RequesterId:     currentApprovalRequest.Spec.RequesterId,
					RequesterName:   currentApprovalRequest.Spec.RequesterName,
					Status:          string(StatusRejected),
					StatusUpdatedAt: time.Now(),
					Telegram: approvals.TelegramResponseSpec{
						ChatId:   update.ChatId,
						UserId:   update.SenderId,
						Username: update.SenderUsername,
					},
					Type: PlatformTelegram,
				},
			}
			if err := CreateApproval(approval); err != nil {
				logrus.Errorf("failed to create approval[%s]: %s", approval.Spec.Id, err)
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
					logrus.Errorf("failed to send error response to user: %s", err)
				}
			}
			currentApprovalRequest.Spec.Approval = &approval.Spec
			if err := UpdateApprovalRequest(*currentApprovalRequest); err != nil {
				logrus.Errorf("failed to update approvalRequest[%s]: %s", currentApprovalRequest.Spec.GetUuid(), err)
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
					logrus.Errorf("failed to send error response to user: %s", err)
				}
			}
			if err := bot.UpdateMessage(
				update.ChatId,
				update.MessageId,
				fmt.Sprintf(
					"❌ Approval request\nID: `%v`\nMessage: `%s`\nRequester: %s \\(`%s`\\)\n\nStatus: *REJECTED*\nApproval ID: `%s`",
					bot.EscapeMarkdown(currentApprovalRequest.Spec.Id),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.Message),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.RequesterName),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.RequesterId),
					bot.EscapeMarkdown(currentApprovalRequest.Spec.Approval.Id),
				),
				nil,
			); err != nil {
				logrus.Errorf("failed to update message[%v]: %s", update.MessageId, err)
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
					logrus.Errorf("failed to send error response to user: %s", err)
				}
				return
			}
		}
		if response != "" {
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, response); err != nil {
				serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send confirmation message: %v", err)
			}
		}
	}
}
func createTelegramApprovalKeyboard(approvalData, rejectionData string) models.ReplyMarkup {
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

func createTelegramApprovalCallbackData(action Action, notificationId string, requestId string) (callbackData string) {
	callbackData = fmt.Sprintf("%s:%s:%s", action, notificationId, requestId)
	return
}

func ParseTelegramApprovalCallbackData(callbackData string) (action Action, notificationId string, requestId string, err error) {
	splitCallbackData := strings.Split(callbackData, ":")
	if len(splitCallbackData) != 3 {
		return "", "", "", fmt.Errorf("failed to parse callback data: expected [{action}:{notificationId}:{requestId}] but received callbackData[%s]", callbackData)
	}
	action = Action(splitCallbackData[0])
	notificationId = splitCallbackData[1]
	requestId = splitCallbackData[2]
	err = nil
	return
}
