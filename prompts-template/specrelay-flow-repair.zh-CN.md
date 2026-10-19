## 修复任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取运行目录下的 `state.json` 和 `acceptance.json`。当前变更：`{{.ChangePath}}/`；diff 基线：`{{.BaselineHead}}`。

任务：

- 仅在当前提案范围内工作：检查实现、修复发现的问题并运行确定性验证。
- `pre_qa_audit`（`audit_N`）：检查验收、完整 diff、源码与测试。运行演示并验证目标能力覆盖情况。如存在上一轮不可变检查点证据，予以检查。仅生成临时的 `test-results/**` 来源，供阶段后置门禁封存；不要写入 `tests/evidence/proposals/<change>/**` 下的已提交证据。修复后继续自审；只有在没有新问题且所有必需测试通过后，才移交独立验收。
- `qa_targeted_repair`（`targeted_repair_N`）：只修复最新的独立验收问题项、失败的验收项和直接回归。不要扩大范围。对每个问题项，针对同一场景生成可复现的修复前失败与修复后通过证据。按照已封存的 `submission_evidence` 映射更新临时来源，供门禁生成快照。
- 每次移交独立验收前，在不创建临时提交的情况下运行仓库显式的 pre-commit hook 入口。应用其变更并重新运行，直到 hook 不再改动任何文件。hook 稳定后，重新运行受影响的测试、所有必需测试和验证命令。
- 每轮迭代都重新核对输入与验证结果。移交前重新运行失败的测试、所有必需测试和验证命令。如果检查失败或结果与当前 diff 不对应，不得声明 `clean`。
- 在执行、审计或定向修复期间不要创建 git 提交。只有归档才会创建完整的交付提交。
- 将当前提案的问题放入 `findings`；将历史债务放入 `non_blocking_findings`。
{{if .HasPreviousRepair}}
- 上一检查点：`{{.LatestPreviousRepairFile}}`。
{{end}}
{{if .HasPreviousQA}}
- 触发本轮的独立验收：`{{.LatestPreviousQAFile}}`。重新检查并处理每个问题项。
{{end}}
{{if .IsRepairConfirmation}}
- 这是上一轮 `clean` 之后的强制确认审计。从状态、验收、完整 diff 和验证结果重新开始；不要未经重新检查就重复上一结论。
- 如发现问题，修复后使用 `needs_more`。只有本轮未发现新问题、无需进一步变更且仍通过验证时，才再次使用 `clean`。
{{end}}

{{if not .RepairMode}}在运行目录下写入 `repair-{{.Iteration}}.json`。

在该目录下运行：`specrelay flow validate-repair --artifact "repair-{{.Iteration}}.json" --json`。
{{end}}

严格的 JSON 字段：`summary`、`decision`、`evidence[]`、`findings[]`、`checks`、`non_blocking_findings[]`。
`decision` 必须为 `clean` 或 `needs_more`。发现问题并修复后，如需再进行一轮确认迭代，使用 `needs_more`；仅当范围内已无已知问题时使用 `clean`。
每个问题项：`{title,severity,scope,evidence,recommendation}`。

在后端作用域会话 `{{.RoleSessionKey}}` 中继续同一角色。修复者不能归档；`clean` 之后仍需独立验收批准。
