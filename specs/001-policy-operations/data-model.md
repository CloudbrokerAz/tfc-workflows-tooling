# Data Model: Policy Operations

**Feature**: Policy Evaluation and Override Operations  
**Branch**: 001-policy-operations  
**Date**: 2025-12-04

## Overview

This document defines the data structures for policy evaluation and override operations in tfc-workflows-tooling. These entities normalize data from both legacy (`policy-checks`) and modern (`task-stages/policy-evaluations`) TFC API formats.

---

## Entity: PolicyEvaluation

### Purpose

Represents the aggregated results of Sentinel policy evaluations for a Terraform Cloud run, including pass/fail counts and details of failed policies.

### Attributes

| Attribute              | Type             | Required    | Description                                                                   |
| ---------------------- | ---------------- | ----------- | ----------------------------------------------------------------------------- |
| `RunID`                | `string`         | Yes         | TFC run identifier (format: `run-[a-zA-Z0-9]+`)                               |
| `PolicyStageID`        | `string`         | Conditional | Task stage ID when using modern API (mutually exclusive with PolicyCheckID)   |
| `PolicyCheckID`        | `string`         | Conditional | Policy check ID when using legacy API (mutually exclusive with PolicyStageID) |
| `TotalCount`           | `int`            | Yes         | Total number of policies evaluated                                            |
| `PassedCount`          | `int`            | Yes         | Number of policies that passed                                                |
| `AdvisoryFailedCount`  | `int`            | Yes         | Number of advisory policies that failed (non-blocking)                        |
| `MandatoryFailedCount` | `int`            | Yes         | Number of mandatory policies that failed (requires override)                  |
| `ErroredCount`         | `int`            | Yes         | Number of policies that encountered errors during evaluation                  |
| `FailedPolicies`       | `[]PolicyDetail` | Yes         | Details of failed mandatory policies (empty if none)                          |
| `Status`               | `string`         | Yes         | Overall policy evaluation status (`passed`, `failed`, `errored`, `pending`)   |
| `RequiresOverride`     | `bool`           | Yes         | True if mandatory policies failed and override is needed                      |

### Relationships

- **Belongs to**: One `Run` (via `RunID`)
- **Has many**: `PolicyDetail` (failed policies)

### Validation Rules

1. `RunID` must match regex pattern `^run-[a-zA-Z0-9]+$`
2. Exactly one of `PolicyStageID` or `PolicyCheckID` must be non-empty
3. All count fields must be non-negative integers (`>= 0`)
4. `TotalCount` should equal sum of individual counts: `PassedCount + AdvisoryFailedCount + MandatoryFailedCount + ErroredCount`
5. `RequiresOverride` must be `true` if and only if `MandatoryFailedCount > 0`
6. `Status` must be one of: `passed`, `failed`, `errored`, `pending`, `running`
7. `FailedPolicies` length must not exceed `MandatoryFailedCount + AdvisoryFailedCount`

### State Transitions

PolicyEvaluation is a **read-only** entity representing a snapshot of evaluation results. It does not undergo state transitions. The underlying Run's status changes, but PolicyEvaluation is re-fetched rather than updated.

### Example (JSON)

```json
{
  "run_id": "run-abc123def456",
  "policy_stage_id": "ts-xyz789",
  "total_count": 8,
  "passed_count": 5,
  "advisory_failed_count": 1,
  "mandatory_failed_count": 2,
  "errored_count": 0,
  "failed_policies": [
    {
      "policy_name": "aws-cost-limit",
      "enforcement_level": "mandatory",
      "status": "failed",
      "description": "Terraform run exceeds $500 daily cost threshold"
    },
    {
      "policy_name": "security-group-ingress",
      "enforcement_level": "mandatory",
      "status": "failed",
      "description": "Security group allows unrestricted ingress from 0.0.0.0/0"
    }
  ],
  "status": "failed",
  "requires_override": true
}
```

### Go Struct

```go
// PolicyEvaluation represents normalized policy evaluation results
type PolicyEvaluation struct {
    RunID                 string         `json:"run_id"`
    PolicyStageID         string         `json:"policy_stage_id,omitempty"`
    PolicyCheckID         string         `json:"policy_check_id,omitempty"`
    TotalCount            int            `json:"total_count"`
    PassedCount           int            `json:"passed_count"`
    AdvisoryFailedCount   int            `json:"advisory_failed_count"`
    MandatoryFailedCount  int            `json:"mandatory_failed_count"`
    ErroredCount          int            `json:"errored_count"`
    FailedPolicies        []PolicyDetail `json:"failed_policies"`
    Status                string         `json:"status"`
    RequiresOverride      bool           `json:"requires_override"`
}

// Validate checks PolicyEvaluation data integrity
func (pe *PolicyEvaluation) Validate() error {
    if !validStringID(&pe.RunID) {
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
```

