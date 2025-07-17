package approver

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/auth"
	"opsicle/internal/common"
	"time"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// getDefaultSlackHandler returns the default handler for slack
func getDefaultSlackHandler(
	app *slack.Client,
	socket *socketmode.Client,
	serviceLogs chan<- common.ServiceLog,
) func(event socketmode.Event) error {
	return func(event socketmode.Event) error {

		switch event.Type {
		case socketmode.EventTypeInteractive:
			cb, ok := event.Data.(slack.InteractionCallback)
			if !ok {
				serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "failed to receive data that fits slack.InteractionCallback")
				return nil
			}
			switch cb.Type {
			case slack.InteractionTypeBlockActions:
				serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received interaction data of type slack.InteractionTypeBlockActions")
				handleSlackInteraction(handleSlackInteractionOpts{
					App:         app,
					Callback:    cb,
					ServiceLogs: serviceLogs,
				})
				socket.Ack(*event.Request)
			case slack.InteractionTypeViewSubmission:
				serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "received interaction data of type slack.InteractionTypeViewSubmission")
				handleSlackViewSubmission(handleSlackViewSubmissionOpts{
					App:         app,
					Callback:    cb,
					ServiceLogs: serviceLogs,
				})
				socket.Ack(*event.Request)
			default:
				socket.Ack(*event.Request)
			}
		default:
			// Unhandled event
		}
		return nil
	}
}

type handleSlackApprovalOpts struct {
	App         *slack.Client
	Req         *ApprovalRequest
	ChannelId   string
	MessageId   string
	ServiceLogs chan<- common.ServiceLog
	SlackTarget *approvals.AuthorizedResponder
	TriggerId   string
	UserId      string
	UserName    string
	Callback    slack.InteractionCallback
}

func handleSlackApproval(opts handleSlackApprovalOpts) {
	if opts.SlackTarget.MfaSeed != nil {
		pendingMfaCacheKey := CreatePendingMfaCacheKey(opts.ChannelId, opts.UserId)
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "creating cache with key[%s]...", pendingMfaCacheKey)
		pendingMfaData := pendingMfa{
			ApprovalRequestMessageId: opts.MessageId,
			ChatId:                   opts.ChannelId,
			MfaSeed:                  *opts.SlackTarget.MfaSeed,
			RequestId:                opts.Req.Spec.Id,
			RequestUuid:              opts.Req.Spec.GetUuid(),
			UserId:                   opts.UserId,
		}
		pendingMfaString, _ := json.Marshal(pendingMfaData)
		if err := Cache.Set(pendingMfaCacheKey, string(pendingMfaString), 60*time.Second); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to set cache item with key[%s]", pendingMfaCacheKey)
			return
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "created cache with key[%s]", pendingMfaCacheKey)
		msg := getSlackPendingMfaMessage(opts.UserId)
		if _, _, err := opts.App.PostMessage(
			opts.ChannelId,
			slack.MsgOptionText(msg, false),
			slack.MsgOptionTS(opts.MessageId),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to respond: %s", err)
			return
		}
		if err := handleSlackMfaRequest(
			opts.App,
			opts.TriggerId,
			opts.MessageId,
			opts.ChannelId,
			opts.Req.Spec.Id,
			opts.Req.Spec.GetUuid(),
			opts.UserId,
			opts.UserName,
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to handle sending of mfa request: %s", err)
			return
		}

		opts.Req.Spec.Actions = append(
			opts.Req.Spec.Actions,
			approvals.Action{
				HappenedAt:  time.Now(),
				MessageId:   opts.MessageId,
				Platform:    string(approvals.PlatformTelegram),
				RequestUuid: opts.Req.Spec.GetUuid(),
				TargetId:    opts.ChannelId,
				Status:      approvals.StatusMfaTriggered,
				UserId:      opts.UserId,
				UserName:    opts.UserName,
			},
		)
		if err := opts.Req.Update(); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "sent mfa request for request[%s:%s] in channel[%s] to user[%s]", opts.Req.Spec.Id, opts.Req.Spec.GetUuid(), opts.ChannelId, opts.UserId)
		return
	} else {
		if err := processSlackApproval(processSlackApprovalOpts{
			ApprovalRequestMessageTs: opts.MessageId,
			App:                      opts.App,
			ChannelId:                opts.ChannelId,
			Req:                      opts.Req,
			SenderId:                 opts.UserId,
			SenderName:               opts.Callback.User.Profile.RealName,
			ServiceLogs:              opts.ServiceLogs,
			Status:                   approvals.StatusApproved,
		}); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process slack approval in channel[%s] from user[%s]: %s", opts.ChannelId, opts.UserId, err)
			return
		}
	}
}

