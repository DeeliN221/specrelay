## 归档任务
阶段：`{{.Stage}}`（迭代 `{{.Iteration}}`）
运行目录：`{{.RunDirectory}}`
读取运行目录下的 `state.json`、`acceptance.json` 以及历史审计/修复/独立验收产物。

操作：

- 调用 `specrelay-archive` 技能；使用 `state.json.change_name` 中的变更名称。
- 这是最终独立验收之后的只读边界。只允许机械性的提案移动；长期规格与规格测试必须在最终独立验收之前完成。
- 进入此阶段前，引擎会从最终通过的不可变快照提升证据。验证最终已提交证据位于 `tests/evidence/proposals/<change>/**` 下、未被 git 忽略，并包含在归档提交中。不要从 `test-results/**` 复制或重建它。
- 最终独立验收之后，不要先触发会改动文件的 pre-commit hook。如果重复的 hook 检查仍然改动文件，保留这些变更，让门禁将本次运行退回审计和独立验收。
- 从 state 中已封存的 `delivery_base_head` 开始，在 HEAD 创建恰好一个完整的交付提交，包含实现、已归档提案和最终证据。不要 amend、squash 或复用执行/审计提交。
- 归档命令返回后，不要编辑、格式化或恢复提案目录或源码文件。可以原样暂存并提交命令结果；不要仅为获得干净的工作区而重写文件。
- 归档后保留任何剩余差异，供只读门禁评估。

在运行目录下写入 `delivery-summary.md`。
包含 `Final review` 章节。
