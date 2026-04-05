# T1：`thinking.ts` / `processUserInput` ↔ Go（Phase 5 → TUI）

**规则**：`PHASE_ITERATION_RULES.md` **§三**；主清单 **`PHASE05_CONTINUATION.md`** **TUI 行 A** 与 **§3.0 T1 子计划**。

---

## §3.0 T1 有序迭代计划

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`thinking.ts`** 核心 → **`internal/utils/thinking`**；**`engine`** 合并 **ultrathink 关键词** 与 **`features.UltrathinkEnabled`**（**`FormatHeadlessModeTags`** + **`ApplyUserTextHints`**） | **`go test ./internal/utils/thinking/... ./internal/query/engine/... -short`** |
| 2 | ☐ | **`processUserInput.ts`** 全量（slash、附件、**`processTextPrompt`**、hooks 循环）；headless 已有 **`processuserinput.TruncateHookOutput`**（**`MAX_HOOK_OUTPUT_LENGTH`**），宿主 **`ProcessUserInputHook`** 可自选使用 | 后续 T1/T5 |
| 3 | ☐ | **系统块 / thinking beta** 与 **`APIContextManagementOpts`**、TUI 展示一致 | **H4** / T3 交叉 |

---

## 映射表（当前）

| TS | Go | 状态 |
|----|-----|------|
| **`utils/thinking.ts`** | **`internal/utils/thinking`** | **[~]**（无 GB / `get3PModelCapabilityOverride` / `USER_TYPE=ant` **resolveAntModel**） |
| **`processUserInput.ts`** `applyTruncation` | **`processuserinput.TruncateHookOutput`** + **`PlainString`** | **[~]** 子集 |
| **`QueryEngine` / `processUserInput` 管线** | **`engine.Config.ProcessUserInputHook`** | **[~]** 宿主接线 |
| **ultrathink 关键词 → 用户文提示** | **`thinking.HasUltrathinkKeyword`** + **`query.ApplyUserTextHints`**（**`engine.runTurnLoop`**） | **[x]** headless |

---

## 维护

完成 §3.0 一项：更新上表、**`PHASE05_CONTINUATION.md`**、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
