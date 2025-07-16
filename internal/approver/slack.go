package approver

import (
	"fmt"
	"log"
	"opsicle/internal/approvals"
	"opsicle/internal/common"
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

// resolveChannelId is a helper function that resolves the channel ID given
// the name of the channel
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

func (s *slackNotifier) SendApprovalRequest(req *ApprovalRequest) (string, notificationMessages, error) {
	requestUuid := req.Spec.GetUuid()
	requestId := req.Spec.Id
	s.ServiceLogs <- common.ServiceLogf(common.LogLevelInfo, "sending via slack...")
	notifications := notificationMessages{}

	for i, target := range req.Spec.Slack {
		if req.Spec.Slack[i].ChannelIds == nil {
			req.Spec.Slack[i].ChannelIds = []string{}
		}
		for _, channelName := range target.ChannelNames {
			var err error
			channelId, err := s.resolveChannelId(channelName)
			if err != nil {
				s.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to resolve the id of channel[%s]: %s", channelId, err)
				continue
			}
			s.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "resolved the id of channel[%s] to channel[%s]", channelId, channelName)
			req.Spec.Slack[i].ChannelIds = append(req.Spec.Slack[i].ChannelIds, channelId)

			notification := approvals.Notification{
				RequestUuid: requestUuid,
				TargetId:    channelId,
				Platform:    NotifierPlatformSlack,
			}
			callbackData := createSlackApprovalCallbackData(
				channelId,
				channelName,
				requestId,
				requestUuid,
			)
			blocks := getSlackApprovalRequestBlocks(req, callbackData)

			s.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "sending slack message to channel[%s/%s]", channelId, channelName)
			sentChannelId, messageTimestamp, err := s.Client.PostMessage(channelId, slack.MsgOptionBlocks(blocks.BlockSet...))
			if err != nil {
				s.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to send slack message to channel[%s] with id[%s]: %w", channelName, channelId, err)
				notification.Error = err
				continue
			} else {
				notification.IsSuccess = true
				notification.TargetId = sentChannelId
				notification.MessageId = messageTimestamp
				notification.SentAt = time.Now()
			}
			req.Spec.Slack[i].Notifications = append(
				req.Spec.Slack[i].Notifications,
				notification,
			)
		}
	}

	return requestUuid, notifications, nil
}

func (s *slackNotifier) StartListening() {
	defaultHandler := getDefaultSlackHandler(
		s.Client,
		s.Socket,
		s.ServiceLogs,
	)

	go func() {
		for evt := range s.Socket.Events {
			if err := defaultHandler(evt); err != nil {
				s.ServiceLogs <- common.ServiceLogf(common.LogLevelError, "failed to handle slack event: %s", err)
			}
		}
	}()

	go func() {
		if err := s.Socket.Run(); err != nil {
			log.Fatalf("socket mode failed: %v", err)
		}
	}()
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
