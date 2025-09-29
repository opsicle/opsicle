package models

var (
	cachePrefixAutomationPending = "automation"
	sessionSigningToken          = "supersecretkey"
)

func SetSessionSigningToken(token string) {
	sessionSigningToken = token
}
