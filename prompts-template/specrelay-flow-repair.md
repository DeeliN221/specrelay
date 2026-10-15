## Repair task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read `state.json` and `acceptance.json`, relative to the run directory. Current change: `{{.ChangePath}}/`; diff baseline: `{{.BaselineHead}}`.

Tasks:

- Work only within the current proposal scope: inspect implementation, fix discovered issues and run deterministic validation.
- `pre_qa_audit` (`audit_N`): inspect acceptance, the complete diff, source and tests. Run the demo and verify target capability coverage. Inspect the preceding immutable checkpoint evidence when present. Produce only temporary `test-results/**` sources for the post-stage gate to seal; do not write the committed evidence under `tests/evidence/proposals/<change>/**`. Continue self-review after repairs; hand off to QA only with no new issues and all required tests passing.
- `qa_targeted_repair` (`targeted_repair_N`): fix only the latest QA findings, failed acceptance items and direct regressions. Do not expand scope. For each finding produce reproducible before-failure and after-success evidence for the same scenario. Update temporary sources according to the sealed `submission_evidence` mappings for the gate to snapshot.
- Before every independent QA handoff, run the repository's explicit pre-commit hook entry without creating a temporary commit. Apply its changes and rerun until it changes no files. After hooks stabilize, rerun affected tests, all required tests and validation commands.
- Recheck inputs and validation every iteration. Before handoff rerun failed tests, all required tests and validation commands. Do not declare `clean` if checks fail or their results do not correspond to the current diff.
- Do not create git commits during execution, audit or targeted repair. Only archiving creates the complete delivery commit.
- Put current proposal issues in `findings`; put historical debt in `non_blocking_findings`.
{{if .HasPreviousRepair}}
- Previous checkpoint: `{{.LatestPreviousRepairFile}}`.
{{end}}
{{if .HasPreviousQA}}
- Independent QA that triggered this iteration: `{{.LatestPreviousQAFile}}`. Recheck and address every finding.
{{end}}
{{if .IsRepairConfirmation}}
- This is the mandatory confirmation audit after the preceding `clean`. Start again from state, acceptance, the complete diff and validation results; do not repeat the previous conclusion without rechecking.
- If any issue is found, repair it and use `needs_more`. Use `clean` again only if this iteration finds no new issues, requires no further changes and still passes validation.
{{end}}

{{if not .RepairMode}}Write `repair-{{.Iteration}}.json` relative to the run directory.

Run there: `specrelay flow validate-repair --artifact "repair-{{.Iteration}}.json" --json`.
{{end}}

Strict JSON fields: `summary`, `decision`, `evidence[]`, `findings[]`, `checks`, `non_blocking_findings[]`.
`decision` must be `clean` or `needs_more`. Use `needs_more` after finding and fixing issues when another confirmation iteration is required; use `clean` only when no known in-scope issues remain.
Each finding: `{title,severity,scope,evidence,recommendation}`.

Continue the same role in backend-scoped session `{{.RoleSessionKey}}`. The repairer cannot archive; independent QA approval is still required after clean.
