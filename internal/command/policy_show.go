// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/tfci/internal/cloud"
)

type PolicyShowCommand struct {
	*Meta

	RunID  string
	NoWait bool
}

func (c *PolicyShowCommand) flags() *flag.FlagSet {
	f := c.flagSet("policy show")
	f.StringVar(&c.RunID, "run-id", "", "HCP Terraform Run ID to check policies for.")
	f.BoolVar(&c.NoWait, "no-wait", false, "Fail immediately if policies not yet evaluated (default: wait with retry).")

	return f
}

func (c *PolicyShowCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	if c.RunID == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("checking policies requires a valid run ID (use --run-id)")
		return 1
	}

	// Fetch policy evaluation
	eval, err := c.cloud.GetPolicyEvaluation(c.appCtx, cloud.GetPolicyEvaluationOptions{
		RunID:  c.RunID,
		NoWait: c.NoWait,
	})

	if err != nil {
		status := c.resolveStatus(err)
		c.addOutput("status", string(status))
		c.writer.ErrorResult(fmt.Sprintf("error retrieving policy evaluation for run '%s': %s", c.RunID, err.Error()))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addPolicyEvaluationDetails(eval)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *PolicyShowCommand) addPolicyEvaluationDetails(eval *cloud.PolicyEvaluation) {
	if eval == nil {
		return
	}

	// Add structured outputs
	c.addOutput("run_id", eval.RunID)
	c.addOutput("total_count", fmt.Sprintf("%d", eval.TotalCount))
	c.addOutput("passed_count", fmt.Sprintf("%d", eval.PassedCount))
	c.addOutput("advisory_failed_count", fmt.Sprintf("%d", eval.AdvisoryFailedCount))
	c.addOutput("mandatory_failed_count", fmt.Sprintf("%d", eval.MandatoryFailedCount))
	c.addOutput("errored_count", fmt.Sprintf("%d", eval.ErroredCount))
	c.addOutput("requires_override", fmt.Sprintf("%t", eval.RequiresOverride))
	c.addOutput("policy_status", eval.Status)

	// Add failed policies if any
	if len(eval.FailedPolicies) > 0 {
		failedPoliciesJSON, _ := json.Marshal(eval.FailedPolicies)
		c.addOutput("failed_policies", string(failedPoliciesJSON))
	}

	// Add full payload for JSON output
	c.addOutputWithOpts("payload", eval, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})

	// Human-readable output (when not in JSON mode)
	if !c.json {
		c.writer.Output("\n📊 Policy Evaluation Summary")
		c.writer.Output(fmt.Sprintf("   Total Policies: %d", eval.TotalCount))
		c.writer.Output(fmt.Sprintf("   ✅ Passed: %d", eval.PassedCount))
		c.writer.Output(fmt.Sprintf("   ⚠️  Failed (Advisory): %d", eval.AdvisoryFailedCount))
		c.writer.Output(fmt.Sprintf("   🚫 Failed (Mandatory): %d", eval.MandatoryFailedCount))
		c.writer.Output(fmt.Sprintf("   ❌ Errored: %d", eval.ErroredCount))

		if eval.MandatoryFailedCount > 0 {
			c.writer.Output("\n🚫 Failed Mandatory Policies:")
			for _, policy := range eval.FailedPolicies {
				if policy.EnforcementLevel == "mandatory" {
					c.writer.Output(fmt.Sprintf("   - %s (%s)", policy.PolicyName, policy.EnforcementLevel))
					if policy.Description != "" {
						c.writer.Output(fmt.Sprintf("     %s", policy.Description))
					}
				}
			}
		}

		if eval.RequiresOverride {
			c.writer.Output("\nℹ️  Override Required: Policy override needed to proceed")
		} else {
			c.writer.Output("\n✅ All policies passed or only advisory policies failed")
		}

		// Add run link with simple construction
		c.writer.Output(fmt.Sprintf("\n🔗 View in HCP Terraform: https://app.terraform.io/app/%s/runs/%s", c.organization, eval.RunID))
		c.writer.Output("")
	}
}

func (c *PolicyShowCommand) Help() string {
	helpText := `
Usage: tfci [global options] policy show [options]

	Retrieves and displays Sentinel policy evaluation results for a Terraform Cloud run.
	Automatically waits for policy evaluation to complete unless --no-wait is specified.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with HCP Terraform. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   HCP Terraform Organization Name.

Options:

	-run-id         HCP Terraform Run ID to check policies for (required).

	-no-wait        Fail immediately if policies not yet evaluated. Default behavior is to wait with retry until policies are evaluated.

Exit Codes:

	0   Success, policies retrieved
	1   Error (invalid run ID, API error, network failure)
	`
	return strings.TrimSpace(helpText)
}

func (c *PolicyShowCommand) Synopsis() string {
	return "Retrieves Sentinel policy evaluation results for a run"
}
