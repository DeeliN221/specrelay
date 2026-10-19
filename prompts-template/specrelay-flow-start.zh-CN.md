## 执行任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取运行目录下的 `state.json` 和 `acceptance.json`。

根据 `state.json.change_name` 确定当前的 specrelay 变更，并保持在其提案范围内。

为当前 specrelay 变更调用 specrelay-exec 技能：`{{.ChangePath}}/`。

完成执行前，运行一个覆盖目标能力的演示。仅生成 `submission_evidence.source_path` 声明的临时 `test-results/**` 证据；阶段后置门禁会为本次运行封存不可变快照。不要直接写入 `tests/evidence/proposals/<change>/**`，也不要创建 git 提交。只有引擎会在归档前提升最终的已提交证据。
