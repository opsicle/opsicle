package session

import (
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"strings"
	"time"
)

type Session struct {
	Id        string    `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
	StartedAt time.Time `json:"startedAt"`
	TimeLeft  string    `json:"timeLeft"`
	UserId    string    `json:"userId"`
}

type GetV1Opts struct {
	BearerToken string
	CachePrefix string
}

func GetV1(opts GetV1Opts) (*Session, error) {
	claims, err := auth.ValidateJWT(secretSessionKey, opts.BearerToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %s", err)
	}
	sessionId := claims.ID

	cacheKey := strings.Join([]string{opts.CachePrefix, claims.UserID, sessionId}, ":")
	sessionInfo, err := cache.Get().Get(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session from cache: %s", err)
	}
	if sessionInfo != sessionId {
		return nil, fmt.Errorf("failed to retrieve a valid session from cache: %s", err)
	}
	return &Session{
		Id:        sessionId,
		ExpiresAt: claims.ExpiresAt.Time,
		StartedAt: claims.IssuedAt.Time,
		TimeLeft:  time.Until(claims.ExpiresAt.Time).String(),
		UserId:    claims.UserID,
	}, nil
}
