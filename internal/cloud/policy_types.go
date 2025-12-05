// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"fmt"
	"regexp"
	"time"
)

// PolicyEvaluation represents normalized policy evaluation results
type PolicyEvaluation struct {
	RunID                string         `json:"run_id"`
	PolicyStageID        string         `json:"policy_stage_id,omitempty"`
	PolicyCheckID        string         `json:"policy_check_id,omitempty"`
	TotalCount           int            `json:"total_count"`
	PassedCount          int            `json:"passed_count"`
	AdvisoryFailedCount  int            `json:"advisory_failed_count"`
	MandatoryFailedCount int            `json:"mandatory_failed_count"`
	ErroredCount         int            `json:"errored_count"`
	FailedPolicies       []PolicyDetail `json:"failed_policies"`
	Status               string         `json:"status"`
	RequiresOverride     bool           `json:"requires_override"`
}

// Validate checks PolicyEvaluation data integrity
func (pe *PolicyEvaluation) Validate() error {
	if !validStringID(pe.RunID) {
		return fmt.Errorf("invalid run ID: %s", pe.RunID)
	}

	if pe.PolicyStageID == "" && pe.PolicyCheckID == "" {
		return fmt.Errorf("either PolicyStageID or PolicyCheckID must be set")
	}

	if pe.PolicyStageID != "" && pe.PolicyCheckID != "" {
		return fmt.Errorf("PolicyStageID and PolicyCheckID are mutually exclusive")
	}

	if pe.TotalCount < 0 || pe.PassedCount < 0 || pe.AdvisoryFailedCount < 0 ||
		pe.MandatoryFailedCount < 0 || pe.ErroredCount < 0 {
		return fmt.Errorf("counts must be non-negative")
	}

	expectedTotal := pe.PassedCount + pe.AdvisoryFailedCount + pe.MandatoryFailedCount + pe.ErroredCount
	if pe.TotalCount != expectedTotal {
		return fmt.Errorf("total count mismatch: expected %d, got %d", expectedTotal, pe.TotalCount)
	}

	if pe.RequiresOverride != (pe.MandatoryFailedCount > 0) {
		return fmt.Errorf("RequiresOverride mismatch with MandatoryFailedCount")
	}

	return nil
}

// PolicyDetail represents individual policy failure information
type PolicyDetail struct {
	PolicyName       string `json:"policy_name"`
	EnforcementLevel string `json:"enforcement_level"`
	Status           string `json:"status"`
	Description      string `json:"description,omitempty"`
}

// Validate checks PolicyDetail data integrity
func (pd *PolicyDetail) Validate() error {
	if pd.PolicyName == "" {
		return fmt.Errorf("policy name must not be empty")
	}

	if pd.EnforcementLevel != "mandatory" && pd.EnforcementLevel != "advisory" {
		return fmt.Errorf("invalid enforcement level: %s", pd.EnforcementLevel)
	}

	if pd.Status != "failed" && pd.Status != "errored" {
		return fmt.Errorf("invalid status: %s", pd.Status)
	}

	return nil
}

// PolicyOverride represents a policy override action
type PolicyOverride struct {
	RunID            string    `json:"run_id"`
	PolicyStageID    string    `json:"policy_stage_id,omitempty"`
	PolicyCheckID    string    `json:"policy_check_id,omitempty"`
	Justification    string    `json:"justification"`
	InitialStatus    string    `json:"initial_status"`
	FinalStatus      string    `json:"final_status"`
	OverrideComplete bool      `json:"override_complete"`
	Timestamp        time.Time `json:"timestamp"`
}

// Validate checks PolicyOverride data integrity
func (po *PolicyOverride) Validate() error {
	if !validStringID(po.RunID) {
		return fmt.Errorf("invalid run ID: %s", po.RunID)
	}

	if po.PolicyStageID == "" && po.PolicyCheckID == "" {
		return fmt.Errorf("either PolicyStageID or PolicyCheckID must be set")
	}

	if po.Justification == "" {
		return fmt.Errorf("justification is required")
	}

	if po.InitialStatus != "post_plan_awaiting_decision" {
		return fmt.Errorf("invalid initial status: %s, expected post_plan_awaiting_decision", po.InitialStatus)
	}

	validFinalStatuses := []string{
		"policy_override",
		"post_plan_completed",
		"apply_queued",
		"discarded",
		"errored",
	}
	valid := false
	for _, s := range validFinalStatuses {
		if po.FinalStatus == s {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid final status: %s", po.FinalStatus)
	}

	return nil
}

// GetPolicyEvaluationOptions configures policy evaluation retrieval
type GetPolicyEvaluationOptions struct {
	RunID  string // Required: TFC run ID
	NoWait bool   // Optional: Fail fast if policies not yet evaluated
}

// Validate checks if options are valid
func (o GetPolicyEvaluationOptions) Validate() error {
	if !validStringID(o.RunID) {
		return ErrInvalidRunID
	}
	return nil
}

// OverridePolicyOptions configures policy override operation
type OverridePolicyOptions struct {
	RunID         string // Required: TFC run ID
	Justification string // Required: Override reason
}

// Validate checks if options are valid
func (o OverridePolicyOptions) Validate() error {
	if !validStringID(o.RunID) {
		return ErrInvalidRunID
	}
	if o.Justification == "" {
		return ErrInvalidJustification
	}
	return nil
}

// validStringID checks if a string is a valid TFC resource ID
func validStringID(id string) bool {
	if id == "" {
		return false
	}
	return regexp.MustCompile(`^[a-z]+-[a-zA-Z0-9]+$`).MatchString(id)
}
