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

	OrgCode  *string
	Email    string
	Password string

	IpAddress string
	UserAgent string
	Hostname  string
	Source    string
	ExpiresIn time.Duration
}

func CreateSessionV1(opts CreateSessionV1Opts) (*SessionToken, error) {
	orgInstance := &Org{}
	if opts.OrgCode != nil {
		var err error
		orgInstance, err = GetOrgV1(GetOrgV1Opts{
			Db:   opts.Db,
			Code: opts.OrgCode,
		})
		if err != nil {
			return nil, fmt.Errorf("models.CreateSessionV1: failed to get org instance: %s", err)
		}
	}
	userInstance, err := GetUserV1(GetUserV1Opts{
		Db:    opts.Db,
		Email: &opts.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("models.CreateSessionV1: failed to get user instance: %s", err)
	}
	if orgInstance != nil {
		if orgInstance.Id != nil {
			orgUserInstance, err := orgInstance.GetUserV1(GetOrgUserV1Opts{
				Db:     opts.Db,
				UserId: *userInstance.Id,
			})
			if err != nil {
				return nil, fmt.Errorf("models.CreateSessionV1: failed to get user org membership: %s", err)
			}
			if orgUserInstance == nil {
				return nil, fmt.Errorf("models.CreateSessionV1: failed to get a user belonging to the specified organisation")
			}
		}
	}

	userInstance.Password = &opts.Password
	if !userInstance.ValidatePassword() {
		return nil, nil
	}
	sessionId := uuid.NewString()
	orgCode := ""
	orgId := ""
	if orgInstance != nil {
		if orgInstance.Code != "" {
			orgCode = orgInstance.Code
		}
		if orgInstance.Id != nil {
			orgId = *orgInstance.Id
		}
	}
	jwtToken, err := auth.GenerateJwt(auth.GenerateJwtOpts{
		Audience: opts.Source,
		Ext: map[string]string{
			"ip": opts.IpAddress,
			"ua": opts.UserAgent,
			"hn": opts.Hostname,
		},
		Id:       sessionId,
		Issuer:   "opsicle/controller",
		OrgCode:  orgCode,
		OrgId:    orgId,
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
