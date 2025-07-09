package approver

import (
	"context"
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"opsicle/internal/integrations/telegram"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

func getDefaultHandler(
	serviceLogs chan<- common.ServiceLog,
) func(context.Context, *telegram.Bot, *telegram.Update) {
	return func(ctx context.Context, bot *telegram.Bot, update *telegram.Update) {
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "chat[%v] << %s", update.ChatId, update.Message)
		// handle potential mfa authentications
		if update.Message != "" && update.IsReply && update.ReplyMessageId != 0 {
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "processing potential mfa response from chat[%v]", update.ChatId)
			handleTelegramMfaResponse(handleTelegramMfaResponseOpts{
				Bot:            bot,
				ChatId:         update.ChatId,
				Message:        update.Message,
				MessageId:      update.MessageId,
				SenderId:       update.SenderId,
				SenderUsername: update.SenderUsername,
				ServiceLogs:    serviceLogs,
			})
		}

		// if it's not an approve/reject button call, do nothing
		if update.CallbackData == "" {
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "ignoring message in chat[%v], not a callback", update.ChatId)
			return
		}

		// handle callbacks
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "processing callback data from chat[%v]", update.ChatId)
		action, requestUuid, requestId, err := parseTelegramApprovalCallbackData(update.CallbackData)
		if err != nil {
			serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to parse callback: %s", err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send response to user: %s", err)
			}
			return
		}
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received callback with action[%s] on request[%s:%s]", action, requestUuid, requestId)
		currentApprovalRequest, err := LoadApprovalRequest(ApprovalRequest{
			Spec: approvals.RequestSpec{
				Id:   requestId,
				Uuid: &requestUuid,
			},
		})
		if err != nil {
			serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to fetch request from cache: %v", err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
			}
			return
		}
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "retrieved approvalRequest[%s:%s]", currentApprovalRequest.Spec.Id, currentApprovalRequest.Spec.GetUuid())

		var response = ""
		if currentApprovalRequest.Spec.Approval != nil {
			switch currentApprovalRequest.Spec.Approval.Status {
			case approvals.StatusApproved:
				response = "This request has already been approved"
			case approvals.StatusRejected:
				response = "This request has already been rejected"
			default:
				response = "We've detected an unknown status, please try again from the start"
			}
		} else if action == ActionApprove {
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "handling approval action by user[%v/%s] in chat[%v]", update.SenderId, update.SenderUsername, update.ChatId)
			handleTelegramApproval(handleTelegramApprovalOpts{
				Bot:            bot,
				ChatId:         update.ChatId,
				MessageId:      update.MessageId,
				Req:            *currentApprovalRequest,
				SenderId:       update.SenderId,
				SenderUsername: update.SenderUsername,
				ServiceLogs:    serviceLogs,
			})
		} else if action == ActionReject {
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "handling rejection action by user[%v/%s] in chat[%v]", update.SenderId, update.SenderUsername, update.ChatId)
			handleTelegramRejection(handleTelegramRejectionOpts{
				Bot:            bot,
				ChatId:         update.ChatId,
				MessageId:      update.MessageId,
				Req:            *currentApprovalRequest,
				SenderId:       update.SenderId,
				SenderUsername: update.SenderUsername,
			})
		} else {
			serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "unknown action or state reached; callbackData[%s]; approvalRequest[%s:%s]", update.CallbackData, currentApprovalRequest.Spec.Id, currentApprovalRequest.Spec.GetUuid())
			response = "An unknown action has been triggered, please try again from the start"
		}
		if response != "" {
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "responding to chat[%v]: %v", update.ChatId, err)
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

func createTelegramApprovalCallbackData(action Action, requestUuid string, requestId string) (callbackData string) {
	callbackData = fmt.Sprintf("%s:%s:%s", action, requestUuid, requestId)
	return
}

type handleTelegramApprovalOpts struct {
	Bot            *telegram.Bot
	ChatId         int64
	MessageId      int
	Req            ApprovalRequest
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan<- common.ServiceLog
}

