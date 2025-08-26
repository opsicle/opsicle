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

// OrgMemberType defines the type of organisation user
type OrgMemberType string

const (
	// TypeOrgAdmin should be able to do everything in the orgniastion
	TypeOrgAdmin OrgMemberType = "admin"

	// TypeOrgBilling should be able to only view and edit billing information
	TypeOrgBilling OrgMemberType = "billing"

	// TypeOrgOperator should be able to update system parameters
	TypeOrgOperator OrgMemberType = "operator"

	// TypeOrgManager should be able to add and remove users
	TypeOrgManager OrgMemberType = "manager"

	// TypeOrgMember is a deferred role, get the permissions from the
	// `org_roles` table. This role assumes all permisisons of a `TypeOrgReporter`
	// role and adds onto the permissions via the deferred permissions
	TypeOrgMember OrgMemberType = "member"

	// TypeOrgReporter is a view-only role for everything in the organisation
	// except secrets and credentials of any kind
	TypeOrgReporter OrgMemberType = "reporter"
)

var OrgMemberTypeMap = map[string]struct{}{
	string(TypeOrgAdmin):    {},
	string(TypeOrgBilling):  {},
	string(TypeOrgOperator): {},
	string(TypeOrgManager):  {},
	string(TypeOrgMember):   {},
	string(TypeOrgReporter): {},
}
