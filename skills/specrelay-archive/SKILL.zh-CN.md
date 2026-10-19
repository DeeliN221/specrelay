---
name: specrelay-archive
description: 校验并归档已完成的 specrelay 提案，将其已验证的行为保留在长期规格和规格测试中。
---

# specrelay 归档

将已验证的变更意图保留在长期维护的规格和测试中，并归档其提案目录。

## 条目类型

对于 small 提案，阅读 `brief.md`、`acceptance.json` 和 `tests/`；从简报中提炼持久行为。对于 standard 工作，阅读完整提案、验收契约和测试；从 `spec.md` 中提炼持久行为。两者都必须将适当的行为合并到 `docs/specs/` 和 `tests/specs/`。

## 可选的外部技能

单独安装的 Matt Pocock 技能不由 `specrelay install` 安装。检查其可用性，若不可用则回退到本流程。

- `/writing-for-agents` 可以审查 `DELIVERY.md` 的信息层级、上下文引用和完成标准。它不得改写引擎封存的观察结果或证据路径。
- `/domain-modeling` 可以帮助审查应纳入维护词汇表的术语。最终 QA 之后，它不得触发新的编辑。

请调用斜杠命令，不要重复其流程。归档阶段是最终 QA 之后的只读边界；外部技能可以审查结果并提出未来改进建议，但不能改写源码、提案或证据。请在最终 QA 之前完成长期规格和测试的修改。

## 流程

- 检查工作树，找出所有与提案相关的变更；排除无关文件。
- 运行 `specrelay validate <change> --json`。
- 不要创建或编辑 `task.md`。完成与否取决于验证、验收结果和最终审查。
- 确认最终通过的测试轮次和相关结果。`test-results/` 只是临时输出。
- 在 SpecRelay 工作流中，引擎根据最终 QA 观察结果创建 `DELIVERY.md`，并根据不可变快照创建证据包。请审查它们，但不要改写。
- 对于独立归档，使用 `DELIVERY.md`、`README.md` 和 `manifest.json` 收集最终证据。
- 为审核者编写交付报告：用户收益、准备工作、操作、预期结果、实际观察、直接证据和限制。对于修复，包含前后差异。
- 命令、退出码、HTTP 200、元素存在、测试通过消息、哈希和硬编码字符串可以补充证据，但不能替代可理解的截图、视频、业务日志或真实输入/输出。
- 运行 `specrelay archive <change> --yes`。CLI 会把包含 `tests/` 的提案目录移动到 `docs/changes/archive/<number>-<change-name>`（不添加日期前缀），并改写脚本、`acceptance.json`、`brief.md`、`spec.md` 及其他文本中的提案测试路径。它不会将测试按逻辑合并到长期 `tests/specs/`。
- 理解每个归档测试的业务契约和断言。按能力而非提案编号组织维护的测试；不要只是整目录复制提案测试。
- 源码注释可以引用提案（例如 `// Sources: 1-login, 3-access-control`），而文件名应描述能力。
- 将归档规格中的持久行为合并到 `docs/specs/*.md`；对于 small 变更，使用简报。更新路径后运行受影响的规格测试。这些编辑必须在最终工作流 QA 边界之前完成。
- 只暂存显式路径。不要使用 `git add .`、`git add -A` 或 `git add -f`。保持 `test-results/` 被忽略；通过尽可能窄的必要忽略例外，使 `tests/evidence/proposals/<change>/` 可被跟踪。
- 检查 `git diff --cached --name-only`，并对每个候选项运行 `git check-ignore --no-index -- <path>`。移除误暂存的被忽略输出，并确认最终证据已存在。
- 任何超过 20 MiB 的证据文件都应使用 Git LFS；使用 `git check-attr filter -- <path>` 确认 `filter=lfs`。
- 从已封存的 `delivery_base_head` 开始，创建恰好一个完整的交付提交。不要 amend、rebase、squash 或改写更早的历史。归档的提案、代码、测试、长期规格和最终证据必须位于同一 HEAD；`delivery_base_head..HEAD` 必须恰好包含一个提交。
- 使用提交信息 `<number>: <change-name>`。报告归档路径、按逻辑合并的测试和规格、已运行的命令以及剩余风险。

归档命令返回后，不要通过编辑、格式化或还原文件来强行获得干净的工作树。保留意外差异，让门禁将其退回审计和独立 QA。不要在最终 QA 之后才首次运行会修改文件的 pre-commit hook。

## 退出标准

验证、验收测试和最终审查均已通过。提案位于 `docs/changes/archive/` 下。其持久行为和测试已按能力合并，或有具体原因说明为何无需合并。受影响的测试已运行。

最终证据目录包含 `DELIVERY.md`、真实可播放的演示视频、审核入口和必要的附件，并与代码一起提交。每个修复都必须提供彼此不同且清晰可读的前后证据，二者使用相同的入口、角色、数据与环境，并从交付报告直接链接。

交付范围内只有一个提交，且相关工作树干净。审查每个已暂存路径，排除临时输出、失败的迭代、缓存、敏感数据和无关产物。

## 完整性检查

仅靠 CLI 移动并不能完成归档。合并历史断言前，先阅读它们。不要捆绑无关工作，也不要提交整个 `test-results/`；只使用已提升到提案证据目录中的最终通过证据。
