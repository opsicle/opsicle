package auth

import (
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
)

func CreateTotpSeed(issuer, accountName string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

func ValidateTotpToken(secret string, token string) (bool, error) {
	validTokenTimestamps := []time.Time{
		time.Now().Add(-1 * time.Minute).Truncate(30 * time.Second),
		time.Now().Truncate(30 * time.Second),
		time.Now().Add(time.Minute).Truncate(30 * time.Second),
	}
	fmt.Println(validTokenTimestamps)
	for _, validTokenTimestamp := range validTokenTimestamps {
		code, err := totp.GenerateCode(secret, validTokenTimestamp)
		if err != nil {
			return false, fmt.Errorf("failed to generate totp token: %s", err)
		}
		if token == code {
			return true, nil
		}
	}
	return false, nil
}

func CreateTotpTokens(secret string, validity time.Duration) ([]string, error) {
	instanceCount := validity / time.Minute
	results := []string{}
	for i := 0; i < int(instanceCount); i++ {
		forTime := time.Now().Add(time.Minute * time.Duration(i)).Truncate(30 * time.Second)
		logrus.Infof("generating code for time: %s", forTime.Format("15:04:05"))
		code, err := totp.GenerateCode(secret, forTime)
		if err != nil {
			return nil, fmt.Errorf("failed to generate totp token: %s", err)
		}
		results = append(results, code)
	}
	return results, nil
}
