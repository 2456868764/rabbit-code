# T3：预算 meter / 附件 UX ↔ Go（Phase 5 → TUI）

**规则**：`PHASE_ITERATION_RULES.md` **§三**；主清单 **`PHASE05_CONTINUATION.md`** **TUI 表行 C** 与 **§3.0 T3 子计划**。

---

## §3.0 T3 有序迭代计划

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`rabbit-code context budget`**：stdin **resolved submit body**；**`query.BuildSubmitTokenBudgetSnapshotPayload`** → JSON **`kind`=`submit_token_budget_snapshot`**（**`total_tokens`** / **`inject_raw_bytes`** / **`mode_detail`** ↔ **`EngineEvent`** **H5.3**）；默认模式读 **`RABBIT_CODE_TOKEN_SUBMIT_ESTIMATE_MODE`**（不必 **`TOKEN_BUDGET=1`**，便于脚本诊断） | **`go test ./internal/query/... ./internal/commands/contextcmd/... -short`** |
| 2 | ☐ | Bubble Tea **meter** 订阅 **`EventKindSubmitTokenBudgetSnapshot`**；附件条 UI；**`utils/attachments.ts`** 全量仍 Follow-on | **Phase 9** |

---

## 映射表（当前）

| TS / 运行时 | Go | 状态 |
|-------------|-----|------|
| **`query.ts`** submit 路径 token 估计 + **`tokenBudget`** | **`engine`** **`EventKindSubmitTokenBudgetSnapshot`** | **[x]** 引擎 |
| 脚本 / SDK 诊断 | **`context budget`** | **[x]** headless |
| TUI meter | — | **T3 序 2** |

---

## 维护

完成 §3.0 一项：更新上表、**`PHASE05_CONTINUATION.md`**、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
