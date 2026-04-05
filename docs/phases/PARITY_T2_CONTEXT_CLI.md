# T2：`commands/context` ↔ Go CLI（Phase 5 → REPL / 脚本）

**规则**：`PHASE_ITERATION_RULES.md` **§三**；主清单 **`PHASE05_CONTINUATION.md`** **TUI 行 B** 与 **§3.0 T2 子计划**。

---

## §3.0 T2 有序迭代计划

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`rabbit-code context`** 子命令路由：**`help`**、**`break-cache`**（委托 **`breakcache`**） | **`go test ./internal/commands/contextcmd/... -short`** |
| 2 | ☑ | **`context report`**：stdin **Messages JSON** + flags → **`query.BuildHeadlessContextReport`** 输出 JSON（headless 子集，对照 **`context-noninteractive.ts`** **`collectContextData`** / **`get_context_usage`** 的数据深度，非 Markdown 全表） | 同上 + 单测 |
| 3 | ☐ | Markdown / 分类表 / **`microcompactMessages`** 全链 parity；TUI 网格（**`context.tsx`**） | **T3** 穿插 |

---

## 映射表（当前）

| TS | Go | 状态 |
|----|-----|------|
| **`commands/context/index.ts`**（slash / local-jsx） | **`contextcmd`** 仅 **headless CLI** | **[~]** |
| **`context-noninteractive.ts`** **`call` / `formatContextAsMarkdownTable`** | **`context report`** → **`HeadlessContextReport`** JSON | **[~]** 子集 |
| **`commands/break-cache`**（TS stub） | **`breakcache`** + **`context break-cache`** | **[x]** headless |

---

## 维护

完成 §3.0 一项：更新上表、**`PHASE05_CONTINUATION.md`**、**`PHASE05_SPEC_AND_ACCEPTANCE.md` §6**。
