package templates

import (
	"bytes"
	_ "embed"
)

//go:embed email_verification.html
var emailVerificationTemplate []byte

//go:embed password_reset.html
var passwordResetTemplate []byte

func GetEmailVerificationMessage(
	serverAddress string,
	verificationCode string,
	triggererAddr string,
	triggererUserAgent string,
) []byte {
	return bytes.ReplaceAll(
		bytes.ReplaceAll(
			bytes.ReplaceAll(
				bytes.ReplaceAll(
					emailVerificationTemplate,
					[]byte("${EMAIL_VERIFICATION_CODE}"), []byte(verificationCode),
				),
				[]byte("${CONTROLLER_URL}"), []byte(serverAddress),
			),
			[]byte("${REMOTE_ADDR}"), []byte(triggererAddr),
		),
		[]byte("${USER_AGENT}"), []byte(triggererUserAgent),
	)
}

func GetPasswordResetMessage(
	verificationCode string,
	triggererAddr string,
	triggererUserAgent string,
) []byte {
	return bytes.ReplaceAll(
		bytes.ReplaceAll(
			bytes.ReplaceAll(
				passwordResetTemplate,
				[]byte("${PWRESET_VERIFICATION_CODE}"), []byte(verificationCode),
			),
			[]byte("${REMOTE_ADDR}"), []byte(triggererAddr),
		),
		[]byte("${USER_AGENT}"), []byte(triggererUserAgent),
	)
}
