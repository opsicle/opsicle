package controller

import (
	"errors"
	"fmt"
	"opsicle/internal/controller/models"
)

type validateRequesterCanManageOrgUsersOpts struct {
	OrgId           string
	RequesterUserId string
}

// validateRequesterCanManageOrgUsers runs a check on the provided
// `RequesterUserId` and checks if they are allowed to manage other org
// users in the organisation identified by `OrgId`.
//
// Returns an `error` if the validation failed, returns `nil` otherwise
func validateRequesterCanManageOrgUsers(opts validateRequesterCanManageOrgUsersOpts) error {
	org := models.Org{Id: &opts.OrgId}
	requester, err := org.GetUserV1(models.GetOrgUserV1Opts{
		Db:     db,
		UserId: opts.RequesterUserId,
	})
	if err != nil {
		if errors.Is(err, models.ErrorNotFound) {
			return fmt.Errorf("failed to verify requester: %w", ErrorInsufficientPermissions)
		}
		return fmt.Errorf("failed to verify requester: %w", ErrorDatabaseIssue)
	}
	isAllowedToUpdateOrgUser := false
	switch models.OrgMemberType(*requester.Org.MemberType) {
	case models.TypeOrgAdmin:
		fallthrough
	case models.TypeOrgManager:
		isAllowedToUpdateOrgUser = true
	}
	if !isAllowedToUpdateOrgUser {
		return fmt.Errorf("requester is not an admin or manager: %w", ErrorInsufficientPermissions)
	}
	return nil
}
