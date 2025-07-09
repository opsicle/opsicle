package approver

import (
	"fmt"
	"log"
	"opsicle/internal/common"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type slackNotifier struct {
	ChannelID   string
	Client      *slack.Client
	Done        chan common.Done
	ServiceLogs chan<- common.ServiceLog
	Socket      *socketmode.Client
}

func (s *slackNotifier) resolveChannelId(channelName string) (string, error) {
	conversationCursor := ""
	for {
		channels, nextCursor, err := s.Client.GetConversations(&slack.GetConversationsParameters{
			Cursor:          conversationCursor,
			ExcludeArchived: true,
			Limit:           200,
			Types:           []string{"public_channel", "private_channel"},
		})
		if err != nil {
			return "", fmt.Errorf("failed to list channels: %w", err)
		}

		for _, ch := range channels {
			if ch.Name == channelName || "#"+ch.Name == channelName {
				return ch.ID, nil
			}
		}

		if nextCursor == "" {
			break
		}
		conversationCursor = nextCursor
	}
	return "", fmt.Errorf("channel %s not found", channelName)
}

func (s *slackNotifier) SendApprovalRequest(req ApprovalRequest) (string, notificationMessages, error) {
	text := getApprovalRequestMessage(req)

	requestUuid := req.Spec.GetUuid()
	requestId := req.Spec.Id
	s.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "sending via slack...")
	notifications := notificationMessages{}

	for _, target := range req.Spec.Slack {
		channelId, err := s.resolveChannelId(target.ChannelName)
		if err != nil {
			s.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to resolve the id of channel[%s]: %s", target.ChannelName, err)
			continue
		}
		notification := notificationMessage{
			Id:       requestUuid,
			TargetId: channelId,
			Platform: NotifierPlatformSlack,
		}

		blocks := slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", text, false, false), nil, nil),
				slack.NewActionBlock("approval_actions_"+requestUuid,
					slack.NewButtonBlockElement(string(ActionApprove), createSlackApprovalCallbackData(channelId, requestUuid, requestId), slack.NewTextBlockObject("plain_text", "Approve", false, false)),
					slack.NewButtonBlockElement(string(ActionReject), createSlackApprovalCallbackData(channelId, requestUuid, requestId), slack.NewTextBlockObject("plain_text", "Reject", false, false)),
				),
			},
		}

		sentChannelId, messageTimestamp, err := s.Client.PostMessage(
			channelId,
			slack.MsgOptionBlocks(blocks.BlockSet...),
		)
		if err != nil {
			s.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send slack message to channel[%s] with id[%s]: %w", target.ChannelName, channelId, err)
			notification.Error = err
			continue
		} else {
			notification.IsSuccess = true
			notification.MessageId = fmt.Sprintf("%s_%s", sentChannelId, messageTimestamp)
			notification.SentAt = time.Now()
		}
		notifications = append(notifications, notification)
	}

	return requestUuid, notifications, nil
}

func (s *slackNotifier) StartListening() {
	go func() {
		for evt := range s.Socket.Events {
			switch evt.Type {
			case socketmode.EventTypeInteractive:
				cb, ok := evt.Data.(slack.InteractionCallback)
				if !ok {
					continue
				}
				switch cb.Type {
				case slack.InteractionTypeBlockActions:
					s.handleInteractive(cb)
					s.Socket.Ack(*evt.Request)
				case slack.InteractionTypeViewSubmission:
					s.handleViewSubmission(cb)
					s.Socket.Ack(*evt.Request)
				default:
					s.Socket.Ack(*evt.Request)
				}
			default:
				// Unhandled event
			}
		}
	}()

	go func() {
		if err := s.Socket.Run(); err != nil {
			log.Fatalf("socket mode failed: %v", err)
		}
	}()
}

func (s *slackNotifier) handleInteractive(callback slack.InteractionCallback) {
	if len(callback.ActionCallback.BlockActions) == 0 {
		return
	}

	action := callback.ActionCallback.BlockActions[0]
	userId := callback.User.ID
	channelId, requestUuid, _ := parseSlackApprovalCallbackData(action.Value)

	switch action.ActionID {
	case string(ActionApprove):
		s.openMfaModal(callback.TriggerID, channelId, requestUuid, userId)
	case string(ActionReject):
		msg := fmt.Sprintf(":x: *Rejected* by `%s` (request ID: `%s`)", userId, requestUuid)
		s.Client.PostMessage(channelId, slack.MsgOptionText(msg, false))
	}
}

func (s *slackNotifier) openMfaModal(triggerId, channelId, requestUuid, user string) {
	mfaInput := slack.NewPlainTextInputBlockElement(slack.NewTextBlockObject("plain_text", "Enter MFA Code", false, false), "mfa_token")
	mfaBlock := slack.NewInputBlock("mfa_input", slack.NewTextBlockObject("plain_text", "MFA Code (optional)", false, false), nil, mfaInput)

	modalRequest := slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      "mfa_modal_" + requestUuid,
		Title:           slack.NewTextBlockObject("plain_text", "MFA Approval", false, false),
		Close:           slack.NewTextBlockObject("plain_text", "Cancel", false, false),
		Submit:          slack.NewTextBlockObject("plain_text", "Submit", false, false),
		PrivateMetadata: fmt.Sprintf("%s|%s|%s", channelId, requestUuid, user),
		Blocks:          slack.Blocks{BlockSet: []slack.Block{mfaBlock}},
	}

	_, err := s.Client.OpenView(triggerId, modalRequest)
	if err != nil {
		log.Printf("failed to open MFA modal: %v", err)
	}
}

func (s *slackNotifier) handleViewSubmission(cb slack.InteractionCallback) {
	meta := cb.View.PrivateMetadata // format: requestID|username
	parts := strings.SplitN(meta, "|", 3)
	if len(parts) != 3 {
		log.Printf("invalid private metadata: %s", meta)
		return
	}
	channelId := parts[0]
	requestUuid := parts[1]
	userId := parts[2]

	mfaToken := cb.View.State.Values["mfa_input"]["mfa_token"].Value
	msg := fmt.Sprintf(":white_check_mark: *Approved* by `%s` (request ID: `%s`)", userId, requestUuid)

	if mfaToken != "" {
		msg += fmt.Sprintf("\nMFA token provided: `%s`", mfaToken)
	} else {
		msg += "\n_No MFA token provided._"
	}

	s.Client.PostMessage(channelId, slack.MsgOptionText(msg, false))
}

func (s *slackNotifier) Stop() {
	s.Done <- common.Done{}
}

type InitSlackNotifierOpts struct {
	AppToken    string
	BotToken    string
	Done        chan common.Done
	ServiceLogs chan<- common.ServiceLog
}

func InitSlackNotifier(opts InitSlackNotifierOpts) {
	client := slack.New(
		opts.BotToken,
		slack.OptionDebug(false),
		slack.OptionAppLevelToken(opts.AppToken),
	)

	socket := socketmode.New(
		client,
		socketmode.OptionDebug(false),
	)

	Notifiers = append(
		Notifiers,
		&slackNotifier{
			Client:      client,
			Done:        opts.Done,
			Socket:      socket,
			ServiceLogs: opts.ServiceLogs,
		},
	)
}
