# Feature Specification: Policy Evaluation and Override Operations

**Feature Branch**: `001-policy-operations`  
**Created**: 2025-12-04  
**Status**: Draft  
**Input**: User description: "Add policy evaluation and override capabilities to tfc-workflows-tooling CLI for Sentinel policy management"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Evaluate Sentinel Policy Results (Priority: P1)

As a DevOps engineer running Terraform workflows, I need to retrieve and evaluate Sentinel policy check results from a TFC/TFE run so that I can understand which policies passed or failed and make informed decisions about whether to proceed with deployment.

**Why this priority**: Policy evaluation is the foundation for policy-based workflow decisions. Without this capability, users cannot programmatically determine if a run has policy failures that require attention. This is the minimum viable functionality needed for any policy-aware automation.

**Independent Test**: Can be fully tested by running `tfci policy show --run-id <run-id>` against a TFC run that has completed policy evaluations, and verifying that the command returns policy status, counts, and individual policy results in both human-readable and JSON formats.

**Acceptance Scenarios**:

1. **Given** a TFC run with completed Sentinel policy evaluations, **When** user executes `tfci policy show --run-id run-abc123`, **Then** the system displays policy evaluation summary including total policies, passed count, advisory failed count, mandatory failed count, and errored count
2. **Given** a TFC run with failed mandatory policies, **When** user executes `tfci policy show --run-id run-abc123`, **Then** the system lists each failed mandatory policy with its name and enforcement level
3. **Given** a TFC run with no policies configured, **When** user executes `tfci policy show --run-id run-abc123`, **Then** the system reports "No policies configured for this workspace"
4. **Given** user wants machine-readable output, **When** user executes `tfci policy show --run-id run-abc123 --json`, **Then** the system outputs policy evaluation results in JSON format
5. **Given** a non-existent run ID, **When** user executes `tfci policy show --run-id invalid`, **Then** the system returns an error with meaningful message

---

### User Story 2 - Override Mandatory Policy Failures (Priority: P2)

As a platform administrator with override permissions, I need to programmatically override mandatory Sentinel policy failures with justification so that I can unblock deployments when policy violations are acceptable for specific business reasons while maintaining an audit trail.

**Why this priority**: Override capability is essential for production workflows where policy exceptions are sometimes necessary. However, it depends on policy evaluation (P1) to identify what needs overriding. This enables complete policy workflow automation but is not needed for read-only policy inspection.

**Independent Test**: Can be fully tested by running `tfci policy override --run-id <run-id> --justification "reason"` against a TFC run in `post_plan_awaiting_decision` status with mandatory policy failures, and verifying that the override is applied and justification is recorded as a comment in TFC.

**Acceptance Scenarios**:

1. **Given** a TFC run with mandatory policy failures in `post_plan_awaiting_decision` status, **When** user executes `tfci policy override --run-id run-abc123 --justification "Emergency hotfix approved by CTO"`, **Then** the system applies the policy override and adds justification comment to the run
2. **Given** an override was successfully applied, **When** the override completes, **Then** the system reports the new run status (e.g., `policy_override`, `post_plan_completed`, or `apply_queued`)
3. **Given** a run without mandatory policy failures, **When** user attempts to override, **Then** the system returns an error indicating override is not needed
4. **Given** a run in incorrect status (e.g., `applied`, `discarded`), **When** user attempts to override, **Then** the system returns an error with current status and expected status
5. **Given** user has insufficient permissions, **When** user attempts to override, **Then** the system returns a permissions error
6. **Given** user wants JSON output, **When** user executes `tfci policy override --run-id run-abc123 --justification "reason" --json`, **Then** the system outputs override result in JSON format

---

### User Story 3 - Wait for Policy Decision (Priority: P3)

As a CI/CD pipeline author, I need to wait for manual policy override decisions with a configurable timeout so that my automated workflows can pause for human approval without failing prematurely or waiting indefinitely.

**Why this priority**: This enables fully automated policy-aware workflows that can gracefully handle manual intervention. It's a convenience feature that builds on P1 and P2 but is not strictly required - users can implement their own polling logic externally.

**Independent Test**: Can be fully tested by running `tfci policy wait --run-id <run-id> --timeout 5m` against a run awaiting policy decision, then either applying an override or discarding the run, and verifying the wait command exits appropriately with correct status code.

