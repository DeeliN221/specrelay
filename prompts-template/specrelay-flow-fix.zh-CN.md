## 修复任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取运行目录下的：`state.json`、`acceptance.json`、`review-{{.Iteration}}.json`{{if .QAPath}}、`qa-{{.Iteration}}.json`{{end}}。

任务：

- 只修复当前复核/独立验收产物中列出的问题项。
{{if .IsFirstRoleTurn}}- 定位根因，而不是修补错误文本。不要删除、弱化或绕过已封存的 `acceptance.json`。
{{end}}{{if .HasRoleSession}}- 复用当前角色会话：`{{.RoleSessionKey}}`。
{{end}}{{if .FixEscalated}}- 连续 {{.ConsecutiveReviewFailures}} 次失败后的升级迭代：在摘要中说明上一轮修复失败的原因，以及重复出现的问题项的根因。
{{end}}

在运行目录下写入 `fix-{{.Iteration}}-summary.md`。

使用 Markdown，包含已修复问题、根因、变更、验证命令与结果，以及剩余风险。
