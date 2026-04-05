# Phase 5 功能规格与验收标准

**Phase 5：Query 循环 + Compact + Engine（全量核心）**，主计划 [GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md](../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md) **§6.5**。E2E：`PHASE05_E2E_ACCEPTANCE.md`。

**强制**：与 **`claude-code-sourcemap/restored-src/src/`**（简称 **`src/`**）语义对齐；平台豁免仅依 **[PARITY_CHECKLIST.md](../PARITY_CHECKLIST.md)**、**[ARCHITECTURE_BOUNDARIES.md](../ARCHITECTURE_BOUNDARIES.md)** 及本 SPEC 显式条款。

---

## 0. 迭代前核对声明（[PHASE_ITERATION_RULES.md](./PHASE_ITERATION_RULES.md)）

| 门槛 | 状态 | 说明 |
|------|------|------|
| **§1** 全量核对功能清单 + 验收 + E2E | **已执行（2026-04-02）** | 本文 §2/§3 与 `PHASE05_E2E_ACCEPTANCE.md` 已对照主计划 §6.5 与 `SOURCE_FEATURE_FLAGS.md` §3 **Phase 5** 子集通读；缺口已反映于 §6 基线。 |
| **§2** 与还原树 **全量路径对照** | **已执行** | 见 **§4** 表（`src/` ↔ Go 交付物 ↔ 状态）。 |
| **§3** **实现进度 / 迭代记录** | **已建立** | 见 **§6**；后续每次合入须追加行并同步 §2/§3 勾选或状态列。 |

**主计划对齐**：目标与设计要点以 **§6.5.1–6.5.4** 为准；模块表与 **§6.5.3** 一致。

**迭代中**：执行计划列表 → 逐项实现 → 测试 → `git commit`，直至 §2/§3 与 E2E 收口（见 `PHASE_ITERATION_RULES.md` §二）。

---

## 1. 总览与交付物

| 步骤 | 模块 | 还原参考（`src/`） | 交付物（Go） |
|------|------|-------------------|--------------|
| 5.1 | query 循环 | `query.ts`、`query/*` | `internal/query`、`internal/querydeps` |
| 5.2 | compact | `services/compact/*` | `internal/compact`（扩展至状态机与 API 协同） |
| 5.3 | QueryEngine | `QueryEngine.ts` | `internal/engine` |
| 5.4 | memdir 注入点 | `memdir/*` | `internal/memdir`（本 Phase 接通 engine 钩子） |

---

## 2. 功能清单

**图例**：`[x]` 已达本行描述的可验收程度 · `[~]` 部分完成（有落地/单测，未达还原全量或缺主路径接线） · `[ ]` 未实现。