---

## Entity: PolicyDetail

### Purpose

Represents detailed information about a single failed policy within a policy evaluation.

### Attributes

| Attribute          | Type     | Required | Description                                                      |
| ------------------ | -------- | -------- | ---------------------------------------------------------------- |
| `PolicyName`       | `string` | Yes      | Name of the policy (e.g., `aws-cost-limit`, `security-baseline`) |
| `EnforcementLevel` | `string` | Yes      | Enforcement level: `mandatory` or `advisory`                     |
| `Status`           | `string` | Yes      | Policy evaluation status: `failed` or `errored`                  |
| `Description`      | `string` | No       | Human-readable failure description                               |

### Validation Rules

1. `PolicyName` must not be empty
2. `EnforcementLevel` must be exactly `mandatory` or `advisory`
3. `Status` must be exactly `failed` or `errored`
4. `Description` is optional but recommended for user clarity

### Example (JSON)

```json
{
  "policy_name": "aws-cost-limit",
  "enforcement_level": "mandatory",
  "status": "failed",
  "description": "Terraform run exceeds $500 daily cost threshold"
}
```

### Go Struct

```go
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
```

---

## Entity: PolicyOverride

### Purpose

Represents a policy override action, including justification, status transitions, and completion tracking.

### Attributes

| Attribute          | Type        | Required    | Description                                                          |
| ------------------ | ----------- | ----------- | -------------------------------------------------------------------- |
| `RunID`            | `string`    | Yes         | TFC run identifier (format: `run-[a-zA-Z0-9]+`)                      |
| `PolicyStageID`    | `string`    | Conditional | Task stage ID when using modern API                                  |
| `PolicyCheckID`    | `string`    | Conditional | Policy check ID when using legacy API                                |
| `Justification`    | `string`    | Yes         | Reason for override (minimum 10 characters)                          |
| `InitialStatus`    | `string`    | Yes         | Run status before override (should be `post_plan_awaiting_decision`) |
| `FinalStatus`      | `string`    | Yes         | Run status after override (e.g., `policy_override`, `apply_queued`)  |
| `OverrideComplete` | `bool`      | Yes         | Whether override operation completed successfully                    |
| `Timestamp`        | `time.Time` | Yes         | When override was applied (UTC)                                      |

### Relationships

- **Belongs to**: One `Run` (via `RunID`)
- **References**: One `PolicyEvaluation` (the evaluation being overridden)

### Validation Rules

1. `RunID` must match regex pattern `^run-[a-zA-Z0-9]+$`
2. Exactly one of `PolicyStageID` or `PolicyCheckID` must be non-empty
3. `Justification` must be at least 10 characters (enforces meaningful explanation)
4. `InitialStatus` must be `post_plan_awaiting_decision` (only status where override is valid)
5. `FinalStatus` must be one of:
   - `policy_override` (override in progress)
   - `post_plan_completed` (override complete, awaiting apply)
   - `apply_queued` (workspace has auto-apply enabled)
   - `discarded` (run was discarded during override)
   - `errored` (override failed)
6. `OverrideComplete` must be `true` if `FinalStatus` is terminal state
7. `Timestamp` must be in UTC timezone

### State Transitions

```
┌─────────────────────────────┐
│ post_plan_awaiting_decision │  (Initial: mandatory policies failed)
└──────────────┬──────────────┘
               │
               │ [Override Applied]
               ▼
       ┌───────────────┐
       │ policy_override│  (Transient: override processing)
       └───────┬────────┘
               │
     ┌─────────┴─────────────┬──────────────────┐
     │                       │                  │
     ▼                       ▼                  ▼
┌────────────────┐  ┌───────────────┐  ┌──────────┐
│post_plan_      │  │ apply_queued  │  │ discarded│
│completed       │  │ (auto-apply)  │  │          │
└────────────────┘  └───────────────┘  └──────────┘
(terminal)          (terminal)         (terminal)
```

### Example (JSON)

```json
{
  "run_id": "run-abc123def456",
  "policy_stage_id": "ts-xyz789",
  "justification": "Emergency hotfix approved by CTO - Incident INC-12345",
  "initial_status": "post_plan_awaiting_decision",
  "final_status": "policy_override",
  "override_complete": true,
  "timestamp": "2025-12-04T10:30:45Z"
}
```

