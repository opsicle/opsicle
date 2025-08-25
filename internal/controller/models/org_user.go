package models

import "time"

type OrgUser struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	OrgId      string    `json:"orgId"`
	OrgCode    string    `json:"orgCode"`
	OrgName    string    `json:"orgName"`
	UserId     string    `json:"userId"`
	UserEmail  string    `json:"userEmail"`
	UserType   string    `json:"userType"`
}