type handleSlackInteractionOpts struct {
	App         *slack.Client
	Callback    slack.InteractionCallback
	ServiceLogs chan<- common.ServiceLog
}

// handleSlackInteraction is called when a button is pressed
func handleSlackInteraction(opts handleSlackInteractionOpts) {
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "beginning slack interaction handling")
	if len(opts.Callback.ActionCallback.BlockActions) == 0 {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "ignoring slack interaction, no block actions found")
		return
	}

	channelId := opts.Callback.Channel.ID
	messageId := opts.Callback.Message.Timestamp
	triggerId := opts.Callback.TriggerID
	userId := opts.Callback.User.ID
	userName := opts.Callback.User.Profile.RealName
	action := opts.Callback.ActionCallback.BlockActions[0]
	callbackData, err := parseSlackApprovalCallbackData(action.Value)
	if err != nil {
		if err := respondSlackSystemError(opts.App, channelId, messageId); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to respond: %s", err)
		}
		return
	}

	requestId := callbackData.RequestId
	requestUuid := callbackData.RequestUuid

	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "loading approval request[%s:%s]", requestId, requestUuid)

	approvalRequest := &ApprovalRequest{
		Spec: approvals.RequestSpec{
			Uuid: &requestUuid,
		},
	}
	if err := approvalRequest.Load(); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to fetch request from cache: %v", err)
		opts.App.PostMessage(channelId, slack.MsgOptionText(getSlackSystemErrorMessage(), false))
		return
	}

	authorizedSlackTargets := getAuthorizedSlackTargets(getAuthorizedSlackTargetsOpts{
		ChannelId:   channelId,
		Req:         *approvalRequest,
		SenderId:    userId,
		ServiceLogs: opts.ServiceLogs,
	})
	if len(authorizedSlackTargets) == 0 {
		approvalRequest.Spec.Actions = append(
			approvalRequest.Spec.Actions,
			approvals.Action{
				HappenedAt:  time.Now(),
				MessageId:   opts.Callback.MessageTs,
				Platform:    string(approvals.PlatformSlack),
				RequestUuid: approvalRequest.Spec.GetUuid(),
				TargetId:    opts.Callback.Channel.ID,
				Status:      approvals.StatusUnauthorized,
				UserId:      opts.Callback.User.ID,
				UserName:    opts.Callback.User.Name,
			},
		)
		if err := approvalRequest.Update(); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approvalRequest[%s]: %s", approvalRequest.Spec.GetUuid(), err)
		}
		if err := respondSlackUnauthorized(opts.App, channelId, userId, action.ActionID, messageId); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to respond: %s", err)
		}
		return
	}

	isTargetFound := false
	for _, slackTarget := range approvalRequest.Spec.Slack {
		isAuthorized, authorizedResponder := getSlackTargetMatchingSender(getSlackTargetMatchingSenderOpts{
			ChannelId:   channelId,
			SenderId:    userId,
			ServiceLogs: opts.ServiceLogs,
			Target:      slackTarget,
		})
		if isAuthorized {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "channel[%s].user[%s]:authorized", channelId, userId)
			isTargetFound = true
			switch action.ActionID {
			case string(ActionApprove):
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "channel[%s].user[%s]:approve", channelId, userId)
				handleSlackApproval(handleSlackApprovalOpts{
					App:         opts.App,
					Req:         approvalRequest,
					ChannelId:   channelId,
					MessageId:   messageId,
					ServiceLogs: opts.ServiceLogs,
					SlackTarget: authorizedResponder,
					TriggerId:   triggerId,
					UserId:      userId,
					UserName:    userName,
				})
			case string(ActionReject):
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "channel[%s].user[%s]:reject", channelId, userId)
				if err := processSlackApproval(processSlackApprovalOpts{
					ApprovalRequestMessageTs: messageId,
					App:                      opts.App,
					ChannelId:                channelId,
					Req:                      approvalRequest,
					SenderId:                 userId,
					SenderName:               opts.Callback.User.Profile.RealName,
					ServiceLogs:              opts.ServiceLogs,
					Status:                   approvals.StatusRejected,
				}); err != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to process slack approval in channel[%s] from user[%s]: %s", channelId, userId, err)
					return
				}
			}
		}
		if !isTargetFound {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "channel[%s].user[%s]:unauthorized", channelId, userId)
			msg := getSlackUnauthorizedMessage(userId, action.ActionID)
			opts.App.PostMessage(
				channelId,
				slack.MsgOptionText(msg, false),
				slack.MsgOptionTS(messageId),
			)
		}
	}
}

