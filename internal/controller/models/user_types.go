package models

type UserType string

const (
	// TypeSystemAdmin is used to indicate a system administrator
	TypeSystemAdmin UserType = "system_admin"

	// TypeSupportUser is used to indicate a support user
	TypeSupportUser UserType = "support_user"

	// TypeUser is used to indicate a normal user of the system
	TypeUser UserType = "user"
)
