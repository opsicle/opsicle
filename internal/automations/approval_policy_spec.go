package automations

import "opsicle/internal/approvals"

// ApprovalPolicySpec defines the approval mechanism in play for
// the automation
type ApprovalPolicySpec struct {
	// PolicyRef when defined should be a string that references an
	// existing policy which can be retrieved by the controller
	PolicyRef *string `json:"policyRef" yaml:"policyRef"`

	// Spec contains an inline approval policy
	Spec *approvals.PolicySpec `json:"spec" yaml:"spec"`
}
