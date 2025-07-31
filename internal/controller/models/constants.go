package models

var (
	sessionSigningToken = "supersecretkey"
)

func SetSessionSigningToken(token string) {
	sessionSigningToken = token
}