func handleTelegramApproval(opts handleTelegramApprovalOpts) {
	if isSenderAuthorized := isSenderAuthorizedToRespond(isSenderAuthorizedToRespondOpts{
		Bot:            opts.Bot,
		ChatId:         opts.ChatId,
		MessageId:      opts.MessageId,
		Req:            opts.Req,
		SenderId:       opts.SenderId,
		SenderUsername: opts.SenderUsername,
		ServiceLogs:    opts.ServiceLogs,
	}); !isSenderAuthorized {
		if err := opts.Bot.ReplyMessage(
			opts.ChatId,
			opts.MessageId,
			getUnauthorizedMessage(),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", opts.MessageId, err)
			if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
			}
		}
		return
	}
	for _, telegramTarget := range opts.Req.Spec.Telegram {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "evaluating telegram target: user[%v] in chat[%v]", opts.SenderId, opts.ChatId)
		if isTelegramTargetMatchingSender(isTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
			Target:         telegramTarget,
		}) {
			if telegramTarget.MfaSeed != nil {
				if err := opts.Bot.UpdateMessage(
					opts.ChatId,
					opts.MessageId,
					getPendingMfaMessage(opts.Req),
				); err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v] in chat[%v]: %s", opts.MessageId, opts.ChatId, err)
				}
				pendingMfaCacheKey := CreatePendingMfaCacheKey(
					fmt.Sprintf("%v", opts.ChatId),
					fmt.Sprintf("%v", opts.SenderId),
				)
				pendingMfaData := pendingMfa{
					ApprovalRequestMessageId: fmt.Sprintf("%v", opts.MessageId),
					ChatId:                   fmt.Sprintf("%v", opts.ChatId),
					MfaSeed:                  *telegramTarget.MfaSeed,
					RequestId:                opts.Req.Spec.Id,
					RequestUuid:              opts.Req.Spec.GetUuid(),
					UserId:                   fmt.Sprintf("%v", opts.SenderId),
				}
				pendingMfaString, _ := json.Marshal(pendingMfaData)
				Cache.Set(pendingMfaCacheKey, string(pendingMfaString), 60*time.Second)
				if err := opts.Bot.SendMessage(
					opts.ChatId,
					getMfaRequestMessage(opts.Req),
					&models.ForceReply{
						ForceReply: true,
					},
				); err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v] in chat[%v]: %s", opts.MessageId, opts.ChatId, err)
				}
				return
			}
			if err := processApproval(processApprovalOpts{
				ApprovalMessageId: opts.MessageId,
				CurrentMessageId:  opts.MessageId,
				Bot:               opts.Bot,
				ChatId:            opts.ChatId,
				Req:               opts.Req,
				SenderId:          opts.SenderId,
				SenderUsername:    opts.SenderUsername,
				ServiceLogs:       opts.ServiceLogs,
			}); err != nil {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
				return
			}
		}
	}
}

type handleTelegramMfaResponseOpts struct {
	Bot            *telegram.Bot
	ChatId         int64
	Message        string
	MessageId      int
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan<- common.ServiceLog
}

