# Quickstart: Policy Operations

**Feature**: Policy Evaluation and Override Operations  
**CLI Tool**: tfc-workflows-tooling (`tfci`)  
**Version**: 1.x (unreleased - in development on branch 001-policy-operations)

## Prerequisites

### Required

1. **TFC/TFE Account** with workspace configured with Sentinel policies
2. **API Token**: User or team token with permissions to:
   - Read runs
   - Read policy checks/evaluations
   - Override policies (if using `policy override`)
3. **Docker** (recommended) or Go 1.24+ for local builds

### Environment Variables

```bash
# Required
export TF_API_TOKEN="your-tfc-api-token"

# Optional (defaults)
export TF_HOSTNAME="app.terraform.io"         # Default: Terraform Cloud
export TF_CLOUD_ORGANIZATION="your-org-name"  # Can also use --organization flag
export TF_LOG="DEBUG"                          # Enable debug logging
```

### Get Your API Token

**Terraform Cloud**:

1. Go to https://app.terraform.io/app/settings/tokens
2. Click "Create an API token"
3. Copy token and export: `export TF_API_TOKEN="..."`

**Terraform Enterprise**:

1. Go to `https://<your-tfe-hostname>/app/settings/tokens`
2. Follow same process as TFC

---

## Installation

### Docker (Recommended)

```bash
# Pull latest image
docker pull hashicorp/tfci:latest

# Verify installation
docker run --rm hashicorp/tfci:latest tfci --version
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/hashicorp/tfc-workflows-tooling.git
cd tfc-workflows-tooling

# Checkout feature branch
git checkout 001-policy-operations

# Build binary
make build

# Verify installation
./tfci --version
```

---

## Basic Usage

### 1. Check Policy Evaluation Results

**Scenario**: After a Terraform plan, you want to see which policies passed or failed.

```bash
# Check policy results for a run
tfci policy show --run-id run-abc123def456

# With JSON output (for parsing in CI/CD)
tfci policy show --run-id run-abc123def456 --json
```

**Human-Readable Output**:

```
📊 Policy Evaluation Summary
   Total Policies: 8
   ✅ Passed: 5
   ⚠️  Failed (Advisory): 1
   🚫 Failed (Mandatory): 2
   ❌ Errored: 0

🚫 Failed Mandatory Policies:
   - aws-cost-limit (mandatory)
     Terraform run exceeds $500 daily cost threshold

   - security-group-ingress (mandatory)
     Security group allows unrestricted ingress from 0.0.0.0/0

ℹ️  Override Required: Policy override needed to proceed

View detailed results: https://app.terraform.io/app/my-org/workspaces/my-workspace/runs/run-abc123def456
```

**JSON Output**:

```json
{
  "run_id": "run-abc123def456",
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

**Exit Codes**:

- `0`: Success, policies retrieved
- `1`: Error (invalid run ID, API error, network failure)

---

### 2. Override Mandatory Policy Failures

**Scenario**: Mandatory policies failed, but you've received approval to override them.

```bash
# Apply policy override with justification
tfci policy override \
  --run-id run-abc123def456 \
  --justification "Emergency hotfix approved by CTO - Incident INC-12345"

# With JSON output
tfci policy override \
  --run-id run-abc123def456 \
  --justification "Emergency hotfix approved by CTO - Incident INC-12345" \
  --json
```

**Human-Readable Output**:

```
🛡️  Applying policy override...
Justification: Emergency hotfix approved by CTO - Incident INC-12345

✅ Policy override applied successfully
📝 Justification comment added to run
⏳ Waiting for override to complete...

✅ Override complete! Run status: policy_override

