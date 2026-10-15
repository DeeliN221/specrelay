#!/usr/bin/env bash
# Verify self review repair loop contract.
# Sources: Proposal 44, Proposal 45

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$REPO_ROOT"

# verify_source_contract: Verify maintained specs, roles and stage entries preserve the contract.
verify_source_contract() {
  rg -q 'Sources: 44-简化审核修正为自审自修循环' docs/specs/codex-workflow-cli/spec.md
  rg -q 'Sources: 45-收敛全量自查与QA定向修复闭环' docs/specs/codex-workflow-cli/spec.md
  rg -q 'Session: "repairer"' internal/app/stage_role.go
  rg -q 'max_repair_iterations|MaxRepairIterations' internal/app
  rg -q 'max_audit_iterations|MaxAuditIterations' internal/app
  rg -q 'repair_' internal/app/workflow_stage.go internal/app/stage_decision.go
  rg -q 'pre_qa_audit' internal/app
  rg -q 'qa_targeted_repair' internal/app
  rg -q 'blocked_environment' internal/app
  rg -q 'blocked_stalled' internal/app
  rg -q 'RepairConfirmationPending' internal/app/state_model.go internal/app/stage_decision.go
  rg -q '强制重审确认' prompts-template/specrelay-flow-repair.md
}

# verify_runtime_contract: Exercise self-review, targeted repair, recovery gates and legacy snapshots.
verify_runtime_contract() {
  go test ./internal/app \
    -run '^(TestPreQAFullAuditRequiresCleanConfirmation|TestPreQAAuditLimitMovesToQA|TestQAFailureTargetsFindingsAndSelfValidates|TestRepairLoopHasNoIterationCeiling|TestRepairEnvironmentBlockResumesWithoutBudget|TestRepairStalledBlockResumesWithProgress|TestQualityLoopResumeChecksLockBeforeUnblocking|TestQualityLoopQAGateRejectsCheckpointDriftDuringRun|TestQualityLoopArchiveGateRejectsFinalQAContentDrift|TestRepairWorkflowDAGResumeEvidence|TestLegacyRepairWorkflowResumeEvidence)$' \
    -count=1
}

verify_source_contract
verify_runtime_contract
