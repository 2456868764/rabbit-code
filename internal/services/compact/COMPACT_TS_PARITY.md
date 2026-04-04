# compact ↔ TS parity checklist

Reference snapshot: `claude-code-sourcemap` @ **a8a678c** — `restored-src/src/services/compact/*.ts`. Re-diff when updating TS.

This package is **headless**: UI-only TS (e.g. `compactWarningHook.ts` / React) has no Go counterpart; behavior is approximated via `SubscribeCompactWarningSuppression` and callers.

**Verification:** `go test ./internal/services/compact/...` (includes `compact_parity_verification_test.go` for constants, strip, prompt smoke, grouping, time-based message, auto buffer tokens).

---

## 1. File mapping (TS → Go)

| TS module | Go file(s) | Notes |
|-----------|------------|--------|
| `apiMicrocompact.ts` | `api_microcompact.go` | Request `context_management` JSON; HTTP merge may live in `internal/services/api` |
| `autoCompact.ts` | `auto_compact.go`, `auto_compact_if_needed.go` | `isAutoCompactEnabled` → **`internal/features.IsAutoCompactEnabled`** |
| `compact.ts` | `compact.go`, `compact_conversation.go`, `compact_context.go`, `post_compact_attachments.go` | Streaming orchestration → **`internal/query`** (`anthropic_compact.go`, `compact_executor.go`; see §5) |
| `microCompact.ts` | `micro_compact.go`, `cached_microcompact.go`, `time_based_trigger.go`, `time_based_microcompact.go` | Cached path split to `cached_microcompact.go` |
| `timeBasedMCConfig.ts` | `time_based_trigger.go` | `getTimeBasedMCConfig` colocated with eval |
| `sessionMemoryCompact.ts` | `session_memory_compact.go` | `session_memory_compact_test.go` |
| `prompt.ts` | `prompt_compact.go` | `prompt_compact_test.go` |
| `postCompactCleanup.ts` | `post_compact_cleanup.go` | Hooks wired from engine; `PostCompactCleanupHooks` |
| `grouping.ts` | `grouping.go` (+ `GroupRawMessagesByAPIRound` in `compact.go`) | `grouping_test.go` |
| `compactWarningState.ts` | `compact_warning.go` | `compact_warning_test.go` |
| `compactWarningHook.ts` | — | **N/A (UI)** |

---

## 2. Diff matrix (symbol / behavior)

| Area | TS surface | Go | Status |
|------|------------|-----|--------|
| API microcompact | `getAPIContextManagement` | `GetAPIContextManagement` | verified (logic vs TS; env: `AntUserType`, `UseAPIClearTool*`, `APIMaxInputTokens`) |
| Auto compact thresholds | buffers, `getAutoCompactThreshold` | `AutocompactBufferTokens` …, `GetAutoCompactThreshold` | **verified** (const test) |
| Token warning | `calculateTokenWarningState` | `CalculateTokenWarningState*` | verified (`auto_compact_test.go`) |
| Auto gate | `isAutoCompactEnabled` | `features.IsAutoCompactEnabled` | split |
| Auto run | `autoCompactIfNeeded` | `AutoCompactIfNeeded` + engine | split |
| Strip / reinject | `strip*`, `feature('EXPERIMENTAL_SKILL_SEARCH')` | `Strip*`, `features.ExperimentalSkillSearchEnabled` | **verified** (`compact_parity_verification_test.go` + `compact_test.go`) |
| Post-compact budgets | `POST_COMPACT_*` | `PostCompact*` consts | **verified** (const test) |
| Error strings | `ERROR_MESSAGE_*`, `PTL_RETRY_MARKER` | `ErrorMessage*`, `PTLRetryMarker` | **verified** (const test) |
| `createCompactCanUseTool` message | literal | `CompactToolUseDenyMessage` | **verified** (const test) |
| Build transcript / boundary | `buildPostCompactMessages`, … | `BuildPostCompactMessagesJSON`, `compact_conversation.go` | verified (`compact_conversation_test.go`, …) |
| Full/partial stream | `compactConversation`, `partialCompactConversation` | `StreamCompact*`, `Streaming*Executor*` | split → §5 |
| Post-compact files | `createPostCompactFileAttachments` | `FilterAttachmentMessagesByRoughTokenBudget` + `engine/post_compact_runtime.go` | split |
| Microcompact | `microcompactMessages`, token est | `MicrocompactMessagesAPIJSON`, … | verified (`micro_compact_test.go`) |
| Cached MC | bundle | `RunCachedMicrocompactTranscriptJSON` | verified (`cached_microcompact_test.go`) |
| Time-based | `evaluateTimeBasedTrigger`, cleared message | `EvaluateTimeBasedTrigger*`, `TimeBasedMCClearedMessage` | **verified** (`time_based_trigger_test.go` + const test) |
| Session memory | `trySessionMemoryCompaction`, indices | `TrySessionMemoryCompactionTranscriptJSON`, … | verified (`session_memory_compact_test.go`) |
| Prompts | `getCompactPrompt`, … | `GetCompactPrompt`, … | **verified** (smoke test) |
| Post-compact cleanup | `runPostCompactCleanup` | `RunPostCompactCleanup` | verified (hook shape vs TS; engine wires) |
| Grouping | `groupMessagesByApiRound` | `GroupMessagesByApiRound`, `GroupRawMessagesByAPIRound` | **verified** (parity test) |
| Compact warning | store / suppress | `compact_warning.go` | verified (`compact_warning_test.go`) |
| Prompt cache break / notifyCacheDeletion | `promptCacheBreakDetection` | `features` + query | **gap** (non-goal; see §6) |
| Analytics / GrowthBook | live | `internal/features` env | **gap** (deliberate) |

