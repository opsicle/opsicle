package validate

const (
	minimumPasswordLength = 12
)

func Password(password string) error {
	return do(
		password,
		andS(
			hasMinLength(minimumPasswordLength),
			hasUppercase(),
			hasLowercase(),
			hasDigit(),
			hasSymbol(),
		),
	)
}
