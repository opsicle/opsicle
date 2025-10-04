package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func NewOrgUser() OrgUser {
	return OrgUser{
		Org:  &Org{},
		User: &User{},
		Role: &OrgRole{},
	}
}

type OrgUsers []OrgUser
type OrgUser struct {
	JoinedAt   time.Time `json:"joinedAt"`
	MemberType string    `json:"memberType"`
	Org        *Org      `json:"org"`
	User       *User     `json:"user"`

	Role *OrgRole `json:"role"`
}

func (ou OrgUser) validate() error {
	if ou.Org == nil {
		return fmt.Errorf("org undefined")
	} else if ou.Org.Id == nil {
		return fmt.Errorf("org id undefined")
	} else if _, err := uuid.Parse(ou.Org.GetId()); err != nil {
		return fmt.Errorf("org id is not a uuid: %w", ErrorInvalidInput)
	}

	if ou.User == nil {
		return fmt.Errorf("user undefined")
	} else if ou.User.Id == nil {
		return fmt.Errorf("user id undefined")
	} else if _, err := uuid.Parse(ou.User.GetId()); err != nil {
		return fmt.Errorf("user id is not a uuid: %w", ErrorInvalidInput)
	}
	return nil
}
