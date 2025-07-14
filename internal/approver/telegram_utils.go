package approver

import (
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/common"
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

func getAuthorizedTelegramTargets(opts getAuthorizedTelegramTargetsOpts) []approvals.TelegramRequestSpec {
	authorizedTargets := []approvals.TelegramRequestSpec{}
	for _, telegramTarget := range opts.Req.Spec.Telegram {
		if isTelegramTargetMatchingSender(isTelegramTargetMatchingSenderOpts{
			ChatId:         opts.ChatId,
			SenderId:       opts.SenderId,
			SenderUsername: opts.SenderUsername,
			Target:         telegramTarget,
		}) {
			authorizedTargets = append(authorizedTargets, telegramTarget)
			break
		}
	}
	return authorizedTargets
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