Next Steps:
- Run the Apply workflow to deploy changes
- View run: https://app.terraform.io/app/my-org/workspaces/my-workspace/runs/run-abc123def456
```

**JSON Output**:

```json
{
  "run_id": "run-abc123def456",
  "policy_stage_id": "ts-xyz789uvw012",
  "justification": "Emergency hotfix approved by CTO - Incident INC-12345",
  "initial_status": "post_plan_awaiting_decision",
  "final_status": "policy_override",
  "override_complete": true,
  "timestamp": "2025-12-04T10:30:45Z"
}
```

**Exit Codes**:

- `0`: Override applied successfully
- `1`: Error (wrong status, no mandatory failures, permissions error)
- `2`: Run discarded during override
- `3`: Override timeout

**Justification Requirements**:

- Minimum 10 characters
- Should reference approval source (e.g., incident ticket, change request)
- Added as comment to run for audit trail

---

### 3. Understanding Built-in Wait Behavior

**Scenario**: The `policy show` command automatically waits for policy evaluation to complete.

```bash
# Default behavior: Waits for policy evaluation (follows WorkspaceService pattern)
tfci policy show --run-id run-abc123def456

# This will poll with retry until policies are evaluated
# Uses Fibonacci backoff (10s, 10s, 20s, 30s, 30s...)
# Respects context timeout
```

**Fast-fail mode** (skip waiting):

```bash
# Fail immediately if policies not yet evaluated
tfci policy show --run-id run-abc123def456 --no-wait

if [ $? -eq 1 ]; then
  echo "Policies still evaluating, check back later"
  exit 0
fi
```

**Typical workflow in CI/CD**:

```bash
# Create run
RUN_ID=$(tfci run create --workspace my-workspace --output json | jq -r '.id')

# Check policies (automatically waits for evaluation)
tfci policy show --run-id $RUN_ID --json > policy-results.json

# Parse results and decide
REQUIRES_OVERRIDE=$(jq -r '.requires_override' policy-results.json)

if [ "$REQUIRES_OVERRIDE" = "true" ]; then
  echo "Manual override required"
  # Notify team or apply automated override
else
  echo "Policies passed, proceeding to apply"
  tfci run apply --run-id $RUN_ID
fi
```

---

## Docker Usage

### Run with Docker

```bash
# Check policies
docker run --rm \
  -e TF_API_TOKEN="${TF_API_TOKEN}" \
  -e TF_CLOUD_ORGANIZATION="my-org" \
  hashicorp/tfci:latest \
  tfci policy show --run-id run-abc123def456

# Override with JSON output
docker run --rm \
  -e TF_API_TOKEN="${TF_API_TOKEN}" \
  -e TF_CLOUD_ORGANIZATION="my-org" \
  hashicorp/tfci:latest \
  tfci policy override \
    --run-id run-abc123def456 \
    --justification "Emergency fix - INC-12345" \
    --json

# Check policies with --no-wait (fail fast if not ready)
docker run --rm \
  -e TF_API_TOKEN="${TF_API_TOKEN}" \
  -e TF_CLOUD_ORGANIZATION="my-org" \
  hashicorp/tfci:latest \
  tfci policy show \
    --run-id run-abc123def456 \
    --no-wait
```

### Create Shell Alias

```bash
# Add to ~/.bashrc or ~/.zshrc
alias tfci='docker run --rm \
  -e TF_API_TOKEN \
  -e TF_CLOUD_ORGANIZATION \
  -e TF_HOSTNAME \
  hashicorp/tfci:latest \
  tfci'

# Use like native command
tfci policy show --run-id run-abc123
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Terraform Deploy with Policy Override

on:
  workflow_dispatch:
    inputs:
      run_id:
        description: "TFC Run ID"
        required: true
      justification:
        description: "Policy override justification"
        required: false

