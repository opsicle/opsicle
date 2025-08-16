package models

import (
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"strings"
	"time"
)

type GetSessionV1Opts struct {
	BearerToken string
	CachePrefix string
}

func GetSessionV1(opts GetSessionV1Opts) (*Session, error) {
	var output *Session = nil
	claims, err := auth.ValidateJWT(sessionSigningToken, opts.BearerToken)
	if claims != nil {
		output = &Session{
			Id:        claims.ID,
			ExpiresAt: claims.ExpiresAt.Time,
			StartedAt: claims.IssuedAt.Time,
			TimeLeft:  time.Until(claims.ExpiresAt.Time).String(),
			UserId:    claims.UserID,
			Username:  claims.Username,
		}
	}
	if err != nil {
		return output, fmt.Errorf("failed to validate token: %w", err)
	}
	cacheKey := strings.Join([]string{opts.CachePrefix, claims.UserID, claims.ID}, ":")
	sessionInfo, err := cache.Get().Get(cacheKey)
	if err != nil {
		return output, fmt.Errorf("failed to retrieve session from cache: %w", err)
	}
	if sessionInfo != claims.ID {
		return output, fmt.Errorf("failed to retrieve a valid session from cache: %w", err)
	}
	return output, nil
}
