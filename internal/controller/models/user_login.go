package models

import "time"

type UserLogin struct {
	Id           string    `json:"id"`
	UserId       string    `json:"userId"`
	AttemptedAt  time.Time `json:"attemptedAt"`
	IpAddress    string    `json:"ipAddress"`
	UserAgent    string    `json:"userAgent"`
	IsPendingMfa bool      `json:"isPendingMfa"`
	ExpiresAt    time.Time `json:"expiresAt"`
}
