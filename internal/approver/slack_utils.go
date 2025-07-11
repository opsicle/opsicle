package approver

import (
	"encoding/json"
	"fmt"
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
