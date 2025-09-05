package audit

import "fmt"

func Interpret(log LogEntry) string {
	switch log.Verb {
	case Create:
		switch log.ResourceType {
		case AutomationTemplateResource:
			return fmt.Sprintf("Created an automation template (ID: %s)", log.ResourceId)
		case OrgUserInvitationResource:
			return "Invited a user"
		case OrgResource:
			return fmt.Sprintf("Created an organisation (ID: %s)", log.ResourceId)
		case UserResource:
			return "Created account"
		}
	case ForcedLogout:
		return "Session was invalidated with automatic logout triggered"
	case Get:
		return "Retrieved own account information"
	case List:
		switch log.ResourceType {
		case OrgMemberTypesResource:
			return "Listed types of org membership"
		case OrgResource:
			return "Listed orgs"
		}
	case Login:
		return "Logged into Opsicle"
	case Logout:
		return "Logged out of Opsicle"
	case Update:
		switch log.ResourceType {
		case AutomationTemplateResource:
			return fmt.Sprintf("Updated automation template with ID %s", log.ResourceId)
		}
	case Verify:
		switch log.ResourceType {
		case UserEmailVerificationCodeResource:
			return "Verified email"
		}
	}
	return fmt.Sprintf(
		"Entity[%s[%s]] performed action[%s] on Resource[%s[%s]]",
		log.EntityType,
		log.EntityId,
		log.Verb,
		log.ResourceType,
		log.ResourceId,
	)
}
