package approvals

import "time"

type Actions []Action

type Action struct {
	Data        any       `json:"data"`
	Error       *string   `json:"error"`
	HappenedAt  time.Time `json:"happenedAt"`
	MessageId   string    `json:"messageId"`
	Platform    string    `json:"platform"`
	RequestUuid string    `json:"requestUuid"`
	TargetId    string    `json:"targetId"`
	Status      Status    `json:"status"`
	UserId      string    `json:"userId"`
	UserName    string    `json:"userName"`
}
