package auth

import (
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
)

var totpOpts = totp.ValidateOpts{
	Period:    30,
	Skew:      1,
	Digits:    6,
	Algorithm: otp.AlgorithmSHA1,
}

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
	return totp.ValidateCustom(token, secret, time.Now().UTC(), totpOpts)
}

func CreateTotpTokens(secret string, validity time.Duration) ([]string, error) {
	instanceCount := validity / time.Minute
	results := []string{}
	for i := 0; i < int(instanceCount); i++ {
		forTime := time.Now().Add(time.Minute * time.Duration(i)).Truncate(30 * time.Second)
		logrus.Infof("generating code for time: %s", forTime.Format("15:04:05"))
		code, err := totp.GenerateCodeCustom(secret, forTime, totpOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate totp token: %s", err)
		}
		results = append(results, code)
	}
	return results, nil
}