jobs:
  check-policies:
    runs-on: ubuntu-latest
    outputs:
      requires_override: ${{ steps.policy-check.outputs.requires_override }}
    steps:
      - name: Check Policy Results
        id: policy-check
        run: |
          docker run --rm \
            -e TF_API_TOKEN="${{ secrets.TF_API_TOKEN }}" \
            -e TF_CLOUD_ORGANIZATION="${{ vars.TF_CLOUD_ORGANIZATION }}" \
            hashicorp/tfci:latest \
            tfci policy show --run-id ${{ github.event.inputs.run_id }} --json \
            > policy-results.json

          cat policy-results.json

          REQUIRES_OVERRIDE=$(jq -r '.requires_override' policy-results.json)
          echo "requires_override=$REQUIRES_OVERRIDE" >> $GITHUB_OUTPUT

          MANDATORY_FAILED=$(jq -r '.mandatory_failed_count' policy-results.json)
          echo "❌ Mandatory Policies Failed: $MANDATORY_FAILED"

      - name: Upload Policy Results
        uses: actions/upload-artifact@v3
        with:
          name: policy-results
          path: policy-results.json

  override-if-needed:
    runs-on: ubuntu-latest
    needs: check-policies
    if: needs.check-policies.outputs.requires_override == 'true'
    steps:
      - name: Apply Policy Override
        run: |
          docker run --rm \
            -e TF_API_TOKEN="${{ secrets.TF_API_TOKEN }}" \
            -e TF_CLOUD_ORGANIZATION="${{ vars.TF_CLOUD_ORGANIZATION }}" \
            hashicorp/tfci:latest \
            tfci policy override \
              --run-id ${{ github.event.inputs.run_id }} \
              --justification "${{ github.event.inputs.justification }}" \
              --json

  apply-terraform:
    runs-on: ubuntu-latest
    needs: [check-policies, override-if-needed]
    if: always() && (needs.check-policies.outputs.requires_override == 'false' || needs.override-if-needed.result == 'success')
    steps:
      - name: Apply Terraform Run
        run: |
          docker run --rm \
            -e TF_API_TOKEN="${{ secrets.TF_API_TOKEN }}" \
            -e TF_CLOUD_ORGANIZATION="${{ vars.TF_CLOUD_ORGANIZATION }}" \
            hashicorp/tfci:latest \
            tfci run apply --run-id ${{ github.event.inputs.run_id }}
```

### GitLab CI

```yaml
stages:
  - check-policies
  - override
  - apply

variables:
  TFCI_IMAGE: "hashicorp/tfci:latest"

check-policies:
  stage: check-policies
  image: $TFCI_IMAGE
  script:
    # policy show automatically waits for evaluation to complete
    - tfci policy show --run-id $RUN_ID --json > policy-results.json
    - cat policy-results.json
    - export REQUIRES_OVERRIDE=$(jq -r '.requires_override' policy-results.json)
    - echo "REQUIRES_OVERRIDE=$REQUIRES_OVERRIDE" >> build.env
  artifacts:
    reports:
      dotenv: build.env
    paths:
      - policy-results.json

override-policies:
  stage: override
  image: $TFCI_IMAGE
  rules:
    - if: '$REQUIRES_OVERRIDE == "true" && $CI_COMMIT_MESSAGE =~ /\[override\]/'
  script:
    - |
      tfci policy override \
        --run-id $RUN_ID \
        --justification "$CI_COMMIT_MESSAGE - $CI_PIPELINE_URL" \
        --json

apply-terraform:
  stage: apply
  image: $TFCI_IMAGE
  script:
    - tfci run apply --run-id $RUN_ID
```

---

## Advanced Usage

### Chaining Commands

```bash
# Check policies, override if needed, then apply
RUN_ID="run-abc123def456"

# Step 1: Check policy status
POLICY_RESULT=$(tfci policy show --run-id $RUN_ID --json)
REQUIRES_OVERRIDE=$(echo $POLICY_RESULT | jq -r '.requires_override')

# Step 2: Override if mandatory failures exist
if [ "$REQUIRES_OVERRIDE" = "true" ]; then
  echo "Mandatory policies failed, applying override..."
  tfci policy override \
    --run-id $RUN_ID \
    --justification "Automated deployment - Approved in CHG-67890"
