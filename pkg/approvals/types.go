package approvals

import "opsicle/internal/approvals"

// CreateRequestInput represents data meant to be sent to the approval
// request creation endpoint
type CreateRequestInput struct {
	// Id is the ID of a request which will be the same for all
	// requests of a given type
	Id string `json:"id" yaml:"id"`

	// Message is an additional message describing the request
	Message string `json:"message" yaml:"message"`

	// RequesterName indicates the requester's system ID
	RequesterId string `json:"requesterId" yaml:"requesterId"`

	// RequesterName indicates the requester's name
	RequesterName string `json:"requesterName" yaml:"requesterName"`

	// Slack specifies the targets in Slack to send this request to
	Slack []approvals.SlackRequestSpec `json:"slack" yaml:"slack"`

	// Telegram specifies the targets in Telegram to send this request to
	Telegram []approvals.TelegramRequestSpec `json:"telegram" yaml:"telegram"`
}
