# Implementation Plan: Policy Evaluation and Override Operations

**Branch**: `001-policy-operations` | **Date**: 2025-12-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-policy-operations/spec.md`

## Summary

Add Sentinel policy evaluation and override capabilities to tfc-workflows-tooling CLI by implementing two new service layers (`PolicyService`) and two new commands (`policy show`, `policy override`). The implementation uses the go-tfe SDK to interact with both legacy `policy-checks` and modern `task-stages/policy-evaluations` API endpoints, following existing service layer patterns with cloudMeta aggregation, Writer abstraction for output, and comprehensive retry logic. The `policy show` command includes built-in wait logic (similar to `WorkspaceService.ReadStateOutputs()`) that automatically polls until policy evaluation completes.

## Technical Context

**Language/Version**: Go 1.24.0  
**Primary Dependencies**:

- `github.com/hashicorp/go-tfe` v1.95.0 (TFC/TFE API client)
- `github.com/mitchellh/cli` v1.1.5 (CLI framework)
- `github.com/sethvargo/go-retry` v0.3.0 (Retry with exponential backoff)
- `go.uber.org/mock` v0.6.0 (Mock generation for testing)
- `github.com/hashicorp/go-hclog` v1.6.3 (Structured logging)

**Storage**: N/A (stateless CLI, all state in TFC/TFE)  
**Testing**:

- Unit tests with `go.uber.org/mock` mocks from `github.com/hashicorp/go-tfe/mocks`
- Integration tests with `testClient()` pattern against live TFC API
- Test execution: `go test ./internal/cloud/... -timeout=15m`

**Target Platform**: Linux AMD64 (Docker), also macOS/Windows for local development  
**Project Type**: CLI tool (single binary with subcommands)

**Performance Goals**:

- Policy evaluation retrieval: <5 seconds (SC-001)
- Policy override application: <10 seconds (SC-003)
- Wait command polling interval: 10-30 seconds with configurable timeout

**Constraints**:

- Must support both TFC API formats: legacy `policy-checks` and modern `task-stages/policy-evaluations`
- Must work in Docker containers with environment variable configuration
- Must not break existing commands or service interfaces (backwards compatibility)
- Must follow existing code patterns (service layer, cloudMeta, Writer interface)
- API tokens must never be logged or exposed in output

**Scale/Scope**:

- 2 new service files (~300-400 LOC each): `policy_evaluation.go`, `policy_override.go`
- 3 new CLI command files (~150-200 LOC each): `policy_show.go`, `policy_override.go`, `policy_wait.go`
- 3 test files with unit and integration tests (~200-300 LOC each)
- Estimated total: ~1500-2000 LOC

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

### ✅ Principle 1: API-First Design

**Compliance**: PASS  
**Evidence**: Spec FR-011 mandates "System MUST use go-tfe SDK client for all TFC API interactions". Implementation uses `client.TaskStages.List()`, `client.PolicyEvaluations.Read()`, and SDK methods exclusively. No direct HTTP calls.

### ✅ Principle 2: Context-Aware Operations

**Compliance**: PASS  
**Evidence**: Spec FR-012 requires "System MUST accept context.Context as first parameter in all service methods". All PolicyService methods signature: `func (ctx context.Context, options XxxOptions) (*Result, error)`

### ✅ Principle 3: Retry & Resilience

**Compliance**: PASS  
**Evidence**: Spec FR-013 mandates "System MUST implement retry logic with exponential backoff for transient failures". Edge cases document rate limiting handling. NFR-006 specifies use of `sethvargo/go-retry` with Fibonacci backoff.

### ✅ Principle 4: Service Layer Architecture

**Compliance**: PASS  
**Evidence**: Spec FR-010 requires "System MUST follow existing tfc-workflows-tooling patterns for service layer architecture". Implementation adds `PolicyService` interface with `policyService` struct embedding `*cloudMeta`, aggregated in `Cloud` struct.

### ✅ Principle 5: Test-Driven Development

**Compliance**: PASS  
**Evidence**: Spec SC-007 mandates ">80% code coverage for policy service layer". SC-008 requires integration tests. NFR-007 specifies "Tests MUST use go-tfe/mocks for unit tests and testClient() pattern for integration tests".

### ✅ Principle 6: Structured Logging

**Compliance**: PASS  
**Evidence**: Spec FR-014 requires "System MUST log operations at appropriate levels (DEBUG, INFO, ERROR)". NFR-005 specifies structured logging with level prefixes matching existing patterns.

### ✅ Principle 7: Dockerized Deployment

**Compliance**: PASS  
**Evidence**: Spec technical constraints include "Must work correctly when running in Docker container". Existing Dockerfile requires no changes. Commands use environment variables (TF_API_TOKEN, TF_HOSTNAME, TF_CLOUD_ORGANIZATION).

### ✅ Principle 8: CI/CD Platform Integration

**Compliance**: PASS  
**Evidence**: Spec SC-010 requires "Commands work correctly when run from CI/CD environments (GitHub Actions, GitLab CI) using environment variables". FR-020 mandates exit codes for workflow decisions. JSON output flag supported.

**GATE STATUS**: ✅ **PASS** - All 8 principles satisfied. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/001-policy-operations/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (API endpoint compatibility analysis)
├── data-model.md        # Phase 1 output (PolicyEvaluation, PolicyOverride entities)
├── quickstart.md        # Phase 1 output (Command usage examples)
├── contracts/           # Phase 1 output (Service interface definitions)
│   └── policy-service.go
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (tfc-workflows-tooling repository root)

```text
tfc-workflows-tooling/
├── go.mod                           # Already exists, no changes needed
├── cli.go                           # UPDATE: Add policy command factory
├── internal/
│   ├── cloud/
│   │   ├── cloud.go                 # UPDATE: Add PolicyService to Cloud struct
│   │   ├── policy_evaluation.go    # NEW: Policy retrieval service
│   │   ├── policy_evaluation_test.go # NEW: Unit and integration tests
│   │   ├── policy_override.go      # NEW: Override and wait service
│   │   └── policy_override_test.go # NEW: Unit and integration tests
│   └── command/
│       ├── policy_show.go          # NEW: CLI command for policy show (with built-in wait)
│       └── policy_override.go      # NEW: CLI command for policy override
└── docs/
    └── USAGE.md                    # UPDATE: Add policy commands documentation
