# Tasks: Policy Evaluation and Override Operations

**Feature**: Policy Evaluation and Override Operations  
**Branch**: 001-policy-operations  
**Input**: Design documents from `/specs/001-policy-operations/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: This feature follows TDD principles with unit tests using go-tfe mocks and integration tests using testClient() pattern.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- All file paths are relative to `/workspace/tfc-workflows-tooling/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure for policy operations feature

- [X] T001 Create feature branch `001-policy-operations` from main
- [X] T002 [P] Create directory structure for policy service files in `internal/cloud/`
- [X] T003 [P] Create directory structure for policy command files in `internal/command/`

**Checkpoint**: Project structure ready for implementation

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Define error types in `internal/cloud/policy_errors.go` (ErrInvalidRunID, ErrInvalidJustification, ErrInvalidRunStatus, ErrNoPolicyCheck, ErrPolicyPending, ErrRunNotFound, ErrPermissionDenied)
- [X] T005 [P] Define PolicyEvaluation struct in `internal/cloud/policy_types.go` with JSON tags and Validate() method
- [X] T006 [P] Define PolicyDetail struct in `internal/cloud/policy_types.go` with JSON tags and Validate() method
- [X] T007 [P] Define PolicyOverride struct in `internal/cloud/policy_types.go` with JSON tags and Validate() method
- [X] T008 [P] Define GetPolicyEvaluationOptions struct in `internal/cloud/policy_types.go` with Validate() method
- [X] T009 [P] Define OverridePolicyOptions struct in `internal/cloud/policy_types.go` with Validate() method
- [X] T010 Define PolicyService interface in `internal/cloud/policy_evaluation.go` with GetPolicyEvaluation and OverridePolicy methods
- [X] T011 Create policyService struct embedding \*cloudMeta in `internal/cloud/policy_evaluation.go`
- [X] T012 Implement NewPolicyService constructor in `internal/cloud/policy_evaluation.go`
- [X] T013 Add PolicyService field to Cloud struct in `internal/cloud/cloud.go`
- [X] T014 Update NewCloud function in `internal/cloud/cloud.go` to initialize PolicyService

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Evaluate Sentinel Policy Results (Priority: P1) 🎯 MVP

**Goal**: Retrieve and display Sentinel policy evaluation results from TFC runs, supporting both legacy and modern API formats

**Independent Test**: Run `tfci policy show --run-id <run-id>` against a TFC run with completed policy evaluations and verify output shows policy counts and failed policy details

### Unit Tests for User Story 1 (TDD - Write FIRST, ensure FAIL)

- [ ] T015 [P] [US1] Create test file `internal/cloud/policy_evaluation_test.go` with testClient helper
- [ ] T016 [P] [US1] Write unit test `TestGetPolicyEvaluation_ModernAPI` using go-tfe mocks for task-stages format
- [ ] T017 [P] [US1] Write unit test `TestGetPolicyEvaluation_LegacyAPI` using go-tfe mocks for policy-checks format
- [ ] T018 [P] [US1] Write unit test `TestGetPolicyEvaluation_InvalidRunID` to verify error handling
- [ ] T019 [P] [US1] Write unit test `TestGetPolicyEvaluation_NoPolicies` to handle workspace without policies
- [ ] T020 [P] [US1] Write unit test `TestGetPolicyEvaluation_NoWait_Pending` to verify fast-fail when NoWait=true