**Acceptance Scenarios**:

1. **Given** a run awaiting policy override decision, **When** user executes `tfci policy wait --run-id run-abc123 --timeout 10m`, **Then** the system polls the run status until it transitions to a terminal state or timeout is reached
2. **Given** an override is applied during the wait, **When** the run status changes to `policy_override`, **Then** the wait command exits with success (exit code 0)
3. **Given** the run is discarded during the wait, **When** the run status changes to `discarded`, **Then** the wait command exits with specific exit code indicating discard (exit code 1)
4. **Given** the timeout is reached before a decision, **When** the wait duration exceeds the timeout, **Then** the system exits with timeout error and appropriate exit code
5. **Given** user wants status updates during wait, **When** polling is active, **Then** the system logs periodic status checks at DEBUG level

---

### Edge Cases

- What happens when TFC API returns 429 (rate limit) during policy checks? System should retry with exponential backoff
- What happens when a policy evaluation is still in progress? System should indicate evaluation is not yet complete
- How does system handle policy evaluation IDs vs policy check IDs? System should support both legacy `policy-checks` and modern `policy-evaluations` API endpoints
- What happens when TFC returns partial policy outcome data? System should gracefully handle missing policy names and show available information
- What happens if override is applied but run transitions to error state? System should detect and report the error status
- How does system handle runs with only advisory policy failures? System should report them but indicate override is not required

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST retrieve policy evaluation results from TFC/TFE API using the run ID
- **FR-002**: System MUST support both legacy `policy-checks` API and modern `task-stages/policy-evaluations` API endpoints
- **FR-003**: System MUST display policy evaluation summary including total count, passed, advisory failed, mandatory failed, and errored counts
- **FR-004**: System MUST list individual failed mandatory policies with policy name and enforcement level
- **FR-005**: System MUST apply policy overrides via the appropriate API endpoint (policy-checks or task-stages)
- **FR-006**: System MUST add justification as a comment to the TFC run when overriding policies
- **FR-007**: System MUST validate run status before attempting override (must be `post_plan_awaiting_decision`)
- **FR-008**: System MUST poll run status after override to confirm status transition
- **FR-009**: System MUST output results in JSON format when `--json` flag is provided
- **FR-010**: System MUST follow existing tfc-workflows-tooling patterns for service layer architecture
- **FR-011**: System MUST use go-tfe SDK client for all TFC API interactions
- **FR-012**: System MUST accept context.Context as first parameter in all service methods
- **FR-013**: System MUST implement retry logic with exponential backoff for transient failures
- **FR-014**: System MUST log operations at appropriate levels (DEBUG, INFO, ERROR)
- **FR-015**: System MUST return structured errors with meaningful messages for all failure cases
- **FR-016**: System MUST respect global CLI flags (--hostname, --organization, --token, --json)
- **FR-017**: System MUST integrate with existing cloudMeta and Writer interfaces
- **FR-018**: System MUST handle cases where workspace has no policies configured
- **FR-019**: System MUST distinguish between different override scenarios (already overridden, not needed, wrong status)
- **FR-020**: System MUST provide exit codes that CI/CD systems can use for workflow decisions

### Key Entities _(include if feature involves data)_

- **PolicyEvaluation**: Represents Sentinel policy evaluation results for a run, including status counts (passed, failed, errored) and individual policy outcomes
- **PolicyOverride**: Represents an override action with justification, target run ID, and resulting status
- **TaskStage**: TFC task stage entity containing policy evaluations, particularly the `post_plan` stage
- **PolicySetOutcome**: Individual policy results nested within policy evaluations, containing policy names, statuses, and enforcement levels
- **Run**: TFC run entity with relationships to policy-checks or task-stages depending on TFC/TFE version

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Users can retrieve policy evaluation results for any TFC run in under 5 seconds
- **SC-002**: System correctly identifies and reports mandatory vs advisory policy failures 100% of the time
- **SC-003**: Policy overrides are successfully applied with justification comments in under 10 seconds
- **SC-004**: System handles both old (policy-checks) and new (policy-evaluations) API formats without user intervention
- **SC-005**: JSON output format is valid and parseable by standard JSON tools (jq, Python json module)
- **SC-006**: Error messages clearly indicate the problem and suggest corrective action (e.g., "Run not in correct state - current: applied, expected: post_plan_awaiting_decision")
- **SC-007**: Unit tests achieve >80% code coverage for policy service layer
- **SC-008**: Integration tests validate both policy evaluation and override against live TFC API
- **SC-009**: CLI follows existing command patterns and help text is clear and consistent with other commands
- **SC-010**: Commands work correctly when run from CI/CD environments (GitHub Actions, GitLab CI) using environment variables

