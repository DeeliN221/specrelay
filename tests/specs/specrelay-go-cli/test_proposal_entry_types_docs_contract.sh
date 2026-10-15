#!/usr/bin/env bash
# Verify proposal size eligibility, artifacts, escalation and maintained-spec obligations.
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
export ROOT
python3 - <<'PY'
from pathlib import Path
import os
root=Path(os.environ['ROOT'])
def require(path,*terms):
 text=(root/path).read_text().lower()
 for term in terms: assert term.lower() in text, f'{path}: missing {term!r}'
 return text
for path in ['README.md','docs/specs/specrelay-go-cli.md']:
 require(path,'micro','small','standard','brief.md','acceptance.json','tests/','proposal.md','design.md','spec.md','two','long-term','TDD','commit','invent')
require('skills/specrelay-plan/SKILL.md','micro','small','standard','at most two acceptance scenarios or two required tests','Escalate to standard','long-term specification')
require('skills/specrelay-create/SKILL.md','brief.md','acceptance.json','tests/','standard','proposal.md','design.md','spec.md','do not invent tasks or tests')
require('skills/specrelay-exec/SKILL.md','For small work','brief.md','acceptance.json','tests/')
require('skills/specrelay-archive/SKILL.md','For a small proposal','brief.md','docs/specs/','tests/specs/')
text=require('skills/specrelay-create/SKILL.md','Standard proposals must also','always required')
assert 'all changes require proposal.md, design.md and spec.md' not in text
print('PASS: micro/small/standard artifacts and responsibility boundaries')
PY
