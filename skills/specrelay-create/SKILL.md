---
name: specrelay-create
description: Create a small brief-only or standard specrelay proposal with an executable acceptance contract and real tests.
---

# specrelay Create

Create an specrelay proposal and its delivery contract. `brief.md`, `acceptance.json` and `tests/` are always required. Standard proposals must also keep `proposal.md`, `design.md` and `spec.md` aligned with that contract.

## Entry types

| Type | Scope | Required artifacts |
| --- | --- | --- |
| small | One business intent, at most two acceptance scenarios or two required tests, without complex design disagreements | `brief.md`, `acceptance.json`, `tests/` |
| standard | Cross-module, high-risk, multi-scenario work or work beyond the small limits | `brief.md`, `proposal.md`, `design.md`, `spec.md`, `acceptance.json`, `tests/` |

Micro changes use TDD and a git commit without `specrelay create`. Escalate to standard because of risk, scope or complexity; do not invent tasks or tests to justify the category.

## Optional external skills

Separately installed Matt Pocock skills are not installed by `specrelay install`. Check availability and fall back to this skill if absent.

- `/to-spec`: synthesize a sufficiently developed discussion into standard `spec.md` or small `brief.md` behavior, then map it to specrelay's documents and acceptance structure.
- `/to-tickets`: split standard work into tracer-bullet tickets and blocking edges, mapping each ticket to acceptance tests/evidence.
- `/prototype`: use a disposable prototype to resolve state-model or UI uncertainty. Record conclusions in design/spec; do not treat prototype code as delivery code. Its branch can be design-preview evidence.
- `/domain-modeling`: establish consistent terms before writing brief/spec.

These skills do not replace `acceptance.json` or real tests. Every generated behavior and ticket needs coverage. Invoke slash commands rather than duplicating their procedures.

## Process

1. Run `specrelay create` and use only its integer output as `<number>`.
2. Create `docs/changes/<number>-<change-name>/` from the agreed plan.
3. First map small brief acceptance items, or standard specification scenarios, to required tests and evidence.
4. Write the required artifacts for the chosen entry type.
5. Run `specrelay validate <change> --json` and check that documents, tests and JSON agree.
6. Commit the proposal before execution so it cannot be lost or mixed with unrelated changes.

## Artifact contract

- `brief.md`: explain the user problem, delivery goals, non-goals, acceptance entry points and default execution context without requiring a long document to understand the change.
- `proposal.md`: required for standard work; state what and why.
- `design.md`: required for standard work; record decisions, tradeoffs and risks.
- `spec.md`: required for standard work; use `### Requirement:` and `#### Scenario:` headings. Every scenario identifies its real test file, data source, entry point, assertions and remaining risk. Existing translated headings remain compatible.
- `acceptance.json`: cover every requirement/scenario, reuse contract tests, and include necessary root end-to-end/regression tests and screenshot, trace, network, console or runtime evidence.
- `tests/`: real project test code expressing the core contract. No empty directory, placeholder file or test-description-only document. Write understandable English comments by default.

Do not create or edit `task.md`. Creation defines intent, scope, scenarios and acceptance rather than fixed implementation steps. Dynamic implementation plans belong in Todo or runtime state, not tracked files.

Creation defines the contract; execution makes it pass. Do not defer core behavior standards to the executor or promise more than tests can demonstrate.

## Coverage requirements

- Map each scenario to at least one required test; key user paths also require evidence. Build the matrix before writing the prose.
- Unsupported promises belong under risks, non-goals or open questions until they can be verified.
- Cover observable behavior: rendered results, CLI output, API contracts, state changes, persistence or permission boundaries.
- Use real business examples, entry points and call chains. Web acceptance must include relevant login, permissions, navigation, APIs and database behavior.
- Do not mock APIs or databases, fabricate authentication, hardcode success or rely only on HTTP 200, element existence or shallow component rendering unless the user explicitly requests this and design records the limits.
- A repository fixture must explain the real business example and field semantics it represents; it cannot silently masquerade as production data.
- Resolve missing accounts, example data, external-service access or executable environments with the user rather than asking execution to invent them.
- Even without an existing test framework, provide a minimal runnable test entry. Confirm a new framework is the lowest-cost option before introducing it.
- Test assertions must check business output, stored records, API fields, authorization rejection, persisted state, audit results or observable user flows—not merely that a test starts.
- `test-results/` contains regenerable runtime output and stays untracked. Stable expected snapshots may follow project conventions; actual output is not a baseline.
- Map every `required_evidence` entry through `submission_evidence`; at least one uses `kind=demo_video`. Use any genuinely playable format supported by the environment. Videos, screenshots and traces must be real files, never renamed command output, exit codes or hardcoded strings.
- `delivery_report` explains user benefits, actions, expected results and direct evidence for reviewers. Fix scenarios need understandable, distinct before/after evidence for the same scenario.
- Before completion, ask whether a superficial implementation could still pass. If so, strengthen assertions or narrow the promised behavior.

