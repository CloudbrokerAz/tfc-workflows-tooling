# Research: Policy Evaluation & Override Technical Investigation

**Date**: 2025-12-04  
**Branch**: 001-policy-operations  
**Purpose**: Resolve "NEEDS CLARIFICATION" items from Technical Context and document technology decisions

## API Endpoint Compatibility

### Decision

Use **automatic detection** strategy that tries modern API first, falls back to legacy with graceful degradation.

### Rationale

1. **TFC Evolution**: Terraform Cloud transitioned from `policy-checks` to `task-stages/policy-evaluations` in 2022-2023 timeframe
2. **TFE Versions**: Enterprise installations may still use legacy endpoints depending on version
3. **SDK Support**: go-tfe v1.95.0 supports both formats with distinct client methods
4. **Backwards Compatibility**: Cannot break existing workflows on older TFE instances

### Implementation Strategy

```go
// Detection logic in policy_evaluation.go
func (s *policyService) GetPolicyEvaluation(ctx context.Context, options GetPolicyEvaluationOptions) (*PolicyEvaluation, error) {
    // Step 1: Read run to get workspace relationship
    run, err := s.tfe.Runs.ReadWithOptions(ctx, options.RunID, &tfe.RunReadOptions{
        Include: []tfe.RunIncludeOpt{tfe.RunWorkspace},
    })
    if err != nil {
        return nil, fmt.Errorf("error reading run: %w", err)
    }

    // Step 2: Try modern API (task-stages) first
    taskStages, err := s.tfe.TaskStages.List(ctx, run.ID, &tfe.TaskStageListOptions{})
    if err == nil && len(taskStages.Items) > 0 {
        // Modern API available - use task-stages/policy-evaluations
        return s.getPolicyFromTaskStages(ctx, run, taskStages)
    }

    // Step 3: Fall back to legacy policy-checks API
    log.Printf("[DEBUG] Task stages not available, falling back to legacy policy-checks API")
    policyCheck, err := s.tfe.PolicyChecks.Read(ctx, run.PolicyCheckID)
    if err != nil {
        return nil, fmt.Errorf("error reading policy check: %w", err)
    }

    return s.getPolicyFromPolicyCheck(ctx, run, policyCheck), nil
}
```

### Alternatives Considered

1. **Configuration Flag**: Require users to specify `--api-version modern|legacy`
   - **Rejected**: Poor UX, users don't know their TFE version details
2. **Version Detection**: Query TFE version via `client.Meta.Version()`

   - **Rejected**: Adds extra API call, version-to-feature mapping fragile

3. **Legacy Only**: Only support policy-checks

   - **Rejected**: Modern TFC users miss improved features, forward compatibility lost

4. **Modern Only**: Only support task-stages
   - **Rejected**: Breaks existing TFE users, violates constitution principle of backwards compatibility

### Validation Plan

- Integration tests against TFC (modern API)
- Manual testing against TFE 2022.x (legacy API)
- Error message clarity when both APIs fail

---

## Policy Data Structure Mapping

### Decision

Create **unified PolicyEvaluation struct** that normalizes data from both API formats.

### Rationale

1. **Consistent UX**: Users shouldn't need to know which API is used
2. **Testability**: Single struct easier to mock and validate
3. **Command Simplicity**: CLI commands work with one data type
4. **Future-Proof**: Easy to add new fields as API evolves

### Data Structure

```go
// PolicyEvaluation represents normalized policy results
type PolicyEvaluation struct {
    RunID                 string          `json:"run_id"`
    PolicyStageID         string          `json:"policy_stage_id,omitempty"`   // Modern API
    PolicyCheckID         string          `json:"policy_check_id,omitempty"`   // Legacy API
    TotalCount            int             `json:"total_count"`
    PassedCount           int             `json:"passed_count"`
    AdvisoryFailedCount   int             `json:"advisory_failed_count"`
    MandatoryFailedCount  int             `json:"mandatory_failed_count"`
    ErroredCount          int             `json:"errored_count"`
    FailedPolicies        []PolicyDetail  `json:"failed_policies"`
    Status                string          `json:"status"`
    RequiresOverride      bool            `json:"requires_override"`
}

// PolicyDetail represents individual policy failure
type PolicyDetail struct {
    PolicyName        string `json:"policy_name"`
    EnforcementLevel  string `json:"enforcement_level"`  // "mandatory" | "advisory"
    Status            string `json:"status"`             // "failed" | "errored"
    Description       string `json:"description,omitempty"`
}
```

