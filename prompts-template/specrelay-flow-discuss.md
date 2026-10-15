## Planning task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read `state.json` and `acceptance.json`, relative to the run directory.

Invoke the `specrelay-plan` skill to discuss and plan the change.

Include demonstrable capabilities and committed evidence in the acceptance design. Explain how the demo covers the target capability. Map each mutable `source_path` to an `archive_path` under `tests/evidence/proposals/<change>/**` in `submission_evidence`. `test-results/**` contains temporary runtime output, not final delivery evidence.