### Go Struct

```go
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
    if !validStringID(&po.RunID) {
        return fmt.Errorf("invalid run ID: %s", po.RunID)
    }

    if po.PolicyStageID == "" && po.PolicyCheckID == "" {
        return fmt.Errorf("either PolicyStageID or PolicyCheckID must be set")
    }

    if len(po.Justification) < 10 {
        return fmt.Errorf("justification must be at least 10 characters")
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
```

---

## Supporting Types

### Run Status Constants

From `github.com/hashicorp/go-tfe`:

```go
const (
    // Pre-override statuses
    RunPostPlanAwaitingDecision = tfe.RunStatus("post_plan_awaiting_decision")

    // Override-related statuses
    RunPolicyOverride        = tfe.RunStatus("policy_override")
    RunPostPlanCompleted     = tfe.RunStatus("post_plan_completed")
    RunApplyQueued           = tfe.RunStatus("apply_queued")

    // Terminal failure statuses
    RunDiscarded      = tfe.RunStatus("discarded")
    RunCanceled       = tfe.RunStatus("canceled")
    RunForceCanceled  = tfe.RunStatus("force_canceled")
    RunErrored        = tfe.RunStatus("errored")
)
```

### Helper Functions

```go
// validStringID checks if a string is a valid TFC resource ID
func validStringID(id *string) bool {
    if id == nil || *id == "" {
        return false
    }
    return regexp.MustCompile(`^[a-z]+-[a-zA-Z0-9]+$`).MatchString(*id)
}

// isTerminalStatus checks if a run status is terminal (no more transitions)
func isTerminalStatus(status tfe.RunStatus) bool {
    terminalStatuses := []tfe.RunStatus{
        tfe.RunApplied,
        tfe.RunPlannedAndFinished,
        tfe.RunDiscarded,
        tfe.RunCanceled,
        tfe.RunForceCanceled,
        tfe.RunErrored,
    }
    for _, ts := range terminalStatuses {
        if status == ts {
            return true
        }
    }
    return false
}

// canOverride checks if a run status allows policy override
func canOverride(status tfe.RunStatus) bool {
    return status == tfe.RunPostPlanAwaitingDecision
}
```

---

## Data Flow

### Policy Evaluation Retrieval

```
User Command
    │
    ├─> policy_show.go (CLI)
    │       │
    │       └─> PolicyService.GetPolicyEvaluation(ctx, runID)
    │               │
    │               ├─> [Try Modern API] tfe.TaskStages.List(ctx, runID)
    │               │       │
    │               │       ├─> SUCCESS: tfe.PolicyEvaluations.Read(ctx, stageID)
    │               │       │       │
    │               │       │       └─> Normalize to PolicyEvaluation
    │               │       │
    │               │       └─> FAIL: Fall back to legacy
    │               │
    │               └─> [Legacy API] tfe.PolicyChecks.Read(ctx, policyCheckID)
    │                       │
    │                       └─> Normalize to PolicyEvaluation
    │
    └─> Output (JSON or human-readable)
```

### Policy Override Application

```
User Command
    │
    ├─> policy_override.go (CLI)
    │       │
    │       └─> PolicyService.OverridePolicy(ctx, runID, justification)
    │               │
    │               ├─> Validate: Run status == post_plan_awaiting_decision
    │               │
    │               ├─> [Detect API Format]
    │               │       │
    │               │       ├─> Modern: tfe.TaskStages.Override(ctx, stageID)
    │               │       │
    │               │       └─> Legacy: tfe.PolicyChecks.Override(ctx, checkID)
    │               │
    │               ├─> Add Comment: tfe.Comments.Create(ctx, runID, justification)
    │               │
    │               └─> Return PolicyOverride struct
    │
    └─> Output confirmation
```

### Wait for Decision

```
User Command
    │
    ├─> policy_wait.go (CLI)
    │       │
    │       └─> PolicyService.WaitForPolicyDecision(ctx, runID, timeout)
    │               │
    │               └─> [Polling Loop with Retry]
    │                       │
    │                       ├─> tfe.Runs.Read(ctx, runID) every 10-30s
    │                       │
    │                       ├─> Check status:
    │                       │       ├─> policy_override → EXIT 0 (success)
    │                       │       ├─> discarded → EXIT 1 (discarded)
    │                       │       ├─> canceled → EXIT 2 (canceled)
    │                       │       ├─> errored → EXIT 4 (error)
    │                       │       └─> still awaiting → RETRY
    │                       │
    │                       └─> Timeout → EXIT 3 (timeout)
    │
    └─> Exit with appropriate code
```

