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

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

func getDefaultHandler(
	serviceLogs chan<- common.ServiceLog,
) func(context.Context, *telegram.Bot, *telegram.Update) {
	return func(ctx context.Context, bot *telegram.Bot, update *telegram.Update) {

		if update.Message != "" {

			if strings.HasPrefix(update.Message, "/") {

				// handle commands

				serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "processing command from chat[%v]", update.ChatId)
				handleTelegramCommands(handleTelegramCommandsOpts{
					Bot:         bot,
					Update:      update,
					ServiceLogs: serviceLogs,
				})
				return

			} else if update.IsReply && update.ReplyMessageId != 0 {

				// handle potential mfa authentications

				serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "processing potential mfa response from chat[%v]", update.ChatId)
				handleTelegramMfaResponse(handleTelegramMfaResponseOpts{
					Bot:                 bot,
					ChatId:              update.ChatId,
					Message:             update.Message,
					MessageId:           update.MessageId,
					MfaRequestMessageId: update.ReplyMessageId,
					SenderId:            update.SenderId,
					SenderUsername:      update.SenderUsername,
					ServiceLogs:         serviceLogs,
				})
				return
			}
		}

		// if it's not an approve/reject button call, do nothing

		if update.CallbackData == "" {
			serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "processing invalid message from chat[%v]", update.ChatId)
			handleTelegramInvalidEvent(handleTelegramInvalidEventOpts{
				Bot:         bot,
				ChatId:      update.ChatId,
				MessageId:   update.MessageId,
				SenderId:    update.SenderId,
				ServiceLogs: serviceLogs,
			})
			return
		}

		// handle user responses via buttons

		serviceLogs <- common.ServiceLogf(common.LogLevelDebug, "processing callback data from chat[%v]", update.ChatId)
		action, requestUuid, err := parseTelegramApprovalCallbackData(update.CallbackData)
		if err != nil {
			serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to parse callback: %s", err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send response to user: %s", err)
			}
			return
		}
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received callback with action[%s] on approvalRequest[%s]", action, requestUuid)
		approvalRequest := &ApprovalRequest{
			Spec: approvals.RequestSpec{
				Uuid: &requestUuid,
			},
		}
		if err := approvalRequest.Load(); err != nil {
			serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to fetch request from cache: %v", err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
			}
			return
		}
		serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "retrieved approvalRequest[%s:%s]", approvalRequest.Spec.Id, approvalRequest.Spec.GetUuid())

		var response = ""
		if approvalRequest.Spec.Approval != nil {
			switch approvalRequest.Spec.Approval.Status {
			case approvals.StatusApproved:
				response = "This request has already been approved"
			case approvals.StatusRejected:
				response = "This request has already been rejected"
			default:
				response = "We've detected an unknown status, please try again from the start"
			}
			serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "responding to chat[%v]: %v", update.ChatId, err)
			if err := bot.ReplyMessage(update.ChatId, update.MessageId, response); err != nil {
				serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send confirmation message: %v", err)
			}
		} else {
			authorizedTargets := getAuthorizedTelegramTargets(getAuthorizedTelegramTargetsOpts{
				ChatId:         update.ChatId,
				Req:            *approvalRequest,
				SenderId:       update.SenderId,
				SenderUsername: update.SenderUsername,
				ServiceLogs:    serviceLogs,
			})
			isSenderAuthorised := len(authorizedTargets) > 0
			if !isSenderAuthorised {
				if err := bot.ReplyMessage(update.ChatId, update.MessageId, getTelegramUnauthorizedMessage()); err != nil {
					serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", update.MessageId, err)
					if err := bot.ReplyMessage(update.ChatId, update.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
						serviceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
					}
				}
				return
			}
			handleTelegramResponse(handleTelegramResponseOpts{
				Action:         action,
				Bot:            bot,
				ChatId:         update.ChatId,
				MessageId:      update.MessageId,
				Req:            *approvalRequest,
				SenderId:       update.SenderId,
				SenderUsername: update.SenderUsername,
				ServiceLogs:    serviceLogs,
			})
			return
		}
	}
}

type handleTelegramResponseOpts struct {
	Action         Action
	Bot            *telegram.Bot
	ChatId         int64
	MessageId      int
	Req            ApprovalRequest
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan<- common.ServiceLog
}

