## 规划任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取运行目录下的 `state.json` 和 `acceptance.json`。

调用 `specrelay-plan` 技能讨论并规划本次变更。

在验收设计中纳入可演示的能力和已提交的证据。说明演示如何覆盖目标能力。在 `submission_evidence` 中，将每个可变的 `source_path` 映射到 `tests/evidence/proposals/<change>/**` 下的 `archive_path`。`test-results/**` 是临时运行时输出，不是最终交付证据。