type handleSlackViewSubmissionOpts struct {
	App         *slack.Client
	Callback    slack.InteractionCallback
	ServiceLogs chan<- common.ServiceLog
}

// handleSlackViewSubmission is mainly for handling the MFA response
func handleSlackViewSubmission(opts handleSlackViewSubmissionOpts) {
	metadata := opts.Callback.View.PrivateMetadata
	mfaModalMetadata, err := parseSlackMfaModalMetadata(metadata)
	if err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to parse mfa modal metadata: %s", err)
		return
	}
	mfaToken := opts.Callback.View.State.Values["mfa_input"]["mfa_token"].Value
	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "received mfaToken[%s]", mfaToken)
	approvalRequest := &ApprovalRequest{
		Spec: approvals.RequestSpec{
			Uuid: &mfaModalMetadata.RequestUuid,
		},
	}
	if err := approvalRequest.Load(); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to fetch request from cache: %v", err)
		opts.App.PostMessage(mfaModalMetadata.ChannelId, slack.MsgOptionText(getSlackSystemErrorMessage(), false))
		return
	}

	if mfaToken != "" {
		for _, slackTarget := range approvalRequest.Spec.Slack {
			isAuthorized, authorizedResponder := getSlackTargetMatchingSender(getSlackTargetMatchingSenderOpts{
				ChannelId:   mfaModalMetadata.ChannelId,
				SenderId:    opts.Callback.User.ID,
				ServiceLogs: opts.ServiceLogs,
				Target:      slackTarget,
			})
			if isAuthorized {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "channel[%s].user[%s]:mfa:authorized", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
				if authorizedResponder.MfaSeed != nil {
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "validating totp token from channel[%s].user[%s]", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
					totpValid, err := auth.ValidateTotpToken(*authorizedResponder.MfaSeed, mfaToken)
					if err != nil {
						opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to validate totp token in slack: %s", err)
						msg := getSlackSystemErrorMessage()
						opts.App.PostMessage(opts.Callback.Channel.ID, slack.MsgOptionText(msg, false))
						return
					}
					if totpValid {
						opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "totp token from channel[%s].user[%s] was valid", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
						if err := processSlackApproval(processSlackApprovalOpts{
							ApprovalRequestMessageTs: mfaModalMetadata.MessageTs,
							App:                      opts.App,
							ChannelId:                mfaModalMetadata.ChannelId,
							Req:                      approvalRequest,
							SenderId:                 opts.Callback.User.ID,
							SenderName:               mfaModalMetadata.UserName,
							ServiceLogs:              opts.ServiceLogs,
							Status:                   approvals.StatusApproved,
						}); err != nil {
							opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to handle slack approval: %s", err)
							msg := getSlackSystemErrorMessage()
							opts.App.PostMessage(opts.Callback.Channel.ID, slack.MsgOptionText(msg, false))
							return
						}
						opts.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "channel[%s].user[%s] successfully verified totp token", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
						return
					}
					opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "totp token from channel[%s].user[%s] was invalid", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
					msg := getSlackMfaRejectedMessage(opts.Callback.User.ID)
					channelId, messageTs, _ := opts.App.PostMessage(
						mfaModalMetadata.ChannelId,
						slack.MsgOptionText(msg, false),
						slack.MsgOptionTS(mfaModalMetadata.MessageTs),
					)
					approvalRequest.Spec.Actions = append(
						approvalRequest.Spec.Actions,
						approvals.Action{
							HappenedAt:  time.Now(),
							MessageId:   messageTs,
							Platform:    string(approvals.PlatformTelegram),
							RequestUuid: approvalRequest.Spec.GetUuid(),
							TargetId:    channelId,
							Status:      approvals.StatusMfaInvalid,
							UserId:      opts.Callback.User.ID,
							UserName:    opts.Callback.User.Name,
						},
					)
					if err := approvalRequest.Update(); err != nil {
						opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approvalRequest[%s]: %s", approvalRequest.Spec.GetUuid(), err)
					}
					return
				}
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "channel[%s].user[%s] submitted a token when a token wasn't expected", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
				msg := getSlackSystemErrorMessage()
				opts.App.PostMessage(
					opts.Callback.Channel.ID,
					slack.MsgOptionText(msg, false),
					slack.MsgOptionTS(mfaModalMetadata.MessageTs),
				)
				break
			} else {
				opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "channel[%s].user[%s]:unauthorized", mfaModalMetadata.ChannelId, opts.Callback.User.ID)
				msg := getSlackUnauthorizedMessage(opts.Callback.User.ID, string(ActionMfa))
				opts.App.PostMessage(opts.Callback.Channel.ID, slack.MsgOptionText(msg, false), slack.MsgOptionTS(mfaModalMetadata.MessageTs))
				return
			}
		}
	} else {
		msg := getSlackMfaRejectedMessage(opts.Callback.User.ID)
		opts.App.PostMessage(
			mfaModalMetadata.ChannelId,
			slack.MsgOptionText(msg, false),
			slack.MsgOptionTS(mfaModalMetadata.MessageTs),
		)
		return
	}

	opts.ServiceLogs <- common.ServiceLogf(common.LogLevelWarn, "handleSlackViewSubmission has reached an unexpected state")
	msg := getSlackSystemErrorMessage()
	opts.App.PostMessage(mfaModalMetadata.ChannelId, slack.MsgOptionText(msg, false), slack.MsgOptionTS(mfaModalMetadata.MessageTs))
}