---

## 3. Execution list (逐项：验证 → 提交)

工作方式：每一项在 Go 侧有**可运行验证**（测试或文档中的明确命令），再 **git commit**。历史上有批量文档+回归测试提交 **`cb097ca`**；`apiMicrocompact` 补测单独为 **`e978cbd`**。

| # | 项 | 如何验证 | 提交 / 备注 |
|---|-----|----------|-------------|
| 1 | Export inventory | 附录 A–B–C 与 `doc.go` 映射 | `cb097ca` |
| 2 | `apiMicrocompact.ts` | `go test ./internal/services/compact/... -run 'APIContext|APIClear|DefaultAPI'` | `cb097ca` + **`e978cbd`**（thinking `clear_all` / redact / `exclude_tools`） |
| 3 | `autoCompact.ts` | `go test ./internal/services/compact/... -run AutoCompact\|CalculateToken\|EffectiveContext` | `cb097ca`（常量 `TestAutoCompactBufferConstants_*`）+ `auto_compact_test.go` |
| 4 | `compact.ts` 常量/错误串 | `go test ./internal/services/compact/... -run TestCompactConstantsMatchTS` | `cb097ca` |
| 5 | `compact.ts` strip | `go test ./internal/services/compact/... -run StripReinjected\|StripImages` | `cb097ca` + `compact_test.go` |
| 6 | `compact.ts` streaming | `go test ./internal/query/... -run Compact\|Partial`；对照附录 C 源文件 | 行为在 **`internal/query`**；TS 大改时人工 diff |
| 7 | `createCompactCanUseTool` | `go test ./internal/services/compact/... -run TestCompactConstantsMatchTS`（含 deny 文案） | `cb097ca` |
| 8 | Post-compact 文件 | `go test ./internal/services/compact/... -run PostCompact\|FilterAttachment`；`engine/post_compact_runtime_test.go`（若有） | `cb097ca` + engine 包测试 |
| 9 | `microCompact.ts` | `go test ./internal/services/compact/... -run Microcompact\|EstimateMessage` | `cb097ca` + `micro_compact_test.go` |
| 10 | cached microcompact | `go test ./internal/services/compact/... -run CachedMicrocompact` | `cb097ca` |
| 11 | Time-based | `go test ./internal/services/compact/... -run TimeBased\|TestTimeBasedMCCleared` | `cb097ca` |
| 12 | `sessionMemoryCompact.ts` | `go test ./internal/services/compact/... -run SessionMemory` | `cb097ca` |
| 13 | `prompt.ts` | `go test ./internal/services/compact/... -run TestPromptCompact` | `cb097ca` |
| 14 | `postCompactCleanup.ts` | `post_compact_cleanup.go` 与 engine 注册 hooks；必要时单测 mock | 文档 + 引擎接线 |
| 15 | `grouping.ts` | `go test ./internal/services/compact/... -run GroupMessages\|GroupRaw` | `cb097ca` |
| 16 | `compactWarning*` | `go test ./internal/services/compact/... -run CompactWarning` | `cb097ca` |
| 17 | 文档拆分 | `doc.go` + 本文件 §1/§4/附录 | `cb097ca` |
| 18 | `notifyCacheDeletion` | 不实现；§2 **gap** | 非目标（见 `doc.go`） |