### Mapping Logic

**From Modern API (task-stages/policy-evaluations)**:

```go
func (s *policyService) getPolicyFromTaskStages(ctx context.Context, run *tfe.Run, taskStages *tfe.TaskStageList) (*PolicyEvaluation, error) {
    // Find policy-evaluation stage
    var policyStage *tfe.TaskStage
    for _, stage := range taskStages.Items {
        if stage.Stage == tfe.PrePlan || stage.Stage == tfe.PostPlan {
            policyStage = stage
            break
        }
    }

    // Read policy evaluation details
    policyEval, err := s.tfe.PolicyEvaluations.Read(ctx, policyStage.ID)
    if err != nil {
        return nil, err
    }

    result := &PolicyEvaluation{
        RunID:                run.ID,
        PolicyStageID:        policyStage.ID,
        Status:               string(policyEval.Status),
        RequiresOverride:     policyEval.ResultCount.MandatoryFailed > 0,
    }

    // Map counts
    result.TotalCount = policyEval.ResultCount.Passed +
                        policyEval.ResultCount.AdvisoryFailed +
                        policyEval.ResultCount.MandatoryFailed +
                        policyEval.ResultCount.Errored
    result.PassedCount = policyEval.ResultCount.Passed
    result.AdvisoryFailedCount = policyEval.ResultCount.AdvisoryFailed
    result.MandatoryFailedCount = policyEval.ResultCount.MandatoryFailed
    result.ErroredCount = policyEval.ResultCount.Errored

    // Extract failed policies (mandatory only for brevity)
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

    return result, nil
}
```

**From Legacy API (policy-checks)**:

```go
func (s *policyService) getPolicyFromPolicyCheck(ctx context.Context, run *tfe.Run, check *tfe.PolicyCheck) *PolicyEvaluation {
    result := &PolicyEvaluation{
        RunID:         run.ID,
        PolicyCheckID: check.ID,
        Status:        string(check.Status),
    }

    // Count outcomes by enforcement level
    for _, outcome := range check.PolicySetOutcomes {
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

### Alternatives Considered

1. **Expose Both Structs**: Return `*tfe.PolicyCheck` or `*tfe.TaskStage` directly

   - **Rejected**: Leaks implementation details, commands need conditional logic

2. **Interface Abstraction**: `type PolicyResult interface { GetCount() int }`
   - **Rejected**: Over-engineered for simple data normalization

---

## Override Endpoint Selection

### Decision

Use **same detection strategy** as evaluation - try modern `TaskStages.Override()`, fall back to legacy `PolicyChecks.Override()`.

### Rationale

1. **Consistency**: Matches evaluation endpoint detection
2. **Correctness**: Override endpoint must match evaluation endpoint used
3. **Error Clarity**: If evaluation used modern API but override tries legacy, failure is confusing

### Implementation

```go
func (s *policyService) OverridePolicy(ctx context.Context, options OverridePolicyOptions) (*PolicyOverride, error) {
    // Validate run status first
    run, err := s.validateOverrideEligibility(ctx, options.RunID)
    if err != nil {
        return nil, err
    }

    // Detect API format
    if run.TaskStage != nil && run.TaskStage.ID != "" {
        // Modern API: override via TaskStages
        return s.overrideViaTaskStage(ctx, run, options.Justification)
    }

    // Legacy API: override via PolicyChecks
    if run.PolicyCheck != nil && run.PolicyCheck.ID != "" {
        return s.overrideViaPolicyCheck(ctx, run, options.Justification)
    }

    return nil, fmt.Errorf("no policy check or task stage found for run %s", run.ID)
}

