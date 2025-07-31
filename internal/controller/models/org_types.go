package models

// OrgType defines the type of the organisation
type OrgType string

const (
	// TypeAdminOrg indicates an organisation that should be
	// considered the administrative unit of the deployment
	TypeAdminOrg OrgType = "admin"

	// TypeDedicatedOrg indicates an organisation which has
	// dedicated resources to it such as a dedicated database,
	// controllers, and workers
	TypeDedicatedOrg OrgType = "dedicated"

	// TypeTenantOrg indicates an organisation that is under a
	// standard SaaS contract with us using shared resources
	TypeTenantOrg OrgType = "tenant"
)

// OrgUserType defines the type of organisation user
type OrgUserType string

const (
	// TypeOrgAdmin should be able to do everything in the orgniastion
	TypeOrgAdmin OrgUserType = "admin"

	// TypeOrgBilling should be able to only view and edit billing information
	TypeOrgBilling OrgUserType = "billing"

	// TypeOrgMember is a deferred role, get the permissions from the
	// `org_roles` table. This role assumes all permisisons of a `TypeOrgReporter`
	// role and adds onto the permissions via the deferred permissions
	TypeOrgMember OrgUserType = "member"

	// TypeOrgReporter is a view-only role for everything in the organisation
	// except secrets and credentials of any kind
	TypeOrgReporter OrgUserType = "reporter"
)
