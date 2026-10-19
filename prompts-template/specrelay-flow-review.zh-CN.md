## 复核任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取运行目录下的 `state.json` 和 `acceptance.json`。当前变更：`{{.ChangePath}}/`；diff 基线：`{{.BaselineHead}}`。

任务：

- 仅复核当前提案范围；不要修改源码。
- 将当前提案的问题放入 `findings`。将历史债务或无关问题放入 `non_blocking_findings`，scope 为 `out_of_scope_existing`。
- 阻塞范围必须为 `current_change` 或 `introduced_regression`。未满足的 acceptance_contract 必须可复现。
- `required_evidence` 必须可独立复核；不要求临时 `test-results/`、截图、trace 或运行时日志被 git 跟踪。要求跟踪临时输出属于验收契约设计错误。
{{if .HasPreviousReview}}
上一轮复核：`{{.LatestPreviousReviewPath}}`；历史复核次数：{{.PreviousReviewCount}}。
{{if .HasPreviousFixSummary}}修复摘要：`{{.LatestPreviousFixSummaryPath}}`。
{{end}}{{end}}

在运行目录下写入 `review-{{.Iteration}}.json`。

在该目录下运行：`specrelay flow validate-review --artifact "review-{{.Iteration}}.json" --json`。

{{if .IsFirstRoleTurn}}
只写一个严格的 JSON 对象。

字段：`summary`、`decision`（0=clean，1=needs_fix）、`evidence[]`、`findings[]`、`checks`、`non_blocking_findings[]`、`workflow_failure`。

每个问题项：`{title,severity,scope,evidence,recommendation}`。`severity`：1=blocker，2=major，3=minor。`scope`：1=current_change，2=introduced_regression，0=out_of_scope_existing。每个 `evidence[]` 条目都必须是字符串，引用可验证的命令、产物、截图、trace、独立验收结果、控制台或网络观测。

对于 clean：`decision=0`、`findings=[]`、每个 `checks` 值均为 true，且 `evidence` 非空并覆盖 acceptance_contract。对于 needs_fix：`decision=1` 且 `findings` 非空。
{{else}}
续接：复用角色会话 `{{.RoleSessionKey}}`，按相同 schema 重写 `review-{{.Iteration}}.json`。复核上一轮的复核/修复结果，只报告仍然存在的问题。
{{end}}
