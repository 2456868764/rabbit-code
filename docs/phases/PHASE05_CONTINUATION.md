# Phase 5 待深化清单（对照 SPEC §4「部分」与 PARITY Follow-on）

主清单 **§2** 已全 `[x]`；下列 **4–14** 已在 **迭代 12** 落地 **可测子集**（每步单独 commit）；与 `src/` **全量**语义之差仍见 **PARITY_PHASE5_DEFERRED.md** Follow-on。

### §3.0 当前有序迭代计划（对照 `PHASE_ITERATION_RULES.md` §三-3.0）

按顺序执行；完成一项则更新本段勾号与 **PHASE05_SPEC_AND_ACCEPTANCE.md §6**。

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`cmd/rabbit-code`**：`Bootstrap` 成功后凡退出路径经 **`app.QuitRuntime`** / **`app.FailBootstrap`**，确保 **`Runtime.Close`**（含未来 **`RegisterEngineShutdown`**）在 **`os.Exit` 前执行 | **`go test ./internal/app/... ./cmd/rabbit-code/... -short`**；手测 **`RABBIT_CODE_EXIT_AFTER_INIT=1`** 仍 0 |
| 2 | ☑ | **H8 接线**：**`Bootstrap`** 成功后 **`app.WireHeadlessEngineForShutdown`**（最小 **`engine.New(parent, nil)`** + **`RegisterEngineShutdown`**）；**`cmd/rabbit-code`** 主路径在 **`QuitRuntime`** 前经 **`Runtime.Close`** drain | **`go test ./internal/app/... ./cmd/rabbit-code/... -short`** |
| 3 | ☑ | **PARITY `[~]` 文档扫尾**：**`QueryEngineConfig`** 字段级映射（**§4**）、**`cost-tracker`/`accumulateUsage`**（**§5** → **`internal/cost`**）、**JSONL Map**（**§6** → Phase 8）— 全量单 struct / USD 聚合 / JSONL 仍 Follow-on | **`PARITY_QUERY_QUERYENGINE.md`** §4–§7 已更新 |

### §3.0 H9 子计划（Headless 表 **行 9**：Bash / 权限）

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`BashExecToolRunner`** null 字节拒绝 + **`PARITY_H9_BASH_PERMISSIONS.md`** + H9 进度段 | **`go test ./internal/query/... -short`** |
| 2 | ☑ | **`readOnlyCommandValidation` / `readOnlyValidation`** ↔ **`IsExtractReadOnlyBash`**（**git** 子集扩展、**NUL** 拒绝）；**`pathValidation`** / **`BashExec`** 对照见 **PARITY_H9** **§4** | **`go test ./internal/memdir/... -short`** |
| 3 | ☑ | **`canUseTool` / 孤儿** ↔ **`OrphanPermissionError`**、**`OrphanPermissionAdvisor`**、**`EventKindOrphanPermission`**（**PARITY_H9** **§5**）；全量 **`ToolRunner`/`canUseTool`** 仍 **DEFERRED** | 文档 + **`go test ./internal/engine/... -short`** |

