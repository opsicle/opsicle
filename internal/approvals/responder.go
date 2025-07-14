package approvals

type AuthorizedResponders []AuthorizedResponder

type AuthorizedResponder struct {
	UserId   *string `json:"userId" yaml:"userId"`
	Username *string `json:"username" yaml:"username"`
	MfaSeed  *string `json:"mfaSeed" yaml:"mfaSeed"`
}