func handleSlackMfaRequest(
	app *slack.Client,
	triggerId,
	messageTs,
	channelId,
	requestId,
	requestUuid,
	userId,
	userName string,
) error {
	mfaInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject(
			"plain_text",
			"Enter MFA Code",
			false,
			false,
		),
		"mfa_token",
	)
	mfaBlock := slack.NewInputBlock(
		"mfa_input",
		slack.NewTextBlockObject(
			"plain_text", "MFA Code",
			false,
			false,
		),
		nil,
		mfaInput,
	)

	modalRequest := slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: "mfa_modal_" + requestUuid,
		Title:      slack.NewTextBlockObject("plain_text", "MFA Approval", false, false),
		Close:      slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		Submit:     slack.NewTextBlockObject("plain_text", "Submit", false, false),
		PrivateMetadata: createSlackMfaModalMetadata(
			channelId,
			messageTs,
			requestId,
			requestUuid,
			userId,
			userName,
		),
		Blocks: slack.Blocks{BlockSet: []slack.Block{mfaBlock}},
	}

	_, err := app.OpenView(triggerId, modalRequest)
	if err != nil {
		return fmt.Errorf("failed to open MFA modal: %v", err)
	}

	return nil
}

type processSlackApprovalOpts struct {
	ApprovalRequestMessageTs string
	App                      *slack.Client
	ChannelId                string
	Req                      *ApprovalRequest
	SenderId                 string
	SenderName               string
	ServiceLogs              chan<- common.ServiceLog
	Status                   approvals.Status
}