### §3.0 T1 子计划（**TUI 表行 A**：`thinking` / `processUserInput` / 系统块）

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`thinking.ts`** → **`internal/utils/thinking`**；**ultrathink** 关键词与 **`RABBIT_CODE_ULTRATHINK`** 在 **`EventKindUserSubmit.PhaseDetail`** 与 **`ApplyUserTextHints`** 中**或**关系 | **`go test ./internal/utils/thinking/... ./internal/query/engine/... -short`** |
| 2 | ☑ | **`processUserInput`** headless：**`user_prompt_keywords`**、**`PlainPromptSignals`**；**`Config.TruncateProcessUserInputHookOutput`**；全量 slash/附件/**`processTextPrompt`** 仍 TUI | **`PARITY_T1_THINKING_PROCESSUSERINPUT.md`** |
| 3 | ☑ | **`InterleavedAPIContextManagementOpts`** + **`ApplyEngineCompactIntegration`** 默认 **`APIContextManagementOpts`**；展示层仍 **H4** / T3 | **`PARITY_T1_THINKING_PROCESSUSERINPUT.md`** |

### §3.0 T2 子计划（**TUI 表行 B**：`context` 类子命令）

| 序 | 状态 | 项 | 验收 |
|----|------|-----|------|
| 1 | ☑ | **`internal/commands/contextcmd`**：**`rabbit-code context`** 路由 **`help`**、**`break-cache`** | **`go test ./internal/commands/contextcmd/... -short`** |
| 2 | ☑ | **`context report`**：**stdin** Messages JSON + **`query.BuildHeadlessContextReport`** JSON | **`PARITY_T2_CONTEXT_CLI.md`** |
| 3 | ☐ | Markdown 全表 / **`analyzeContextUsage`** 深度 parity；**`context.tsx`** 网格仍 TUI | **T2/T3** Follow-on |

---

| # | 项 | 说明 | 目标 Phase / 状态 |
|---|----|------|-------------------|
| 1 | **P5.2.2 SnipCompact 接循环** | `RunTurnLoop` + **`EventKindSnipCompactApplied`** | **已完成**（迭代 11） |
| 2 | **P5.F.10 滚动条** | **`messages.StripHistorySnipPieces`** | **已完成**（迭代 11） |
| 3 | **P5.F.8 请求体 beta** | 占位 **`anthropic_beta`** | **已完成**（迭代 11） |
| 4 | **P5.F.1 token 估计** | **`query.EstimateUTF8BytesAsTokens`**；**`RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_TOKENS`**、**`_MAX_ATTACHMENT_BYTES`**（memdir 原始注入字节） | **已完成**（迭代 12 子集；真 tokenizer / TUI 仍 defer） |
| 5 | **P5.F.2 analyzeContext** | **`query.ReactiveCompactByTranscript`** + **`RABBIT_CODE_REACTIVE_COMPACT_MIN_TOKENS`**；**`CompactAdvisor(transcript []byte)`** | **已完成**（迭代 12 子集） |
| 6 | **P5.F.3 sessionRestore** | **`RABBIT_CODE_SESSION_RESTORE`** 用户文提示；coordinator / 工具链仍 defer | **已完成**（迭代 12 headless） |
| 7 | **P5.F.4 / F.5 + TUI** | **`EventKindUserSubmit.PhaseDetail`** = **`query.FormatHeadlessModeTags`** | **已完成**（迭代 12 headless；`thinking.ts` / TUI 仍 defer） |
| 8 | **P5.F.6 CLI** | **`rabbit-code context break-cache`** → **`internal/commands/breakcache`** JSON（对齐 **`src/commands/break-cache`**） | **已完成**（迭代 12） |
| 9 | **P5.F.7 模板正文** | **`RABBIT_CODE_TEMPLATE_DIR`** / **`Config.TemplateDir`** + **`query.LoadTemplateMarkdownAppendix`**；磁盘 **stopHooks** 仍 defer | **已完成**（迭代 12 子集） |
| 10 | **P5.F.9 + compact** | **`RABBIT_CODE_PROMPT_CACHE_BREAK_SUGGEST_COMPACT`** → 成功后 **reactive compact suggest**（迭代 12）；**SSE** 路径 **trim / 重发 / compact 恢复** 的 headless 子集见 **H1 进度**（与 suggest 正交） | **已完成**（迭代 12 + H1 子集） |
| 11 | **P5.1.1 query State** | **`LoopState`** 增 **`MaxOutputTokensRecoveryCount`** 等；transition 行为未全镜像 | **已完成**（迭代 12 子集） |
| 12 | **bash** | **`querydeps.BashExecToolRunner`** + **`RABBIT_CODE_BASH_EXEC`** | **已完成**（迭代 12 桥接；权限栈 Phase 6） |
| 13 | **memdir findRelevant** | **`memdir.FindRelevantMemoryPaths`**（启发式，无 LLM） | **已完成**（迭代 12 子集） |
| 14 | **compact 链** | **`compact.ReactiveSuggestFromTranscript`**、**`ExecuteStubWithMeta`** / **`FormatStubCompactSummary`** | **已完成**（迭代 12 子集） |

---

## 全量还原推荐顺序（`src/` 对齐）

**原则**：先 **headless 引擎**（`engine` / `query` / `querydeps` / `compact` / `memdir` / `anthropic`），再 **TUI/REPL**。下列编号与对话约定 **H*** / **T*** 一致；实施时仍按 **PARITY_PHASE5_DEFERRED.md** Follow-on 与 **§4** 路径对照。

### Headless

| 阶段 | 代号 | 内容 | 主要 `src/` 参考 |
|------|------|------|------------------|
| 1 | **H6** | **`query.ts` `State` 全字段 + transition / continue 语义** 与 Go **`LoopState`**、**`ApplyTransition`** 对齐 | **`headless 已收口`**（迭代 20）：见下 **H6 进度**；TS **`ToolUseContext`** 全对象见 **PARITY_PHASE5_DEFERRED.md** |
| 2 | **H1** | **Prompt cache break** 后 **trim / 重发 / compact 协同**（不仅是 suggest）：Go **可测子集**已接（**`RABBIT_CODE_PROMPT_CACHE_BREAK_*`**，见下 **H1 进度** 与模块根 **README** `Config` 段）；`src/` 全量仍对照 TS | `services/api/promptCacheBreakDetection.ts`、**`internal/query` / `internal/engine` / `internal/features/rabbit_env.go`**、compact 调用链 |
| 3 | **H2** | **全量 `analyzeContext` + auto/reactive compact 触发** | `utils/analyzeContext.ts`、`services/compact/autoCompact.ts` |
| 4 | **H3** | **`services/compact/*` 全链路**（与 H2 强相关） | `services/compact/*` |
| 5 | **H4** | **Micro-compact / cache 请求体**（正式 beta、字段与上游一致） | `services/compact/microCompact.ts`、`query.ts` |
| 6 | **H5** | **Token / 附件预算全量**（API tokenizer、附件计入；引擎侧先落地） | `query.ts`、`utils/attachments.ts`；进度见下 **§H5** |
| 7 | **H7** | **Snip 元数据与持久化**（UUID map、会话往返） | **headless 子集已落地**（见下 **§H7**）；JSONL Map+parentUuid 全量仍 PARITY |
| 8 | **H8** | **memdir 全量**（含 LLM 选记忆、TEAMMEM、extract fork 等） | `memdir/*`、`findRelevantMemories.ts`、`services/extractMemories/*`；**子集已落地**，**缺口见下 §H8 仍待实现** |
| 9 | **H9** | **Bash / 权限真实栈**（Phase 6 工具层） | bash / permissions 与上游工具栈；**§3.0 子计划** **`PARITY_H9_BASH_PERMISSIONS.md`** |

#### H9 进度（Bash / 权限；Phase 6 headless 桥）

- **`RABBIT_CODE_BASH_EXEC`**：**`query.BashExecToolRunner`**（**`sh -c`**，**`command` / `cmd`** JSON）；未开启时 **`BashStubToolRunner`**（**`PARITY_PHASE5_DEFERRED` P5 Tools**）。
- **卫生**：命令串 **null 字节**拒绝（**`BashExecToolRunner`** + **`IsExtractReadOnlyBash`**；**`PARITY_H9_BASH_PERMISSIONS.md` §3.0 序 1–2**）。
- **Extract**：**`memdir.IsExtractReadOnlyBash`** — **`readOnlyCommandValidation` / `readOnlyValidation`** 保守子集（**git** **`stash list` / `remote` / `config --get`** 等；**§4**）。
- **孤儿权限**：**`query.OrphanPermissionError`**、**`engine.Config.OrphanPermissionAdvisor`** → **`EventKindOrphanPermission`**（**§5**）。
- **全量**：**`src/tools/BashTool/*`**（**pathValidation**、**bashPermissions**、**sandbox**、全量 **`canUseTool`**）仍 **Phase 6** / **PARITY_QUERY**；见 **`PARITY_H9_BASH_PERMISSIONS.md` §3.0**。

#### H6 进度（迭代 14 起）

- **`LoopState`**：`LoopContinue`（对应 **`transition`**）、**`AutoCompactTracking`**、**`MaxOutputTokensOverride*`**、**`PendingToolUseSummary`**；常量 **`ContinueReason*`** 与 `query.ts` continue 站点对齐。
- **`RunTurnLoop`**：在追加 **tool_result** 后进入下一轮前 **`RecordLoopContinue(..., ContinueReasonNextTurn)`**。
- **`engine`（迭代 15）**：**`RecoverStrategy`** 第二次 **`RunTurnLoop`** 前 **`resetLoopStateForRetryAttempt`** 保留 H6 字段；可恢复错误路径 **`ContinueReasonReactiveCompactRetry`**（compact executor 成功）、**`ContinueReasonSubmitRecoverRetry`** / **`ContinueReasonMaxOutputTokensRecovery`**（即将重试）；成功后 **cache-break** 与 **post-loop compact** 成功执行时写入 **`LoopContinue`**（reactive / **`ContinueReasonAutoCompactExecuted`**）。
- **`ApplyTransition`（迭代 16）**：**`TranStartCompact`** 写入 **`AutoCompactTracking`**（**`TurnID`** = `autocompact:<n>`，与 **`autoCompact.ts`** 记账对齐）。
- **`engine`（迭代 16）**：可选 **`Config.ContextCollapseDrain`**（**`CONTEXT_COLLAPSE`** + **`prompt_too_long`** 时 trim + **`ContinueReasonCollapseDrainRetry`**，供同轮 compact 使用）；**`StopHookBlockingContinue`** / **`TokenBudgetContinueAfterTurn`** 在成功后按 **`query.ts`** 顺序（先 stop-hook 语义，再 token budget）可触发额外 **`RunTurnLoop`**（**`PrepareLoopStateForStopHookBlockingContinuation`** / **`PrepareLoopStateForTokenBudgetContinuation`**）；**`executeRunTurnLoopAttempts`**（**`run_attempts.go`**）抽取原重试循环。
- **迭代 17**：**`RecoverStrategy`** 重试且本轮 **`ContextCollapseDrain`** 已提交时，下一轮 **`RunTurnLoopFromMessages`** 使用 drain 后的 **`msgs`**（对齐 **`query.ts`** `collapse_drain_retry` → continue 携带 **`drained.messages`**）。
- **迭代 18**：**`LoopState.MessagesJSON`** 镜像 **`query.ts` `state.messages`**（**`RunTurnLoop`** 每次变更 transcript 时同步）；**`ToolUseContextMirror`**（**`AgentID`** / **`MainLoopModel`** / **`NonInteractive`** / **`QueryChainID`** / **`QueryDepth`** 占位）+ **`engine.Config.AgentID`**、**`NonInteractive`** → **`LoopDriver`**；**`resetLoopStateForRetryAttempt`** 保留 **`MessagesJSON`** 与 **`ToolUseContext`**；**`ApplyTransition`** 不修改二者（文档 + 单测）。
- **迭代 19**：**`CompactExecutor`** 扩展为 **`(summary, nextTranscriptJSON, err)`**；可恢复错误路径上 **`RecoverStrategy`** 重试时若 **`nextTranscriptJSON` 非空** 则优先于 **drain-only** 作为 **`RunTurnLoopFromMessages`** 种子（**`compact.ExecuteStub`** / **`ExecuteStubWithMeta`** 返回 **`nil`** 第二值保持旧行为）。
- **迭代 20（H6 headless 收口）**：**`ContinueReasonStopHookPrevented`**；**`Config.StopHooksAfterSuccessfulTurn`**（**`StopHookAfterTurnFunc`**：`preventContinuation` → **`EventKindDone`** + **`PhaseDetail`**，**`blockingContinue`** 对齐 **`stop_hook_blocking`**，顺序在 **`TokenBudgetContinueAfterTurn`** 之前）；**`ToolUseContextMirror`** 增 **`SessionID`**、**`Debug`**、**`AbortSignalAborted`**；**`engine.Config.SessionID`** / **`Debug`**；与 **`StopHookBlockingContinue`** 合并阻塞继续语义；**`RunTurnLoop`** 每轮开端重置 **`AbortSignalAborted`**，**`ctx` 已取消**时置 **`true`**（**`feat(phase5/h6)`** 单测覆盖）。
- **非 H6（PARITY）**：TS **`ToolUseContext`** 全量（AppState、MCP、完整 **`abortController`** 等）与磁盘 **`stopHooks` / `handleStopHooks`** 工具执行链 — 见 **`PARITY_PHASE5_DEFERRED.md`** Follow-on。

#### H1 进度（headless 子集；与 **README** § Headless engine、`feat(phase5/h6)` 前续提交一致）

- **Env**（**`internal/features/rabbit_env.go`**）：**`RABBIT_CODE_PROMPT_CACHE_BREAK_DETECTION`**；**`RABBIT_CODE_PROMPT_CACHE_BREAK_TRIM_RESEND`**（默认开启 strip + 重试 **`AssistantTurn`** 一次，设 **`0`** 关闭）；**`RABBIT_CODE_PROMPT_CACHE_BREAK_AUTO_COMPACT`** 且 **`Config.CompactExecutor`** 返回 **`nextTranscriptJSON`** 时走 compact 恢复；与迭代 12 的 **`RABBIT_CODE_PROMPT_CACHE_BREAK_SUGGEST_COMPACT`**（post-loop suggest）正交。
- **`query`**：**`assistantTurnWithPromptCacheBreakHandling`**、**`StripCacheControlFromMessagesJSON`**（**`internal/query/transcript_strip_cache.go`**）；**`ContinueReasonPromptCacheBreakTrimResend`** / **`ContinueReasonPromptCacheBreakCompactRetry`**；**`LoopDriver.PromptCacheBreakRecovery`**（**`engine`** 在 **`run_attempts.go`** 中注入 **`promptCacheBreakCompactRecovery`**）。
- **`engine`**：**`EventKindPromptCacheBreakDetected`**（**`PhaseDetail`**: **`sse`**）；**`EventKindPromptCacheBreakRecovery`**（**`PhaseDetail`**: **`trim_resend`** / **`compact_retry`**）；**`query.LoopObservers.OnPromptCacheBreakRecovery`** → **`EngineEvent`** 总线。
- **H1 仍相对 `src/` 全量的缺口（可后续迭代）**：多轮 SSE 细粒度对齐、与 **Messages** 非-Turn 路径的完全统一、上游 **promptCacheBreakDetection.ts** 的边角字符串与遥测字段 1:1 等 — 见 **PARITY** Follow-on。
- **本迭代已补齐（单测覆盖）**：**strip** 遇非法 / 空 **`messages` JSON** 时**直接返回 strip 错误**（不再静默落入 compact）；**trim 重发后仍 cache break → compact 种子 → 第三次 `AssistantTurn` 成功** 全链（**`TestRunTurnLoop_promptCacheBreak_trimThenCompact_chain`**）；**`StripCacheControlFromMessagesJSON`** 空输入与非法 JSON（**`transcript_strip_cache_test.go`**）；**`tool_use` 块同级 `cache_control` 递归剔除**（**`TestStripCacheControlFromMessagesJSON_nestedInToolInput`**）；**同一 `AssistantTurn` 波次内至多两轮 compact 恢复**（**`maxPromptCacheBreakCompactRounds`**，**`TestRunTurnLoop_promptCacheBreak_secondCompactRound`**）；**SSE cache break 启发式扩展**（**`IsPromptCacheBreakStreamJSON`**，`prompt_cache_key` / `cached…block…invalid` / `ephemeral…cache…stale`）；**`AnthropicAssistant` 从 `ContextWithOnPromptCacheBreak` 注入 `readOpts`**（**`StreamAssistant`**：**`TestAnthropicAssistant_StreamAssistant_promptCacheBreakFromContext`**；**`AssistantTurn`**：**`TestAnthropicAssistant_AssistantTurn_promptCacheBreakFromContext`**）；**`engine` 双次 `compact_retry` 事件 + 双次 `CompactExecutor`**（**`TestEngine_promptCacheBreak_twoCompactRetry_events`**）。

#### H2 / H3 / H4 进度（相对上表 **Headless** 行 3–5 的可测子集）

| 代号 | 全量目标（`src/`） | 本轮 Go 增量 |
|------|-------------------|-------------|
| **H2** | **`analyzeContext` + auto/reactive compact 触发** | 上列 + **`query.ProactiveAutoCompactSuggested`** / **`query.EffectiveContextInputWindow`** / **`AutoCompactThresholdTokens`** / **`CalculateTokenWarningState`**（对照 **`autoCompact.ts`** 阈值与 **`calculateTokenWarningState`**）；**`features`**：**`IsAutoCompactEnabled`**、**`RABBIT_CODE_DISABLE_*` / `RABBIT_CODE_AUTO_COMPACT`**、**`RABBIT_CODE_CONTEXT_WINDOW_TOKENS`**、**`RABBIT_CODE_AUTO_COMPACT_WINDOW`**、**`RABBIT_CODE_SUPPRESS_PROACTIVE_AUTO_COMPACT`** 等；**`engine`** post-loop 在无 **`CompactAdvisor`** 时亦可因转录超阈触发 **`SuggestAutoCompact`**（**`TestEngine_ProactiveAutoCompact_suggestWithoutAdvisor`**）；**`compact.TranscriptProactiveAutoSuggest`** 封装；**`query.ReactiveCompactByTranscript`** 在 **`RABBIT_CODE_DISABLE_COMPACT`** 时为 false（对齐 **`autoCompactIfNeeded`** 入口早退）。 |
| **H3** | **`services/compact/*` 全链路** | **`AfterSuccessfulCompactExecution`**；**`ExecutorPhaseAfterSchedule`**（pending→**`executing`**，传入 **`CompactExecutor`**）；**`ResultPhaseAfterCompactExecutor`**（成功→**`idle`**，失败保留 **executing**）；**`engine` / `run_attempts`** 的 **Suggest** 仍报 **pending**，**Result** 报 **idle/executing**（**`TestExecutorPhaseAfterSchedule`**、**`TestResultPhaseAfterCompactExecutor`**）；**`query.MirrorAutocompactConsecutiveFailures`**：**`Engine`** 跨 **Submit** 的 **`autoCompactConsecutiveFailures`** 在每次 proactive auto **executor** 结果后写入 **`LoopState.AutoCompactTracking.ConsecutiveFailures`**（对齐 **`autoCompact.ts`** `tracking`，便于会话/观测；断路器仍以 **Engine** 计数为准）。 |
| **H4** | **Micro-compact / 请求体 beta 与上游一致** | **`EventKindCachedMicrocompactActive.PhaseDetail`** = **`BetaCachedMicrocompactBody`**；**httptest** 校验 POST JSON **`anthropic_beta`**（**`TestAnthropicAssistant_streamBody_anthropicBetaCachedMicrocompact`**）；**`TestCompactableToolNames_matchMicroCompactTS`** 防 **`COMPACTABLE_TOOLS`** 与 **`microCompact.ts`** 漂移。 |

**H2/H3/H4 仍 defer**：TS **`analyzeContext.ts`** 全量（API tokenizer、MCP/Skills 分解、网格 UI）、**`microCompact.ts`** 全量、**`autoCompact.ts`** 内 **session memory 优先 / 电路条 / fork querySource** 等、非 headless UI、Bedrock extra-body 差异 — 见 **PARITY**。

#### H5 功能列表（Phase 5 headless；逐项核对 `PARITY_PHASE5_DEFERRED`）

| # | 项 | 说明 | 状态 |
|---|----|------|------|
| H5.1 | **提交文本 token 估计模式** | **`RABBIT_CODE_TOKEN_SUBMIT_ESTIMATE_MODE`**：`bytes4`（默认）或 **`structured`**（resolved 为 Messages JSON 数组时用 **`EstimateMessageTokensFromTranscriptJSON`**） | **[x]** `features.SubmitTokenEstimateMode`、`query.EstimateResolvedSubmitTextTokens` |
| H5.2 | **附件原始字节计入 MAX_INPUT_TOKENS** | memdir 注入字节按 **⌈raw/4⌉** 与 resolved 文本估计**相加**后对比 **`RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_TOKENS`**（**`MAX_ATTACHMENT_BYTES`** 仍为独立硬上限） | **[x]** `query.EstimateSubmitTokenBudgetTotal`、`engine.runTurnLoop` |
| H5.3 | **预算遥测事件** | **`EventKindSubmitTokenBudgetSnapshot`**：`PhaseAuxInt`=合计估计 token，`PhaseAuxInt2`=inject 原始字节，`PhaseDetail`=mode（供 T3 meter / 遥测） | **[x]** |
| H5.4 | **Anthropic API tokenizer** | 请求前与上游一致的 **count_tokens**（或等价 SDK）；需密钥 / 网络 | **[x]** `RABBIT_CODE_TOKEN_SUBMIT_ESTIMATE_MODE=api` + `CountMessagesInputTokens` + fallback |
| H5.5 | **用户消息 token budget 解析 + continuation** | `utils/tokenBudget.ts` **`parseTokenBudget`**、`query/tokenBudget.ts` **`checkTokenBudget`**（+500k 续写、递减回报停）与引擎 **`TokenBudgetContinueAfterTurn`** 全量对齐 | **[x]** headless：`ParseTokenBudget` / `CheckTokenBudget`、内置续跑 + 事件；无预算时仍可用回调 |
| H5.6 | **附件类型全量** | `utils/attachments.ts` 图片缩放、file_ref、MCP 资源等计入（非仅 memdir 原始字节） | **[x]** 子集：`EstimateMessageTokensFromTranscriptJSON` 对 image/document base64 启发式加强 |

#### H7 功能列表（Snip 元数据与持久化；Messages 数组 headless 模型）

| # | 项 | 说明 | 状态 |
|---|----|------|------|
| H7.1 | **稳定 Snip ID** | 每次 history_snip / snip_compact 前缀裁剪生成 **`NewSnipRemovalID`**（32 hex） | **[x]** |
| H7.2 | **结构化元数据** | **`query.SnipRemovalEntry`**（kind、removedMessageCount、bytesBefore/After）追加到 **`LoopState.SnipRemovalLog`** | **[x]** |
| H7.3 | **侧车 JSON** | **`MarshalSnipRemovalLogJSON` / `UnmarshalSnipRemovalLogJSON`** 供会话保存 | **[x]** |
| H7.4 | **重放往返** | **`ReplaySnipRemovals`**：对「完整前缀」转录按序执行 **`SnipDropFirstMessages`**，与运行时裁剪一致 | **[x]** |
| H7.5 | **引擎与事件** | **`EventKindHistorySnipApplied` / `SnipCompactApplied`** 填 **`EngineEvent.SnipID`**；**`Engine.SnipRemovalLogForPersistence`**；**`Config.RestoredSnipRemovalLog`** 注入下一轮 **`LoopState`** | **[x]** |
| H7.6 | **中区删除重放** | **`SnipRemovalEntry.removedIndices`** + **`snipRemoveMessageIndices`**；**`ReplaySnipRemovalsEx`** 按条重放（与仅前缀互补） | **[x]** |
| H7.7 | **removedUuids 互操作** | JSON 字段 **`removedUuids`**；**`ReplaySnipRemovalsEx` + `SnipReplayOptions.UUIDToIndex`** 重放（宿主提供 message UUID→下标映射） | **[x]** |
| H7.8 | **日志合并** | **`MergeSnipRemovalLogs`**（按 **`id`** 去重追加） | **[x]** |
| H7.9 | **转录内 UUID 侧车** | 默认键 **`rabbit_message_uuid`**：**`BuildUUIDToIndexFromMessagesJSON`**、**`StripMessageFieldFromTranscriptJSON`**、**`ReplaySnipRemovalsAuto`**（无需宿主单独传 `UUIDToIndex`） | **[x]** |
| H7.10 | **UUID 写入转录** | **`AnnotateTranscriptWithUUIDs`**（与消息条数对齐）；**`TranscriptMessageCount`** | **[x]** |
| H7.11 | **多字段剥离** | **`StripMessageFieldsFromTranscriptJSON`**（API 前批量去掉多个侧车键） | **[x]** |

**仍属 PARITY / 非本仓库 headless 范围**：TS **`sessionStorage.ts`** 的 **`Map<UUID, TranscriptMessage>`** 整表加载、**`parentUuid`** 自动重链、JSONL 追加存储与 **SnipTool** 运行时中区执行 — 需 **Phase 8 / 会话层**；Go 已提供 **前缀/索引/`removedUuids` 重放**、**侧车 JSON** 与 **H7.9 转录内 UUID** 对接点。

#### H8 进度（memdir 全量 headless 子集）

- **`memdir.ScanMemoryFiles`**：递归 **`.md`**、跳过 **`MEMORY.md`**、前 **30** 行 **`description:` / `type:`**、**`MaxMemoryFiles`**、按 **mtime** 新→旧。
- **`FormatMemoryManifest`**、**`FindRelevantMemories`**：**`heuristic`**（与 **`FindRelevantMemoryPaths`** 同套 token 重叠）与 **`llm`**（侧向 **`TextComplete`** + **`ParseSelectedMemoriesJSON`**；失败回退启发式）；**`RecentTools`** / **`AlreadySurfaced`**。
- **`RABBIT_CODE_MEMDIR_RELEVANCE_MODE`**（**`heuristic`** | **`llm`** / **`side_query`**）；**`engine.Config`**：**`MemdirMemoryDir`**、**`MemdirRecentTools`**、**`MemdirTextComplete`**、**`MemdirRelevanceModeOverride`**、**`MemdirAlreadySurfaced`**；每轮 **`memdirPathsForSubmit`**，注入成功后累积 **`memdirSurfaced`**；LLM 路径默认 **`PostMessagesStreamReadAssistant`**（system+user 合并为单条 user）。
- **`paths.ts` `autoMemoryDirectory`（可信来源）**：**`config.LoadTrustedAutoMemoryDirectory`**（policy → **`RABBIT_CODE_SETTINGS_JSON`** → **`.rabbit-code.local.json`** → **`config.json`**；**不读** project **`.rabbit-code.json`**）；**`memdir.ResolveAutoMemDirWithOptions`** 与 **`Config.MemdirTrustedAutoMemoryDirectory`**；**`RABBIT_CODE_AUTO_MEMDIR`** 与 trusted 二选一即可在 **`AutoMemoryEnabled`** 下走自动解析。
- **单测**：**`memory_scan_test`**、**`find_relevant_memories_test`**、**`select_llm_test`**、**`engine` MemdirMemoryDir + surfaced**、**`auto_memory_settings_test`**、trusted-only memdir inject。
- **§3.0 子计划（memdir `[~]` 扫尾顺序）**：**`internal/memdir/MEMDIR_TS_PARITY.md`** §3.0，执行方式同 **`PHASE_ITERATION_RULES.md`** §三（**3.0 + 3.2**）。
- **H8 续（prompt / TEAMMEM / extract fork）**：**`promptdata/*.txt` + `promptembed.go`**；**`BuildCombinedMemoryPrompt`**、**`BuildSearchingPastContextSection`**（**`RABBIT_CODE_MEMORY_SEARCH_PAST_CONTEXT`**）；**`BuildExtractAutoOnlyPrompt` / `BuildExtractCombinedPrompt`**；**`RABBIT_CODE_TEAMMEM`**、**`TeamMemoryEnabledFromMerged`**、**`team_mem.go`**（**`team/`** 路径、**`SanitizeTeamMemPathKey`**）；**`ExtractController` + `RunForkedExtractMemory`**、**`AutoMemToolRunner`**、transcript 上 **`HasMemoryWritesSince` / `CountModelVisibleMessagesSince`**；**`query.QuerySourceExtractMemories`**；引擎 **stop hook 异步 extract**、**`DrainExtractMemories`**、可选 **`Config.ExtractMemoriesSaved`**；相关 env：**`RABBIT_CODE_EXTRACT_MEMORIES`**、**`_NON_INTERACTIVE`**、**`_INTERVAL`**、**`_SKIP_INDEX`**。
- **H8 续二（系统块 / 守卫 / fork 对齐，Go 已落地）**：
  - **`LoadMemorySystemPrompt`**（**`memdir.ts` `loadMemoryPrompt`**  analogue）：**auto-only / TEAMMEM 合并 / KAIROS daily-log** 分支（**`KairosDailyLogMemoryEnabled`** 优先于 team）；**`AppendClaudeMdStyleMemoryEntrypoints`**（私享 + team **MEMORY.md** 截断正文）；**`RABBIT_CODE_MEMORY_SYSTEM_PROMPT`** 关闭注入；**`CoworkMemoryExtraGuidelineLines`**（**`RABBIT_CODE_` / `CLAUDE_COWORK_MEMORY_EXTRA_GUIDELINES`**）。
  - **引擎**：每轮 **`runTurnLoop`** 开头 **`refreshMemorySystemPromptForAssistant`** → **`AnthropicAssistant.SystemPrompt`**；**`MemdirProjectRoot`**（**`engineMemdirProjectRoot`**）供 prompt 内项目根；**`MessagesStreamBody.system`**（**`internal/services/api/client.go`** + **`querydeps/anthropic_assistant.go`**）。
  - **TEAMMEM**：**`SanitizeTeamMemPathKey` + Unicode NFKC**（**`golang.org/x/text/unicode/norm`**）；**`ValidateTeamMemWritePath`**（目标文件可不存在时对 **父目录** **`EvalSymlinks`**，缓解 macOS **`/var` vs `/private/var`**）；**`TeamMemSecretGuardRunner` + `ScanTeamMemorySecrets`**（gitleaks 式规则 **子集**）：**`Engine.New`** 在 **TEAMMEM + memdir 已解析** 时包裹 **`Deps.Tools`**；**`RunForkedExtractMemory`** 在 team 开启时在 **`AutoMemToolRunner` 内层** 再包裹同一守卫。
  - **Extract**：**`AutoMemToolRunner`** 放行 **`REPL`**；只读 **bash** 扩展 **`git`** 子命令子集（仍拒管道/重定向等）；**`IsExtractReadOnlyBash`** 仍为 **保守子集**（非 TS 全量 **`readOnlyValidation`**）。
  - **类型**：**`internal/types/memory.go`**：**`MemoryFileKindPrivate` / `MemoryFileKindTeam`** 常量（非 **`memory/types.ts`** 完整模型）。

##### H8 仍待实现 / PARITY 缺口（对照 `restored-src`）

下列为 **与 TS 全量仍有差距** 或 **未接线** 的项；实施时以 **`claude-code-sourcemap/restored-src/src/`** 为准。

**系统提示与上下文装载**

- **system 块**与 **用户消息侧 memdir 片段**（**`FindRelevantMemories`** + attachment 头）的 **顺序、去重、是否重复装载** 与 **`memdir.ts` / `claudemd.ts`** 未必 1:1。
- **`claudemd` 式安全读**（**`safelyReadMemoryFileAsync`**、错误重试、team 双入口）：未做同等 **异步读管线**。

**TEAMMEM（团队记忆）**

- **`teamMemPaths.ts` 全量**：**`realpathDeepestExisting`**、悬空 symlink、**`validateTeamMemKey` 全量** 等；Go 为 **子集**（已有 **`ValidateTeamMemWritePath` + `IsTeamMemPathUnderAutoMem`**，非整条 TS 链路）。
- **`teamMemorySync`**：**watcher**（如 **fsnotify**）、会话开始同步、**远程 push/pull**：未实现。
- **折叠/摘要中的 team 统计**（**`teamMemoryOps` / `collapseReadSearch`** 等）：未实现。
- **配置 `team_mem_path`（向导）** 与 **固定 `…/memory/team/`** 布局的 **产品说明 / 行为对齐**：待文档与实现交叉核对。

**工具层与类型**

- **密钥守卫**：经 **`Engine`** 装配的 **`Deps.Tools`** 已包裹；**不经该路径的自定义 ToolRunner** 与 TS「凡 Write/Edit 必经 **File*Tool** 校验」**未必一致**。
- **`memory/types.ts` 中 `TeamMem` 等**：Go 仅有 **常量**；**消息/附件 Content 分支** 未对等扩展。

**Extract 子代理（`extractMemories.ts` / `forkedAgent.ts`）**

- **Bash `isReadOnly`**：非 **`readOnlyValidation.ts` / `checkReadOnlyConstraints`** 全量。
- **HTTP / cache**：无 **`cacheSafeParams` / `skipCacheWrite`** 等与父请求 **1:1**；仍为 **同 Turn + 父 transcript** 近似。
- **Analytics**：无 **`tengu_extract_memories_*`**、**`tengu_fork_agent_query`** 等。
- **GrowthBook**：仍为 **env** 映射（如 **`RABBIT_CODE_EXTRACT_MEMORIES`**），非动态 GB。

**引擎 / 产品接线**

- **进程退出前 drain**：**`Engine.DrainExtractMemories`** 已有；**`cmd/rabbit-code`** 在 **`Bootstrap` 成功**后调用 **`app.WireHeadlessEngineForShutdown`**（**`engine.New` + `RegisterEngineShutdown`**），**`QuitRuntime`/`FailBootstrap`** 在 **`os.Exit` 前 `Runtime.Close`** 时 bounded drain（**`internal/app/engine_shutdown.go`**）。全功能 REPL 若持有独立 **`Engine`** 实例，仍应 **`RegisterEngineShutdown`** 该实例（或复用同一 **`Runtime`** 接线）。

**其它**

- **多 agent**：各子会话 **extract 策略** 未单独设计。
- **测试隔离**：TS **`initExtractMemories()`** 每测一闭包；Go **每 Engine 一个 `ExtractController`**，**隔离语义不同**（非功能缺口，parity 测试需注意）。

### TUI / REPL（在 Headless 主干之后）

| 阶段 | 代号 | 内容 | 主要 `src/` 参考 |
|------|------|------|------------------|
| A | **T1** | **`thinking.ts`、系统块、`processUserInput`** 与展示/输入一致（**§3.0 T1 子计划**、**`PARITY_T1_THINKING_PROCESSUSERINPUT.md`**） | `utils/thinking.ts`、`utils/processUserInput` |
| B | **T2** | **完整 REPL `context.ts` 类子命令**（在现有 **`context break-cache`** 上扩展） | `context.ts` 等 |
| C | **T3** | **预算 meter、附件 UX**（消费引擎暴露的预算信号） | 同 F.1，偏 UI |

**穿插**：**T4**（**session restore 协调与工具**，`sessionRestore.ts` / `query.ts`）与 **T5**（**job 分类、磁盘 stopHooks、模板与 REPL 集成**）在 **T1→T2→T3** 推进过程中按需插入（例如 T1 后与 session 恢复 UI 绑定时做 T4；模板/命令面扩展时做 T5），不强制严格串行，但 **不应早于** 对应 headless 能力就绪（避免 UI 空转）。

#### T1 进度（**§3.0 T1 子计划**）

- **序 1 ☑**：**`internal/utils/thinking`** 对照 **`utils/thinking.ts`**（关键词、模型门控、默认 thinking、彩虹 token 名）；**`engine`** 在 **`Submit`** 与 **`ApplyUserTextHints`** 中对 **ultrathink** 使用 **`features.UltrathinkEnabled() || thinking.HasUltrathinkKeyword(...)`**。
- **序 2 ☑**：**`user_prompt_keywords`** / **`PlainPromptSignals`**；**`engine.Config.TruncateProcessUserInputHookOutput`** 在钩 **`replace`** 后应用 **`TruncateHookOutput`**。
- **序 3 ☑**：**`thinking.InterleavedAPIContextManagementOpts`**；**`features.RedactThinkingEnabled`** / **`ThinkingClearAllLatched`**；**`ApplyEngineCompactIntegration`** 默认 **`AnthropicAssistant.APIContextManagementOpts`**（**`aa.Client != nil`** 且未预设时）。
- **Hook 截断**：**`internal/utils/processuserinput`** **`TruncateHookOutput`**（**`processUserInput.ts`** **`MAX_HOOK_OUTPUT_LENGTH`**）；宿主可另选自调用或开启 **`TruncateProcessUserInputHookOutput`**。
- **PARITY**：**`docs/phases/PARITY_T1_THINKING_PROCESSUSERINPUT.md`**。

#### T2 进度（**§3.0 T2 子计划**）

- **序 1–2 ☑**：**`internal/commands/contextcmd`**（**`rabbit-code context`** **`help`** / **`break-cache`** / **`report`**）；**`report`** → **`query.BuildHeadlessContextReport`** JSON（stdin Messages JSON）。
- **PARITY**：**`docs/phases/PARITY_T2_CONTEXT_CLI.md`**。

---

更新本表时同步 **PARITY_PHASE5_DEFERRED.md**、**PARITY_H9_BASH_PERMISSIONS.md**（H9 迭代）、**PARITY_T1_THINKING_PROCESSUSERINPUT.md**（T1 迭代）、**PARITY_T2_CONTEXT_CLI.md**（T2 迭代）、**PHASE05_SPEC_AND_ACCEPTANCE.md** §6，并与模块根 **README.md**（**`engine.Config` highlights / Phase 5 headless**）交叉核对。