func (s *policyService) overrideViaTaskStage(ctx context.Context, run *tfe.Run, justification string) (*PolicyOverride, error) {
    // Apply override
    taskStage, err := s.tfe.TaskStages.Override(ctx, run.TaskStage.ID, tfe.TaskStageOverrideOptions{
        Comment: tfe.String(justification),
    })
    if err != nil {
        return nil, fmt.Errorf("error overriding task stage: %w", err)
    }

    // Add comment to run
    _, err = s.tfe.Comments.Create(ctx, run.ID, tfe.CommentCreateOptions{
        Body: tfe.String(fmt.Sprintf("Policy Override: %s", justification)),
    })
    if err != nil {
        log.Printf("[WARN] failed to add comment to run: %s", err)
    }

    return &PolicyOverride{
        RunID:            run.ID,
        PolicyStageID:    taskStage.ID,
        Justification:    justification,
        InitialStatus:    string(run.Status),
        OverrideComplete: taskStage.Status == tfe.TaskStageOverridden,
    }, nil
}

func (s *policyService) overrideViaPolicyCheck(ctx context.Context, run *tfe.Run, justification string) (*PolicyOverride, error) {
    // Legacy API uses PolicyChecks.Override
    err := s.tfe.PolicyChecks.Override(ctx, run.PolicyCheck.ID)
    if err != nil {
        return nil, fmt.Errorf("error overriding policy check: %w", err)
    }

    // Add comment to run
    _, err = s.tfe.Comments.Create(ctx, run.ID, tfe.CommentCreateOptions{
        Body: tfe.String(fmt.Sprintf("Policy Override: %s", justification)),
    })
    if err != nil {
        log.Printf("[WARN] failed to add comment to run: %s", err)
    }

    return &PolicyOverride{
        RunID:            run.ID,
        PolicyCheckID:    run.PolicyCheck.ID,
        Justification:    justification,
        InitialStatus:    string(run.Status),
        OverrideComplete: true, // Legacy API is synchronous
    }, nil
}
```

### Validation

**Pre-Override Checks**:

1. Run status must be `post_plan_awaiting_decision`
2. Run must have mandatory policy failures
3. Workspace must allow overrides (checked by API)

```go
func (s *policyService) validateOverrideEligibility(ctx context.Context, runID string) (*tfe.Run, error) {
    run, err := s.tfe.Runs.Read(ctx, runID)
    if err != nil {
        return nil, fmt.Errorf("error reading run: %w", err)
    }

    if run.Status != tfe.RunPostPlanAwaitingDecision {
        return nil, fmt.Errorf("run status is %s, expected post_plan_awaiting_decision", run.Status)
    }

    return run, nil
}
```

---

## Retry Strategy

### Decision

Use **Fibonacci backoff** with context-aware timeout for polling operations.

### Rationale

1. **Existing Pattern**: tfc-workflows-tooling already uses this (see `configuration_version.go`)
2. **Constitution Compliance**: Principle 3 mandates exponential backoff
3. **API Friendliness**: Fibonacci grows slower than pure exponential, reduces load
4. **Proven**: sethvargo/go-retry battle-tested in production

### Parameters

```go
const (
    PolicyWaitMaxDuration = 30 * time.Minute  // Max overall wait time
    PolicyWaitInitialBackoff = 10 * time.Second  // Start with 10s intervals
    PolicyWaitMaxBackoff = 30 * time.Second  // Cap at 30s intervals
)

