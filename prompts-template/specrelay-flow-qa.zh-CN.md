## 独立验收任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取 `state.json`、`acceptance.json`，{{if .HasRepairCheckpoint}}最新的修复检查点，{{else}}本轮未配置修复检查点，{{end}}以及当前变更 `{{.ChangePath}}/`。Diff 基线：`{{.BaselineHead}}`。

任务：

- 仅验证当前提案范围。不要修改源码或已封存的 `acceptance.json`。
- 使用独立验收会话评估{{if .HasRepairCheckpoint}}最新的修复检查点{{else}}执行结果（零修复模式没有检查点）{{end}}。不要继承修复者的自评。
- 拒绝结果时，在 findings 和 acceptance_matrix 中都包含可复现的失败证据；下一阶段将依据这些记录进行定向修复。
- 验证演示是否覆盖目标能力，并检查本次运行中封存的不可变 required_evidence 快照。保持只读：不要修改临时 `test-results/**` 来源，也不要写入 `tests/evidence/proposals/<change>/**`。{{if .HasRepairCheckpoint}}对于已修复的问题项，验证同一场景下可复现的修复前失败与修复后通过证据。{{end}}
- 从已封存 `acceptance.json` 的 required_tests/required_evidence 中原样复制 `acceptance_matrix[].id`，并覆盖 acceptance_contract。
- 将当前提案的问题放入 `findings`；将历史债务或无关问题放入 `non_blocking_findings`，scope 为 `out_of_scope_existing`。
- 阻塞范围必须为 `current_change` 或 `introduced_regression`。不要读取可变的 `source_path` 来检测漂移，也不要把 `test-results/**` 哈希变化视为源码回归。

在运行目录下写入 `qa-{{.Iteration}}.json`。

在该目录下运行：`specrelay flow validate-qa --artifact "qa-{{.Iteration}}.json" --acceptance "acceptance.json" --json`。

{{if .IsFirstRoleTurn}}
只写一个 JSON 对象。

字段：`summary`、`decision`（0=clean，1=needs_fix）、`evidence[]`、`acceptance_matrix[]`、`findings[]`、`non_blocking_findings[]`。

每个 `acceptance_matrix[]` 条目：`{id,status,artifact,evidence}`，其中 `status`：0=passed，1=failed。每个问题项：`{title,severity,scope,evidence,recommendation}`，其中 `severity`：1=blocker，2=major，3=minor，`scope`：1=current_change，2=introduced_regression，0=out_of_scope_existing。每个 `evidence[]` 条目都必须是字符串，引用可复现的测试、截图、trace、控制台、网络或运行时证据。

对于 clean：`decision=0`、`findings=[]`、`evidence` 非空，且 `acceptance_matrix` 覆盖 required_tests/required_evidence。对于 needs_fix：`decision=1` 且 `findings` 非空。
{{else}}
续接：复用角色会话 `{{.RoleSessionKey}}`，按相同 schema 重写 `qa-{{.Iteration}}.json`。
{{end}}
