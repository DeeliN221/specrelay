# SpecRelay CLI specification

This document defines the default English command contract. The [Chinese specification](../zh-CN/specs/specrelay-go-cli.md) retains the earlier wording; the old requirement for a Chinese character in change names is superseded by Unicode-compatible English defaults.

## Commands

The CLI MUST expose `list | l [--json]`, `install | i [--global | -g]`, and `language [en|zh-CN] [--global] [--json]` as daily commands. Automation retains `create`, `status <change> [--json]`, `validate <change> [--json]`, and `archive <change> --yes`. `flow` exposes the workflow runner. Removed top-level `init`, `plan`, and `exec` commands MUST fail.

`specrelay create` MUST return the next numeric proposal identifier without creating a directory or files. The maximum number is computed across active and dated archived proposals. `specrelay create` MUST NOT accept a proposal-name argument.

Default help, statuses, diagnostics, generated delivery reports, and bundled instructions MUST be English. JSON keys and machine status values MUST remain stable. Change names MUST accept English and Unicode letters, digits, and hyphens while rejecting path separators, spaces, and a leading hyphen. Numbered active changes use `<positive-number>-<description>`.

### Language

English is the default user-visible language; `zh-CN` is the other supported language. `specrelay language` with no value MUST print the effective language. With a value it MUST persist `language` in the repository `specrelay-flow.yaml` (or `~/specrelay-flow.yaml` with `--global`), creating the configuration file when absent. Language resolution order is repository configuration, user-level configuration, then English.

The effective language MUST select the bundled `SKILL.md` variant installed by `specrelay install`, the default flow prompts written by `specrelay flow config`, and CLI help, status, and error output. Switching language MUST refresh skill directories that are already installed for the project and user. User-customized prompts MUST be preserved; untouched built-in prompt defaults MUST follow the selected language. Machine-readable JSON keys and status values MUST remain stable English identifiers.

## Change sizes

| Size | Eligibility | Required artifacts |
| --- | --- | --- |
| micro | Implementation-only change without observable behavior, command, state, or lasting-specification changes | TDD and a git commit; no proposal directory |
| small | One business intent, no complex design disagreement, at most two scenarios or required tests | `brief.md`, `acceptance.json`, `tests/` |
| standard | Cross-module, high-risk, multi-scenario, or larger than small | Small artifacts plus `proposal.md`, `design.md`, `spec.md` |

Counts MUST NOT become quotas for inventing tests. Small changes still identify the long-term specification and test destination. Active proposals MUST NOT contain `task.md`; dynamic plans belong to the executor's Todo or runtime state.

## Validation

`specrelay validate --json` MUST return `valid`, `change`, `errors`, `warnings`, and `artifacts`; lifecycle diagnostics add stable machine codes where available. Standard specifications require a requirement heading, normative language, and a scenario. English forms are `### Requirement:`, MUST/SHALL/SHOULD, and `#### Scenario:`. Existing translated headings remain accepted.

The acceptance contract MUST be parsed strictly. `required_tests` needs at least one complete test and business-level assertion. IDs must be unique. Coverage must reference existing tests and evidence. Evidence producers must be traceable through the coverage-linked test command, purpose, assertions, test body, or adjacent shell wrapper. Shared validation resides in `internal/acceptance` and is reused by CLI validation, workflow preflight, acceptance execution, and QA matrix validation.

Superficial assertions such as HTTP 200, an element merely existing, a component rendering, or a page opening MUST NOT replace observable business behavior. Unknown fields, missing artifacts, unresolved IDs, non-test notes in `tests/`, or untraceable evidence MUST fail validation. Temporary `test-results/` output MUST remain ignored; contracts MUST NOT require it to be tracked with `git ls-files` or add ignore exceptions.

## Acceptance evidence

`run-acceptance` MUST retain its summary, per-test results, evidence, diagnostics, and producer/coverage relationships. Each run binds results to the run, stage, attempt, input diff, contract, and immutable evidence. Runtime sources stay under `test-results/`; the engine seals reviewable copies before final promotion.

A submission bundle MUST map each required evidence item from its declared source to a canonical path under `tests/evidence/proposals/<change>/`. Paths must stay inside the repository and current proposal. Files must be nonempty regular files, not symbolic links or ignored files. Files larger than 20 MiB require Git LFS tracking. The bundle MUST cover all required evidence and include a playable final demonstration; before/after material does not replace that demonstration.

Evidence quality MUST be checked by kind: videos must be plausible playable media, screenshots decodable images of useful dimensions, traces readable nonempty ZIP archives, structured evidence meaningful JSON, and text enough readable scenario/action/outcome information. Renamed text, empty JSON, command-only logs, fixed status strings, or identical before/after files MUST fail.

## Reviewer-facing delivery

`delivery_report` MUST identify user benefits and executable user scenarios with expected behavior and directly linked evidence. At least one scenario references the demonstration. Fix comparisons must describe distinct outcomes and bind distinct evidence files within the same scenario.

Final independent QA MUST record an actual observation for each delivery scenario. The generated `DELIVERY.md` contains User benefits, How to verify, Before and after, Reviewable evidence, and Known limitations. Existing translated reports remain readable. Claims consisting only of commands, exit codes, hashes, internal state fields, or a passing-test message are insufficient.

## Archive

`specrelay archive <change> --yes` requires explicit confirmation and a valid submission evidence bundle. It MUST move the active proposal, including its `tests/`, to the dated archive and rewrite proposal-relative test references in scripts, contracts, and documentation. It MUST reject an existing destination.

The CLI performs the mechanical archive; the archive skill owns consolidation of lasting specifications and tests. In a sealed workflow, those substantive changes MUST already be complete before final QA. Only mechanical movement, delivery summary creation, and committing the engine's final output may follow final QA.

Implementation, archived proposal, and final evidence MUST belong to exactly one complete delivery commit descended directly from the sealed delivery baseline. No amended or reused intermediate implementation commit may substitute for it. A successful archive requires the actual committed files, not merely a plausible report.

## Installation

`specrelay install` installs bundled skills for the project or, with `--global`/`-g`, to the user skill directory. It MUST NOT install optional external skills or silently depend on them. Generated defaults and documentation MUST describe the actual current command interface.

## Regression coverage

The Go tests exercise filesystem-backed validation, unsafe paths, Unicode names, English proposals, archive reference rewriting, submission evidence quality, delivery reports, and real CLI help. Language changes MUST preserve these behavior checks rather than remove them.
