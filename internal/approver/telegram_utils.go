package approver

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/common"
	"strconv"
	"strings"
)

func createTelegramApprovalCallbackData(action Action, requestUuid string) (callbackData string) {
	callbackData = fmt.Sprintf("%s:%s", action, requestUuid)
	return
}

func parseTelegramApprovalCallbackData(callbackData string) (action Action, requestUuid string, err error) {
	splitCallbackData := strings.Split(callbackData, ":")
	if len(splitCallbackData) != 2 {
		return "", "", fmt.Errorf("failed to parse callback data: expected [{action}:{requestUuid}] but received callbackData[%s]", callbackData)
	}
	action = Action(splitCallbackData[0])
	requestUuid = splitCallbackData[1]
	err = nil
	return
}

type getAuthorizedTelegramTargetsOpts struct {
	ChatId         int64
	Req            ApprovalRequest
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan<- common.ServiceLog
}

func getAuthorizedTelegramTargets(opts getAuthorizedTelegramTargetsOpts) approvals.AuthorizedResponders {
	authorizedResponders := approvals.AuthorizedResponders{}
	for _, telegramTarget := range opts.Req.Spec.Telegram {
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelDebug, "DEPRECATED - running through list of telegram chat specifications")
		isAuthorized, authorizedResponder := getTelegramTargetMatchingSender(getTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
			ServiceLogs:    opts.ServiceLogs,
			Target:         telegramTarget,
		})
		if isAuthorized {
			authorizedResponders = append(authorizedResponders, *authorizedResponder)
			break
		}
	}
	return authorizedResponders
}

type getTelegramTargetMatchingSenderOpts struct {
	ChatId         int64
	SenderId       int64
	SenderUsername string
	ServiceLogs    chan<- common.ServiceLog
	Target         approvals.TelegramRequestSpec
}

func getTelegramTargetMatchingSender(opts getTelegramTargetMatchingSenderOpts) (bool, *approvals.AuthorizedResponder) {
	isChatMatched := false
	for _, chatId := range opts.Target.ChatIds {
		if chatId == opts.ChatId {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "matched chat[%v]", chatId)
			isChatMatched = true
			break
		}
	}

	isSenderMatched := len(opts.Target.AuthorizedResponders) == 0
	matchedSender := approvals.AuthorizedResponder{}
	for _, authorizedResponder := range opts.Target.AuthorizedResponders {
		o, _ := json.Marshal(authorizedResponder)
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "comparing authorizedResponder[%s] with sender[%s:%s]\n", string(o), opts.SenderId, opts.SenderUsername)
		isUserIdDefined := authorizedResponder.UserId != nil
		isUserIdMatch := true
		isUsernameDefined := authorizedResponder.Username != nil
		isUsernameMatch := true
		if isUserIdDefined {
			userId, err := strconv.ParseInt(*authorizedResponder.UserId, 10, 64)
			if err == nil {
				if userId != opts.SenderId {
					isUserIdMatch = false
				}
			}
		}
		if isUsernameDefined {
			if opts.SenderUsername != *authorizedResponder.Username {
				isUsernameMatch = false
			}
		}
		if isUserIdMatch && isUsernameMatch {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "matched authorised responder in chat[%v]", opts.ChatId)
			matchedSender = authorizedResponder
			isSenderMatched = true
			break
		}
	}
	return isChatMatched && isSenderMatched, &matchedSender
}