func policyWaitBackoffStrategy() retry.Backoff {
    backoff := retry.NewFibonacci(PolicyWaitInitialBackoff)
    backoff = retry.WithCappedDuration(PolicyWaitMaxBackoff, backoff)
    backoff = retry.WithMaxDuration(PolicyWaitMaxDuration, backoff)
    return backoff
}
```

### Wait Implementation

```go
func (s *policyService) WaitForPolicyDecision(ctx context.Context, options WaitForPolicyOptions) (*PolicyOverride, error) {
    // Apply user timeout if provided
    if options.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, options.Timeout)
        defer cancel()
    }

    var finalRun *tfe.Run
    err := retry.Do(ctx, policyWaitBackoffStrategy(), func(ctx context.Context) error {
        run, err := s.tfe.Runs.Read(ctx, options.RunID)
        if err != nil {
            return err  // Non-retryable
        }

        log.Printf("[DEBUG] Waiting for policy decision, current status: %s", run.Status)

        // Check for terminal states
        switch run.Status {
        case tfe.RunPolicyOverride, tfe.RunPostPlanCompleted:
            // Override detected
            finalRun = run
            return nil
        case tfe.RunDiscarded:
            return fmt.Errorf("run was discarded")
        case tfe.RunCanceled, tfe.RunForceCanceled:
            return fmt.Errorf("run was canceled")
        case tfe.RunErrored:
            return fmt.Errorf("run entered error state")
        case tfe.RunPostPlanAwaitingDecision:
            // Still waiting
            return retry.RetryableError(fmt.Errorf("still awaiting decision"))
        default:
            // Unexpected status
            log.Printf("[WARN] Unexpected run status during wait: %s", run.Status)
            return retry.RetryableError(fmt.Errorf("unexpected status: %s", run.Status))
        }
    })

    if err != nil {
        return nil, err
    }

    return &PolicyOverride{
        RunID:            finalRun.ID,
        InitialStatus:    string(tfe.RunPostPlanAwaitingDecision),
        FinalStatus:      string(finalRun.Status),
        OverrideComplete: true,
    }, nil
}
```

### Error Handling Categories

**Retryable Errors** (with backoff):

- `429 Too Many Requests` (rate limiting)
- `5xx Server Error` (transient TFC issues)
- Run still in `post_plan_awaiting_decision` (expected during wait)

**Non-Retryable Errors** (fail fast):

- `401 Unauthorized` (invalid token)
- `403 Forbidden` (insufficient permissions)
- `404 Not Found` (invalid run ID)
- Run status: `discarded`, `canceled`, `errored` (terminal states)

---

## Best Practices Summary

### From Existing Codebase

1. **Client Retry Configuration** (`tfe_client.go`):

   ```go
   client.RetryServerErrors(true)  // Automatic 5xx retry
   ```

2. **Service Method Pattern** (`run.go`, `workspace.go`):

   - Accept `context.Context` as first parameter
   - Embed `*cloudMeta` for client and writer access
   - Return structured results, not primitives
   - Log at appropriate levels (DEBUG, INFO, ERROR)

3. **Testing Pattern** (`run_test.go`, `configuration_version_test.go`):

   - Unit tests with `gomock` and `go-tfe/mocks`
   - Integration tests with `testClient()` helper
   - Cleanup functions registered with `t.Cleanup()`
   - Parallel test execution with `t.Parallel()` where safe

4. **Output Abstraction** (`writer/writer.go`):
   - Use `Writer` interface for all user-facing output
   - Support JSON mode with `UseJson(true)`
   - Emit structured objects in JSON mode, formatted text in human mode

### New Patterns for Policy Operations

1. **Dual API Support**: Always try modern API first, graceful fallback
2. **Normalization Layer**: Convert both APIs to unified struct
3. **Pre-Flight Validation**: Check run status before expensive operations
4. **Justification Enforcement**: Require non-empty justification for overrides
5. **Status Polling**: Use retry with context timeout, log each attempt

---

## References

- go-tfe SDK documentation: https://pkg.go.dev/github.com/hashicorp/go-tfe
- TFC API documentation: https://developer.hashicorp.com/terraform/cloud-docs/api-docs
- Existing retry patterns: `internal/cloud/configuration_version.go`
- Service layer examples: `internal/cloud/run.go`, `internal/cloud/workspace.go`
- Testing patterns: `internal/cloud/run_test.go`
