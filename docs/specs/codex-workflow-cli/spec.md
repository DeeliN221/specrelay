# SpecRelay workflow runner specification

## Scope and compatibility

`specrelay flow` runs proposals through planning, execution, self-review, independent QA, targeted repair, and final delivery. This English specification consolidates the current behavior. The [earlier Chinese specification](../../zh-CN/specs/codex-workflow-cli/spec.md) is retained as a translated historical reference, including superseded generation-specific requirements. Existing sealed runs continue under their captured workflow generation and configuration; current defaults MUST NOT silently reinterpret old snapshots.

## Proposal selection and queues

### Requirement: Select valid active proposals

The runner MUST list executable active changes, support one or multiple selections, validate numeric/range input, and order a batch by proposal number. Invalid selections MUST fail without starting work. The human `run` shortcut selects all active changes. A single selected proposal still uses the same appendable queue abstraction.

### Requirement: Execute batches serially

A batch MUST start the next proposal only after the previous one finishes. Failures stop automatic advancement. Running batches accept new active proposals, excluding duplicates and already queued items. Completed, failed, and stopped batches MUST reject inappropriate append attempts. Interactive startup MUST offer appropriate resume, append, planning, stop, or new-run choices.

Planning uses the configured agent interactively and delegates to `specrelay-plan`. After planning, newly created proposals become selectable. Inserting an unrelated new proposal while another runs MUST NOT be mistaken for unauthorized modification of the current change.

## Durable sealed execution

### Requirement: Capture immutable inputs

Before creating a sealed run, the runner MUST validate the proposal, configuration, required agent executables, and Git baseline. A repository without an initial commit MUST fail with an actionable diagnostic. The run captures its workflow configuration, prompts, acceptance contract, baseline, sessions, and state under the project-specific runtime directory. Later user configuration changes MUST NOT alter an existing sealed run.

The acceptance snapshot MUST carry integrity hashes and reject changes, missing hashes, or mismatched run/stage/attempt/diff bindings. Runtime results, source files, and evidence MUST NOT be trusted only because their filenames match.

### Requirement: Advance through explicit stages

Stage decisions MUST remain separate from execution and persistence. The current quality loop proceeds from execution to audit, then independent QA. Findings return to the appropriate repair stage. QA-targeted repairs address only the latest findings and directly related regressions; they must not expand proposal scope.

Self-review may repeat as needed. The configured `max_audit_iterations` defaults to 3; reaching it transfers control to independent QA instead of running audits forever. A zero limit preserves unlimited audit rounds. The required clean confirmation and stage transitions remain explicit in the generation-specific decision logic. Legacy repair/review limits continue to apply only to their historical workflow generations.

After independent QA fails, targeted repair MUST rerun failed tests, all required tests, and validation commands. A clean result with failing or stale checks cannot advance. The current targeted-repair loop MUST NOT inherit the historical review iteration ceiling.

### Requirement: Stabilize hooks before QA

Execution and audit MUST use the repository's existing explicit pre-commit entry point without temporary commits. Changes made by hooks must be absorbed, followed by another invocation that leaves files unchanged. Affected tests, required tests, and validation commands then run against the final inputs.

No implementation, audit, or targeted-repair stage may create a Git commit. The archive stage alone creates the complete delivery commit. Post-QA hooks MUST NOT introduce previously untested changes; such changes return the run to audit and independent QA.

### Requirement: Protect independent QA inputs

QA MUST be an independent role/session that observes the latest trusted inputs without modifying tracked source, the sealed acceptance contract, temporary evidence sources, or the final submission bundle. Before and after QA, the runner verifies the tested diff and the latest passing audit/repair checkpoint. Drift during preparation, execution, or gate validation invalidates trust and reroutes to a full audit.

QA must assess user-visible results, not merely command success, file existence, a hash, or HTTP 200. Findings include reproducible failure evidence. Historical debt belongs in `non_blocking_findings` with scope `out_of_scope_existing`; it must not introduce unrelated IDs into the acceptance matrix.

### Requirement: Preserve immutable acceptance evidence

Each required test and evidence item MUST be bound to the acceptance contract and stage attempt. The engine seals actual evidence files and logs, verifies their hashes, and records immutable locations. Temporary files in `test-results/` remain mutable runtime sources, never the final source of truth after sealing.

Final independent QA must cover every required test and evidence ID exactly as declared. Its delivery observations must cover all user scenarios. The engine promotes only the final passing immutable snapshots into `tests/evidence/proposals/<change>/`. Missing or tampered sealed data MUST fail rather than be rebuilt from mutable sources.

### Requirement: Read-only delivery after final QA

Archive checks the trusted final QA and promotes its evidence before invoking the archive agent. The agent verifies the reviewer-facing report and performs only mechanical proposal movement and delivery summary creation. After the archive command returns, it must not edit, format, or restore proposal/source content merely to make the worktree clean.

