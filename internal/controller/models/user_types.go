package models

type UserType string

const (
	TypeSystemAdmin UserType = "system_admin"
	TypeSupportUser UserType = "support_user"
	TypeUser        UserType = "user"
)
