# Service Contract: PolicyService

**Feature**: Policy Evaluation and Override Operations  
**Branch**: 001-policy-operations  
**Date**: 2025-12-04

## Overview

`PolicyService` provides methods for retrieving Sentinel policy evaluation results, applying policy overrides, and waiting for policy decisions. The service abstracts away differences between legacy (`policy-checks`) and modern (`task-stages/policy-evaluations`) TFC API formats, providing a unified interface.

---

## Interface Definition

```go
package cloud

import (
    "context"
    "time"
)

// PolicyService handles Sentinel policy operations for TFC/TFE runs
type PolicyService interface {
    // GetPolicyEvaluation retrieves policy evaluation results for a run.
    // Returns normalized PolicyEvaluation regardless of API format (legacy or modern).
    // Automatically waits (with retry) for policy evaluation to complete unless NoWait is true.
    //
    // Errors:
    //   - ErrInvalidRunID: Run ID format invalid
    //   - ErrRunNotFound: Run does not exist
    //   - ErrNoPolicyCheck: Run has no policy evaluation
    //   - ErrPolicyPending: Policies not yet evaluated (only if NoWait=true)
    //   - API errors from go-tfe SDK
    GetPolicyEvaluation(ctx context.Context, options GetPolicyEvaluationOptions) (*PolicyEvaluation, error)

    // OverridePolicy applies a policy override with justification.
    // Pre-conditions: Run status must be post_plan_awaiting_decision.
    //
    // Errors:
    //   - ErrInvalidRunID: Run ID format invalid
    //   - ErrInvalidJustification: Justification too short
    //   - ErrInvalidRunStatus: Run not in awaiting_decision status
    //   - ErrNoPolicyCheck: Run has no policy check/stage to override
    //   - ErrPermissionDenied: Insufficient permissions to override
    //   - API errors from go-tfe SDK
    OverridePolicy(ctx context.Context, options OverridePolicyOptions) (*PolicyOverride, error)
}
```

---

## Implementation Struct

```go
// policyService implements PolicyService using go-tfe SDK
type policyService struct {
    *cloudMeta  // Embeds tfe client and writer
}

// NewPolicyService creates a new policy service instance
func NewPolicyService(meta *cloudMeta) PolicyService {
    return &policyService{cloudMeta: meta}
}
```

---

## Request/Response Types

### GetPolicyEvaluationOptions

```go
// GetPolicyEvaluationOptions configures policy evaluation retrieval
type GetPolicyEvaluationOptions struct {
    // RunID is the TFC run identifier (required)
    // Format: "run-[a-zA-Z0-9]+"
    RunID string
    
    // NoWait skips retry logic, fails immediately if policies not yet evaluated (optional)
    // Default: false (waits with retry)
    NoWait bool
}

// Validate checks if options are valid
func (o GetPolicyEvaluationOptions) Validate() error {
    if !validStringID(&o.RunID) {
        return ErrInvalidRunID
    }
    return nil
}
```

### PolicyEvaluation (Response)

