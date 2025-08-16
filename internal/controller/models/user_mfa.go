package models

import "time"

const (
	MfaTypeTotp = "totp"
)

type UserMfas []UserMfa

func (m UserMfas) GetRedacted() UserMfas {
	output := UserMfas{}
	for _, n := range m {
		output = append(output, n.GetRedacted())
	}
	return output
}

type UserMfa struct {
	Id            string     `json:"id"`
	Type          string     `json:"type"`
	ConfigJson    *string    `json:"configJson"`
	Secret        *string    `json:"secret"`
	UserId        string     `json:"userId"`
	UserEmail     *string    `json:"userEmail"`
	IsVerified    bool       `json:"isVerified"`
	VerifiedAt    *time.Time `json:"verifiedAt"`
	CreatedAt     *time.Time `json:"createdAt"`
	LastUpdatedAt *time.Time `json:"lastUpdatedAt"`
}

func (m UserMfa) GetRedacted() UserMfa {
	n := m
	n.Secret = nil
	return n
}