```

## Phase 0: Research & Technology Decisions

### Research Questions

1. **API Endpoint Compatibility**: How do we detect and handle both legacy `policy-checks` and modern `task-stages/policy-evaluations` endpoints?
2. **Policy Data Structure**: What are the exact API response formats for policy evaluations vs policy checks?
3. **Override Endpoint Selection**: Which override endpoint should be used based on API version detection?

These questions will be researched and documented in `research.md` with:

- Decision: Chosen approach
- Rationale: Why selected
- Alternatives considered: Other options evaluated

### Technology Choices

All technologies are predetermined by existing codebase:

- **Go 1.24.0**: Already in use (go.mod)
- **go-tfe v1.95.0**: Already in use, supports both API formats
- **mitchellh/cli**: Already in use for command structure
- **sethvargo/go-retry**: Already in use for retry logic
- **go.uber.org/mock**: Already in use for testing

**No new external dependencies required.**

## Phase 1: Design & Contracts

### Data Model (data-model.md)

#### Entity: PolicyEvaluation

**Purpose**: Represents the results of Sentinel policy evaluations for a TFC/TFE run

**Attributes**:

- `RunID` (string): TFC run identifier
- `PolicyStageID` (string, optional): Task stage ID for modern API
- `PolicyCheckID` (string, optional): Policy check ID for legacy API
- `TotalCount` (int): Total number of policies evaluated
- `PassedCount` (int): Number of policies that passed
- `AdvisoryFailedCount` (int): Number of advisory policies that failed
- `MandatoryFailedCount` (int): Number of mandatory policies that failed
- `ErroredCount` (int): Number of policies that errored
- `FailedPolicies` ([]PolicyDetail): Details of failed mandatory policies
- `Status` (string): Overall evaluation status

**Relationships**:

- Belongs to one Run (via RunID)
- Has many PolicyDetail (failed policies)

**Validation Rules**:

- RunID must match pattern `run-[a-zA-Z0-9]+`
- Exactly one of PolicyStageID or PolicyCheckID must be present
- All counts must be non-negative
- TotalCount should equal sum of individual counts

**State Transitions**:

- N/A (read-only entity, no state changes)

#### Entity: PolicyDetail

**Purpose**: Individual policy failure information

**Attributes**:

- `PolicyName` (string): Name of the policy
- `EnforcementLevel` (string): "mandatory" or "advisory"
- `Status` (string): "failed" or "errored"

**Validation Rules**:

- PolicyName must not be empty
- EnforcementLevel must be "mandatory" or "advisory"
- Status must be "failed" or "errored"

#### Entity: PolicyOverride

**Purpose**: Represents a policy override action

**Attributes**:

- `RunID` (string): TFC run identifier
- `Justification` (string): Reason for override (required)
- `PolicyStageID` (string, optional): For modern API override
- `PolicyCheckID` (string, optional): For legacy API override
- `InitialStatus` (string): Run status before override
- `FinalStatus` (string): Run status after override
- `OverrideComplete` (bool): Whether override succeeded

**Validation Rules**:

- RunID must match pattern `run-[a-zA-Z0-9]+`
- Justification must not be empty (minimum 10 characters)
- Exactly one of PolicyStageID or PolicyCheckID must be present
- InitialStatus must be `post_plan_awaiting_decision`

**State Transitions**:

- `post_plan_awaiting_decision` → `policy_override` (override applied)
- `post_plan_awaiting_decision` → `post_plan_completed` (override complete)
- `post_plan_awaiting_decision` → `apply_queued` (workspace auto-apply enabled)
- `post_plan_awaiting_decision` → `errored` (override failed)

### Service Contracts (contracts/policy-service.go)

```go
// PolicyService handles Sentinel policy operations
type PolicyService interface {
    // GetPolicyEvaluation retrieves policy evaluation results for a run
    // Automatically waits (with retry) for policy evaluation to complete
    GetPolicyEvaluation(ctx context.Context, options GetPolicyEvaluationOptions) (*PolicyEvaluation, error)

    // OverridePolicy applies a policy override with justification
    OverridePolicy(ctx context.Context, options OverridePolicyOptions) (*PolicyOverride, error)
}