func handleTelegramMfaResponse(opts handleTelegramMfaResponseOpts) {
	mfaToken := opts.Message
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "processing mfa token from chat[%v] << %s", opts.ChatId, mfaToken)
	pendingMfaString, err := Cache.Get(CreatePendingMfaCacheKey(
		fmt.Sprintf("%v", opts.ChatId),
		fmt.Sprintf("%v", opts.SenderId),
	))
	if err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to retrieve a cache mfa: %s", err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send response to user: %s", err)
		}
		return
	}
	var pendingMfaData pendingMfa
	if err := json.Unmarshal([]byte(pendingMfaString), &pendingMfaData); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to parse pending mfa error: %s", err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send response to user: %s", err)
		}
		return
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "processing approvalRequest[%s:%s]...", pendingMfaData.RequestId, pendingMfaData.RequestUuid)
	approvalRequestQuery := ApprovalRequest{
		Spec: approvals.RequestSpec{
			Id:   pendingMfaData.RequestId,
			Uuid: &pendingMfaData.RequestUuid,
		},
	}
	currentApprovalRequest, err := LoadApprovalRequest(approvalRequestQuery)
	if err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to load approvalRequest[%s:%s]: %s", pendingMfaData.RequestId, pendingMfaData.RequestUuid, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "successfully retrieved approvalRequest[%s:%s]...", currentApprovalRequest.Spec.Id, currentApprovalRequest.Spec.GetUuid())

	if isSenderAuthorized := isSenderAuthorizedToRespond(isSenderAuthorizedToRespondOpts{
		Bot:            opts.Bot,
		ChatId:         opts.ChatId,
		MessageId:      opts.MessageId,
		Req:            *currentApprovalRequest,
		SenderId:       opts.SenderId,
		SenderUsername: opts.SenderUsername,
		ServiceLogs:    opts.ServiceLogs,
	}); !isSenderAuthorized {
		if err := opts.Bot.ReplyMessage(
			opts.ChatId,
			opts.MessageId,
			getUnauthorizedMessage(),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", opts.MessageId, err)
			if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
			}
		}
		return
	}
	for _, telegramTarget := range currentApprovalRequest.Spec.Telegram {
		if isTelegramTargetMatchingSender(isTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
			Target:         telegramTarget,
		}) {
			if telegramTarget.MfaSeed != nil {
				valid, err := auth.ValidateTotpToken(*telegramTarget.MfaSeed, mfaToken)
				if err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to validate totpToken[%v]: %s", mfaToken, err)
					continue
				}
				if !valid {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "totpToken[%v] was not vavlid", mfaToken)
					continue
				}
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "user[%v] in chat[%v] approved the transaction", opts.SenderId, opts.ChatId)

				approvalMessageId, err := strconv.ParseInt(pendingMfaData.ApprovalRequestMessageId, 10, 0)
				if err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to parse approvalRequestId[%s]: %s", pendingMfaData.ApprovalRequestMessageId, err)
					if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
						opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
					}
					continue
				}
				if err := processApproval(processApprovalOpts{
					ApprovalMessageId: int(approvalMessageId),
					CurrentMessageId:  opts.MessageId,
					Bot:               opts.Bot,
					ChatId:            opts.ChatId,
					Req:               *currentApprovalRequest,
					SenderId:          opts.SenderId,
					SenderUsername:    opts.SenderUsername,
					ServiceLogs:       opts.ServiceLogs,
				}); err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process approvalRequest[%s]: %s", currentApprovalRequest.Spec.GetUuid(), err)
					return
				}
			}
		}
	}
}

type handleTelegramRejectionOpts struct {
	Bot            *telegram.Bot
	ChatId         int64
	MessageId      int
	Req            ApprovalRequest
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan common.ServiceLog
}

func handleTelegramRejection(opts handleTelegramRejectionOpts) {
	if isSenderAuthorized := isSenderAuthorizedToRespond(isSenderAuthorizedToRespondOpts{
		Bot:            opts.Bot,
		ChatId:         opts.ChatId,
		MessageId:      opts.MessageId,
		Req:            opts.Req,
		SenderId:       opts.SenderId,
		SenderUsername: opts.SenderUsername,
		ServiceLogs:    opts.ServiceLogs,
	}); !isSenderAuthorized {
		if err := opts.Bot.ReplyMessage(
			opts.ChatId,
			opts.MessageId,
			getUnauthorizedMessage(),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", opts.MessageId, err)
			if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
			}
		}
		return
	}
	approval := Approval{
		Spec: approvals.ApprovalSpec{
			ApproverId:      strconv.FormatInt(opts.SenderId, 10),
			ApproverName:    opts.SenderUsername,
			Id:              uuid.New().String(),
			RequestId:       opts.Req.Spec.Id,
			RequesterId:     opts.Req.Spec.RequesterId,
			RequesterName:   opts.Req.Spec.RequesterName,
			Status:          approvals.StatusRejected,
			StatusUpdatedAt: time.Now(),
			Telegram: approvals.TelegramResponseSpec{
				ChatId:   opts.ChatId,
				UserId:   opts.SenderId,
				Username: opts.SenderUsername,
			},
			Type: approvals.PlatformTelegram,
		},
	}
	if err := CreateApproval(approval); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to create approval[%s]: %s", approval.Spec.Id, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
	}
	opts.Req.Spec.Approval = &approval.Spec
	if err := UpdateApprovalRequest(opts.Req); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
	}
	if err := opts.Bot.UpdateMessage(
		opts.ChatId,
		opts.MessageId,
		getRejectedMessage(opts.Req),
		nil,
	); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", opts.MessageId, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return
	}
}