---

## Mapping from go-tfe SDK Types

### Modern API: TaskStage → PolicyEvaluation

```go
// Source: *tfe.TaskStage with *tfe.PolicyEvaluation
// Target: PolicyEvaluation

func mapFromTaskStage(taskStage *tfe.TaskStage, policyEval *tfe.PolicyEvaluation) *PolicyEvaluation {
    result := &PolicyEvaluation{
        PolicyStageID:        taskStage.ID,
        Status:               string(policyEval.Status),
        PassedCount:          policyEval.ResultCount.Passed,
        AdvisoryFailedCount:  policyEval.ResultCount.AdvisoryFailed,
        MandatoryFailedCount: policyEval.ResultCount.MandatoryFailed,
        ErroredCount:         policyEval.ResultCount.Errored,
        RequiresOverride:     policyEval.ResultCount.MandatoryFailed > 0,
    }

    result.TotalCount = result.PassedCount + result.AdvisoryFailedCount +
                        result.MandatoryFailedCount + result.ErroredCount

    // Extract failed mandatory policies
    for _, outcome := range policyEval.PolicySetOutcomes {
        if outcome.Outcomes.MandatoryFailed > 0 {
            result.FailedPolicies = append(result.FailedPolicies, PolicyDetail{
                PolicyName:       outcome.PolicySetName,
                EnforcementLevel: "mandatory",
                Status:           "failed",
                Description:      outcome.PolicySetDescription,
            })
        }
    }

    return result
}
```

### Legacy API: PolicyCheck → PolicyEvaluation

```go
// Source: *tfe.PolicyCheck
// Target: PolicyEvaluation

func mapFromPolicyCheck(policyCheck *tfe.PolicyCheck) *PolicyEvaluation {
    result := &PolicyEvaluation{
        PolicyCheckID: policyCheck.ID,
        Status:        string(policyCheck.Status),
    }

    // Aggregate counts from outcomes
    for _, outcome := range policyCheck.PolicySetOutcomes {
        switch outcome.Outcome {
        case "passed":
            result.PassedCount++
        case "failed":
            if outcome.EnforcementLevel == "mandatory" {
                result.MandatoryFailedCount++
                result.FailedPolicies = append(result.FailedPolicies, PolicyDetail{
                    PolicyName:       outcome.PolicyName,
                    EnforcementLevel: "mandatory",
                    Status:           "failed",
                })
            } else {
                result.AdvisoryFailedCount++
            }
        case "errored":
            result.ErroredCount++
        }
    }

    result.TotalCount = result.PassedCount + result.AdvisoryFailedCount +
                        result.MandatoryFailedCount + result.ErroredCount
    result.RequiresOverride = result.MandatoryFailedCount > 0

    return result
}
```

---

## Testing Strategy

### Unit Tests

1. **Validation Tests**: Verify each entity's `Validate()` method catches invalid data
2. **Mapping Tests**: Test `mapFromTaskStage()` and `mapFromPolicyCheck()` with mock data
3. **State Transition Tests**: Verify `canOverride()`, `isTerminalStatus()` logic

### Integration Tests

1. **Round-Trip Tests**: Create run with policies, retrieve evaluation, verify counts match
2. **API Format Tests**: Test against TFC (modern) and TFE 2022.x (legacy)
3. **Override Flow Tests**: Apply override, wait for completion, verify final status

### Example Test

```go
func TestPolicyEvaluationValidation(t *testing.T) {
    tests := []struct {
        name    string
        eval    *PolicyEvaluation
        wantErr bool
    }{
        {
            name: "valid modern API",
            eval: &PolicyEvaluation{
                RunID:                "run-abc123",
                PolicyStageID:        "ts-xyz789",
                TotalCount:           5,
                PassedCount:          3,
                MandatoryFailedCount: 2,
                RequiresOverride:     true,
            },
            wantErr: false,
        },
        {
            name: "invalid count mismatch",
            eval: &PolicyEvaluation{
                RunID:                "run-abc123",
                PolicyStageID:        "ts-xyz789",
                TotalCount:           10, // Should be 5
                PassedCount:          3,
                MandatoryFailedCount: 2,
                RequiresOverride:     true,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.eval.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```
