package session

import (
	"database/sql"
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cache"
	"opsicle/internal/controller/org"
	"opsicle/internal/controller/user"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Token struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

type CreateV1Opts struct {
	Db          *sql.DB
	CachePrefix string

	OrgCode  string
	Email    string
	Password string

	IpAddress string
	UserAgent string
	Source    string
	ExpiresIn time.Duration
}

func CreateV1(opts CreateV1Opts) (*Token, error) {
	orgInstance, err := org.GetV1(org.GetV1Opts{
		Db:   opts.Db,
		Code: &opts.OrgCode,
	})
	if err != nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to get org instance: %s", err)
	}
	if orgInstance == nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to find a org")
	}
	userInstance, err := user.GetV1(user.GetV1Opts{
		Db:    opts.Db,
		Email: &opts.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to get user instance: %s", err)
	}
	orgUserInstance, err := orgInstance.GetUserV1(org.GetUserV1Opts{
		Db:     opts.Db,
		UserId: *userInstance.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to get user org membership: %s", err)
	}
	if orgUserInstance == nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to get a user belonging to the specified organisation")
	}

	userInstance.Password = &opts.Password
	if !userInstance.ValidatePassword() {
		return nil, nil
	}
	sessionId := uuid.NewString()
	jwtToken, err := auth.GenerateJwt(auth.GenerateJwtOpts{
		Audience: opts.Source,
		Ext: map[string]string{
			"ip": opts.IpAddress,
			"ua": opts.UserAgent,
		},
		Id:       sessionId,
		Issuer:   "opsicle/controller",
		OrgCode:  orgInstance.Code,
		OrgId:    *orgInstance.Id,
		Secret:   secretSessionKey,
		Subject:  "cli",
		Ttl:      opts.ExpiresIn,
		UserId:   *userInstance.Id,
		Username: userInstance.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to issue jwt: %s", err)
	}
	cacheKey := strings.Join([]string{opts.CachePrefix, *userInstance.Id, sessionId}, ":")
	if err := cache.Get().Set(cacheKey, sessionId, opts.ExpiresIn); err != nil {
		return nil, fmt.Errorf("controller.session.CreateV1: failed to update cache: %s", err)
	}
	return &Token{
		Id:    sessionId,
		Value: jwtToken,
	}, nil
}
