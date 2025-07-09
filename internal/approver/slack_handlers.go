package approver

import (
	"fmt"
	"strings"
)

const (
	slackCallbackDataDelimiter = "__"
)

func createSlackApprovalCallbackData(channelId string, requestUuid string, requestId string) (callbackData string) {
	callbackData = fmt.Sprintf("%s%s%s%s%s",
		channelId,
		slackCallbackDataDelimiter,
		requestUuid,
		slackCallbackDataDelimiter,
		requestId,
	)
	return
}

func parseSlackApprovalCallbackData(callbackData string) (channelId, requestUuid, requestId string) {
	callbackDataSegments := strings.Split(callbackData, slackCallbackDataDelimiter)
	return callbackDataSegments[0], callbackDataSegments[1], callbackDataSegments[2]
}