**Verify**: All tests FAIL (implementations don't exist yet)

### Service Implementation for User Story 1

- [X] T021 [US1] Implement GetPolicyEvaluation method in `internal/cloud/policy_evaluation.go` with context validation
- [X] T022 [US1] Implement getPolicyFromTaskStages helper in `internal/cloud/policy_evaluation.go` for modern API parsing
- [X] T023 [US1] Implement getPolicyFromPolicyCheck helper in `internal/cloud/policy_evaluation.go` for legacy API parsing
- [X] T024 [US1] Add retry logic with Fibonacci backoff in `internal/cloud/policy_evaluation.go` using sethvargo/go-retry
- [X] T025 [US1] Implement wait logic in GetPolicyEvaluation that polls until evaluation completes (unless NoWait=true)
- [X] T026 [US1] Add structured logging (DEBUG, INFO, ERROR) in `internal/cloud/policy_evaluation.go`
- [X] T027 [US1] Add API endpoint detection logic (try modern first, fall back to legacy) in GetPolicyEvaluation

**Verify**: Unit tests for US1 now PASS

### CLI Command for User Story 1

- [X] T028 [US1] Create `internal/command/policy_show.go` with PolicyShowCommand struct embedding Meta
- [X] T029 [US1] Implement Run() method in PolicyShowCommand with flag parsing (--run-id, --json, --no-wait)
- [X] T030 [US1] Implement Help() method in PolicyShowCommand with usage documentation
- [X] T031 [US1] Implement Synopsis() method in PolicyShowCommand
- [X] T032 [US1] Add human-readable output formatting in PolicyShowCommand (policy counts, failed policies list, run URL)
- [X] T033 [US1] Add JSON output formatting in PolicyShowCommand using Writer interface
- [X] T034 [US1] Add global flag support (--hostname, --organization, --token) in PolicyShowCommand
- [X] T035 [US1] Register "policy show" command in `cli.go` command factory

### Integration Tests for User Story 1

- [ ] T036 [US1] Write integration test `TestPolicyShowCommand_Integration` in `internal/command/policy_show_test.go` using testClient() against live TFC
- [ ] T037 [US1] Write integration test for JSON output format validation
- [ ] T038 [US1] Write integration test for --no-wait flag behavior

**Checkpoint**: User Story 1 complete - `policy show` command fully functional and independently testable

---

## Phase 4: User Story 2 - Override Mandatory Policy Failures (Priority: P2)

**Goal**: Apply policy overrides with justification to unblock deployments when mandatory policies fail

**Independent Test**: Run `tfci policy override --run-id <run-id> --justification "reason"` against a run with mandatory failures in post_plan_awaiting_decision status and verify override is applied

### Unit Tests for User Story 2 (TDD - Write FIRST, ensure FAIL)

- [ ] T039 [P] [US2] Create test file `internal/cloud/policy_override_test.go` with test helpers
- [ ] T040 [P] [US2] Write unit test `TestOverridePolicy_ModernAPI` using go-tfe mocks for task-stages override
- [ ] T041 [P] [US2] Write unit test `TestOverridePolicy_LegacyAPI` using go-tfe mocks for policy-checks override
- [ ] T042 [P] [US2] Write unit test `TestOverridePolicy_InvalidStatus` to verify status validation
- [ ] T043 [P] [US2] Write unit test `TestOverridePolicy_ShortJustification` to verify justification length check
- [ ] T044 [P] [US2] Write unit test `TestOverridePolicy_NoPolicyCheck` to handle runs without policy checks
- [ ] T045 [P] [US2] Write unit test `TestOverridePolicy_PermissionDenied` to verify error handling

**Verify**: All tests FAIL (implementations don't exist yet)

### Service Implementation for User Story 2

- [X] T046 [US2] Implement OverridePolicy method in `internal/cloud/policy_override.go` with option validation
- [X] T047 [US2] Implement validateOverrideEligibility helper in `internal/cloud/policy_override.go` to check run status
- [X] T048 [US2] Implement overrideViaTaskStage helper in `internal/cloud/policy_override.go` for modern API
- [X] T049 [US2] Implement overrideViaPolicyCheck helper in `internal/cloud/policy_override.go` for legacy API
- [X] T050 [US2] Add justification comment to run using Comments API in both override helpers
- [X] T051 [US2] Add polling logic in OverridePolicy to wait for status transition after override
- [X] T052 [US2] Add structured logging (DEBUG, INFO, ERROR) in `internal/cloud/policy_override.go`
- [X] T053 [US2] Add error handling for terminal states (discarded, errored, canceled) during override

**Verify**: Unit tests for US2 now PASS

### CLI Command for User Story 2

- [X] T054 [US2] Create `internal/command/policy_override.go` with PolicyOverrideCommand struct embedding Meta
- [X] T055 [US2] Implement Run() method in PolicyOverrideCommand with flag parsing (--run-id, --justification, --json)
- [X] T056 [US2] Implement Help() method in PolicyOverrideCommand with usage documentation
- [X] T057 [US2] Implement Synopsis() method in PolicyOverrideCommand
- [X] T058 [US2] Add human-readable output formatting in PolicyOverrideCommand (override status, next steps, run URL)
- [X] T059 [US2] Add JSON output formatting in PolicyOverrideCommand using Writer interface
- [X] T060 [US2] Add input validation in PolicyOverrideCommand (justification min length, run ID format)
- [X] T061 [US2] Add exit code handling in PolicyOverrideCommand (0=success, 1=error, 2=discarded, 3=timeout)
- [X] T062 [US2] Register "policy override" command in `cli.go` command factory

### Integration Tests for User Story 2

- [ ] T063 [US2] Write integration test `TestPolicyOverrideCommand_Integration` in `internal/command/policy_override_test.go` using testClient()
- [ ] T064 [US2] Write integration test for justification comment verification in TFC run
- [ ] T065 [US2] Write integration test for status transition validation after override
- [ ] T066 [US2] Write integration test for JSON output format validation

**Checkpoint**: User Story 2 complete - `policy override` command fully functional and independently testable

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements, documentation, and validation across all user stories

- [ ] T067 [P] Update `docs/USAGE.md` with policy command documentation (examples, flags, exit codes)
- [ ] T068 [P] Update `CHANGELOG.md` with policy operations feature entry
- [ ] T069 [P] Add policy operations examples to `README.md`
- [ ] T070 [P] Verify all godoc comments are complete for exported types and functions
- [ ] T071 Run `go test ./internal/cloud/... -cover` and verify >80% coverage for policy service
- [ ] T072 Run `go test ./internal/command/... -cover` and verify coverage for policy commands
- [X] T073 Run `make lint` and fix any linting errors
- [X] T074 Run `make build` and verify binary compiles successfully
- [ ] T075 Validate quickstart.md examples against actual implementation
- [ ] T076 Test Docker image build with `docker build -t tfci:test .`
- [ ] T077 Test policy commands in Docker container with environment variables
- [ ] T078 Run integration tests against TFC with `TFE_TOKEN=xxx go test -v ./internal/cloud/... -timeout=15m`
- [ ] T079 Manual testing: Create TFC run → Check policies → Override → Verify in TFC UI
- [ ] T080 Security review: Verify API tokens never logged, justifications stored correctly

**Checkpoint**: Feature complete, tested, documented, and ready for PR

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (Phase 1) completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) completion
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2) completion - Can run in parallel with US1 if team capacity allows
- **Polish (Phase 5)**: Depends on US1 and US2 completion

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories - **MVP CORE**
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Uses PolicyEvaluation types from US1 but no runtime dependency - **MVP CORE**