type isSenderAuthorizedToRespondOpts struct {
	Bot            *telegram.Bot
	ChatId         int64
	MessageId      int
	Req            ApprovalRequest
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan<- common.ServiceLog
}

func isSenderAuthorizedToRespond(opts isSenderAuthorizedToRespondOpts) bool {
	isSenderAuthorized := false
	for _, telegramTarget := range opts.Req.Spec.Telegram {
		if isTelegramTargetMatchingSender(isTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
			Target:         telegramTarget,
		}) {
			isSenderAuthorized = true
			break
		}
	}
	return isSenderAuthorized
}

type isTelegramTargetMatchingSenderOpts struct {
	ChatId         int64
	SenderId       int64
	SenderUsername string
	Target         approvals.TelegramRequestSpec
}

func isTelegramTargetMatchingSender(opts isTelegramTargetMatchingSenderOpts) bool {
	if opts.Target.ChatId == opts.ChatId {
		if opts.Target.UserId != nil && *opts.Target.UserId != opts.SenderId {
			return false
		}
		if opts.Target.Username != nil && *opts.Target.Username != opts.SenderUsername {
			return false
		}
		return true
	}
	return false
}

func parseTelegramApprovalCallbackData(callbackData string) (action Action, notificationId string, requestId string, err error) {
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

type processApprovalOpts struct {
	ApprovalMessageId int
	CurrentMessageId  int
	Bot               *telegram.Bot
	ChatId            int64
	Req               ApprovalRequest
	SenderId          int64
	SenderUsername    string
	ServiceLogs       chan<- common.ServiceLog
}

func processApproval(opts processApprovalOpts) error {
	approval := Approval{
		Spec: approvals.ApprovalSpec{
			ApproverId:      strconv.FormatInt(opts.SenderId, 10),
			ApproverName:    opts.SenderUsername,
			Id:              uuid.New().String(),
			RequestId:       opts.Req.Spec.Id,
			RequesterId:     opts.Req.Spec.RequesterId,
			RequesterName:   opts.Req.Spec.RequesterName,
			Status:          approvals.StatusApproved,
			StatusUpdatedAt: time.Now(),
			Telegram: approvals.TelegramResponseSpec{
				ChatId:   opts.ChatId,
				UserId:   opts.SenderId,
				Username: opts.SenderUsername,
			},
			Type: approvals.PlatformTelegram,
		},
	}
	if err := CreateApproval(approval); err != nil {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.CurrentMessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to create approval[%s]: %s", approval.Spec.Id, err)
	}
	opts.Req.Spec.Approval = &approval.Spec
	if err := UpdateApprovalRequest(opts.Req); err != nil {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.CurrentMessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
	}
	if err := opts.Bot.UpdateMessage(
		opts.ChatId,
		opts.ApprovalMessageId,
		getApprovedMessage(opts.Req),
		nil,
	); err != nil {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.CurrentMessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to update message[%v]: %s", opts.ApprovalMessageId, err)
	}
	return nil
}
