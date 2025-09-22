package validate

import (
	"errors"
	"strings"
	"unicode"
)

var (
	ErrorConsecutiveReservedCharacters = errors.New("consecutive_reserved_characters")
	ErrorNoDigit                       = errors.New("no_digit")
	ErrorNoLowercase                   = errors.New("no_lowercase")
	ErrorNoSymbol                      = errors.New("no_symbol")
	ErrorNoUppercase                   = errors.New("no_uppercase")
	ErrorNotInAllowlistedCharacters    = errors.New("not_in_allowlisted_characters")
	ErrorNotLatinAlnum                 = errors.New("not_latin_alphanumeric")
	ErrorNotUnicodeAlnum               = errors.New("not_unicode_alphanumeric")
	ErrorNotSpace                      = errors.New("not_space")
	ErrorPostfixedWithNonLatinAlnum    = errors.New("cannot_end_with_non_latin_alphanumeric")
	ErrorPostfixedWithNonUnicodeAlnum  = errors.New("cannot_end_with_non_unicode_alphanumeric")
	ErrorPrefixedWithNonLatinAlnum     = errors.New("cannot_start_with_non_latin_alphanumeric")
	ErrorPrefixedWithNonUnicodeAlnum   = errors.New("cannot_start_with_non_unicode_alphanumeric")
	ErrorStringTooShort                = errors.New("string_too_short")
	ErrorStringTooLong                 = errors.New("string_too_long")

	ErrorInvalidUuid = errors.New("invalid_uuid")
)

func hasDigit() StringRule {
	return func(s string) error {
		for _, r := range s {
			if unicode.IsDigit(r) {
				return nil
			}
		}
		return ErrorNoDigit
	}
}

func hasMaxLength(l int) StringRule {
	return func(s string) error {
		if len(s) > l {
			return ErrorStringTooLong
		}
		return nil
	}
}

func hasMinLength(l int) StringRule {
	return func(s string) error {
		if len(s) < l {
			return ErrorStringTooShort
		}
		return nil
	}
}

func hasNoConsecutive(r rune) StringRule {
	return func(s string) error {
		if strings.Contains(s, string([]rune{r, r})) {
			return ErrorConsecutiveReservedCharacters
		}
		return nil
	}
}

func hasLowercase() StringRule {
	return func(s string) error {
		for _, r := range s {
			if unicode.IsLower(r) {
				return nil
			}
		}
		return ErrorNoLowercase
	}
}

func hasSymbol() StringRule {
	return func(s string) error {
		for _, r := range s {
			if unicode.IsPunct(r) || unicode.IsSymbol(r) {
				return nil
			}
		}
		return ErrorNoSymbol
	}
}

func hasUppercase() StringRule {
	return func(s string) error {
		for _, r := range s {
			if unicode.IsUpper(r) {
				return nil
			}
		}
		return ErrorNoUppercase
	}
}

func isCharacterInAllowlist(runes []rune) RuneRule {
	runeMap := map[rune]struct{}{}
	for _, runeInstance := range runes {
		runeMap[runeInstance] = struct{}{}
	}
	return func(curr rune, prev rune) error {
		if _, ok := runeMap[curr]; ok {
			return nil
		}
		return ErrorNotInAllowlistedCharacters
	}
}

func isLatinAlnum() RuneRule {
	return func(curr rune, prev rune) error {
		if (curr >= 'A' && curr <= 'Z') || (curr >= 'a' && curr <= 'z') || (curr >= '0' && curr <= '9') {
			return nil
		}
		return ErrorNotLatinAlnum
	}
}

func isPostfixedWithLatinAlnum() StringRule {
	return func(input string) error {
		if input == "" {
			return nil
		}
		if err := isLatinAlnum()(rune(input[len(input)-1]), ' '); err != nil {
			return ErrorPostfixedWithNonLatinAlnum
		}
		return nil
	}
}

func isPostfixedWithUnicodeAlnum() StringRule {
	return func(input string) error {
		if input == "" {
			return nil
		}
		if err := isUnicodeAlnum()(rune(input[len(input)-1]), ' '); err != nil {
			return ErrorPostfixedWithNonUnicodeAlnum
		}
		return nil
	}
}

func isPrefixedWithUnicodeAlnum() StringRule {
	return func(input string) error {
		if input == "" {
			return nil
		}
		if err := isUnicodeAlnum()(rune(input[0]), ' '); err != nil {
			return ErrorPrefixedWithNonUnicodeAlnum
		}
		return nil
	}
}

func isPrefixedWithLatinAlnum() StringRule {
	return func(input string) error {
		if input == "" {
			return nil
		}
		if err := isLatinAlnum()(rune(input[0]), ' '); err != nil {
			return ErrorPrefixedWithNonLatinAlnum
		}
		return nil
	}
}

func isUnicodeAlnum() RuneRule {
	return func(curr rune, prev rune) error {
		if unicode.IsLetter(curr) || unicode.IsDigit(curr) {
			return nil
		}
		return ErrorNotUnicodeAlnum
	}
}