### Non-Functional Requirements

- **NFR-001**: Code MUST follow tfc-workflows-tooling conventions (service layer, cloudMeta, interfaces)
- **NFR-002**: All public methods MUST have godoc comments
- **NFR-003**: Implementation MUST minimize changes to existing codebase (add new files, avoid modifying core)
- **NFR-004**: Commands MUST provide human-readable output by default and JSON output with --json flag
- **NFR-005**: Implementation MUST use structured logging with [DEBUG], [INFO], [ERROR] prefixes
- **NFR-006**: Retry logic MUST use sethvargo/go-retry with Fibonacci backoff
- **NFR-007**: Tests MUST use go-tfe/mocks for unit tests and testClient() pattern for integration tests
- **NFR-008**: Implementation MUST handle context cancellation gracefully

## Assumptions

1. **TFC API Version**: Implementation assumes TFC API v2 with task-stages support. Legacy policy-checks endpoint is supported for backwards compatibility
2. **Authentication**: User has provided valid TFC/TFE API token with sufficient permissions (read for policy evaluation, write for policy override)
3. **Run State**: For override operations, run must be in `post_plan_awaiting_decision` status with mandatory policy failures
4. **Environment**: Commands run in environments with network access to TFC/TFE API endpoints
5. **Dependencies**: go-tfe SDK version supports PolicyEvaluations and TaskStages (v1.95+)
6. **CLI Framework**: Uses existing mitchellh/cli framework with command factory pattern
7. **Output Interface**: Uses existing Writer interface for output abstraction (supports testing and JSON mode)

## Out of Scope

- Creating or managing Sentinel policy definitions (policy authoring is done in TFC UI/API separately)
- Policy testing or dry-run evaluation (handled by TFC backend)
- Custom policy check implementations (only works with TFC Sentinel policies)
- Policy set management (attaching policies to workspaces)
- Run cost estimation or other non-policy task stages
- Automatic retry of failed runs after policy fixes
- Policy failure notifications or alerting (handled by TFC or external systems)
- Multi-run batch policy operations
- Policy compliance reporting across workspaces

## Dependencies

### External Dependencies

- **go-tfe SDK** (v1.95.0+): Official TFC/TFE Go client library
- **mitchellh/cli** (v1.1+): CLI framework for command structure
- **sethvargo/go-retry** (v0.3+): Retry logic with backoff
- **go.uber.org/mock**: Mock generation for testing

### Internal Dependencies

- **internal/cloud/cloud.go**: cloudMeta struct and Cloud aggregator
- **internal/cloud/tfe_client.go**: TFE client initialization
- **internal/writer/writer.go**: Writer interface for output abstraction
- **internal/command/meta.go**: Command meta with global flags
- **cli.go**: Command factory registration

## Technical Constraints

1. **API Compatibility**: Must support both old (`policy-checks`) and new (`task-stages/policy-evaluations`) API endpoints
2. **Go Version**: Must compile with Go 1.24+
3. **Docker**: Must work correctly when running in Docker container with mounted volumes
4. **CI/CD**: Must integrate with existing GitHub Actions and GitLab CI patterns
5. **Backwards Compatibility**: Must not break existing commands or service interfaces
6. **TFC/TFE Versions**: Must work with both TFC (SaaS) and TFE (self-hosted) deployments

## Security Considerations

- **Token Handling**: API tokens must never be logged or exposed in output
- **Justification Logging**: Override justifications are stored in TFC as comments for audit trail
- **Permissions**: Override operations require appropriate TFC workspace permissions (admin/write)
- **Input Validation**: Run IDs and justification text must be validated before API calls
- **Error Messages**: Must not leak sensitive information in error outputs

## Questions / Clarifications

_None - Requirements are clear based on GitHub Actions workflow examples and existing tfc-workflows-tooling patterns._

## Clarifications

### Session 2025-12-04

_No clarifications needed - Specification completed structured ambiguity scan with all categories marked Clear or Resolved._
