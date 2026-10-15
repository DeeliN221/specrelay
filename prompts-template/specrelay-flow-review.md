## Review task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read `state.json` and `acceptance.json`, relative to the run directory. Current change: `{{.ChangePath}}/`; diff baseline: `{{.BaselineHead}}`.

Tasks:

- Review only the current proposal scope; do not modify source code.
- Put current proposal issues in `findings`. Put historical debt or unrelated issues in `non_blocking_findings` with scope `out_of_scope_existing`.
- Blocking scope must be `current_change` or `introduced_regression`. An unmet acceptance_contract must be reproducible.
- `required_evidence` must be independently reviewable; it does not require temporary `test-results/`, screenshots, traces or runtime logs to be tracked in git. Requiring temporary output to be tracked is a contract design error.
{{if .HasPreviousReview}}
Previous review: `{{.LatestPreviousReviewPath}}`; historical review count: {{.PreviousReviewCount}}.
{{if .HasPreviousFixSummary}}Fix summary: `{{.LatestPreviousFixSummaryPath}}`.
{{end}}{{end}}

Write `review-{{.Iteration}}.json` relative to the run directory.

Run in that directory: `specrelay flow validate-review --artifact "review-{{.Iteration}}.json" --json`.

{{if .IsFirstRoleTurn}}
Write exactly one strict JSON object.

Fields: `summary`, `decision` (0=clean, 1=needs_fix), `evidence[]`, `findings[]`, `checks`, `non_blocking_findings[]`, `workflow_failure`.

Each finding: `{title,severity,scope,evidence,recommendation}`. `severity`: 1=blocker, 2=major, 3=minor. `scope`: 1=current_change, 2=introduced_regression, 0=out_of_scope_existing. Each `evidence[]` entry must be a string citing a verifiable command, artifact, screenshot, trace, QA result, console or network observation.

For clean: `decision=0`, `findings=[]`, every `checks` value true, and nonempty `evidence` covering acceptance_contract. For needs_fix: `decision=1` and nonempty `findings`.
{{else}}
Continuation: reuse the role session `{{.RoleSessionKey}}` and rewrite `review-{{.Iteration}}.json` with the same schema. Review the previous review/fix and report only issues that remain.
{{end}}
