---
name: specrelay-archive
description: Validate and archive a completed specrelay proposal, preserving its verified behavior in long-term specifications and specification tests.
---

# specrelay Archive

Preserve the verified change intent in maintained specifications and tests, as well as archiving its proposal directory.

## Entry types

For a small proposal, read `brief.md`, `acceptance.json` and `tests/`; derive the lasting behavior from the brief. For standard work, read the complete proposal, acceptance contract and tests; derive lasting behavior from `spec.md`. Both must merge appropriate behavior into `docs/specs/` and `tests/specs/`.

## Optional external skills

Separately installed Matt Pocock skills are not installed by `specrelay install`. Check their availability and fall back to this process if unavailable.

- `/writing-for-agents` can review the information hierarchy, context references and completion criteria of `DELIVERY.md`. It must not rewrite the engine's sealed observations or evidence paths.
- `/domain-modeling` can help review terminology that should enter the maintained glossary. It must not trigger new edits after final QA.

Invoke slash commands rather than duplicating their procedures. The archive stage is a read-only boundary after final QA; external skills may review the results and suggest future improvements but cannot rewrite source, proposals or evidence. Finish long-term specification and test changes before final QA.

## Process

- Check the worktree and identify all proposal-related changes; exclude unrelated files.
- Run `specrelay validate <change> --json`.
- Do not create or edit `task.md`. Completion depends on validation, acceptance results and final review.
- Confirm the final passing test iteration and relevant results. `test-results/` is temporary output only.
- In an SpecRelay workflow, the engine creates `DELIVERY.md` from final QA observations and the evidence package from immutable snapshots. Review them without rewriting them.
- For standalone archiving, collect the final evidence with `DELIVERY.md`, `README.md` and `manifest.json`.
- Write delivery reports for reviewers: user benefit, preparation, actions, expected result, actual observation, direct evidence and limitations. For fixes include the before/after difference.
- Commands, exit codes, HTTP 200, element presence, passing-test messages, hashes and hardcoded strings may supplement evidence but cannot replace understandable screenshots, videos, business logs or real input/output.
- Run `specrelay archive <change> --yes`. The CLI moves the proposal directory including `tests/` to `docs/changes/archive/<number>-<change-name>` without a date prefix and rewrites proposal-test paths in scripts, `acceptance.json`, `brief.md`, `spec.md` and other text. It does not logically merge tests into long-term `tests/specs/`.
- Understand each archived test's business contract and assertions. Organize maintained tests by capability, not proposal number; do not merely copy whole proposal test directories.
- Source comments may cite proposals (for example `// Sources: 1-login, 3-access-control`), while filenames describe capabilities.
- Merge lasting behavior from the archived specification into `docs/specs/*.md`; for small changes use the brief. Run affected specification tests after updating paths. These edits must be finished before the final workflow QA boundary.
- Stage explicit paths only. Do not use `git add .`, `git add -A` or `git add -f`. Keep `test-results/` ignored; make `tests/evidence/proposals/<change>/` trackable with only the narrowest necessary ignore exception.
- Inspect `git diff --cached --name-only` and run `git check-ignore --no-index -- <path>` for every candidate. Remove inadvertently staged ignored output and confirm final evidence is present.
- Use Git LFS for any evidence file over 20 MiB; confirm `filter=lfs` using `git check-attr filter -- <path>`.
- Starting from sealed `delivery_base_head`, create exactly one complete delivery commit. Do not amend, rebase, squash or rewrite earlier history. The archived proposal, code, tests, long-term specifications and final evidence must share HEAD; `delivery_base_head..HEAD` must contain exactly one commit.
- Use commit message `<number>: <change-name>`. Report archive paths, logically merged tests and specifications, commands run and remaining risks.

After the archive command returns, do not edit, format or restore files to force a clean worktree. Preserve unexpected differences for the gate to return to audit and independent QA. Do not first run a file-changing pre-commit hook after final QA.

## Exit criteria

Validation, acceptance tests and final review have passed. The proposal is under `docs/changes/archive/`. Its lasting behavior and tests are merged by capability, or a specific reason explains why no merge is needed. Affected tests have run.

The final evidence directory contains `DELIVERY.md`, real playable demo video, a review entry point and necessary attachments, committed with the code. Each fix has distinct, readable before/after evidence for the same entry point, role, data and environment, linked directly from the delivery report.

The delivery range contains one commit and the relevant worktree is clean. Review every staged path to exclude temporary output, failed iterations, caches, sensitive data and unrelated artifacts.

## Completeness check

The CLI move alone does not complete archiving. Read historical assertions before merging them. Do not bundle unrelated work or commit all of `test-results/`; use only final passing evidence promoted into the proposal evidence directory.
