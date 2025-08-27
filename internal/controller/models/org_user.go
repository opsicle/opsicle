package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

func (ou OrgUser) validate() error {
	if _, err := uuid.Parse(ou.OrgId); err != nil {
		return fmt.Errorf("org id is not a uuid: %w", ErrorInvalidInput)
	} else if _, err := uuid.Parse(ou.UserId); err != nil {
		return fmt.Errorf("user id is not a uuid: %w", ErrorInvalidInput)
	}
	return nil
}
