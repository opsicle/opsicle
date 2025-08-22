package validate

import (
	"errors"
	"fmt"
	"unicode"
)

var (
	allowedSymbolsInOrgCode = map[rune]bool{
		'-': true,
	}
)

const (
	OrgCodeMinLength = 6
	OrgCodeMaxLength = 32
)

func OrgCode(orgCode string) error {
	var errs []error

	if len(orgCode) < OrgCodeMinLength {
		errs = append(errs, ErrorTooShort)
	}
	if len(orgCode) > OrgCodeMaxLength {
		errs = append(errs, ErrorTooLong)
	}

	invalidCharacters := map[rune]struct{}{}
	for _, r := range orgCode {
		if !isRuneAllowedInOrgCode(r) {
			invalidCharacters[r] = struct{}{}
		}
	}
	if _, ok := allowedSymbolsInOrgCode[rune(orgCode[0])]; ok {
		errs = append(errs, fmt.Errorf("%w: character[%q] is not allowed as the first character", ErrorInvalidPrefixCharacter, rune(orgCode[0])))
	}
	if _, ok := allowedSymbolsInOrgCode[rune(orgCode[len(orgCode)-1])]; ok {
		errs = append(errs, fmt.Errorf("%w: character[%q] is not allowed as the last character", ErrorInvalidPrefixCharacter, rune(orgCode[len(orgCode)-1])))
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

func isRuneAllowedInOrgCode(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	if _, ok := allowedSymbolsInOrgCode[r]; ok {
		return true
	}
	return false
}
