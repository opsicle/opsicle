package validate

import (
	"errors"
)

var (
	ErrorEmptyString             = errors.New("empty_string")
	ErrorInvalidCharacter        = errors.New("invalid_character")
	ErrorInvalidPostfixCharacter = errors.New("invalid_postfix_character")
	ErrorInvalidPrefixCharacter  = errors.New("invalid_prefix_character")

	allowedSymbolsInOrgName = []rune{
		'.',
		',',
		'-',
		' ',
		'&',
		'!',
		'(',
		')',
		':',
	}
)

const (
	OrgNameMinLength = 6
	OrgNameMaxLength = 255
)

func OrgName(orgName string) error {
	return do(
		orgName,
		andS(
			hasMinLength(OrgNameMinLength),
			hasMaxLength(OrgNameMaxLength),
		),
		orR(
			isCharacterInAllowlist(allowedSymbolsInOrgName),
			isUnicodeAlnum(),
		),
	)
}
