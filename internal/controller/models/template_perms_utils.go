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

func (t *Template) CanUserDeleteV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanDelete)
}

func (t *Template) CanUserInviteV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanInvite)
}

func (t *Template) CanUserUpdateV1(opts DatabaseConnection, userId string) (bool, error) {
	return t.canUserV1(opts, userId, CanUpdate)
}
