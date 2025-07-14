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
		isAuthorized, authorizedResponder := getTelegramTargetMatchingSender(getTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
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
	Target         approvals.TelegramRequestSpec
}

func getTelegramTargetMatchingSender(opts getTelegramTargetMatchingSenderOpts) (bool, *approvals.AuthorizedResponder) {
	isChatMatched := false
	for _, chatId := range opts.Target.ChatIds {
		fmt.Println("matching chat")
		if chatId == opts.ChatId {
			fmt.Println("matched chat id")
			isChatMatched = true
			break
		}
	}

	isSenderMatched := len(opts.Target.AuthorizedResponders) == 0
	matchedSender := approvals.AuthorizedResponder{}
	for _, authorizedResponder := range opts.Target.AuthorizedResponders {
		fmt.Println("matching responder")
		o, _ := json.Marshal(authorizedResponder)
		fmt.Printf("comparing authorizedResponder[%s]\n", string(o))
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
		fmt.Printf(
			"o.chatid: %v\n"+
				"o.uid: %v\n"+
				"o.uname: %v\n"+
				"uid.defined: %v\n"+
				"uid.match: %v\n"+
				"uname.defined: %v\n"+
				"uname.match: %v\n",
			opts.ChatId,
			opts.SenderId,
			opts.SenderUsername,
			isUserIdDefined,
			isUserIdMatch,
			isUsernameDefined,
			isUsernameMatch,
		)
		if isUserIdMatch && isUsernameMatch {
			matchedSender = authorizedResponder
			isSenderMatched = true
			break
		}
	}
	return isChatMatched && isSenderMatched, &matchedSender
}
