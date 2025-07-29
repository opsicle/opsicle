package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the structure of the JWT payload.
type Claims struct {
	Ext      map[string]string `json:"ext"`
	UserID   string            `json:"userId"`
	Username string            `json:"username"`
	OrgCode  string            `json:"orgCode"`
	OrgId    string            `json:"orgId"`
	jwt.RegisteredClaims
}

type GenerateJwtOpts struct {
	Audience string
	Ext      map[string]string
	Id       string
	Issuer   string
	OrgCode  string
	OrgId    string
	Secret   string
	Subject  string
	Ttl      time.Duration
	UserId   string
	Username string
}

// GenerateJwt creates a signed JWT for a user.
func GenerateJwt(opts GenerateJwtOpts) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   opts.UserId,
		Username: opts.Username,
		OrgCode:  opts.OrgCode,
		OrgId:    opts.OrgId,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{opts.Audience},
			ID:        opts.Id,
			Issuer:    opts.Issuer,
			Subject:   opts.Subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(opts.Ttl)),
		},
	}
	if len(opts.Ext) > 0 {
		claims.Ext = map[string]string{}
		for key, value := range opts.Ext {
			claims.Ext[key] = value
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(opts.Secret))
}

// ValidateJWT verifies the token's signature and expiry.
// Returns the Claims if valid, otherwise an error.
func ValidateJWT(jwtSecret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("failed to validate token signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		// if errors.Is(err, jwt.ErrTokenExpired) {

		// }
		return nil, fmt.Errorf("failed to parse token claims: %s", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("failed to validate token claims structure")
	}

	if claims.ExpiresAt == nil || time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("failed to validate token expiry")
	}

	return claims, nil
}
