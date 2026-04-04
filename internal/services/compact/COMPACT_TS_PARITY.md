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
| `compact.ts` | `compact.go`, `compact_conversation.go`, `compact_context.go`, `post_compact_attachments.go` | Streaming orchestration → **`internal/query/querydeps`** (see §5) |
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

## 3. Execution list — completed

1. ~~**Export inventory**~~ — See **Appendix A** (TS exports) and **Appendix B** (Go public symbols in this package).
2. ~~**apiMicrocompact.ts**~~ — Compared branches to `GetAPIContextManagement` (`clear_thinking_20251015`, `clear_tool_uses_20250919`, tool lists in `toolsClearableResults` / `toolsClearableUses`). Re-verify when TS edits `USER_TYPE` / env names.
3. ~~**autoCompact.ts**~~ — Buffer constants locked by `TestAutoCompactBufferConstants_matchAutoCompactTs`; threshold math covered by `auto_compact_test.go`.
4. ~~**compact.ts constants**~~ — `TestCompactConstantsMatchTS_compactTs` (POST_COMPACT_*, retries, errors, PTL marker, deny message).
5. ~~**compact.ts strip**~~ — Reinjected attachments: feature-gated test; images: existing strip tests in `compact_test.go` / integration paths.
6. ~~**compact.ts streaming**~~ — Mapped to Go: `anthropic_compact.go` (`StreamCompactSummaryDetailed`, `StreamPartialCompactSummaryDetailed`), `compact_executor.go` (`StreamingCompactExecutorWithConfig`, partial variant), builders in `compact_conversation.go`. **Manual:** re-diff TS on stream/tool changes.
7. ~~**createCompactCanUseTool**~~ — Message string verified = `CompactToolUseDenyMessage` (orchestration may live entirely in TS client; Go exposes constant for callers).
8. ~~**Post-compact file restore**~~ — Go: `post_compact_attachments.go` + `internal/query/engine/post_compact_runtime.go` (`RecordPostCompactFileRead`). Re-diff TS attachment ordering when `createPostCompactFileAttachments` changes.
9. ~~**microCompact.ts**~~ — Covered by `micro_compact_test.go` (tokens, time-based, cache edits).
10. ~~**cachedMicrocompact**~~ — `cached_microcompact_test.go`.
11. ~~**Time-based**~~ — `time_based_trigger_test.go` + cleared-message const test.
12. ~~**sessionMemoryCompact.ts**~~ — `session_memory_compact_test.go`.
13. ~~**prompt.ts**~~ — `TestPromptCompact_smokeMatchesTSExports` (non-empty prompts + `FormatCompactSummary` unwrap).
14. ~~**postCompactCleanup.ts**~~ — `RunPostCompactCleanup` order matches TS (microcompact reset first; main-thread gates; hooks for TS-only clears). Engine must register hooks.
15. ~~**grouping.ts**~~ — `TestGroupRawMessagesByAPIRound_matchesTypedGrouping`.
16. ~~**compactWarning***~~ — Documented in §1; tests in `compact_warning_test.go`, `micro_compact_test.go` (suppression clear on MC).
17. ~~**Document splits**~~ — `doc.go` + this file; **Appendix C** lists cross-package entry points.
18. ~~**notifyCacheDeletion**~~ — **Permanent non-goal** in this repo unless product adds phase-2 cache read comparison (`doc.go` already notes).

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
| Stream compact / partial summary | `internal/query/querydeps/anthropic_compact.go`, `compact_executor.go` |
| Anthropic assistant config (compact tools JSON) | `internal/query/querydeps/anthropic_assistant.go` |
| Auto-compact chain / loop | `internal/query/engine/auto_compact_chain.go`, `internal/query/loop.go` |
| Post-compact file capture | `internal/query/engine/post_compact_runtime.go` |
| Feature gates (auto compact, ant user, API clear tools, …) | `internal/features/env.go` |
