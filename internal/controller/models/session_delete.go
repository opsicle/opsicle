package models

import (
	"errors"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"strings"
)

type DeleteSessionV1Opts struct {
	BearerToken string
	CachePrefix string
}

func DeleteSessionV1(opts DeleteSessionV1Opts) (string, error) {
	claims, err := auth.ValidateJWT(sessionSigningToken, opts.BearerToken)
	if err != nil {
		switch true {
		case errors.Is(err, auth.ErrorJwtTokenSignature):
			return "", err
		case errors.Is(err, auth.ErrorJwtTokenExpired):
			if claims != nil && claims.ID != "" {
				cache.Get().Del(strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":"))
			}
			return "", nil
		case errors.Is(err, auth.ErrorJwtClaims):
			if claims != nil && claims.ID != "" {
				cache.Get().Del(strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":"))
			}
			return "", err
		default:
			return "", err
		}
	}
	if claims != nil && claims.ID != "" {
		cache.Get().Del(strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":"))
		return claims.ID, nil
	}
	return "", nil
}
