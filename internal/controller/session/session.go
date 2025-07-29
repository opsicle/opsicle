package session

import "time"

type Session struct {
	Id        string    `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
	StartedAt time.Time `json:"startedAt"`
	TimeLeft  string    `json:"timeLeft"`
	UserId    string    `json:"userId"`
	Username  string    `json:"username"`
	UserType  string    `json:"userType"`
	OrgCode   string    `json:"orgCode"`
	OrgId     string    `json:"orgId"`
}
