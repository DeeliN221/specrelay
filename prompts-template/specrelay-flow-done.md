## Archive task
Stage: `{{.Stage}}` (iteration `{{.Iteration}}`)
Run directory: `{{.RunDirectory}}`
Read `state.json`, `acceptance.json` and historical audit/repair/QA artifacts, relative to the run directory.

Actions:

- Invoke the `specrelay-archive` skill; use the change name from `state.json.change_name`.
- This is a read-only boundary after final QA. Only mechanical proposal moves are allowed; long-term specifications and specification tests must already be complete before final QA.
- Before entering this stage, the engine promotes evidence from the final passing immutable snapshots. Verify that the final committed evidence is under `tests/evidence/proposals/<change>/**`, is not ignored by git, and is included in the archive commit. Do not copy or rebuild it from `test-results/**`.
- Do not first trigger a file-changing pre-commit hook after final QA. If a repeated hook check still changes files, preserve those changes and let the gate return the run to audit and independent QA.
- Starting from the sealed `delivery_base_head` in state, create exactly one complete delivery commit containing the implementation, archived proposal and final evidence at HEAD. Do not amend, squash or reuse execution/audit commits.
- After the archive command returns, do not edit, format or restore proposal directories or source files. You may stage and commit the command results unchanged; do not rewrite files merely to obtain a clean worktree.
- Preserve any remaining differences after archiving for the read-only gate to assess.

Write `delivery-summary.md` relative to the run directory.
Include a `Final review` section.