### Within Each User Story

1. **Tests FIRST** (TDD): Write unit tests → Verify they FAIL
2. **Implementation**: Service layer methods → Verify tests PASS
3. **CLI Commands**: Command implementation → Integration tests
4. **Validation**: Run tests, verify independent story functionality

### Parallel Opportunities

**Phase 1 (Setup)**: All 3 tasks can run in parallel

**Phase 2 (Foundational)**: Tasks T005-T009 can run in parallel (all define types in same file)

**Phase 3 (User Story 1)**:

- Unit tests (T015-T020) can all run in parallel
- After T021 completes: T022, T023, T024, T025, T026, T027 can run in parallel (different helpers)
- After T028 completes: T029, T030, T031, T032, T033, T034 can run in parallel (different methods)
- Integration tests (T036-T038) can run in parallel

**Phase 4 (User Story 2)**:

- Unit tests (T039-T045) can all run in parallel
- After T046 completes: T047-T053 can run in parallel (different helpers)
- After T054 completes: T055-T061 can run in parallel (different methods)
- Integration tests (T063-T066) can run in parallel

**Phase 5 (Polish)**: Tasks T067-T070 can run in parallel (different documentation files)

**Cross-Story Parallelism**: Once Phase 2 completes, User Story 1 (Phase 3) and User Story 2 (Phase 4) can be worked on by different developers simultaneously

---

## Parallel Example: User Story 1

```bash
# After T015-T020 complete, launch in parallel:
Task T016: "Write unit test TestGetPolicyEvaluation_ModernAPI"
Task T017: "Write unit test TestGetPolicyEvaluation_LegacyAPI"
Task T018: "Write unit test TestGetPolicyEvaluation_InvalidRunID"
Task T019: "Write unit test TestGetPolicyEvaluation_NoPolicies"
Task T020: "Write unit test TestGetPolicyEvaluation_NoWait_Pending"

# After T021 completes, launch helpers in parallel:
Task T022: "Implement getPolicyFromTaskStages helper"
Task T023: "Implement getPolicyFromPolicyCheck helper"
Task T024: "Add retry logic with Fibonacci backoff"
Task T025: "Implement wait logic"
Task T026: "Add structured logging"
Task T027: "Add API endpoint detection"
```