**一键全量：** `go test ./internal/services/compact/... -count=1` 与 `go test ./internal/... -count=1`。

---

## 4. Tests

```bash
go test ./internal/services/compact/... -count=1
go test ./internal/... -count=1   # full rabbit-code
```

---

## Appendix A — TS exports (@ `src/services/compact/`)

| File | Exports |
|------|---------|
| `apiMicrocompact.ts` | `ContextEditStrategy`, `ContextManagementConfig`, `getAPIContextManagement` |
| `autoCompact.ts` | `getEffectiveContextWindowSize`, `AutoCompactTrackingState`, buffer consts, `getAutoCompactThreshold`, `calculateTokenWarningState`, `isAutoCompactEnabled`, `shouldAutoCompact`, `autoCompactIfNeeded` |
| `compact.ts` | POST_COMPACT_* , strip/truncate/build helpers, errors, `compactionResult` types, `compactConversation`, `partialCompactConversation`, `createCompactCanUseTool`, post-compact attachment factories, … |
| `microCompact.ts` | `TIME_BASED_MC_CLEARED_MESSAGE`, cache API, `estimateMessageTokens`, types, `microcompactMessages`, `evaluateTimeBasedTrigger` |
| `sessionMemoryCompact.ts` | config API, `hasTextBlocks`, `adjustIndexToPreserveAPIInvariants`, `calculateMessagesToKeepIndex`, `shouldUseSessionMemoryCompaction`, `trySessionMemoryCompaction` |
| `prompt.ts` | `getPartialCompactPrompt`, `getCompactPrompt`, `formatCompactSummary`, `getCompactUserSummaryMessage` |
| `postCompactCleanup.ts` | `runPostCompactCleanup` |
| `grouping.ts` | `groupMessagesByApiRound` |
| `compactWarningState.ts` | `compactWarningStore`, `suppressCompactWarning`, `clearCompactWarningSuppression` |
| `compactWarningHook.ts` | `useCompactWarningSuppression` |
| `timeBasedMCConfig.ts` | `TimeBasedMCConfig`, `getTimeBasedMCConfig` |

---

## Appendix B — Go `internal/services/compact` public API (non-test)

Primary types/functions: `compact.go` (`RunPhase`, strip/build/truncate/group PTL), `compact_conversation.go` (boundary, partial/full stream JSON builders), `auto_compact.go`, `auto_compact_if_needed.go`, `micro_compact.go`, `cached_microcompact.go`, `time_based_trigger.go`, `time_based_microcompact.go`, `session_memory_compact.go`, `prompt_compact.go`, `post_compact_cleanup.go`, `post_compact_attachments.go`, `grouping.go`, `compact_warning.go`, `compact_context.go`, `api_microcompact.go`. See `go doc` / ripgrep `^func [A-Z]` on `*.go` excluding `*_test.go`.

---

## Appendix C — Cross-package mirrors (not under `services/compact/`)

| Concern | Go location |
|---------|-------------|
| Stream compact / partial summary | `internal/query/anthropic_compact.go`, `compact_executor.go` |
| Anthropic assistant config (compact tools JSON) | `internal/query/anthropic_assistant.go` |
| Auto-compact chain / loop | `internal/query/engine/auto_compact_chain.go`, `internal/query/loop.go` |
| Post-compact file capture | `internal/query/engine/post_compact_runtime.go` |
| Feature gates (auto compact, ant user, API clear tools, …) | `internal/features/env.go` |
