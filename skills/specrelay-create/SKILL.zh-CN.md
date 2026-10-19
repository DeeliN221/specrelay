---
name: specrelay-create
description: 创建仅含简报的 small 或 standard specrelay 提案，并附带可执行的验收契约与真实测试。
---

# specrelay 创建

创建 specrelay 提案及其交付契约。`brief.md`、`acceptance.json` 和 `tests/` 始终是必需的。standard 提案还必须让 `proposal.md`、`design.md` 和 `spec.md` 与该契约保持一致。

## 条目类型

| 类型 | 范围 | 必需产物 |
| --- | --- | --- |
| small | 单一业务意图，最多两个验收场景或两项必需测试，且没有复杂的设计分歧 | `brief.md`、`acceptance.json`、`tests/` |
| standard | 跨模块、高风险、多场景的工作，或超出 small 上限的工作 | `brief.md`、`proposal.md`、`design.md`、`spec.md`、`acceptance.json`、`tests/` |

micro 变更使用 TDD 和一次 git 提交，不经过 `specrelay create`。应因风险、范围或复杂度升级到 standard；不要为了证明该类别而虚构任务或测试。

## 可选的外部技能

单独安装的 Matt Pocock 技能不由 `specrelay install` 安装。请检查可用性；若不存在，回退到本技能。

- `/to-spec`：将充分展开的讨论综合为 standard 的 `spec.md` 或 small 的 `brief.md` 行为，然后映射到 specrelay 的文档与验收结构。
- `/to-tickets`：将 standard 工作拆分为 tracer-bullet 工单并标明阻塞关系，把每个工单映射到验收测试/证据。
- `/prototype`：用一次性原型解决状态模型或 UI 的不确定性。将结论记录到设计/规格中；不要把原型代码当作交付代码。其分支可以作为设计预览证据。
- `/domain-modeling`：在编写简报/规格之前建立一致的术语。

这些技能不能替代 `acceptance.json` 或真实测试。每个生成的行为和工单都需要覆盖。请调用斜杠命令，不要重复其流程。

## 流程

1. 运行 `specrelay create`，只将其整数输出用作 `<number>`。
2. 根据已商定的规划，创建 `docs/changes/<number>-<change-name>/`。
3. 先将 small 简报中的验收项，或 standard 规格中的场景，映射到必需测试和证据。
4. 编写所选条目类型所需的产物。
5. 运行 `specrelay validate <change> --json`，检查文档、测试和 JSON 是否一致。
6. 在执行前提交提案，以免丢失或与无关变更混杂。

## 产物契约

- `brief.md`：说明用户问题、交付目标、非目标、验收入口和默认执行上下文，使读者无需阅读长篇文档即可理解该变更。
- `proposal.md`：standard 工作必需；说明做什么以及为什么。
- `design.md`：standard 工作必需；记录决策、取舍和风险。
- `spec.md`：standard 工作必需；使用 `### Requirement:` 和 `#### Scenario:` 标题。每个场景都要指明其真实测试文件、数据来源、入口、断言和剩余风险。已有的翻译标题仍保持兼容。
- `acceptance.json`：覆盖每个需求/场景，复用契约测试，并包含必要的根级端到端/回归测试，以及截图、trace、网络、控制台或运行时证据。
- `tests/`：表达核心契约的真实项目测试代码。不得有空目录、占位文件或仅描述测试的文档。默认使用易懂的英文注释。

不要创建或编辑 `task.md`。创建阶段定义意图、范围、场景和验收，而非固定的实现步骤。动态实现计划应放在 Todo 或运行时状态中，而不是受跟踪文件里。

创建定义契约；执行让契约通过。不要把核心行为标准推迟给执行器，也不要承诺超出测试所能证明的内容。

## 覆盖要求

- 将每个场景映射到至少一项必需测试；关键用户路径还需要证据。先构建覆盖矩阵，再撰写正文。
- 无法支撑的承诺应归入风险、非目标或待解决问题，直到可以验证。
- 覆盖可观测行为：渲染结果、CLI 输出、API 契约、状态变更、持久化或权限边界。
- 使用真实业务示例、入口和调用链。Web 验收必须包含相关的登录、权限、导航、API 和数据库行为。
- 除非用户明确要求且设计记录了相关限制，否则不要 mock API 或数据库、伪造认证、硬编码成功结果，或仅依赖 HTTP 200、元素存在或浅层组件渲染。
- 仓库夹具必须说明其代表的真实业务示例和字段语义；它不能悄然冒充生产数据。
- 缺失的账号、示例数据、外部服务访问或可执行环境应与用户一起解决，而不是让执行阶段自行编造。
- 即使没有现成的测试框架，也要提供最小可运行的测试入口。引入新框架前，确认它是成本最低的方案。
- 测试断言必须检查业务输出、存储记录、API 字段、授权拒绝、持久化状态、审计结果或可观测的用户流程——而不仅仅是测试能够启动。
- `test-results/` 保存可重新生成的运行时输出，且保持不被跟踪。稳定的预期快照可以遵循项目惯例；实际输出不是基线。
- 通过 `submission_evidence` 映射每个 `required_evidence` 条目；至少有一个使用 `kind=demo_video`。使用环境支持的、确实可播放的格式。视频、截图和 trace 必须是真实文件，绝不能是重命名后的命令输出、退出码或硬编码字符串。
- `delivery_report` 面向审核者说明用户收益、操作、预期结果和直接证据。修复场景需要为同一场景提供可理解且彼此不同的前后证据。
- 完成之前，先问一问浅层实现是否仍能通过。若是，则加强断言或收窄承诺的行为。

