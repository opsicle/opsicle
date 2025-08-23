package validate

var (
	allowedSymbolsInOrgCode = []rune{'-'}
)

const (
	OrgCodeMinLength = 6
	OrgCodeMaxLength = 32
)

func OrgCode(orgCode string) error {
	return do(
		orgCode,
		andS(
			hasMinLength(OrgCodeMinLength),
			hasMaxLength(OrgCodeMaxLength),
			isPrefixedWithLatinAlnum(),
			isPostfixedWithLatinAlnum(),
		),
		orR(
			isLatinAlnum(),
			isCharacterInAllowlist(allowedSymbolsInOrgCode),
		),
	)
}
