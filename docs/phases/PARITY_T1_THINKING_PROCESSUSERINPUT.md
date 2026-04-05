# T1：`thinking.ts` / `processUserInput` ↔ Go（Phase 5 → TUI）

**规则**：`PHASE_ITERATION_RULES.md` **§三**；主清单 **`PHASE05_CONTINUATION.md`** **TUI 行 A** 与 **§3.0 T1 子计划**。

---

## §3.0 T1 有序迭代计划

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`thinking.ts`** 核心 → **`internal/utils/thinking`**；**`engine`** 合并 **ultrathink 关键词** 与 **`features.UltrathinkEnabled`**（**`FormatHeadlessModeTags`** + **`ApplyUserTextHints`**） | **`go test ./internal/utils/thinking/... ./internal/query/engine/... -short`** |
| 2 | ☑ | **`processUserInput`** headless：**`user_prompt_keywords`**（**`userPromptKeywords.ts`** **`matchesNegativeKeyword` / `matchesKeepGoingKeyword`**）、**`PlainPromptSignals`**；**`engine.Config.TruncateProcessUserInputHookOutput`** → **`runTurnLoop`** 在 **`ProcessUserInputHook`** **`replace`** 后对正文 **`TruncateHookOutput`**；全量 slash/附件/hooks 循环仍 TUI/T5 | **`go test ./internal/utils/processuserinput/... ./internal/query/engine/... -short`** |
| 3 | ☑ | **`thinking.InterleavedAPIContextManagementOpts`**（**`apiMicrocompact.ts`** interleaved + **`RABBIT_CODE_REDACT_THINKING`** / **`RABBIT_CODE_THINKING_CLEAR_ALL`**）；**`ApplyEngineCompactIntegration`** 在 **`aa.Client != nil`** 且 **`APIContextManagementOpts == nil`** 时默认填入 | **`go test ./internal/utils/thinking/... ./internal/app/... -short`**；TUI 展示仍 **H4** / **T3** |

---

## 映射表（当前）

| TS | Go | 状态 |
|----|-----|------|
| **`utils/thinking.ts`** | **`internal/utils/thinking`** | **[~]**（无 GB / `get3PModelCapabilityOverride` / `USER_TYPE=ant` **resolveAntModel**） |
| **`processUserInput.ts`** `applyTruncation` | **`processuserinput.TruncateHookOutput`** + **`PlainString`**；**`engine.Config.TruncateProcessUserInputHookOutput`** | **[x]** headless 钩后截断 |
| **`userPromptKeywords.ts`**（negative / keep-going） | **`processuserinput.MatchesNegativeKeyword`** / **`MatchesKeepGoingKeyword`**、**`PlainPromptSignals`** | **[x]** headless |
| **`QueryEngine` / `processUserInput` 管线** | **`engine.Config.ProcessUserInputHook`** | **[~]** 宿主接线 |
| **`apiMicrocompact.ts`** context_management / interleaved | **`thinking.InterleavedAPIContextManagementOpts`** + **`ApplyEngineCompactIntegration`** | **[x]** 默认 opts |
| **ultrathink 关键词 → 用户文提示** | **`thinking.HasUltrathinkKeyword`** + **`query.ApplyUserTextHints`**（**`engine.runTurnLoop`**） | **[x]** headless |

---

## 维护

完成 §3.0 一项：更新上表、**`PHASE05_CONTINUATION.md`**、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