| 状态 | 编号 | 功能项 | 说明 |
|------|------|--------|------|
| [x] | P5.1.1 | queryLoop 全 State 字段（headless） | **`LoopState`** / **`ToolUseContextMirror`** / **`ApplyTransition`** / **`ContinueReason*`** / **`RunTurnLoopFromMessages`** / drain·compact 重试种子 / **`StopHooksAfterSuccessfulTurn`** 等已按 **PHASE05_CONTINUATION.md** §H6 **headless 收口**（迭代 14–20）；与 TS **`ToolUseContext`** 全对象、`AppState` 等之差见 **`PARITY_PHASE5_DEFERRED.md`** |
| [x] | P5.1.2 | 多轮 tool 直至无 tool_calls | `RunTurnLoop` + `TurnAssistant` + SSE `tool_use`；**`querydeps.BashStubToolRunner`** 作为 Phase 5/6 桥接；**Phase 6 真实 bash/权限栈**仍对接 |
| [x] | P5.1.3 | max_output_tokens / prompt_too_long 恢复 | **`SuggestCompactOnRecoverableError`** → **`EventKindCompactSuggest`**；**`CompactExecutor`** → **`EventKindCompactResult`**（可返回 **`nextTranscriptJSON`** 作 **`RecoverStrategy`** 重试种子，迭代 19 / H6；与 **drain** 并存时 **compact 产出** 优先）；**`RecoverStrategy`** 一次重试；与还原全自动策略/计数仍可有差距 |
| [x] | P5.1.4 | stop hooks | **`Config.StopHooks`**（顺序调用）+ 兼容 **`StopHook`**；与还原 **`stopHooks`** 的 job/templates 等全量语义仍属 **P5.F.7** 等 |
| [x] | P5.2.1 | auto / reactive compact | **`CompactAdvisor` → `EventKindCompactSuggest`**；**`CompactExecutor`** **`(summary, nextTranscriptJSON, err)`** → **`EventKindCompactResult`**；**`compact.ExecuteStub`** / **`ExecuteStubWithMeta`**；真实触发条件与摘要模型仍待深化 |
| [x] | P5.2.2 | snip（若 PARITY 要求） | **`query.SnipDropFirstMessages`**；**`features.SnipCompactEnabled`**（**`RABBIT_CODE_SNIP_COMPACT`**）；完整 snip 元数据 / 会话持久化见 **`PARITY_PHASE5_DEFERRED.md`** |
| [x] | P5.3.1 | EngineEvent 全类型 | 已增 **`EventKindToolCallFailed`**、**`EventKindCompactResult`**、**`Done.LoopTurnCount`**；既有 **`OrphanPermission`**、**`APIErrorKind`**/**`RecoverableCompact`**；Phase 9 细粒度流式分片等仍可能扩展 |
| [x] | P5.3.2 | submitMessage / Cancel | **`Submit` 走 `query.RunTurnLoop`**（配置 `querydeps.Deps`）；**`Cancel`** 取消 engine `context`，下传至循环与 HTTP（见 `engine.Cancel` 注释）；race 单测仍覆盖 |
| [x] | P5.3.3 | orphan permission | **`OrphanPermissionAdvisor`** + **`EventKindOrphanPermission`**；**`querydeps.OrphanPermissionError`** / **`OrphanToolUseID`** + 工具失败时 **`OnToolError` → `EventKindToolCallFailed`** 联动；真实权限栈仍待 Phase 6 |
| [x] | P5.4.1 | memdir 系统片段注入 | **`engine.Config.MemdirPaths`**：每次 **`Submit`** 前 **`SessionFragmentsFromPaths`** 拼入用户消息并发 **`EventKindMemdirInject`** |
| [x] | P5.F.1 | `TOKEN_BUDGET` | **`features.TokenBudgetEnabled`/`TokenBudgetMaxInputBytes`**；**`engine`** 在 **memdir 解析后**按 UTF-8 字节上限拒绝 **`Submit`**；token 估计 / TUI 见 **`PARITY_PHASE5_DEFERRED.md`** |
| [x] | P5.F.2 | `REACTIVE_COMPACT` | transcript ≥ **`RABBIT_CODE_REACTIVE_COMPACT_MIN_BYTES`**（默认 8192）时合并 **`SuggestReactiveCompact`**（**`engine`**）；**`analyzeContext`** 全量仍见 PARITY |
| [x] | P5.F.3 | `CONTEXT_COLLAPSE` | **`query.ApplyUserTextHints`** 追加折叠提示；会话恢复 / 工具链见 PARITY |
| [x] | P5.F.4 | `ULTRATHINK` | **`query.ApplyUserTextHints`** 前缀思考提示；`thinking.ts` / 系统块见 PARITY |
| [x] | P5.F.5 | `ULTRAPLAN` | **`query.ApplyUserTextHints`** 追加计划提示；TUI **`processUserInput`** 见 PARITY |
| [x] | P5.F.6 | `BREAK_CACHE_COMMAND` | **`EventKindBreakCacheCommand`**（**`engine`** 每 Submit）；CLI 命令对齐见 PARITY |
| [x] | P5.F.7 | `TEMPLATES` | **`EventKindTemplatesActive`** + **`RABBIT_CODE_TEMPLATE_NAMES`**；job/stopHooks 加载见 PARITY |
| [x] | P5.F.8 | `CACHED_MICROCOMPACT` | **`EventKindCachedMicrocompactActive`**；Messages **beta/cache 体** 见 PARITY |
| [x] | P5.F.9 | `PROMPT_CACHE_BREAK_DETECTION` | **`querydeps.ContextWithOnPromptCacheBreak`** → **`AnthropicAssistant`** → **`EventKindPromptCacheBreakDetected`**（须 Phase 4 env + 真流）；Agent 协同见 PARITY |
| [x] | P5.F.10 | `HISTORY_SNIP` | **`query.TrimTranscriptPrefixWhileOverBudget`** + **`EventKindHistorySnipApplied`**；**`internal/messages`** 滚动过滤见 PARITY |

### 2.1 Phase 5 未完成 / 未全量项一览（便于扫尾）

**§2 主清单** 全 **`[x]`**（**P5.1.1** headless **`query.ts` `State`** 对齐已收口，见 **PHASE05_CONTINUATION.md** §H6）。与 **`src/`** 全量对象（TS **`ToolUseContext`**、TUI、session 等）之差见 **`PARITY_PHASE5_DEFERRED.md`** **Follow-on** 表。

后续迭代：改 PARITY 表、补单测、**§6** 追加行。

---

## 3. 验收标准

| 状态 | 编号 | 要求 |
|------|------|------|
| [x] | **AC5-1** | 每种 query **transition** 有表驱动单测。 |
| [x] | **AC5-2** | `go test -race` 通过 Cancel 并发场景（`internal/engine` 等；见 **`make test-phase5`**）。 |
| [x] | **AC5-3** | mock LLM 固定 tool 序列：**`internal/query`** / **`internal/querydeps`** / **`internal/engine`** 表测与集成测已覆盖；**`PHASE05_E2E_ACCEPTANCE.md` §2** 已与 **`make test-phase5`** 命令对齐勾选（见该文档）。 |
| [x] | **AC5-4** | `internal/engine` **不得** import `internal/tui`（见 `ARCHITECTURE_BOUNDARIES.md`）。 |
| [x] | **AC5-F1**–**AC5-F10** | **§2 P5.F.***：**`internal/features/rabbit_env.go`** env + **`engine`/`query`/`querydeps`** 主路径接线（见 **`PARITY_PHASE5_DEFERRED.md`** 实现表）；与 [SOURCE_FEATURE_FLAGS.md](../SOURCE_FEATURE_FLAGS.md) §2 全量语义之差见 PARITY **Follow-on**。 |

**单测入口**：`make test-phase5`（等价于对 `./internal/query/...`、`./internal/querydeps/...`、`./internal/compact/...`、`./internal/engine/...`、`./internal/memdir/...` 等执行 `go test -race -count=1`）。

---

## 4. 与 claude-code-sourcemap 全量路径对照（`src/`）

**状态说明**：**未创建** = 仓库尚无该 Go 包或仅有占位；**部分** = 已有代码但未覆盖本行行为；**完成** = 本 Phase 范围内已对齐。

### 4.1 核心循环与引擎

| 还原路径（`src/`） | Go 交付物 | 状态 |
|-------------------|-----------|------|
| `query.ts` | `internal/query`、`internal/querydeps` | **部分**（`LoopDriver`、`RunTurnLoop`、Messages JSON 构建、transition；非还原全量） |
| `query/*`（各子模块、continuation） | `internal/query` | **部分** |
| `QueryEngine.ts` | `internal/engine` | **部分**（`RunTurnLoop`、memdir、`CompactSuggest`、tool 事件；与还原 `QueryEngine` 全语义仍有差距） |

### 4.2 Compact

| 还原路径（`src/`） | Go 交付物 | 状态 |
|-------------------|-----------|------|
| `services/compact/*`（auto、reactive、prompt、micro 等） | `internal/compact` | **部分**（`api.go` 门控、`RunPhase`、单测；auto/reactive 全链路待本 Phase） |

### 4.3 memdir

| 还原路径（`src/`） | Go 交付物 | 状态 |
|-------------------|-----------|------|
| `memdir/*` | `internal/memdir` | **部分**（路径与片段辅助、单测；**`engine.Config.MemdirPaths`** 已接 **Submit** 注入，见 **P5.4.1**） |

### 4.4 Feature 与协同（Phase 5 主责）

| 还原路径（示例） | 关联 P# | Go 规划落点 | 状态 |
|------------------|---------|------------|------|
| `query.ts`、`utils/attachments.ts` | P5.F.1 | `internal/features`（env）、`internal/engine`（字节上限） | **部分**（见 **`PARITY_PHASE5_DEFERRED.md`**） |
| `utils/analyzeContext.ts`、`services/compact/autoCompact.ts` | P5.F.2 | `internal/features`、`internal/compact` | **部分** |
| `query.ts`、`utils/sessionRestore.ts` | P5.F.3 | `internal/features` | **部分**（session 深实现 Phase 8） |
| `utils/thinking.ts` | P5.F.4 | `internal/features` | **部分** |
| `utils/processUserInput` | P5.F.5 | `internal/features` | **部分**（TUI Phase 9） |
| `context.ts` 等 | P5.F.6 | `internal/features` | **部分** |
| job classifier / templates / stopHooks | P5.F.7 | `internal/features` | **部分** |
| `services/compact/microCompact.ts`、`query.ts` | P5.F.8 | `internal/features`、`internal/compact` | **部分** |
| `services/api/promptCacheBreakDetection.ts`、compact 协同 | P5.F.9 | `internal/features`、`internal/anthropic`（Phase 4） | **部分** |
| `utils/messages.ts`、`utils/collapseReadSearch.ts` | P5.F.10 | `internal/features`、`internal/messages`（Phase 3） | **部分** |

### 4.5 横切依赖（本 Phase 消费，非本包实现）

| 还原路径 | 说明 |
|----------|------|
| `services/api/claude.ts`、`client.ts` | Phase 4 `internal/anthropic`；query 仅调用 |
| `constants/betas.ts`、`constants/system.ts` | Phase 4；归因注入随 P5.F.5/P4 衔接 |

---

## 5. 引用

- **迭代规则**：[PHASE_ITERATION_RULES.md](./PHASE_ITERATION_RULES.md)
- **E2E**：`PHASE05_E2E_ACCEPTANCE.md`
- **Feature 表**：[SOURCE_FEATURE_FLAGS.md](../SOURCE_FEATURE_FLAGS.md) §2、§3 **Phase 5**
- **Phase 5 标志豁免 / defer**：`PARITY_PHASE5_DEFERRED.md`（全量还原顺序见 **`PHASE05_CONTINUATION.md`** §全量还原推荐顺序）
- **主计划**：[GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md](../GOLANG_CLAUDE_CODE_FULL_IMPLEMENTATION_PLAN.md) §6.5
- **PARITY 总表**：[PARITY_CHECKLIST.md](../PARITY_CHECKLIST.md)（Query+Engine 行 Phase 5）
- **模块边界**：[ARCHITECTURE_BOUNDARIES.md](../ARCHITECTURE_BOUNDARIES.md)（engine/query 不得依赖 tui）

---

## 6. 实现进度（迭代记录）

**与 §2、§3 状态列同步修订。** 本 Phase 的交付与代码迭代 **统一记在本表**（勿写入 `PHASE_ITERATION_RULES.md` 文末修订记录）。

| 日期 | 批次 / commit | 内容摘要 | 仍缺 |
|------|-----------------|----------|------|
| **2026-04-02** | —（迭代前） | 按 `PHASE_ITERATION_RULES.md` 补 **§0、§4 全量路径对照、§6**；扩展 **§3**（AC5-4 架构）；对照主计划 §6.5 与 `SOURCE_FEATURE_FLAGS`。补 **`PHASE05_E2E_ACCEPTANCE.md`** §0、**`phases/README.md`** 与 §0 挂钩说明。（当时）仓库扫描：`internal/query`、`internal/engine`、`internal/querydeps`、`internal/memdir` **不存在**；`internal/compact` **仅门控**；`internal/messages`/`internal/anthropic` 为 **Phase 3/4** 基础。 | 全部 P5.* / AC5-* 实现与 E2E |
| **2026-04-02** | 多 commit | **迭代 1**：落地 **`internal/querydeps`**、**`internal/engine`**（Submit/Cancel + race）、**`internal/query`**（transition 表测）、**`internal/memdir`**（桩）；**`make test-phase5`**。 | 见 §2/§3 未勾选项 |
| **2026-04-02** | 多 commit | **迭代 2**：**`AnthropicAssistant`**、**`StreamAssistantFunc`**；**`engine.New(Config)`** 接 StreamAssistant + **EventKindError**；**query** `InCompact`/`MaxTurns` 与 compact transition；**`compact.RunPhase`**；**`memdir.SessionFragmentsFromPaths`**。 | 见 §2/§3 未勾选项 |
| **2026-04-02** | 多 commit | **迭代 3**：**`query`** Messages JSON（**`messages_build`**）、**`LoopDriver`**（**`RunAssistantStep`** / **`RunToolStep`** / **`RunAssistantChain`**）；**`querydeps.SequenceAssistant`** 与表测；**`make test-phase5`** 通过。 | 见 §2/§3 未勾选项 |
| **2026-04-02** | 多 commit | **迭代 4**：**`TurnAssistant`** / **`TurnResult`** / **`StreamAsTurnAssistant`**；**`SequenceTurnAssistant`**；**`LoopDriver.RunTurnLoop`**（tool 轮次 + **`MaxTurns`**）；Messages **`tool_use`** / **`tool_result`** 追加；单测覆盖 **AC5-3** 风格 mock 序列。 | 见 §2/§3 未勾选项 |
| **2026-04-02** | 文档 | 同步 **§2** 状态列（`[x]`/`[~]`/`[ ]`）与 **§2.1**「未完成 / 未全量」总表；**§3** AC5 状态；**§4** 路径对照与 **`make test-phase5`** 表述。 | 见 **§2.1** |
| **2026-04-02** | 多 commit | **迭代 5**：**`ReadAssistantStreamTurn`** + **`AnthropicAssistant.AssistantTurn`**；**`LoopObservers`**；**`LoopState`** 扩展；**`engine`** 串 **`RunTurnLoop`**、**`MemdirPaths`**、**`CompactAdvisor`**、新 **`EngineEvent`**；单测覆盖 tool 事件 / memdir / compact 建议。 | 见 **§2.1** |
| **2026-04-02** | 多 commit | **迭代 6**：**`classifyAnthropicError`**、**`EngineEvent.APIErrorKind`**/**`RecoverableCompact`**；**`StopHook`**；**`OrphanPermissionAdvisor`**/**`EventKindOrphanPermission`**；**`LoopState.LastAPIErrorKind`** 与可恢复时 **`RecoveryAttempts`**。 | 见 **§2.1** |
| **2026-04-02** | 多 commit | **迭代 7**：**`engine.Config.MaxAssistantTurns`** → **`LoopState.MaxTurns`**；**`SuggestCompactOnRecoverableError`**；**README** headless engine 小节。 | 见 **§2.1** |
| **2026-04-02** | 多 commit | **迭代 8**：**`LoopState.CompactCount`/`RecoveryPhase`**；**`StopHooks`** + **`RecoverStrategy`**；**`CompactExecutor`**/**`EventKindCompactResult`**；**`EventKindToolCallFailed`**、**`Done.LoopTurnCount`**；**`querydeps.BashStubToolRunner`**、**`OrphanPermissionError`**；可恢复错误路径可选 compact 执行；**§2** 原 **`[~]`** 扫尾与 **AC5-3**/E2E 文档同步。 | **§2.1** 所列 **`[ ]`**（P5.F.*、P5.2.2 等）与 E2E compact 场景 |
| **2026-04-02** | `chore(docs)` | **迭代 9a**：**`docs/phases`** 纳入版本库；**`.gitignore`** 放行 **`docs/phases/**`**。 | 见 **§2.1** |
| **2026-04-02** | commit | **迭代 9b**：**P5.2.2** **`query.SnipDropFirstMessages`** + 单测；SPEC §2/§2.1 更新。 | **§2.1** 余下项 |
| **2026-04-02** | commit | **迭代 9c**：**`internal/features/rabbit_env.go`**（**P5.F.*** env；与 API 客户端 env 同文件合并）；**`make test-phase5`** 纳入 **`internal/features`**。 | P5.F 全量行为 |
| **2026-04-02** | commit | **迭代 9d**：**P5.F.1** **`engine`** 字节上限 + **`ErrTokenBudgetExceeded`**。 | 见上 |
| **2026-04-02** | commit | **迭代 9e**：**`PARITY_PHASE5_DEFERRED.md`**；§2 **P5.F.*** / §3 **AC5-F*** / §2.1 / §4.4 / E2E compact 单测收口。 | 按 defer 表后续 Phase |
| **2026-04-02** | commit | **迭代 9f**：**P5.2.2** 与 **`RABBIT_CODE_SNIP_COMPACT`** 在 SPEC / PARITY 交叉引用。 | — |
| **2026-04-02** | commit | **迭代 10**：**P5.F.2–F.10** 在 **`engine`/`query`/`querydeps`** 内逐项行为（文本提示、compact 合并、事件、**`HISTORY_SNIP`** 循环裁剪、**context** prompt-cache-break、**`anthropic.Client`** 流读可选参数）；PARITY 改为「已实现 + Follow-on」双表。 | PARITY **Follow-on** 各行 |
| **2026-04-01** | commit | **迭代 11**：**P5.2.2** **`RunTurnLoop`** 内 **`SNIP_COMPACT`** 独立阈值 + **`EventKindSnipCompactApplied`**；**`messages.StripHistorySnipPieces`** + 测；**`CACHED_MICROCOMPACT`** 请求体 **`anthropic_beta`**（占位）；**`make test-phase5`** 含 **`internal/messages`**。 | PARITY Follow-on 余项（token 估计、F.9 agent、metadata 等） |
| **2026-04-01** | 多 commit | **迭代 12**：**PHASE05_CONTINUATION** 第 **4–14** 项可测子集（token 启发式与附件字节、reactive min tokens、SESSION_RESTORE、UserSubmit mode tags、**`context break-cache`** CLI、模板目录 markdown、cache-break 后 compact 建议、**`LoopState`** 扩展字段、**`BashExecToolRunner`**、**`FindRelevantMemoryPaths`**、**`compact`/`ExecuteStubWithMeta`** 等）；**`make test-phase5`** / **`make test-phase4`** 通过。 | 见 **PHASE05_CONTINUATION.md** 与 PARITY **Follow-on**（全量 `src/` 仍可有差距） |
| **2026-04-01** | `docs` | **迭代 13**：**`PHASE05_CONTINUATION.md`** 增加 **全量还原推荐顺序**（**H6→H1→H2/H3/H4→H5/H7/H8→H9**，再 **T1→T2→T3** 与 **T4/T5** 穿插）；**`PARITY_PHASE5_DEFERRED.md`**、**§5** 引用。 | 按该顺序推进 headless 再 TUI |
| **2026-04-01** | commit | **迭代 14（H6 启动）**：**`query.LoopState`** 对齐 **`query.ts` `State`** 子集（**`LoopContinue`**、**`AutoCompactTracking`**、**`maxOutputTokensOverride`**、**`pendingToolUseSummary`**）；**`RunTurnLoop`** 在工具轮后 **`ContinueReasonNextTurn`**；单测 **`loop_h6_test`**。 | H6 余量见 **PHASE05_CONTINUATION.md** §H6 进度 |
| **2026-04-01** | commit | **迭代 15（H6 续）**：**`engine`** 在恢复 / compact / cache-break / post-loop advisor 路径 **`RecordLoopContinue`**；**`resetLoopStateForRetryAttempt`** + **`CloneAutoCompactTracking`**；**`ContinueReasonSubmitRecoverRetry`**、**`AutoCompactExecuted`**；**`engine`/`query`** 单测。 | 见 **PHASE05_CONTINUATION.md** §H6 进度 |
| **2026-04-01** | commit | **迭代 16（H6 续）**：**`ApplyTransition(TranStartCompact)`** 填充 **`AutoCompactTracking`**；**`engine`** **`ContextCollapseDrain`** / **`StopHookBlockingContinue`** / **`TokenBudgetContinueAfterTurn`** + **`run_attempts.go`**；**`PrepareLoopStateFor*Continuation`**；单测覆盖 drain / 双轮 Submit。 | **`messages`/`toolUseContext`** 入 **`LoopState`**；drain 后全链重发仍 defer |
| **2026-04-01** | commit | **迭代 17（H6 续）**：**`query.RunTurnLoopFromMessages`**；**`executeRunTurnLoopAttempts`** 在 **collapse drain + `RecoverStrategy`** 重试时用 trim 后的 transcript 种子；单测 **`ContextCollapseDrain_recoverRetryUsesDrainedSeed`**。 | **`messages`/`toolUseContext`** 入 **`LoopState`**；reactive-compact 后重发种子仍 defer |
| **2026-04-01** | commit | **迭代 18（H6 续）**：**`LoopState.MessagesJSON`** + **`ToolUseContextMirror`**；**`LoopDriver`/`engine.Config`** **`AgentID`**/**`NonInteractive`**；**`RunTurnLoop`** 同步 **`SetMessagesJSON`**；**`resetLoopStateForRetryAttempt`** 携带 transcript；**`ApplyTransition`** 语义与表测 **`reflect.DeepEqual`**。 | **`ToolUseContext`** 全量；stopHook 流式；compact 重试种子 |
| **2026-04-01** | commit | **迭代 19（H6 续）**：**`CompactExecutor`** 第二返回值 **`nextTranscriptJSON`**；**`executeRunTurnLoopAttempts`** 与 **drain** 优先级；**`compact` stub** 签名；单测 **`RecoverStrategy_compactNextTranscriptSeedsRetry`**、**`drainThenCompact_compactNextWinsOnRetrySeed`**。 | **`ToolUseContext`** 全量；stopHook 流式 |
| **2026-04-01** | commit | **迭代 20（H6 headless 完成）**：**`StopHooksAfterSuccessfulTurn`**、**`ContinueReasonStopHookPrevented`**；**`ToolUseContextMirror`** **`SessionID`**/**`Debug`**/**`AbortSignalAborted`**；**§2 P5.1.1→`[x]`**。 | TS **`ToolUseContext`** 全量 / 磁盘 stopHooks 链见 **PARITY** |
| **2026-04-01** | commit | **迭代 21（§3.0 计划项 1）**：**`app.QuitRuntime`** / **`app.FailBootstrap`**；**`cmd/rabbit-code`** 主路径不再 **`os.Exit` 跳过 `defer rt.Close()`**；**`PrintBootstrapFailure`** 委托 **`FailBootstrap(nil,·)`**；对齐 **H8** 退出清理与 **`RegisterEngineShutdown`** 前提。 | **§3.0 计划** 项 2（Engine 宿主注册）见 **PHASE05_CONTINUATION.md** |
| **2026-04-01** | commit | **迭代 22（memdir §3.0 / §3.2）**：**`internal/memdir/MEMDIR_TS_PARITY.md`** 写入 **§3.0 有序计划** 与 **§3.2 单次核对**；**`FindRelevantMemoriesClassic`**（**`findRelevantMemories.ts`** 形参薄委托）+ 单测；**`PHASE05_CONTINUATION.md`** H8 子计划指针；**`doc.go`**；**`go build ./...`**、**`go test ./... -short`**。 | **MEMDIR_TS_PARITY** §3.0 序 **2–4**（`memoryScan` / `memdir` / `paths`） |

（后续行：每完成可合并条目追加一行。）