fi

# Step 3: Apply the run
tfci run apply --run-id $RUN_ID
```

### Parsing JSON Output

```bash
# Extract specific fields with jq
MANDATORY_FAILED=$(tfci policy show --run-id run-abc123 --json | jq -r '.mandatory_failed_count')
FAILED_POLICIES=$(tfci policy show --run-id run-abc123 --json | jq -r '.failed_policies[].policy_name')

echo "Mandatory Failed: $MANDATORY_FAILED"
echo "Failed Policies:"
echo "$FAILED_POLICIES"

# Use in conditional logic
if [ "$MANDATORY_FAILED" -gt 0 ]; then
  echo "❌ Deployment blocked by mandatory policies"
  exit 1
fi
```

### Multiple Runs in Parallel

```bash
# Check policies for multiple runs concurrently
RUN_IDS=("run-abc123" "run-def456" "run-ghi789")

for RUN_ID in "${RUN_IDS[@]}"; do
  (
    echo "Checking $RUN_ID..."
    tfci policy show --run-id $RUN_ID --json > "policy-$RUN_ID.json"
  ) &
done

wait
echo "All policy checks complete"
```

---

## Troubleshooting

### Common Errors

#### Error: "invalid run ID format"

```bash
# ❌ Wrong: Missing "run-" prefix
tfci policy show --run-id abc123def456

# ✅ Correct: Include "run-" prefix
tfci policy show --run-id run-abc123def456
```

#### Error: "run status does not allow this operation"

```bash
# Override only works when run is awaiting decision
# Check current run status first
tfci run show --run-id run-abc123def456 | grep "Status:"

# If status is not "post_plan_awaiting_decision", cannot override
```

#### Error: "justification must be at least 10 characters"

```bash
# ❌ Wrong: Justification too short
tfci policy override --run-id run-abc123 --justification "hotfix"

# ✅ Correct: Meaningful justification
tfci policy override --run-id run-abc123 --justification "Emergency hotfix for production incident INC-12345"
```

#### Error: "insufficient permissions for this operation"

```bash
# Your API token needs override permissions
# Solution 1: Use a token with workspace admin role
# Solution 2: Request override permissions from workspace admin
```

### Enable Debug Logging

```bash
# See all API calls and responses
export TF_LOG=DEBUG
tfci policy show --run-id run-abc123def456
```

---

## Tips & Best Practices

### 1. Always Include Meaningful Justifications

```bash
# ❌ Bad: Generic justification
--justification "Need to deploy"

# ✅ Good: Specific with reference
--justification "Emergency hotfix for CVE-2025-1234 - Approved by Security Team in INC-67890"
```

### 2. Use JSON Output in Automation

```bash
# Human-readable for manual use
tfci policy show --run-id run-abc123

# JSON for scripting/CI/CD
tfci policy show --run-id run-abc123 --json | jq '.mandatory_failed_count'
```

### 3. Handle Pending Policy Evaluations

```bash
# Default: Wait for policies to complete (automatic retry)
tfci policy show --run-id run-abc123

# Fast-fail: Return immediately if not ready
tfci policy show --run-id run-abc123 --no-wait
if [ $? -ne 0 ]; then
  echo "Policies not ready yet"
fi
```

### 4. Control Wait Timeout with Context

```bash
# Use shell timeout for overall command timeout
timeout 5m tfci policy show --run-id run-abc123

# Or set via context in automation
export TF_CLI_TIMEOUT="5m"
tfci policy show --run-id run-abc123
```

---

## Next Steps

- **Full Documentation**: See [USAGE.md](../../docs/USAGE.md) for all commands
- **Contributing**: See [CONTRIBUTING.md](../../docs/CONTRIBUTING.md) for development guidelines
- **Support**: Open issues at https://github.com/hashicorp/tfc-workflows-tooling/issues
