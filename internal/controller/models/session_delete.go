package models

import (
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"strings"
)

type DeleteSessionV1Opts struct {
	BearerToken string
	CachePrefix string
}

func DeleteSessionV1(opts DeleteSessionV1Opts) (string, error) {
	claims, err := auth.ValidateJWT(secretSessionKey, opts.BearerToken)
	if err != nil {
		// probably already invalid
		return "", nil
	}
	sessionId := claims.ID

	cacheKey := strings.Join([]string{opts.CachePrefix, claims.UserID, sessionId}, ":")
	if _, err := cache.Get().Get(cacheKey); err != nil {
		return "", fmt.Errorf("failed to retrieve session from cache: %s", err)
	}
	if err := cache.Get().Del(cacheKey); err != nil {
		return "", fmt.Errorf("failed to delete session from cache: %s", err)
	}
	return sessionId, nil
}
