package models

import (
	"errors"
	"fmt"
)

type TemplateUserPerm int

const (
	CanDelete TemplateUserPerm = iota
	CanExecute
	CanInvite
	CanUpdate
	CanView
)

// canUserV1 is the base function for the series of convenience functions
// that look like `CanUser*V1`
func (t *Template) canUserV1(opts DatabaseConnection, userId string, actions ...TemplateUserPerm) (bool, error) {
	if t.Id == nil {
		return false, fmt.Errorf("%w: template id not specified", ErrorInvalidInput)
	}
	templateUser := NewTemplateUser(userId, *t.Id)
	if err := templateUser.LoadV1(opts); err != nil {
		if errors.Is(err, ErrorNotFound) {
			return false, fmt.Errorf("user not found: %w", err)
		}
		return false, fmt.Errorf("failed to load template user: %w", err)
	}
	output := true
	for _, action := range actions {
		switch action {
		case CanDelete:
			output = output && templateUser.CanDelete
		case CanExecute:
			output = output && templateUser.CanExecute
		case CanInvite:
			output = output && templateUser.CanInvite
		case CanUpdate:
			output = output && templateUser.CanUpdate
		case CanView:
			output = output && templateUser.CanView
		}
	}
	return output, nil
}

// CanUserDeleteV1 returns truthy if a user identified by `userId`
// is allowed to delete the template. An error is returned if the
// `Template` instance is invalid or if the database operation failed`
func (t *Template) CanUserDeleteV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanDelete)
}

// CanUserExecuteV1 returns truthy if a user identified by `userId`
// is allowed to create an automation based on the template. An
// error is returned if the `Template` instance is invalid or if
// the database operation failed`
func (t *Template) CanUserExecuteV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanExecute)
}

// CanUserInviteV1 returns truthy if a user identified by `userId`
// is allowed to invite another user to collaborate on the template.
// An error is returned if the `Template` instance is invalid or if
// the database operation failed`
func (t *Template) CanUserInviteV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanInvite)
}

// CanUserUpdateV1 returns truthy if a user identified by `userId`
// is allowed to update the template. An error is returned if the
// `Template` instance is invalid or if the database operation failed`
func (t *Template) CanUserUpdateV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanUpdate)
}

// CanUserViewV1 returns truthy if a user identified by `userId`
// is allowed to view details about the template. An error is
// returned if the `Template` instance is invalid or if the
// database operation failed`
func (t *Template) CanUserViewV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanView)
}