## 验收格式

使用严格 JSON。例如：

```json
{
  "summary": "Verify the main user flow and persisted result",
  "coverage": [{
    "spec": "Requirement: Example capability / Scenario: Main user flow",
    "tests": ["contract-main-flow"],
    "evidence": ["main-flow-demo"],
    "risk": "External-provider outage recovery is outside this scenario"
  }],
  "required_tests": [{
    "id": "contract-main-flow",
    "source": "change_contract",
    "path": "docs/changes/12-example/tests/main-flow.acceptance.test.ts",
    "command": "pnpm exec tsx --test docs/changes/12-example/tests/main-flow.acceptance.test.ts",
    "purpose": "Verify the main behavior promised by proposal/spec",
    "assertions": [
      "Create a business record with required fields through the real entry point",
      "Read the persisted record after refresh or a new query",
      "Reject requests lacking permission with an explicit denial"
    ],
    "expected_initial_failure": "The target behavior is absent before implementation"
  }],
  "required_evidence": [{
    "id": "main-flow-demo", "kind": "demo_video",
    "path": "test-results/main-flow/demo.mp4",
    "purpose": "Demonstrate creation, refresh and restored state"
  }],
  "submission_evidence": [{
    "evidence_id": "main-flow-demo",
    "source_path": "test-results/main-flow/demo.mp4",
    "archive_path": "tests/evidence/proposals/12-example/main-flow-demo.mp4"
  }],
  "delivery_report": {
    "user_benefits": ["Users can complete the operation and see the persisted result."],
    "prerequisites": ["A normal user account with access to this feature."],
    "scenarios": [{
      "id": "main-user-flow", "title": "Complete the main user flow",
      "user_value": "Obtain a usable result without additional manual processing.",
      "steps": [{
        "action": "Create a record through the real entry point, then refresh.",
        "expected": "The record and its required fields remain visible."
      }],
      "evidence_ids": ["main-flow-demo"]
    }],
    "known_limits": ["The demonstration uses a test account and sanitized business data."]
  }
}
```

`.mp4` 扩展名仅用于说明映射关系，并非强制格式。path、source_path 和 archive_path 中的实际扩展名应保持一致。

`coverage[].spec` 引用真实的需求/场景。其 tests 和 evidence 引用已声明的 ID；如果没有证据，使用空数组并说明风险。测试来源为 `change_contract`、`root_e2e`、`existing_regression` 或 `new_regression`，每项测试至少包含一个业务断言。证据类型为 `screenshot`、`trace`、`network`、`console`、`runtime_log`、`state_snapshot`、`demo_video` 或 `other`。

对于修复，在同一个交付场景中包含以下对象：

```json
"comparison": {
  "before": "The actual user-visible failure before the fix",
  "after": "The usable result after the fix",
  "before_evidence_id": "before-proof",
  "after_evidence_id": "after-proof"
}
```

将该场景的 `evidence_ids` 同时包含这两个 ID，并指向内容不同且可读取的文件。在 prerequisites 或 steps 中说明共享的入口、角色、数据和环境。

`required_evidence` 描述可检查的临时证据，`submission_evidence` 定义最终提交路径，`delivery_report` 驱动 `DELIVERY.md`。通过的测试输出可以补充用户证据，但不能替代它。

在创建提案之前，澄清尚未解决的测试策略、真实数据、可观测断言或 QA 证据。执行阶段可以新增测试，但不能删除、削弱或绕过原始契约。只有用户意图发生变化，才允许同步修改契约/规格/设计/测试，并记录原因。

## 退出标准

使用不带日期前缀的 `docs/changes/<number>-<change-name>/`。编号应从 `specrelay create` 获取，而不是通过枚举庞大的提案/归档目录得到。默认使用易懂、面向动作的英文名称；支持 Unicode 名称。

校验通过，简报足以支撑执行，测试包含真实代码而非占位符。人工核对 JSON 映射、审核者收益与步骤、可观测结果和真实演示证据。立即以 `<number> proposal: <change-name>` 提交提案产物。

不要把测试推迟到执行阶段，不要承诺未覆盖的行为，不要提交临时输出，也不要捏造缺失的真实数据。