The archive gate distinguishes expected proposal moves/reference rewrites from untested source drift. Implementation, archived proposal, and final evidence MUST be present in exactly one complete delivery commit from `delivery_base_head`; they must all belong to HEAD. Missing, ignored, untracked, separately committed, symbolic-link, or changed evidence MUST be rejected. Dirty implementation after the delivery commit must remain visible and return to repair rather than be silently overwritten.

Evidence-quality failures can return to repair with the exact diagnostic. Archive recovery must find a unique sealed proposal location and must not overwrite unrelated work. The final `delivery-summary.md` includes the final review; generated `DELIVERY.md` is validated separately by the shared acceptance package.

## Agent backends and sessions

### Requirement: Backend-aware process invocation

The runner supports configured Codex, Pi, Agy, and Claude backends through explicit adapters. Unknown backends fail. Before a sealed run starts, the runner checks only executables actually referenced by its snapshot, rather than unrelated optional tools.

Arguments MUST be passed directly and multiline prompts through stdin. JSONL/stream events track session IDs and diagnostics. Pi session headers and Claude stream-json session identity must be preserved. stderr belongs in process-failure diagnostics. The system must not hide a failed agent behind a successful parser exit.

### Requirement: Role and backend session isolation

Planning, execution, reviewing, repairing, QA, and archiving use their appropriate role sessions. Repeated turns of the same role should resume the same backend-scoped session. A session from one backend MUST NOT be passed to another. Historical fixer sessions remain compatible with their stored backend identity.

Stage configuration controls the agent, model, reasoning, and supported fast mode. Unsupported or obsolete aliases must fail explicitly. A backend that does not implement fast mode must not receive an invented flag.

### Requirement: Output-idle recovery

A process that stops producing output may be interrupted for retry using the runner's explicit retry classification. Its observed session identity and durable artifacts remain available. Retry must not fabricate completion or erase useful partial evidence.

## Configuration and prompts

### Requirement: One source of defaults

Configuration merges embedded defaults, user `~/specrelay-flow.yaml`, repository `specrelay-flow.yaml`, and finally the immutable run snapshot. `specrelay flow config` writes the embedded default, optionally to the user scope with `--global`. It MUST NOT overwrite an existing config or allow both `.yaml` and `.yml` to silently compete.

The public interface has no engine-selection or profile-selection option. Removed `flow init`, `flow install`, legacy top-level fields, and obsolete aliases MUST report their replacement or reject the input. Deprecated review/fix settings can migrate only where the compatibility parser explicitly supports them; conflicting old and new forms must fail.

Generated defaults, checked-in example configuration, stage-role metadata, templates, and documented behavior MUST agree. Workflow engine implementation details remain internal. The current default must not create a fixed external subagent/fan-in framework; legacy unsupported configuration fields must not reactivate removed behavior.

### Requirement: Prompt delegation and scope

Planning delegates to `specrelay-plan`, execution to `specrelay-exec`, and archiving to `specrelay-archive`. Prompts identify the current stage, run directory, proposal, and sealed acceptance contract. Execution must focus on the proposal's hard contract without copying the full skill body or injecting unrelated task queues.

First role turns establish the artifact schema; continuation turns reuse the role session and concise context rather than repeating large examples. Prompts expose the previous checkpoints required for the current decision, not every historical artifact indiscriminately. Missing required variables or snapshot prompt keys MUST fail closed.

Default templates and runtime-added instructions MUST be English. Chinese compatibility fixtures and explicit translations may remain, but English proposals must work without Chinese names or headings.

## Artifacts and validation gates

### Requirement: Strict review and repair artifacts

Review artifacts have a summary, decision, evidence, findings, checks, and optional non-blocking findings. Clean review requires all checks true, no blocking findings, and reproducible validation plus runtime/screenshot/trace/QA evidence. Findings must have valid severity and scope. Historical scope values remain restricted to their compatibility contract.

Repair artifacts use `clean` or `needs_more`. A repair that fixed issues but still needs confirmation uses `needs_more`; it must not claim clean based only on its intent. Artifact numeric enums supported by the parser must map consistently to their stable semantic values. A workflow-failure declaration requires the corresponding failure decision.

### Requirement: Strict independent QA artifacts

QA artifacts have summary, decision, evidence, acceptance matrix, findings, and optional non-blocking findings. Clean QA requires no blocking findings, meaningful evidence, and complete passed coverage of the sealed required tests/evidence. Unknown IDs or scenario evidence from outside the referenced delivery scenario MUST fail. `needs_fix` requires findings.

`user_acceptance` must record actual observations for every delivery scenario. Before/after comparisons must describe distinct observable behavior and use distinct meaningful evidence files.

### Requirement: Uniform stage artifact retry

Missing or invalid artifacts after an agent returns MUST trigger the same role/session retry at that stage. Artifact gates, deterministic validation gates, and acceptance gates remain separate in durable state. An invalid artifact must not appear as a completed stage or cause a batch to skip the proposal.

