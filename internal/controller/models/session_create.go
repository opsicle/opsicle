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
	SessionId string `json:"sessionId"`
	Value     string `json:"value"`
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
	userInstance, err := GetUserV1(GetUserV1Opts{
		Db:    opts.Db,
		Email: &opts.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to get user instance: %s", err)
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
		UserId:   *userInstance.Id,
		Username: userInstance.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to issue jwt: %s", err)
	}
	cacheKey := strings.Join([]string{opts.CachePrefix, *userInstance.Id, sessionId}, ":")
	if err := cache.Get().Set(cacheKey, sessionId, opts.ExpiresIn); err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to update cache: %s", err)
	}
	return &SessionToken{
		SessionId: sessionId,
		Value:     jwtToken,
	}, nil
}
