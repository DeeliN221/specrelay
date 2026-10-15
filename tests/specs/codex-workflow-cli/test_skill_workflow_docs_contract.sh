#!/usr/bin/env bash
# Verify documented workflow responsibilities, skill completion gates and hook discipline.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
export ROOT
python3 - <<'PY'
from pathlib import Path
import os
root=Path(os.environ['ROOT'])
def require(path,*terms):
 text=(root/path).read_text().lower()
 for term in terms:
  assert term.lower() in text, f'{path}: missing contract {term!r}'
require('README.md','Skills define how each role works','change directory stores the proposal and tests','acceptance.json','executable contract and evidence','docs/changes/','specrelay flow','when stages advance through validation gates','pre-commit','again')
for skill in ['specrelay-plan','specrelay-create','specrelay-exec','specrelay-archive']:
 require(f'skills/{skill}/SKILL.md','## Process','## Exit criteria')
require('skills/specrelay-plan/SKILL.md','## Completeness check','scope, non-goals and tests')
require('skills/specrelay-create/SKILL.md','superficial implementation could still pass','Do not defer tests until execution')
require('skills/specrelay-exec/SKILL.md','superficial','pre-commit hook','rerun until no files change','After hooks stabilize')
require('skills/specrelay-archive/SKILL.md','## Completeness check','Do not first run a file-changing pre-commit hook after final QA')
print('PASS: workflow responsibilities, completion checks and hook stability')
PY
