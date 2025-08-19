package auth

import "errors"

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

	// ErrorJwtTokenExpired indicates the token has expired
	ErrorJwtTokenExpired = errors.New("jwt_token_expired")
	// ErrorJwtTokenSignature indicates token signature validation failed
	ErrorJwtTokenSignature = errors.New("jwt_token_signature")
	// ErrorJwtClaims indicates that the claim data couldn't be parsed
	ErrorJwtClaimsInvalid = errors.New("jwt_claims_invalid")

	ErrorPasswordNoUppercase = errors.New("password_requires_uppercase")
	ErrorPasswordNoLowercase = errors.New("password_requires_lowercase")
	ErrorPasswordNoNumber    = errors.New("password_requires_number")
	ErrorPasswordNoSymbol    = errors.New("password_requires_symbol")
)
