package org

import "time"

type Type string

const (
	TypeAdmin  Type = "admin"
	TypeTenant Type = "tenant"
)

type Org struct {
	Id         *string    `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  *time.Time `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
	IsDeleted  bool       `json:"isDeleted"`
	DeletedAt  *time.Time `json:"deletedAt"`
	IsDisabled bool       `json:"isDisabled"`
	DisabledAt *time.Time `json:"disabledAt"`
	Code       string     `json:"code"`
	Icon       string     `json:"icon"`
	Logo       string     `json:"logo"`
	Motd       string     `json:"motd"`
	Type       string     `json:"type"`
}