// GetPolicyEvaluationOptions configures policy evaluation retrieval
type GetPolicyEvaluationOptions struct {
    RunID  string // Required: TFC run ID
    NoWait bool   // Optional: Fail fast if policies not yet evaluated
}

// OverridePolicyOptions configures policy override operation
type OverridePolicyOptions struct {
    RunID         string // Required: TFC run ID
    Justification string // Required: Override reason (min 10 chars)
}
```

### CLI Commands

#### Command: policy show

```bash
tfci policy show --run-id <run-id> [--json] [--no-wait]
```

**Flags**:

- `--run-id` (required): TFC run ID to check policies for
- `--json` (optional): Output in JSON format
- `--no-wait` (optional): Fail immediately if policies not yet evaluated (default: wait with retry)
- Global flags: `--hostname`, `--organization`, `--token`

**Behavior**: Automatically waits (with retry logic) for policy evaluation to complete, following the same pattern as `WorkspaceService.ReadStateOutputs()`. Uses Fibonacci backoff to poll run status until policies are evaluated or timeout reached.

**Output** (human-readable):

```
📊 Policy Evaluation Summary
   Total Policies: 5
   ✅ Passed: 3
   ⚠️  Failed (Advisory): 1
   🚫 Failed (Mandatory): 1
   ❌ Errored: 0

🚫 Failed Mandatory Policies:
   - cost-limit-check (mandatory)

View detailed results: https://app.terraform.io/app/org/workspaces/ws/runs/run-abc123
```

**Output** (JSON):

```json
{
  "run_id": "run-abc123",
  "total_count": 5,
  "passed_count": 3,
  "advisory_failed_count": 1,
  "mandatory_failed_count": 1,
  "errored_count": 0,
  "failed_policies": [
    {
      "policy_name": "cost-limit-check",
      "enforcement_level": "mandatory",
      "status": "failed"
    }
  ],
  "requires_override": true
}
```

**Exit Codes**:

- `0`: Success, policies retrieved
- `1`: Error (invalid run ID, API error, network failure)

#### Command: policy override

```bash
tfci policy override --run-id <run-id> --justification <reason> [--json]
```

**Flags**:

- `--run-id` (required): TFC run ID to override
- `--justification` (required): Reason for override
- `--json` (optional): Output in JSON format

**Output** (human-readable):

```
🛡️ Applying policy override...
Justification: Emergency hotfix approved by CTO