Execution preflight reuses `specrelay validate` and shared lifecycle diagnostics. A missing test, weak contract, forbidden `task.md`, or evidence without a traceable producer returns to the executor for correction. Acceptance execution records the same contract IDs and producer relationships used by validation and QA.

Validation commands support string and structured command forms, with strict field validation. Commands must not declare incompatible shell/run and executable/args forms. Stage validation attempts follow the configured limit and preserve their last passing checkpoint. Failed validation reruns the same stage, not an unrelated stage.

## Resume, restart, locks, and interruption

### Requirement: Resume from durable facts

An existing valid artifact can advance without rerunning the agent; missing or invalid artifacts resume the same stage. State transitions must be persisted atomically. Captured prompts/configuration remain authoritative. Unexpected changes outside the allowed current transition trigger intervention instead of silent absorption.

### Requirement: Recoverable blocks

`blocked_environment` and `blocked_stalled` are resumable pauses and keep their batch association. Missing prerequisites are recorded as variable names or paths, never secret values. A stalled run resumes only after new code, evidence, configuration, or user input provides a reason to progress.

Recovery compares against saved source/diff checkpoints. Environment recovery must not absorb undeclared-path changes or unbound HEAD changes. Historical terminal review/validation/contract blocks, completed runs, and user-stopped runs are not automatically restarted merely because an old record exists.

### Requirement: Targeted restart

`restart` selects the latest recoverable task or an explicit `-bN`/`-wN` alias. A single failed/interrupted run reuses its sealed identity and snapshots. Batch restart resumes the current change; where the batch contract requires replacing a failed run record, it must preserve the batch queue and refuse inconsistent/missing state.

A live worker, active lease, uncertain lock, or protected concurrent state MUST prevent destructive restart or archive. Detached startup failure rolls back only its own unchanged preparation generation. If another worker advances state or acquires a lease, stale rollback must be rejected rather than overwrite that progress.

### Requirement: Cross-platform locks

Lock ownership includes the resource, host, process, and owner token. A lock for a different resource or host must not be treated as locally killable. Reclamation checks that ownership has not changed. Interruption must preserve flushed logs and resumable stage state. Windows behavior follows the platform specification.

## Human and machine observability

### Requirement: Stable JSON runner interface

The CLI supports version/contract discovery and JSON change listing, run start, resume, restart, status, abort, acceptance execution, and artifact validation. JSON commands MUST emit machine-readable output without human checklists or update banners. Errors carry a nonzero exit status and structured diagnostics. State file fields and stable status values remain compatible.

### Requirement: Human status

Human status uses stable short task aliases and compact aligned columns. English stage labels replace the old Chinese two-character defaults; Unicode display widths still align user-provided names correctly. A batch shows each change with its own run details, failures, and appropriate recovery command.

Header elapsed time measures workflow wall time rather than the sum of overlapping stages. Stage durations derive from recorded/session timing; skipped stages do not contribute. A state completed before workflow start is distinct from a stage that actually ran.

Recoverable blocks explicitly show paused/resumable status, original stage, reason, and the relevant resume/restart command. Stale locks and unrecoverable failures give actionable cleanup/archive guidance. Startup summaries must not confuse a failed record with successful delivery.

### Requirement: Watch and loop

`watch` defaults to active work and accepts explicit aliases; it refreshes status and activity indicators without starting a new run. Missing or invalid targets fail clearly.

The loop monitors active batches, archives eligible failures, and continues remaining proposals. It stops for intervention-required states, exhausted proposals, user stop, or repeated unchanged failures. Lock/heartbeat liveness must be checked before reclaiming workers or records.

## Cleanup and updates

### Requirement: Scoped, planned cleanup

`clean` plans before deleting and supports dry-run and JSON output. It acts only on the current project's failed, damaged, or invalid runtime records. Completed usable history and active/protected runs remain intact. Batch cleanup handles referenced runs consistently and reports skipped active work.

Optional agent-session cleanup removes only records belonging to deleted runs across supported backend stores. Missing external stores or unknown schemas produce diagnostics without blocking safe local cleanup. Protected runs' external records must not be removed. Cleanup never rolls back repository code.

### Requirement: Verifiable self-update

Human status may announce an available update; JSON interfaces remain unaffected. `update` queries `DeeliN221/specrelay`, selects the current platform asset, requires a trusted HTTPS download and SHA-256 manifest, enforces download limits, verifies the extracted executable's version, and backs up the current binary.

A failed update retains usable rollback information and must not replace the current executable with an unverified file. Windows replacement may be deferred until the current process exits; output must explain the staged file and rollback command. Unknown versions or missing assets/checksums must not be guessed.

## Regression obligations

Tests MUST cover public CLI behavior, interactive selection, batch ordering, durable stage decisions, strict artifacts, immutable evidence, read-only QA, archive commit integrity, concurrency-safe recovery, Unicode display, English defaults, updater integrity, and Windows adapters. Shared fixtures should reduce duplication without weakening assertions. Language migration MUST update expected wording while retaining the underlying behavior checks; it must not delete tests or change machine fields to make the suite pass.
