package models

import (
	"database/sql"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SessionToken struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type CreateSessionV1Opts struct {
	Db          *sql.DB
	CachePrefix string

	Email string

	IpAddress string
	UserAgent string
	Hostname  string
	Source    string
	ExpiresIn time.Duration
}

func CreateSessionV1(opts CreateSessionV1Opts) (*SessionToken, error) {
	user := User{Email: opts.Email}
	if err := user.LoadByEmailV1(DatabaseConnection{Db: opts.Db}); err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to get user instance: %w", err)
	}

	sessionId := uuid.NewString()

	jwtToken, err := auth.GenerateJwt(auth.GenerateJwtOpts{
		Audience: opts.Source,
		Ext: map[string]string{
			"ip": opts.IpAddress,
			"ua": opts.UserAgent,
			"hn": opts.Hostname,
		},
		Id:       sessionId,
		Issuer:   "opsicle/controller",
		Secret:   sessionSigningToken,
		Subject:  "cli",
		Ttl:      opts.ExpiresIn,
		UserId:   user.GetId(),
		Username: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to issue jwt: %w", err)
	}
	cacheKey := strings.Join([]string{opts.CachePrefix, user.GetId(), sessionId}, ":")
	// random arbitrary cache expiration
	cacheExpiryDuration := time.Hour * 24 * 7
	if err := cache.Get().Set(cacheKey, sessionId, cacheExpiryDuration); err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to update cache: %w", err)
	}
	return &SessionToken{
		Id:    sessionId,
		Value: jwtToken,
	}, nil
}