✅ Policy override applied successfully
📝 Justification comment added to run
⏳ Waiting for override to complete...

✅ Override complete! Run status: policy_override

Next Steps:
- Run the Apply workflow to deploy changes
- View run: https://app.terraform.io/app/org/workspaces/ws/runs/run-abc123
```

**Exit Codes**:

- `0`: Override applied successfully
- `1`: Error (wrong status, no mandatory failures, permissions error)
- `2`: Run discarded during override
- `3`: Override timeout

### Quickstart (quickstart.md)

**Setup**:

```bash
# Pull latest Docker image
docker pull hashicorp/tfci:latest

# Or build locally
docker build -t tfci:local .

# Set environment variables
export TF_API_TOKEN="your-token"
export TF_CLOUD_ORGANIZATION="your-org"
export TF_HOSTNAME="app.terraform.io"  # Optional, defaults to TFC
```

**Basic Usage**:

```bash
# Check policy results for a run (automatically waits if still evaluating)
tfci policy show --run-id run-abc123

# Check without waiting (fail fast if not ready)
tfci policy show --run-id run-abc123 --no-wait

# Override mandatory policy failures
tfci policy override \
  --run-id run-abc123 \
  --justification "Emergency fix approved by security team"
```

**Docker Usage**:

```bash
# Check policies
docker run --rm \
  -e TF_API_TOKEN \
  -e TF_CLOUD_ORGANIZATION \
  hashicorp/tfci:latest \
  tfci policy show --run-id run-abc123

# Override with JSON output
docker run --rm \
  -e TF_API_TOKEN \
  -e TF_CLOUD_ORGANIZATION \
  hashicorp/tfci:latest \
  tfci policy override \
    --run-id run-abc123 \
    --justification "Hotfix deployment" \
    --json | jq '.final_status'
```

**CI/CD Integration** (GitHub Actions):

```yaml
- name: Check Policies
  id: policy-check
  run: |
    docker run --rm \
      -e TF_API_TOKEN="${{ secrets.TF_API_TOKEN }}" \
      -e TF_CLOUD_ORGANIZATION="${{ vars.TF_CLOUD_ORGANIZATION }}" \
      hashicorp/tfci:latest \
      tfci policy show --run-id ${{ steps.plan.outputs.run_id }} --json \
      > policy-results.json

    MANDATORY_FAILED=$(jq -r '.mandatory_failed_count' policy-results.json)
    echo "mandatory_failed=$MANDATORY_FAILED" >> $GITHUB_OUTPUT

- name: Override if Needed
  if: steps.policy-check.outputs.mandatory_failed != '0'
  run: |
    docker run --rm \
      -e TF_API_TOKEN="${{ secrets.TF_API_TOKEN }}" \
      -e TF_CLOUD_ORGANIZATION="${{ vars.TF_CLOUD_ORGANIZATION }}" \
      hashicorp/tfci:latest \
      tfci policy override \
        --run-id ${{ steps.plan.outputs.run_id }} \
        --justification "${{ github.event.inputs.justification }}"
