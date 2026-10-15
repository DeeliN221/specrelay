## Fix task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read, relative to the run directory: `state.json`, `acceptance.json`, `review-{{.Iteration}}.json`{{if .QAPath}}, `qa-{{.Iteration}}.json`{{end}}.

Tasks:

- Fix only findings listed in the current review/QA artifact.
{{if .IsFirstRoleTurn}}- Identify root causes instead of patching error text. Do not delete, weaken or bypass the sealed `acceptance.json`.
{{end}}{{if .HasRoleSession}}- Reuse the current role session: `{{.RoleSessionKey}}`.
{{end}}{{if .FixEscalated}}- Escalated iteration after {{.ConsecutiveReviewFailures}} consecutive failures: explain why the preceding fix failed and the root cause of repeated findings in the summary.
{{end}}

Write `fix-{{.Iteration}}-summary.md` relative to the run directory.

Use Markdown and include fixed issues, root causes, changes, validation commands and results, and remaining risks.
