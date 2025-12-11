package models

import (
	"fmt"

	"github.com/google/uuid"
)

type OrgIdentity struct {
	Id string
}

type AuthOrgTokenV1Opts struct {
	CacheConnection
	ControllerUrl

	TokenId string
	Token   string
}

// AuthOrgTokenV1 authenticates the provided org token using it's ID
// and value
func AuthOrgTokenV1(opts AuthOrgTokenV1Opts) (*OrgIdentity, error) {

	// check cache for a mapping from token-id to org-id
	// if cache has it, return that
	val, err := opts.Cache.Get("org-token-" + opts.TokenId)
	if err != nil {
		return nil, fmt.Errorf("failed to get org token: %w", err)
	}

	if _, err := uuid.Parse(val); err == nil {
		return &OrgIdentity{Id: val}, nil
	}

	// ping controller to get info, store info in cache

	return &OrgIdentity{}, nil
}
