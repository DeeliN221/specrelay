## QA task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read `state.json`, `acceptance.json`, {{if .HasRepairCheckpoint}}the latest repair checkpoint, {{else}}no repair checkpoint is configured for this iteration, {{end}}and the current change `{{.ChangePath}}/`. Diff baseline: `{{.BaselineHead}}`.

Tasks:

- Validate only the current proposal scope. Do not modify source code or the sealed `acceptance.json`.
- Use an independent QA session to assess {{if .HasRepairCheckpoint}}the latest repair checkpoint{{else}}execution results (zero-repair mode has no checkpoint){{end}}. Do not inherit the repairer's self-assessment.
- When rejecting a result, include reproducible failure evidence in both findings and acceptance_matrix; the next stage performs targeted repairs from these records.
- Verify the demo covers the target capability and inspect the immutable required_evidence snapshots sealed within the current run. Remain read-only: do not modify temporary `test-results/**` sources or write `tests/evidence/proposals/<change>/**`. {{if .HasRepairCheckpoint}}For repaired findings, verify reproducible before-failure and after-success evidence for the same scenario.{{end}}
- Copy `acceptance_matrix[].id` exactly from required_tests/required_evidence in the sealed `acceptance.json`, covering acceptance_contract.
- Put current proposal issues in `findings`; put historical debt or unrelated issues in `non_blocking_findings` with scope `out_of_scope_existing`.
- Blocking scope must be `current_change` or `introduced_regression`. Do not read mutable `source_path` to detect drift or treat a changed `test-results/**` hash as a source regression.

Write `qa-{{.Iteration}}.json` relative to the run directory.

Run there: `specrelay flow validate-qa --artifact "qa-{{.Iteration}}.json" --acceptance "acceptance.json" --json`.

{{if .IsFirstRoleTurn}}
Write exactly one JSON object.

Fields: `summary`, `decision` (0=clean, 1=needs_fix), `evidence[]`, `acceptance_matrix[]`, `findings[]`, `non_blocking_findings[]`.

Each `acceptance_matrix[]` entry: `{id,status,artifact,evidence}`, where `status`: 0=passed, 1=failed. Each finding: `{title,severity,scope,evidence,recommendation}`, where `severity`: 1=blocker, 2=major, 3=minor and `scope`: 1=current_change, 2=introduced_regression, 0=out_of_scope_existing. Each `evidence[]` entry must be a string citing reproducible tests, screenshots, traces, console, network or runtime evidence.

For clean: `decision=0`, `findings=[]`, nonempty `evidence`, and an `acceptance_matrix` covering required_tests/required_evidence. For needs_fix: `decision=1` and nonempty `findings`.
{{else}}
Continuation: reuse role session `{{.RoleSessionKey}}` and rewrite `qa-{{.Iteration}}.json` with the same schema.
{{end}}