## Acceptance format

Use strict JSON. For example:

```json
{
  "summary": "Verify the main user flow and persisted result",
  "coverage": [{
    "spec": "Requirement: Example capability / Scenario: Main user flow",
    "tests": ["contract-main-flow"],
    "evidence": ["main-flow-demo"],
    "risk": "External-provider outage recovery is outside this scenario"
  }],
  "required_tests": [{
    "id": "contract-main-flow",
    "source": "change_contract",
    "path": "docs/changes/12-example/tests/main-flow.acceptance.test.ts",
    "command": "pnpm exec tsx --test docs/changes/12-example/tests/main-flow.acceptance.test.ts",
    "purpose": "Verify the main behavior promised by proposal/spec",
    "assertions": [
      "Create a business record with required fields through the real entry point",
      "Read the persisted record after refresh or a new query",
      "Reject requests lacking permission with an explicit denial"
    ],
    "expected_initial_failure": "The target behavior is absent before implementation"
  }],
  "required_evidence": [{
    "id": "main-flow-demo", "kind": "demo_video",
    "path": "test-results/main-flow/demo.mp4",
    "purpose": "Demonstrate creation, refresh and restored state"
  }],
  "submission_evidence": [{
    "evidence_id": "main-flow-demo",
    "source_path": "test-results/main-flow/demo.mp4",
    "archive_path": "tests/evidence/proposals/12-example/main-flow-demo.mp4"
  }],
  "delivery_report": {
    "user_benefits": ["Users can complete the operation and see the persisted result."],
    "prerequisites": ["A normal user account with access to this feature."],
    "scenarios": [{
      "id": "main-user-flow", "title": "Complete the main user flow",
      "user_value": "Obtain a usable result without additional manual processing.",
      "steps": [{
        "action": "Create a record through the real entry point, then refresh.",
        "expected": "The record and its required fields remain visible."
      }],
      "evidence_ids": ["main-flow-demo"]
    }],
    "known_limits": ["The demonstration uses a test account and sanitized business data."]
  }
}
```

The `.mp4` extension illustrates the mapping, not a mandatory format. Match actual extensions across path, source_path and archive_path.

`coverage[].spec` references real requirements/scenarios. Its tests and evidence reference declared IDs; if evidence is absent, use an empty array and explain the risk. Test sources are `change_contract`, `root_e2e`, `existing_regression` or `new_regression`, with at least one business assertion per test. Evidence kinds are `screenshot`, `trace`, `network`, `console`, `runtime_log`, `state_snapshot`, `demo_video` or `other`.

For a fix, include this object in the same delivery scenario:

```json
"comparison": {
  "before": "The actual user-visible failure before the fix",
  "after": "The usable result after the fix",
  "before_evidence_id": "before-proof",
  "after_evidence_id": "after-proof"
}
```

Include both IDs in that scenario's `evidence_ids`, referring to readable files with different content. Describe the shared entry point, role, data and environment in prerequisites or steps.

`required_evidence` describes inspectable temporary evidence, `submission_evidence` defines final commit paths, and `delivery_report` drives `DELIVERY.md`. Passing-test output may supplement user evidence, not replace it.

Clarify unresolved test strategy, real data, observable assertions or QA evidence before creating a proposal. Execution may add tests but cannot delete, weaken or bypass the original contract. Only changed user intent permits synchronized contract/spec/design/test changes with a recorded reason.

## Exit criteria

Use `docs/changes/<number>-<change-name>/` without a date prefix. Obtain the number from `specrelay create`, not by enumerating a large proposal/archive directory. Use an understandable action-oriented English name by default; Unicode names are supported.

Validation passes, the brief supports execution, and tests contain real code without placeholders. Manually verify JSON mappings, reviewer benefits and steps, observable results and real demo evidence. Immediately commit the proposal artifacts with `<number> proposal: <change-name>`.

Do not defer tests until execution, promise uncovered behavior, commit temporary output or fabricate missing real data.
