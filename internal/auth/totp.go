package auth

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

var totpOpts = totp.ValidateOpts{
	Period:    30,
	Skew:      1,
	Digits:    6,
	Algorithm: otp.AlgorithmSHA1,
}

type GetTotpQrCodeOpts struct {
	Issuer    string
	AccountId string
	Secret    string
}

func GetTotpQrCode(opts GetTotpQrCodeOpts) (string, error) {
	label := url.PathEscape(fmt.Sprintf("%s:%s", opts.Issuer, opts.AccountId))
	q := url.Values{}
	q.Set("secret", strings.ToUpper(opts.Secret)) // most apps expect uppercase
	q.Set("issuer", opts.Issuer)
	q.Set("algorithm", totpOpts.Algorithm.String())
	q.Set("digits", fmt.Sprintf("%d", totpOpts.Digits))
	q.Set("period", fmt.Sprintf("%d", totpOpts.Period))

	qr, err := qrcode.New(fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode()), qrcode.Low)
	if err != nil {
		return "", fmt.Errorf("failed to create qr code: %w", err)
	}
	var b strings.Builder
	// Get the QR code bitmap
	bitmap := qr.Bitmap()
	for y := 0; y < len(bitmap); y += 2 {
		for x := 0; x < len(bitmap[y]); x++ {
			// Upper pixel
			top := bitmap[y][x]
			// Lower pixel (if exists)
			bottom := false
			if y+1 < len(bitmap) {
				bottom = bitmap[y+1][x]
			}
			switch {
			case top && bottom:
				fmt.Fprintf(&b, "\u2588") // Full block
			case top && !bottom:
				fmt.Fprintf(&b, "\u2580") // Upper half block
			case !top && bottom:
				fmt.Fprintf(&b, "\u2584") // Lower half block
			default:
				fmt.Fprintf(&b, " ") // Space
			}
		}
		fmt.Fprintf(&b, "\n")
	}
	return b.String(), nil
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
			return nil, fmt.Errorf("failed to generate totp token: %w", err)
		}
		results = append(results, code)
	}
	return results, nil
}