See [data-model.md](../data-model.md#entity-policyevaluation) for full definition.

**Key fields**:
- `RunID` (string): Run identifier
- `TotalCount` (int): Total policies evaluated
- `PassedCount` (int): Policies passed
- `MandatoryFailedCount` (int): Mandatory policies failed
- `FailedPolicies` ([]PolicyDetail): Details of failures
- `RequiresOverride` (bool): Whether override needed

---

### OverridePolicyOptions

```go
// OverridePolicyOptions configures policy override operation
type OverridePolicyOptions struct {
    // RunID is the TFC run identifier (required)
    // Format: "run-[a-zA-Z0-9]+"
    RunID string

    // Justification is the reason for override (required, min 10 characters)
    // Examples: "Emergency hotfix approved by CTO - INC-12345"
    Justification string
}

// Validate checks if options are valid
func (o OverridePolicyOptions) Validate() error {
    if !validStringID(&o.RunID) {
        return ErrInvalidRunID
    }
    if len(o.Justification) < 10 {
        return ErrInvalidJustification
    }
    return nil
}
```

### PolicyOverride (Response)

See [data-model.md](../data-model.md#entity-policyoverride) for full definition.

**Key fields**:
- `RunID` (string): Run identifier
- `Justification` (string): Override reason
- `InitialStatus` (string): Status before override
- `FinalStatus` (string): Status after override
- `OverrideComplete` (bool): Whether operation completed

---

## Error Types

```go
var (
    // ErrInvalidRunID indicates run ID format is invalid
    ErrInvalidRunID = errors.New("invalid run ID format")

    // ErrInvalidJustification indicates justification is too short
    ErrInvalidJustification = errors.New("justification must be at least 10 characters")

    // ErrInvalidRunStatus indicates run is not in correct status for operation
    ErrInvalidRunStatus = errors.New("run status does not allow this operation")

    // ErrNoPolicyCheck indicates run has no policy check or task stage
    ErrNoPolicyCheck = errors.New("run has no policy evaluation")
    
    // ErrPolicyPending indicates policies are still being evaluated (only with NoWait=true)
    ErrPolicyPending = errors.New("policy evaluation still in progress")

    // ErrRunNotFound indicates run does not exist
    ErrRunNotFound = errors.New("run not found")

    // ErrPermissionDenied indicates insufficient permissions
    ErrPermissionDenied = errors.New("insufficient permissions for this operation")

    // ErrTimeout indicates operation timed out
    ErrTimeout = errors.New("operation timed out")

    // ErrRunDiscarded indicates run was discarded
    ErrRunDiscarded = errors.New("run was discarded")

    // ErrRunCanceled indicates run was canceled
    ErrRunCanceled = errors.New("run was canceled")

    // ErrRunErrored indicates run entered error state
    ErrRunErrored = errors.New("run entered error state")
)
```

---

## Method Specifications

### GetPolicyEvaluation

**Purpose**: Retrieve policy evaluation results for a run, with automatic wait for evaluation completion.

**Pre-conditions**:
- Run must exist
- Run must have completed plan phase
- Run must have policy evaluation (either policy check or task stage)

**Post-conditions**:
- Returns normalized PolicyEvaluation
- Counts are accurate and validated
- FailedPolicies populated with mandatory failures

**Algorithm**:
1. Validate options (run ID format)
2. Read run with workspace relationship
3. Check if policy evaluation is complete
4. If not complete and NoWait=false:
   - Enter retry loop with Fibonacci backoff (following WorkspaceService.ReadStateOutputs pattern)
   - Poll run status until policies evaluated or context timeout
5. If not complete and NoWait=true: Return ErrPolicyPending
6. Try modern API: List task stages, find policy stage
7. If modern API available: Read policy evaluation from stage
8. If modern API unavailable: Read legacy policy check
9. Normalize response to PolicyEvaluation struct
10. Validate response data integrity
11. Return normalized result

**Example Usage**:
```go
ctx := context.Background()

// Default: waits for policies to complete
options := cloud.GetPolicyEvaluationOptions{
    RunID: "run-abc123def456",
}

eval, err := policyService.GetPolicyEvaluation(ctx, options)
if err != nil {
    log.Fatalf("Error retrieving policy evaluation: %s", err)
}

// Fast-fail mode (no wait)
optionsNoWait := cloud.GetPolicyEvaluationOptions{
    RunID:  "run-abc123def456",
    NoWait: true,
}

eval, err = policyService.GetPolicyEvaluation(ctx, optionsNoWait)
if err == cloud.ErrPolicyPending {
    fmt.Println("Policies still evaluating, try again later")
    return
}

fmt.Printf("Total: %d, Passed: %d, Mandatory Failed: %d\n",
    eval.TotalCount, eval.PassedCount, eval.MandatoryFailedCount)

if eval.RequiresOverride {
    fmt.Println("Override required!")
    for _, policy := range eval.FailedPolicies {
        fmt.Printf("  - %s (%s): %s\n",
            policy.PolicyName, policy.EnforcementLevel, policy.Status)
    }
}
```

**Performance**: < 5 seconds for completed evaluations (SC-001), variable for pending evaluations (depends on evaluation time, respects context timeout)

---

### OverridePolicy

**Purpose**: Apply a policy override with justification.

**Pre-conditions**:
- Run must exist
- Run status must be `post_plan_awaiting_decision`
- Run must have mandatory policy failures
- User must have override permissions on workspace
- Justification must be at least 10 characters

**Post-conditions**:
- Policy override applied to run
- Justification comment added to run
- Run status transitions to `policy_override` or `post_plan_completed`
- PolicyOverride returned with status information

**Algorithm**:
1. Validate options (run ID, justification length)
2. Read run and check status
3. If status ≠ `post_plan_awaiting_decision`, return ErrInvalidRunStatus
4. Detect API format (modern vs legacy)
5. If modern: Call `TaskStages.Override()` with comment
6. If legacy: Call `PolicyChecks.Override()`
7. Add justification comment to run via `Comments.Create()`
8. Poll run status until override completes
9. Return PolicyOverride with status details

**Example Usage**:
```go
ctx := context.Background()
options := cloud.OverridePolicyOptions{
    RunID:         "run-abc123def456",
    Justification: "Emergency hotfix approved by CTO - Incident INC-12345",
}

override, err := policyService.OverridePolicy(ctx, options)
if err != nil {
    log.Fatalf("Error applying override: %s", err)
}

fmt.Printf("Override applied! Status: %s → %s\n",
    override.InitialStatus, override.FinalStatus)

if override.OverrideComplete {
    fmt.Println("Override complete, ready for apply")
}
```

**Performance**: < 10 seconds (SC-003)

---

## API Endpoint Detection Strategy

The service automatically detects which API format to use:

```go
// Pseudo-code for detection
func (s *policyService) detectAPIFormat(ctx context.Context, run *tfe.Run) (apiFormat, error) {
    // Try modern API first
    taskStages, err := s.tfe.TaskStages.List(ctx, run.ID, nil)
    if err == nil && len(taskStages.Items) > 0 {
        return modernAPI, nil
    }

    // Fall back to legacy
    if run.PolicyCheck != nil && run.PolicyCheck.ID != "" {
        return legacyAPI, nil
    }

    return unknownAPI, ErrNoPolicyCheck
}
```

**Modern API** (TFC/TFE 2023+):
- Endpoints: `/task-stages`, `/policy-evaluations`
- SDK methods: `TaskStages.List()`, `PolicyEvaluations.Read()`, `TaskStages.Override()`
- Features: Richer metadata, policy set outcomes, granular counts

**Legacy API** (TFE < 2023):
- Endpoints: `/policy-checks`
- SDK methods: `PolicyChecks.Read()`, `PolicyChecks.Override()`
- Features: Basic counts, policy outcomes

---

## Integration with Existing Services

### Cloud Struct Update

```go
// cloud.go - Add PolicyService to aggregator
type Cloud struct {
    *cloudMeta
    ConfigVersionService
    RunService
    PlanService
    WorkspaceService
    PolicyService  // NEW
}

func NewCloud(c *tfe.Client, w Writer) *Cloud {
    meta := &cloudMeta{tfe: c, writer: w}

    return &Cloud{
        cloudMeta:            meta,
        ConfigVersionService: NewConfigVersionService(meta),
        RunService:           NewRunService(meta),
        PlanService:          NewPlanService(meta),
        WorkspaceService:     NewWorkspaceService(meta),
        PolicyService:        NewPolicyService(meta),  // NEW
    }
}
```

### Writer Interface Usage

```go
// Human-readable output
s.writer.Output(fmt.Sprintf("📊 Policy Evaluation Summary"))
s.writer.Output(fmt.Sprintf("   Total Policies: %d", eval.TotalCount))
s.writer.Output(fmt.Sprintf("   ✅ Passed: %d", eval.PassedCount))
s.writer.Output(fmt.Sprintf("   🚫 Failed (Mandatory): %d", eval.MandatoryFailedCount))

// JSON output
if jsonMode {
    s.writer.UseJson(true)
    jsonData, _ := json.Marshal(eval)
    s.writer.Output(string(jsonData))
}
```

---

## Testing Strategy

### Unit Tests (with go-tfe/mocks)

```go
func TestGetPolicyEvaluation_ModernAPI(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    // Create mocks
    runsMock := mocks.NewMockRuns(ctrl)
    taskStagesMock := mocks.NewMockTaskStages(ctrl)
    policyEvalMock := mocks.NewMockPolicyEvaluations(ctrl)

    // Set expectations
    runsMock.EXPECT().
        ReadWithOptions(gomock.Any(), "run-abc123", gomock.Any()).
        Return(&tfe.Run{ID: "run-abc123"}, nil)

    taskStagesMock.EXPECT().
        List(gomock.Any(), "run-abc123", nil).
        Return(&tfe.TaskStageList{
            Items: []*tfe.TaskStage{{ID: "ts-xyz789", Stage: tfe.PostPlan}},
        }, nil)

    policyEvalMock.EXPECT().
        Read(gomock.Any(), "ts-xyz789").
        Return(&tfe.PolicyEvaluation{
            ResultCount: &tfe.PolicyResultCount{
                Passed:           3,
                MandatoryFailed:  2,
            },
        }, nil)

    // Inject mocks into client
    client := &tfe.Client{
        Runs:               runsMock,
        TaskStages:         taskStagesMock,
        PolicyEvaluations:  policyEvalMock,
    }

    // Test service
    service := NewPolicyService(&cloudMeta{tfe: client})
    eval, err := service.GetPolicyEvaluation(context.Background(), GetPolicyEvaluationOptions{
        RunID: "run-abc123",
    })

    require.NoError(t, err)
    assert.Equal(t, 3, eval.PassedCount)
    assert.Equal(t, 2, eval.MandatoryFailedCount)
    assert.True(t, eval.RequiresOverride)
}
```

### Integration Tests (with testClient)

```go
func TestOverridePolicy_IntegrationTest(t *testing.T) {
    skipUnlessIntegration(t)
    client := testClient(t)
    ctx := context.Background()

    // Setup test environment
    org, orgCleanup := createOrganization(t, client)
    t.Cleanup(orgCleanup)

    ws, wsCleanup := createWorkspace(t, client, org)
    t.Cleanup(wsCleanup)

    // Create run with failing policy
    run := createRunWithFailedPolicy(t, client, ws)

    // Apply override
    service := NewPolicyService(&cloudMeta{tfe: client})
    override, err := service.OverridePolicy(ctx, OverridePolicyOptions{
        RunID:         run.ID,
        Justification: "Test override justification",
    })

    require.NoError(t, err)
    assert.True(t, override.OverrideComplete)
    assert.Equal(t, "post_plan_awaiting_decision", override.InitialStatus)
}
```

---

## Performance Considerations

1. **API Call Optimization**: Single run read with `Include` options to fetch relationships
2. **Retry Backoff**: Fibonacci backoff reduces API load during polling
3. **Timeout Enforcement**: Context propagation ensures operations don't hang
4. **Parallel Requests**: GetPolicyEvaluation can be called for multiple runs concurrently

---

## Security Considerations

1. **Token Exposure**: API tokens never logged or included in output
2. **Justification Audit**: All overrides logged with justification in TFC audit trail
3. **Permission Checks**: API returns 403 if user lacks override permissions
4. **Input Validation**: All inputs validated before API calls