func handleTelegramResponse(opts handleTelegramResponseOpts) {
	isTargetAuthorized := false
	for _, telegramTarget := range opts.Req.Spec.Telegram {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "evaluating telegram target: user[%v] in chat[%v]", opts.SenderId, opts.ChatId)
		if isTelegramTargetMatchingSender(isTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
			Target:         telegramTarget,
		}) {
			isTargetAuthorized = true
			switch opts.Action {
			case ActionApprove:
				{
					if telegramTarget.MfaSeed != nil {
						handleTelegramMfaRequired(opts, telegramTarget)
					} else {
						handleTelegramApproval(opts)
					}
				}
			case ActionReject:
				{
					handleTelegramRejection(opts)
				}
			}
			break
		}
	}
	if !isTargetAuthorized {

	}
}

func handleTelegramMfaRequired(opts handleTelegramResponseOpts, target approvals.TelegramRequestSpec) {
	if err := opts.Bot.UpdateMessage(
		opts.ChatId,
		opts.MessageId,
		getTelegramPendingMfaMessage(opts.Req),
		nil,
		// getTelegramApprovalKeyboard(
		// 	createTelegramApprovalCallbackData(ActionApprove, opts.Req.Spec.GetUuid(), opts.Req.Spec.Id),
		// 	createTelegramApprovalCallbackData(ActionReject, opts.Req.Spec.GetUuid(), opts.Req.Spec.Id),
		// ),
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
		MfaSeed:                  *target.MfaSeed,
		RequestId:                opts.Req.Spec.Id,
		RequestUuid:              opts.Req.Spec.GetUuid(),
		UserId:                   fmt.Sprintf("%v", opts.SenderId),
	}
	pendingMfaString, _ := json.Marshal(pendingMfaData)
	Cache.Set(pendingMfaCacheKey, string(pendingMfaString), 60*time.Second)
	if err := opts.Bot.ReplyMessage(
		opts.ChatId,
		opts.MessageId,
		getTelegramMfaRequestMessage(opts.Req),
		&models.ForceReply{
			ForceReply: true,
		},
	); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v] in chat[%v]: %s", opts.MessageId, opts.ChatId, err)
	}
}

func handleTelegramApproval(opts handleTelegramResponseOpts) {
	if err := processTelegramApproval(processTelegramApprovalOpts{
		ApprovalMessageId: opts.MessageId,
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

type handleTelegramCommandsOpts struct {
	Bot         *telegram.Bot
	Update      *telegram.Update
	ServiceLogs chan<- common.ServiceLog
}

func handleTelegramCommands(opts handleTelegramCommandsOpts) {
	if strings.HasPrefix(opts.Update.Message, "/info") {
		if err := opts.Bot.ReplyMessage(
			opts.Update.ChatId,
			opts.Update.MessageId,
			getTelegramInfoMessage(opts.Update),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to respond to a /info request: %s", err)
		}
	}
}

type handleTelegramInvalidEventOpts struct {
	Bot         *telegram.Bot
	ChatId      int64
	MessageId   int
	SenderId    int64
	ServiceLogs chan<- common.ServiceLog
}

func handleTelegramInvalidEvent(opts handleTelegramInvalidEventOpts) {
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "handling invalid message[%v] by user[%v] in chat[%v] which is not an mfa token and not a callback", opts.MessageId, opts.SenderId, opts.ChatId)
	if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, we do not respond to unexpected messages"); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send response to user: %s", err)
	}
}

