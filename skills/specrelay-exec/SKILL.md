---
name: specrelay-exec
description: Implement an active specrelay proposal using its brief, intent, acceptance contract and real tests; update obsolete tests when the authorized intent changes.
---

# specrelay Exec

Implement the current specrelay change.

## Entry types and context

For small work, read `brief.md`, `acceptance.json` and `tests/`, plus `proposal.md` when present. For standard work, also read the full `proposal.md`; consult `design.md` and `spec.md` for architectural decisions, conflicting historical tests or gaps in acceptance coverage. A small proposal does not reduce testing requirements or remove responsibility for maintained specifications.

## Optional external skills

Separately installed Matt Pocock skills may support implementation. They are not installed by `specrelay install`; check availability and fall back to this process.

- `/implement` and `/tdd`: use red-green-refactor while preserving proposal tests and aligning additions with `acceptance.json`.
- `/code-review`: review both engineering quality and specification compliance before independent QA; handle findings through self-review and repair.
- `/prototype`: investigate a disproved contract assumption with a disposable prototype. If the authorized contract changes, update its specification, acceptance and tests together and record why. Prototype code is not automatically delivery code.
- `/diagnosing-bugs`: diagnose persistent bugs, regressions or unexplained failures, then add relevant regression coverage within scope.
- `/codebase-design`: use as shared design vocabulary, not a replacement for implementation responsibility.

External skills cannot bypass or weaken the acceptance contract. Invoke their slash commands and follow their own procedures. Contract changes require the user's changed intent and synchronized specifications, acceptance and tests.

## Process

1. Confirm the proposal was committed before workflow startup. SpecRelay seals `delivery_base_head`; execution, audit, repair and tests must not create or rewrite commits after startup.
2. Read the brief, acceptance contract, proposal tests and full intent source. Do not infer intent solely from `acceptance.json` and tests.
3. Run creation-stage contract tests first. Initial failure should demonstrate missing behavior, not syntax, paths or a broken environment.
4. Compare every coverage entry and required assertion with the proposal's success criteria. A behavior or runtime guarantee is not established by a configuration string, file/script existence, HTTP status, element presence, log substring or successful exit alone. Strengthen weak behavioral coverage and document why before proceeding; do not weaken the intended outcome.
5. Implement the smallest verifiable change that fulfills the proposal.
6. Run affected tests and all `acceptance.json.required_tests` commands.
7. Before independent QA, explicitly run the repository's pre-commit hook entry without creating a temporary commit. Apply its edits and rerun until no files change.
8. After hooks stabilize, rerun affected tests, required tests and validation commands so evidence corresponds to the final content.
9. Keep the dynamic plan in executor Todo or runtime state. Report changes, validation and remaining risks.
10. Tell `fix-code` or `fix-webapp` that they are operating inside SpecRelay: change and test code and collect evidence, but let `specrelay-archive` create the final commit.

Check the proposal's committed status with:

```sh
git log --oneline -- docs/changes/<change>/
```

If necessary, commit the proposal before starting SpecRelay, using explicit paths and `Proposal draft: <change>` as the message. Once the workflow starts, do not add such a commit, amend, rebase or squash.

## Implementation boundaries

- Follow the current proposal and latest authorized user intent. Check every success criterion against real assertions rather than exploiting missing acceptance coverage.
- Weak contracts include superficial checks on configuration, scripts, files, HTTP status, elements, substrings or exit codes when the target is behavior, semantics or runtime guarantees. Static fixtures and mocked results cannot substitute for a real business flow.
- Review relevant historical tests in `tests/specs/` and root `tests/`. Organize specification tests by capability. If the new intent invalidates old expectations, update them and explain the conflict in design or delivery notes.
- Do not delete, skip, weaken or bypass creation-stage tests or `acceptance.json` to make implementation pass.
- A requirement to track temporary screenshots, traces or runtime logs under `test-results/` is a contract error. Correct the mapping consistently; final evidence becomes trackable only under `tests/evidence/proposals/<change>/`. Do not force-add ignored output.
- When the user changes acceptance criteria, synchronize `spec.md`, `design.md`, `acceptance.json` and tests, recording the reason before implementation continues.
- Add real project tests as needed. Contract additions belong under `docs/changes/<change>/tests/`; regressions and end-to-end tests may use the existing root test layout. Update the acceptance mapping.
- Do not mock APIs or databases, fabricate authentication, hardcode successful results or rely only on HTTP 200 unless the user explicitly asks for this and the proposal records the limits.
- Do not add placeholder documents to `tests/`. Do not create or modify `task.md` or another tracked dynamic task list; use Todo or runtime state.
- Execution, audit, repair and tests do not commit. Archiving creates one complete delivery commit after `delivery_base_head`.
- Find the declared hook entry (project script, pre-commit, lefthook or husky). Run it before QA, apply edits, and run it again without changes. If it continues changing files, stay in the current stage and diagnose it.
- Keep `test-results/` temporary. Stable expected snapshots may be tracked according to project conventions; actual run results must not masquerade as expected baselines.
- Exercise every `delivery_report.scenarios` user path and collect understandable evidence. Web evidence must show actions and page results; CLI evidence must show real input, output and business state changes.
- Videos, screenshots and traces must be real readable files. Echo/printf strings, exit codes, HTTP 200, element presence, passing-test messages and hashes cannot replace user evidence.
- Before/after fix evidence must use the same entry point, role, data and scenario and clearly show the failed behavior becoming usable.

## Long-running demos

Run long demos such as `paper-orchestra-production-live` in the foreground, with stdout/stderr visible to the terminal or workflow runner. Do not use `nohup ... &`, `setsid ... &`, shell background jobs or redirected logs followed by sleeping and polling. Do not silence the pi node with `sleep N && tail -f log`.

Wait synchronously and preserve progress output as go-dag heartbeats. If a stage could produce no output for more than five minutes, make the demo itself emit periodic progress so the node does not appear stalled.

## Exit criteria

The in-scope implementation is complete and runtime planning has converged. Creation-stage contracts pass; relevant regressions, end-to-end and package tests have run with commands and results recorded. No tests or acceptance boundaries were bypassed. Any changed historical expectation has an explained conflict with new intent.

Hooks are idempotent before final QA and subsequent required tests and validation pass. Every proposal success criterion has meaningful acceptance coverage, or an explicit record explains an exclusion and who accepts its risk. Behavioral changes must include the corresponding implementation or architecture; superficial configuration-only compliance is incomplete.

Every declared user scenario has been demonstrated with real readable evidence. Report implementation, test changes, historical-test updates, commands and remaining risks. Do not treat passing old regressions as proof of new behavior or use acceptance omissions as permission to ignore the proposal.
