package validate

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrorEmailAliasesNotAllowed          = errors.New("email_aliases_not_allowed")
	ErrorEmailDomainInvalid              = errors.New("email_domain_invalid")
	ErrorEmailDomainTldNotAllowlisted    = errors.New("email_domain_tld_not_allowlisted")
	ErrorEmailEmptyDomain                = errors.New("email_empty_domain")
	ErrorEmailInvalidAt                  = errors.New("email_invalid_at")
	ErrorEmailMissing                    = errors.New("email_missing")
	ErrorEmailUserPartInvalidLength      = errors.New("email_user_part_invalid_length")
	ErrorEmailUserPartNonAscii           = errors.New("email_user_part_non_ascii")
	ErrorEmailUserPartIlledgalChar       = errors.New("email_user_part_illegal_char")
	ErrorEmailUserPartConsecutiveSymbols = errors.New("email_user_part_consecutive_symbols")
	ErrorEmailUserPartLeadingSymbols     = errors.New("email_user_part_leading_symbols")
	ErrorEmailUserPartTrailingSymbols    = errors.New("email_user_part_trailing_symbols")
)

var domainRegex = regexp.MustCompile(
	`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*\.[a-z]{2,}$`,
)

var validDomainTlds = regexp.MustCompile(
	`\.(ai|com\.[a-z]+|co|dev|io|me|net|org|gov\.[a-z]+|)$`,
)

func Email(email string) error {
	errs := []error{}

	if len(email) <= 3 {
		return ErrorEmailMissing
	}

	at := strings.IndexByte(email, '@')
	if at <= 0 || at != strings.LastIndexByte(email, '@') {
		errs = append(errs, ErrorEmailInvalidAt)
	}

	user := email[:at]
	domain := email[at+1:]
	if len(domain) == 0 {
		errs = append(errs, ErrorEmailEmptyDomain)
	}
	if !domainRegex.MatchString(domain) {
		errs = append(errs, ErrorEmailDomainInvalid)
	}
	if !validDomainTlds.MatchString(domain) {
		errs = append(errs, ErrorEmailDomainTldNotAllowlisted)
	}

	if len(user) < 1 || len(user) > 64 {
		errs = append(errs, ErrorEmailUserPartInvalidLength)
	}

	var prev rune
	for i, r := range user {
		if r > unicode.MaxASCII {
			errs = append(errs, ErrorEmailUserPartNonAscii)
		}

		// Disallow '+'
		if r == '+' {
			errs = append(errs, ErrorEmailAliasesNotAllowed)
		}

		// Allowed chars: letters, digits, '.', '-'
		if !(isASCIILetterOrDigit(byte(r)) || r == '+' || r == '.' || r == '-' || r == '_') {
			errs = append(errs, ErrorEmailUserPartIlledgalChar)
		}

		// No consecutive '.' or '-'
		if (r == '.' && prev == '.') || (r == '-' && prev == '-') {
			errs = append(errs, ErrorEmailUserPartConsecutiveSymbols)
		}

		// No leading '.' or '-'
		if i == 0 && (r == '.' || r == '-' || r == '_') {
			errs = append(errs, ErrorEmailUserPartLeadingSymbols)
		}

		// No trailing '.' or '-'
		if i == len(user)-1 && (r == '.' || r == '-' || r == '_') {
			errs = append(errs, ErrorEmailUserPartTrailingSymbols)
		}

		prev = r
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func isASCIILetterOrDigit(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9')
}
