---
name: specrelay-plan
description: Plan the scope and acceptance criteria of an specrelay change proposal.
---

# specrelay Plan

Turn an initial idea into a well-defined change that is ready for proposal creation.

## Choose an entry type

| Type | Suitable change | Artifacts |
| --- | --- | --- |
| micro | An implementation-only fix that does not change observable behavior, command contracts, state semantics or long-term specifications | TDD and a git commit; no change directory |
| small | One business intent, at most two acceptance scenarios or two required tests, and no complex design disagreement | `brief.md`, `acceptance.json`, `tests/` |
| standard | Cross-module, high-risk or multi-scenario work, or work exceeding the small limits | Full proposal documents, `acceptance.json`, `tests/` |

Escalate to standard because of risk, scope or scenario complexity; do not invent tests or tasks to reach that category. Even a small proposal must identify its long-term specification destination in `docs/specs/` and `tests/specs/`.

## Optional external skills

The following separately installed Matt Pocock skills can support planning. They are not installed by `specrelay install`. Check availability and use this skill's own process when they are absent.

- `/tdd`: use directly for micro changes without a change directory.
- `/grill-with-docs` and `/domain-modeling`: clarify intent, terminology, acceptance criteria and tradeoffs for small/standard work; produce `CONTEXT.md` and ADRs when useful. Let the interview skill lead the questions, then map its conclusions to an specrelay entry type and scope.
- `/wayfinder`: for a change too large or uncertain for a single session, map decisions in the issue tracker first. Once the map is clear, use `/to-spec` for `proposal.md`/`spec.md` and decide how to split specrelay proposals.
- `/domain-modeling`: resolve ambiguous or conflicting vocabulary and reuse its terminology in the final scope.

Invoke these skills by their slash commands. Their own `SKILL.md` files define their procedures; do not duplicate those procedures here.

## Process

1. Establish the user's problem and current difficulty before writing files.
2. Identify which behavior, documentation, commands, tests or data will change.
3. State non-goals so execution does not absorb adjacent work.
4. Explain the real test coverage needed to establish the user's intent. Test count is a classification signal, not a target to inflate.
5. Present a proposed name for the user to review.

Discuss whether the requested change is necessary and feasible. Challenge mistaken premises and unrealistic scope. Identify affected historical tests and whether the new intent requires updating them. Use ASCII diagrams and trees when they make the proposal easier to understand.

## Exit criteria

Provide a proposal name, problem, scope, non-goals, real tests to implement under `docs/changes/<change>/tests/`, and unresolved questions.

Names should communicate the action and purpose, using English by default. Unicode names remain supported. Before creation, resolve questions that cannot reasonably be assumed and would change scope, acceptance criteria, data sources or permission boundaries.

## Completeness check

Even a simple change needs scope, non-goals and tests. Define real entry points, assertions and likely evidence during planning. The name becomes a directory and audit entry, so choose it deliberately rather than using a placeholder.