```

## Re-evaluate Constitution Post-Design

### ✅ Principle 1: API-First Design

**Post-Design Status**: PASS  
**Evidence**: Service contracts exclusively use go-tfe SDK types (`tfe.Run`, `tfe.TaskStage`, `tfe.PolicyEvaluation`). No custom HTTP client code. API endpoint detection logic uses SDK methods.

### ✅ Principle 2: Context-Aware Operations

**Post-Design Status**: PASS  
**Evidence**: All service method signatures include `ctx context.Context` as first parameter. Wait operation checks `ctx.Done()` in polling loop. Context propagated to all SDK calls.

### ✅ Principle 3: Retry & Resilience

**Post-Design Status**: PASS  
**Evidence**: Policy retrieval and override operations use retry with Fibonacci backoff. Wait command implements polling with configurable interval and timeout. Rate limiting handled via SDK retry configuration.

### ✅ Principle 4: Service Layer Architecture

**Post-Design Status**: PASS  
**Evidence**: PolicyService interface defines clean contract. Implementation uses `policyService` struct embedding `*cloudMeta`. Aggregated in `Cloud` struct alongside RunService, WorkspaceService. Single responsibility maintained (policy operations only).

### ✅ Principle 5: Test-Driven Development

**Post-Design Status**: PASS  
**Evidence**: Test structure planned with unit tests using gomock, integration tests using testClient(). Test files created alongside implementation files. Coverage target >80% specified.

### ✅ Principle 6: Structured Logging

**Post-Design Status**: PASS  
**Evidence**: Log statements planned at DEBUG (polling status), INFO (operation start/complete), ERROR (failures). Format matches existing pattern: `log.Printf("[LEVEL] message: %s", context)`.

### ✅ Principle 7: Dockerized Deployment

**Post-Design Status**: PASS  
**Evidence**: No Dockerfile changes required. Commands use environment variables for configuration. JSON output enables parsing in containerized environments. Exit codes support automated decision-making.

### ✅ Principle 8: CI/CD Platform Integration

**Post-Design Status**: PASS  
**Evidence**: Quickstart includes GitHub Actions example. Exit codes documented for workflow branching. JSON output format enables jq parsing. Environment variable configuration matches existing patterns.

**GATE STATUS**: ✅ **PASS** - All 8 principles satisfied post-design.

## Implementation Phases

### Phase 2: Core Implementation (NOT DONE BY /speckit.plan)

This phase will be planned by `/speckit.tasks`. Key milestones:

1. **Service Layer**: Implement `policy_evaluation.go` (with built-in retry/wait) and `policy_override.go`
2. **CLI Commands**: Implement `policy_show.go` and `policy_override.go`
3. **Integration**: Update `cloud.go` and `cli.go` to wire new services
4. **Testing**: Unit tests with mocks, integration tests with live API
5. **Documentation**: Update USAGE.md with policy commands

### Phase 3: Testing & Validation (NOT DONE BY /speckit.plan)

1. Unit test coverage >80%
2. Integration tests against TFC
3. Manual testing in Docker container
4. CI/CD workflow validation

### Phase 4: Documentation & Release (NOT DONE BY /speckit.plan)

1. Update CHANGELOG.md
2. Update README.md
3. Create pull request
4. Release new version

## Risk Assessment

### Technical Risks

1. **API Compatibility**: Legacy vs modern endpoints may have subtle differences

   - **Mitigation**: Comprehensive research in Phase 0, integration tests for both formats
   - **Impact**: Medium | **Probability**: Low

2. **TFC API Changes**: go-tfe SDK may not support all features

   - **Mitigation**: Use v1.95+ which includes task-stages and policy-evaluations
   - **Impact**: High | **Probability**: Very Low

3. **Override Race Conditions**: Run status may change during override
   - **Mitigation**: Poll status after override, handle state transitions explicitly
   - **Impact**: Low | **Probability**: Medium

### Integration Risks

1. **Docker Environment**: Commands must work in containers

   - **Mitigation**: Existing commands already containerized, follow same patterns
   - **Impact**: Low | **Probability**: Very Low

2. **CI/CD Compatibility**: Exit codes and JSON must work in workflows
   - **Mitigation**: Design validated against GitHub Actions examples
   - **Impact**: Medium | **Probability**: Low

## Success Metrics

- ✅ All 20 functional requirements (FR-001 to FR-020) implemented
- ✅ All 10 success criteria (SC-001 to SC-010) validated
- ✅ Unit test coverage >80%
- ✅ P1 and P2 user stories independently testable (P3 removed - wait logic built into P1)
- ✅ Zero breaking changes to existing codebase
- ✅ All 8 constitution principles satisfied

## Next Steps

**Command Ends Here** - Phase 2 implementation planning requires `/speckit.tasks`.

Run `/speckit.tasks` to generate detailed task breakdown for:

- File creation sequence
- Test-driven development steps
- Integration points
- Validation checkpoints
