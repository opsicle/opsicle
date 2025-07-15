package approver

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/approvals"
	"opsicle/internal/common"
)

const (
	slackDataDelimiter = "___"
)

type slackApprovalCallbackData struct {
	ChannelId   string  `json:"channelId"`
	ChannelName string  `json:"channelName"`
	RequestId   string  `json:"requestId"`
	RequestUuid string  `json:"requestUuid"`
	UserId      *string `json:"userId"`
}

func createSlackApprovalCallbackData(
	channelId string,
	channelName string,
	requestId string,
	requestUuid string,
	userId *string,
) (callbackData string) {
	data := slackApprovalCallbackData{
		ChannelId:   channelId,
		ChannelName: channelName,
		RequestId:   requestId,
		RequestUuid: requestUuid,
		UserId:      userId,
	}
	callbackString, _ := json.Marshal(data)
	return string(callbackString)
}

func parseSlackApprovalCallbackData(callbackData string) (*slackApprovalCallbackData, error) {
	var data slackApprovalCallbackData
	if err := json.Unmarshal([]byte(callbackData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data[%s]: %s", callbackData, err)
	}
	return &data, nil
}

type slackMfaRequestModalMetadata struct {
	ChannelId   string `json:"channelId"`
	MessageTs   string `json:"messageTs"`
	RequestId   string `json:"requestId"`
	RequestUuid string `json:"requestUuid"`
	UserId      string `json:"userId"`
	UserName    string `json:"userName"`
}

func createSlackMfaModalMetadata(channelId, messageTs, requestId, requestUuid, userId, userName string) (modalMetadata string) {
	data := slackMfaRequestModalMetadata{
		ChannelId:   channelId,
		MessageTs:   messageTs,
		RequestId:   requestId,
		RequestUuid: requestUuid,
		UserId:      userId,
		UserName:    userName,
	}
	modalMetadataString, _ := json.Marshal(data)
	return string(modalMetadataString)
}

func parseSlackMfaModalMetadata(privateMetadata string) (*slackMfaRequestModalMetadata, error) {
	var data slackMfaRequestModalMetadata
	if err := json.Unmarshal([]byte(privateMetadata), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data[%s]: %s", privateMetadata, err)
	}
	return &data, nil
}

type getAuthorizedSlackTargetsOpts struct {
	ChannelId   string
	Req         ApprovalRequest
	SenderId    string
	ServiceLogs chan<- common.ServiceLog
}

func getAuthorizedSlackTargets(opts getAuthorizedSlackTargetsOpts) approvals.AuthorizedResponders {
	authorizedResponders := approvals.AuthorizedResponders{}
	for _, target := range opts.Req.Spec.Slack {
		isAuthorized, authorizedResponder := getSlackTargetMatchingSender(getSlackTargetMatchingSenderOpts{
			ChannelId:   opts.ChannelId,
			SenderId:    opts.SenderId,
			Target:      target,
			ServiceLogs: opts.ServiceLogs,
		})
		if isAuthorized {
			authorizedResponders = append(authorizedResponders, *authorizedResponder)
			break
		}
	}
	return authorizedResponders
}

type getSlackTargetMatchingSenderOpts struct {
	ChannelId   string
	SenderId    string
	ServiceLogs chan<- common.ServiceLog
	Target      approvals.SlackRequestSpec
}

func getSlackTargetMatchingSender(opts getSlackTargetMatchingSenderOpts) (bool, *approvals.AuthorizedResponder) {
	isChatMatched := false
	for _, channelId := range opts.Target.ChannelIds {
		if channelId == opts.ChannelId {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "matched channel[%v]", channelId)
			isChatMatched = true
			break
		}
	}

	isSenderMatched := len(opts.Target.AuthorizedResponders) == 0
	matchedSender := approvals.AuthorizedResponder{}
	for _, authorizedResponder := range opts.Target.AuthorizedResponders {
		o, _ := json.Marshal(authorizedResponder)
		opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "comparing authorizedResponder[%s] with sender[%s]\n", string(o), opts.SenderId)
		isUserIdDefined := authorizedResponder.UserId != nil
		isUserIdMatch := true
		if isUserIdDefined {
			if *authorizedResponder.UserId != opts.SenderId {
				isUserIdMatch = false
			}
		}
		if isUserIdMatch {
			opts.ServiceLogs <- common.ServiceLogf(common.LogLevelTrace, "matched authorised responder in channel[%s]", opts.ChannelId)
			matchedSender = authorizedResponder
			isSenderMatched = true
			break
		}
	}
	return isChatMatched && isSenderMatched, &matchedSender
}
