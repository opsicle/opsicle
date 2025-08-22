package validate

import (
	"errors"
	"fmt"
	"unicode"
)

var (
	ErrorEmptyString             = errors.New("empty_string")
	ErrorInvalidCharacter        = errors.New("invalid_character")
	ErrorInvalidPostfixCharacter = errors.New("invalid_postfix_character")
	ErrorInvalidPrefixCharacter  = errors.New("invalid_prefix_character")

	ErrorTooShort = errors.New("too_short")
	ErrorTooLong  = errors.New("too_long")

	allowedSymbolsInOrgName = map[rune]bool{
		'.': true,
		',': true,
		'-': true,
		' ': true,
		'&': true,
		'!': true,
		'(': true,
		')': true,
		':': true,
	}
)

const (
	OrgNameMinLength = 6
	OrgNameMaxLength = 255
)

func OrgName(orgName string) error {
	var errs []error

	if len(orgName) < OrgNameMinLength {
		errs = append(errs, ErrorTooShort)
	}
	if len(orgName) > OrgNameMaxLength {
		errs = append(errs, ErrorTooLong)
	}

	invalidCharacters := map[rune]struct{}{}
	for _, r := range orgName {
		if !isRuneAllowedInOrgName(r) {
			invalidCharacters[r] = struct{}{}
		}
	}
	if len(invalidCharacters) > 0 {
		for invalidCharacter := range invalidCharacters {
			errs = append(errs, fmt.Errorf("%w: character[%q] is not allowed", ErrorInvalidCharacter, invalidCharacter))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func isRuneAllowedInOrgName(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	if _, ok := allowedSymbolsInOrgName[r]; ok {
		return true
	}
	return false
}
