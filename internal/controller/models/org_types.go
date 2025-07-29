package models

type OrgType string

const (
	TypeAdminOrg  OrgType = "admin"
	TypeTenantOrg OrgType = "tenant"
)