func processSlackApproval(opts processSlackApprovalOpts) error {
	slackResponse := approvals.SlackResponseSpec{
		ChannelId:  opts.ChannelId,
		ReceivedAt: time.Now(),
		Status:     opts.Status,
		UserId:     opts.SenderId,
		UserName:   opts.SenderName,
	}
	if opts.Req.Spec.Approval == nil {
		opts.Req.Spec.Approval = &approvals.ApprovalSpec{
			ApproverId:      opts.SenderId,
			ApproverName:    opts.SenderName,
			Id:              uuid.New().String(),
			RequestId:       opts.Req.Spec.Id,
			RequestUuid:     opts.Req.Spec.GetUuid(),
			RequesterId:     opts.Req.Spec.RequesterId,
			RequesterName:   opts.Req.Spec.RequesterName,
			Status:          opts.Status,
			StatusUpdatedAt: time.Now(),
			Slack:           []approvals.SlackResponseSpec{},
			Type:            approvals.PlatformSlack,
		}
	}
	opts.Req.Spec.Approval.Slack = append(
		opts.Req.Spec.Approval.Slack,
		slackResponse,
	)
	approval := Approval{Spec: *opts.Req.Spec.Approval}
	if err := approval.Create(); err != nil {
		if err := respondSlackSystemError(opts.App, opts.ChannelId, opts.ApprovalRequestMessageTs); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to respond: %s", err)
		}
		return fmt.Errorf("failed to create approval[%s]: %s", approval.Spec.Id, err)
	}
	if err := opts.Req.Update(); err != nil {
		msg := getSlackSystemErrorMessage()
		if _, _, err := opts.App.PostMessage(
			opts.ChannelId,
			slack.MsgOptionText(msg, false),
			slack.MsgOptionTS(opts.ApprovalRequestMessageTs),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		return fmt.Errorf("failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
	}

	var channelId, messageTs string
	msg := slack.Blocks{}
	if opts.Status == approvals.StatusApproved {
		msg = getSlackApprovedBlocks(opts.Req)

		threadMessage := getSlackApprovalDetailsMessage(opts.SenderId, approval.Spec.StatusUpdatedAt)
		channelId, messageTs, _ = opts.App.PostMessage(
			opts.ChannelId,
			slack.MsgOptionText(threadMessage, false),
			slack.MsgOptionTS(opts.ApprovalRequestMessageTs),
		)
	} else {
		msg = getSlackRejectedBlocks(opts.Req)

		threadMessage := getSlackRejectionDetailsMessage(opts.SenderId, approval.Spec.StatusUpdatedAt)
		channelId, messageTs, _ = opts.App.PostMessage(
			opts.ChannelId,
			slack.MsgOptionText(threadMessage, false),
			slack.MsgOptionTS(opts.ApprovalRequestMessageTs),
		)
	}

	opts.Req.Spec.Actions = append(
		opts.Req.Spec.Actions,
		approvals.Action{
			HappenedAt:  time.Now(),
			MessageId:   messageTs,
			Platform:    string(approvals.PlatformTelegram),
			RequestUuid: opts.Req.Spec.GetUuid(),
			TargetId:    channelId,
			Status:      opts.Status,
			UserId:      opts.SenderId,
			UserName:    opts.SenderName,
		},
	)
	if err := opts.Req.Update(); err != nil {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to update approvalRequest[%s]: %s", opts.Req.Spec.GetUuid(), err)
	}
	if _, _, _, err := opts.App.UpdateMessage(
		opts.ChannelId,
		opts.ApprovalRequestMessageTs,
		slack.MsgOptionBlocks(msg.BlockSet...),
	); err != nil {
		msg := getSlackSystemErrorMessage()
		if _, _, err := opts.App.PostMessage(
			opts.ChannelId,
			slack.MsgOptionText(msg, false),
			slack.MsgOptionTS(opts.ApprovalRequestMessageTs),
		); err != nil {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		}
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send error response to user: %s", err)
		return fmt.Errorf("failed to update message[%v]: %s", opts.ApprovalRequestMessageTs, err)
	}
	return nil
}

func respondSlackSystemError(
	client *slack.Client,
	channelId string,
	thread ...string,
) error {
	msg := getSlackSystemErrorMessage()
	slackMessageOptions := []slack.MsgOption{
		slack.MsgOptionText(msg, false),
	}
	if len(thread) > 0 {
		slackMessageOptions = append(
			slackMessageOptions,
			slack.MsgOptionTS(thread[0]),
		)
	}
	if _, _, err := client.PostMessage(
		channelId,
		slackMessageOptions...,
	); err != nil {
		return fmt.Errorf("failed to send unauthorized message: %s", err)
	}
	return nil
}

func respondSlackUnauthorized(
	client *slack.Client,
	channelId string,
	senderId string,
	action string,
	thread ...string,
) error {
	msg := getSlackUnauthorizedMessage(senderId, action)
	slackMessageOptions := []slack.MsgOption{
		slack.MsgOptionText(msg, false),
	}
	if len(thread) > 0 {
		slackMessageOptions = append(
			slackMessageOptions,
			slack.MsgOptionTS(thread[0]),
		)
	}
	if _, _, err := client.PostMessage(
		channelId,
		slackMessageOptions...,
	); err != nil {
		return fmt.Errorf("failed to send unauthorized message: %s", err)
	}
	return nil
}
