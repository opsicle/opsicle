package models

import (
	"errors"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"strings"
)

type DeleteSessionV1Opts struct {
	Cache       cache.Cache
	BearerToken string
	CachePrefix string
}

func DeleteSessionV1(opts DeleteSessionV1Opts) (string, error) {
	claims, err := auth.ValidateJWT(sessionSigningToken, opts.BearerToken)
	if err != nil {
		switch true {
		case errors.Is(err, auth.ErrorJwtTokenSignature):
			return "", auth.ErrorJwtTokenSignature
		case errors.Is(err, auth.ErrorJwtTokenExpired):
			if claims != nil && claims.ID != "" {
				opts.Cache.Del(strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":"))
			}
			return "", auth.ErrorJwtTokenExpired
		case errors.Is(err, auth.ErrorJwtClaimsInvalid):
			if claims != nil && claims.ID != "" {
				opts.Cache.Del(strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":"))
			}
			return "", auth.ErrorJwtClaimsInvalid
		default:
			return "", fmt.Errorf("%w: %w", ErrorUnknown, err)
		}
	}
	if claims == nil || claims.ID == "" {
		return "", nil
	}
	opts.Cache.Del(strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":"))
	return claims.ID, nil
}
