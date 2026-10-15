## Execution task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read `state.json` and `acceptance.json`, relative to the run directory.

Identify the current specrelay change from `state.json.change_name` and stay within its proposal scope.

Invoke the specrelay-exec skill for the current specrelay change: `{{.ChangePath}}/`.

Before completing execution, run a demo that covers the target capability. Produce only the temporary `test-results/**` evidence declared by `submission_evidence.source_path`; the post-stage gate seals immutable snapshots for the current run. Do not write directly to `tests/evidence/proposals/<change>/**` or create a git commit. Only the engine promotes final committed evidence before archiving.