type handleTelegramMfaResponseOpts struct {
	Bot                 *telegram.Bot
	ChatId              int64
	Message             string
	MessageId           int
	MfaRequestMessageId int
	SenderId            int64
	SenderUsername      string
	ServiceLogs         chan<- common.ServiceLog
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
	approvalRequest := &ApprovalRequest{
		Spec: approvals.RequestSpec{
			Uuid: &pendingMfaData.RequestUuid,
		},
	}
	if err := approvalRequest.Load(); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to load approvalRequest[%s:%s]: %s", pendingMfaData.RequestId, pendingMfaData.RequestUuid, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "successfully retrieved approvalRequest[%s:%s]...", approvalRequest.Spec.Id, approvalRequest.Spec.GetUuid())

	authorizedTargets := getAuthorizedTelegramTargets(getAuthorizedTelegramTargetsOpts{
		ChatId:         opts.ChatId,
		Req:            *approvalRequest,
		SenderId:       opts.SenderId,
		SenderUsername: opts.SenderUsername,
		ServiceLogs:    opts.ServiceLogs,
	})
	isSenderAuthorized := len(authorizedTargets) > 0
	if !isSenderAuthorized {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, getTelegramUnauthorizedMessage()); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", opts.MessageId, err)
			if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
			}
		}
		return
	}
	if authorizedTargets[0].MfaSeed == nil {
		requestSpecification, _ := json.MarshalIndent(authorizedTargets[0], "", "  ")
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to receive an mfa seed where expected, request specification follows:\n%s", requestSpecification)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, getTelegramSystemErrorMessage()); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return
	}
	approvalMessageId, err := strconv.ParseInt(pendingMfaData.ApprovalRequestMessageId, 10, 0)
	if err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to parse approvalRequestId[%s]: %s", pendingMfaData.ApprovalRequestMessageId, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return
	}

	mfaSeed := *authorizedTargets[0].MfaSeed
	valid, err := auth.ValidateTotpToken(mfaSeed, mfaToken)
	if err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to validate totpToken[%v]: %s", mfaToken, err)
		return
	}
	if !valid {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "removing mfa response message with id[%v]", opts.MessageId)
		if isDeleted, err := opts.Bot.Client.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
			ChatID:    opts.ChatId,
			MessageID: opts.MessageId,
		}); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa response message with id[%v] due to error: %s", opts.MessageId, err)
		} else if !isDeleted {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa response message with id[%v] for unknown reasons", opts.MessageId)
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "totpToken[%v] was not vavlid", mfaToken)
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "removing mfa request message with id[%v]", opts.MfaRequestMessageId)
		if isDeleted, err := opts.Bot.Client.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
			ChatID:    opts.ChatId,
			MessageID: opts.MfaRequestMessageId,
		}); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa request message with id[%v] due to error: %s", opts.MfaRequestMessageId, err)
		} else if !isDeleted {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa request message with id[%v] for unknown reasons", opts.MfaRequestMessageId)
		}
		opts.Bot.ReplyMessage(opts.ChatId, int(approvalMessageId), getTelegramMfaRejectedMessage(*approvalRequest, opts.SenderId, opts.SenderUsername))
		// if err := opts.Bot.UpdateMessage(
		// 	opts.ChatId,
		// 	int(approvalMessageId),
		// 	getTelegramApprovalRequestMessage(*approvalRequest),
		// ); err != nil {
		// 	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approval message: %s", err)
		// }
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "updating message[%v] in chat[%v]", int(approvalMessageId), opts.ChatId)
		if err := opts.Bot.UpdateMessage(
			opts.ChatId,
			int(approvalMessageId),
			getTelegramApprovalRequestMessage(*approvalRequest),
			getTelegramApprovalKeyboard(
				createTelegramApprovalCallbackData(ActionApprove, approvalRequest.Spec.GetUuid()),
				createTelegramApprovalCallbackData(ActionReject, approvalRequest.Spec.GetUuid()),
			),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update markup for approval message: %s", err)
		}
		return
	}
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "user[%v] in chat[%v] approved the transaction", opts.SenderId, opts.ChatId)

	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "removing mfa response message with id[%v]", opts.MessageId)
	if isDeleted, err := opts.Bot.Client.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
		ChatID:    opts.ChatId,
		MessageID: opts.MessageId,
	}); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa response message with id[%v] due to error: %s", opts.MessageId, err)
	} else if !isDeleted {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa response message with id[%v] for unknown reasons", opts.MessageId)
	}

	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "removing mfa request message with id[%v]", opts.MfaRequestMessageId)
	if isDeleted, err := opts.Bot.Client.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
		ChatID:    opts.ChatId,
		MessageID: opts.MfaRequestMessageId,
	}); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa request message with id[%v] due to error: %s", opts.MfaRequestMessageId, err)
	} else if !isDeleted {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to remove mfa request message with id[%v] for unknown reasons", opts.MfaRequestMessageId)
	}
	if err := processTelegramApproval(processTelegramApprovalOpts{
		ApprovalMessageId: int(approvalMessageId),
		Bot:               opts.Bot,
		ChatId:            opts.ChatId,
		Req:               *approvalRequest,
		SenderId:          opts.SenderId,
		SenderUsername:    opts.SenderUsername,
		ServiceLogs:       opts.ServiceLogs,
	}); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process approvalRequest[%s]: %s", approvalRequest.Spec.GetUuid(), err)
		return
	}
}