---

## Implementation Strategy

### MVP First (User Story 1 + User Story 2)

1. Complete Phase 1: Setup → Branch and directories ready
2. Complete Phase 2: Foundational → All types, interfaces, foundation ready
3. Complete Phase 3: User Story 1 → `policy show` command working
4. **VALIDATE US1**: Test independently against TFC
5. Complete Phase 4: User Story 2 → `policy override` command working
6. **VALIDATE US2**: Test independently against TFC
7. Complete Phase 5: Polish → Documentation and final validation
8. **STOP and DEMO**: Policy operations MVP ready for PR

**Why this MVP?**

- US1 enables policy inspection (read-only, safe)
- US2 enables policy overrides (write operation, completes workflow)
- Together they provide complete policy workflow automation
- US3 is redundant (wait behavior built into US1)

### Incremental Delivery

1. **Week 1**: Setup + Foundational + User Story 1
   - Deliverable: `tfci policy show` command
   - Demo: Show policy evaluation results in CI/CD
2. **Week 2**: User Story 2
   - Deliverable: `tfci policy override` command
   - Demo: Complete policy workflow (evaluate → override → proceed)
3. **Week 3**: Polish
   - Deliverable: Documentation, tests, validation
   - Demo: Production-ready feature

### Parallel Team Strategy

With 2 developers after Phase 2 completes:

1. **Developer A**: User Story 1 (Policy Evaluation)
   - Tests → Service → CLI → Integration tests
2. **Developer B**: User Story 2 (Policy Override)

   - Tests → Service → CLI → Integration tests

3. **Both**: Phase 5 Polish together
   - Documentation, validation, integration testing

---

## Task Count Summary

- **Phase 1 (Setup)**: 3 tasks
- **Phase 2 (Foundational)**: 11 tasks (BLOCKING)
- **Phase 3 (User Story 1)**: 24 tasks (MVP Core)
- **Phase 4 (User Story 2)**: 28 tasks (MVP Core)
- **Phase 5 (Polish)**: 14 tasks

**Total**: 80 tasks

**MVP Scope** (US1 + US2): 80 tasks (All phases)

**Parallel Opportunities**: 30+ tasks can run in parallel within phases

**Estimated Effort**:

- Solo developer: 2-3 weeks
- Team of 2: 1-2 weeks
- With parallel execution: 1 week sprint possible

---

## Notes

- All tasks follow strict checklist format: `- [ ] [TaskID] [P?] [Story?] Description with file path`
- [P] indicates tasks that can run in parallel (different files, no blocking dependencies)
- [Story] label (US1, US2) maps each task to its user story for traceability
- Each user story is independently completable and testable
- TDD approach: Write tests FIRST, verify they FAIL, implement, verify they PASS
- Integration tests use testClient() pattern from existing codebase
- Unit tests use go-tfe/mocks from `github.com/hashicorp/go-tfe/mocks`
- All service methods accept context.Context as first parameter
- All errors use structured error types defined in Phase 2
- Structured logging follows existing patterns: [DEBUG], [INFO], [ERROR]
- Exit codes for CI/CD: 0=success, 1=error, 2=discarded, 3=timeout
- API compatibility: Automatic detection (try modern, fall back to legacy)

---

## Validation Checklist

Before marking feature complete:

- ✅ All unit tests pass with >80% coverage
- ✅ All integration tests pass against live TFC
- ✅ `policy show` command works with both modern and legacy TFC APIs
- ✅ `policy override` command applies overrides and adds justification comments
- ✅ Commands work in Docker container with environment variables
- ✅ JSON output is valid and parseable
- ✅ Error messages are clear and actionable
- ✅ Documentation matches implementation
- ✅ Quickstart examples are validated
- ✅ No API tokens logged or exposed
- ✅ Follows existing codebase patterns (service layer, cloudMeta, Writer interface)
- ✅ All godoc comments complete
- ✅ Linting passes
- ✅ Binary compiles successfully