func handleTelegramRejection(opts handleTelegramResponseOpts) {
	approval := &Approval{
		Spec: approvals.ApprovalSpec{
			ApproverId:      strconv.FormatInt(opts.SenderId, 10),
			ApproverName:    opts.SenderUsername,
			Id:              uuid.New().String(),
			RequestId:       opts.Req.Spec.Id,
			RequestUuid:     opts.Req.Spec.GetUuid(),
			RequesterId:     opts.Req.Spec.RequesterId,
			RequesterName:   opts.Req.Spec.RequesterName,
			Status:          approvals.StatusRejected,
			StatusUpdatedAt: time.Now(),
			Telegram: &approvals.TelegramResponseSpec{
				ChatId:   opts.ChatId,
				UserId:   opts.SenderId,
				Username: opts.SenderUsername,
			},
			Type: approvals.PlatformTelegram,
		},
	}
	if err := approval.Create(); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to create approval[%s]: %s", approval.Spec.Id, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
	}
	opts.Req.Spec.Approval = &approval.Spec
	if err := opts.Req.Update(); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
	}
	if err := opts.Bot.UpdateMessage(
		opts.ChatId,
		opts.MessageId,
		getTelegramRejectedMessage(opts.Req),
		nil,
	); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update message[%v]: %s", opts.MessageId, err)
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return
	}
	if err := opts.Bot.ReplyMessage(opts.ChatId, opts.MessageId, getTelegramRejectMessage(opts.Req, opts.SenderId, opts.SenderUsername)); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send approval message to chat[%v]: %s", opts.ChatId, err)
	}
	if err := handleCallback(handleCallbackOpts{
		Req:         opts.Req,
		ServiceLogs: opts.ServiceLogs,
	}); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process webhook for request[%s:%s]: %s", opts.Req.Spec.Id, opts.Req.Spec.GetUuid(), err)
	}
}

type processTelegramApprovalOpts struct {
	ApprovalMessageId int
	Bot               *telegram.Bot
	ChatId            int64
	Req               ApprovalRequest
	SenderId          int64
	SenderUsername    string
	ServiceLogs       chan<- common.ServiceLog
}

// processTelegramApproval processes an approval via Telegram
func processTelegramApproval(opts processTelegramApprovalOpts) error {
	approval := &Approval{
		Spec: approvals.ApprovalSpec{
			ApproverId:      strconv.FormatInt(opts.SenderId, 10),
			ApproverName:    opts.SenderUsername,
			Id:              uuid.New().String(),
			RequestId:       opts.Req.Spec.Id,
			RequestUuid:     opts.Req.Spec.GetUuid(),
			RequesterId:     opts.Req.Spec.RequesterId,
			RequesterName:   opts.Req.Spec.RequesterName,
			Status:          approvals.StatusApproved,
			StatusUpdatedAt: time.Now(),
			Telegram: &approvals.TelegramResponseSpec{
				ChatId:   opts.ChatId,
				UserId:   opts.SenderId,
				Username: opts.SenderUsername,
			},
			Type: approvals.PlatformTelegram,
		},
	}
	if err := approval.Create(); err != nil {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.ApprovalMessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to create approval[%s]: %s", approval.Spec.Id, err)
	}
	opts.Req.Spec.Approval = &approval.Spec
	if err := opts.Req.Update(); err != nil {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.ApprovalMessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
	}
	if err := opts.Bot.UpdateMessage(
		opts.ChatId,
		opts.ApprovalMessageId,
		getTelegramApprovedMessage(opts.Req),
		nil,
	); err != nil {
		if err := opts.Bot.ReplyMessage(opts.ChatId, opts.ApprovalMessageId, "🙇🏼 Apologies, something went wrong internally"); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to update message[%v]: %s", opts.ApprovalMessageId, err)
	}
	if err := opts.Bot.ReplyMessage(opts.ChatId, opts.ApprovalMessageId, getTelegramApproveMessage(opts.Req, opts.SenderId, opts.SenderUsername)); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send approval message to chat[%v]: %s", opts.ChatId, err)
	}
	if err := handleCallback(handleCallbackOpts{
		Req:         opts.Req,
		ServiceLogs: opts.ServiceLogs,
	}); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process webhook for request[%s:%s]: %s", opts.Req.Spec.Id, opts.Req.Spec.GetUuid(), err)
	}
	return nil
}
