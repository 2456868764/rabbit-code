# Phase 5 feature flags — PARITY status & follow-on work

See **PHASE05_SPEC_AND_ACCEPTANCE.md** §2 **P5.F.*** and **`internal/features/rabbit_env.go`** for env names.

## Implemented in `engine` / `query` / `querydeps` (headless path)

| SPEC ID | Flag | Runtime behavior |
|---------|------|-------------------|
| **P5.F.1** | `TOKEN_BUDGET` | UTF-8 byte cap + optional **`RABBIT_CODE_TOKEN_BUDGET_MAX_INPUT_TOKENS`** (heuristic via **`query.EstimateUTF8BytesAsTokens`**) + optional **`RABBIT_CODE_TOKEN_BUDGET_MAX_ATTACHMENT_BYTES`** (raw memdir inject bytes). |
| **P5.F.2** | `REACTIVE_COMPACT` | Byte threshold (**`RABBIT_CODE_REACTIVE_COMPACT_MIN_BYTES`**, default 8192) **or** heuristic token threshold (**`RABBIT_CODE_REACTIVE_COMPACT_MIN_TOKENS`**) via **`query.ReactiveCompactByTranscript`**; **`CompactAdvisor`** receives full transcript JSON. |
| **P5.F.3** | `CONTEXT_COLLAPSE` | Appends collapse hint (`query.ApplyUserTextHints`). **`SESSION_RESTORE`** (**`RABBIT_CODE_SESSION_RESTORE`**) appends session-restore hint. |
| **P5.F.4** | `ULTRATHINK` | Prepends thinking hint to resolved user text (`query`). |
| **P5.F.5** | `ULTRAPLAN` | Appends plan hint to resolved user text (`query`). |
| **P5.F.6** | `BREAK_CACHE_COMMAND` | **`EventKindBreakCacheCommand`** at start of each `runTurnLoop` Submit. CLI: **`rabbit-code context break-cache`** (**`internal/commands/breakcache`**，对齐 **`src/commands/break-cache`**). |
| **P5.F.7** | `TEMPLATES` | **`EventKindTemplatesActive`** + names; loads **`<name>.md`** from **`RABBIT_CODE_TEMPLATE_DIR`** or **`engine.Config.TemplateDir`** when set. |
| **P5.F.8** | `CACHED_MICROCOMPACT` | **`EventKindCachedMicrocompactActive`**; streaming request body sets **`anthropic_beta`** (placeholder string **`BetaCachedMicrocompactBody`** in `internal/anthropic/betas.go` until upstream names a dedicated cache-editing beta). |
| **P5.F.9** | `PROMPT_CACHE_BREAK_DETECTION` | Per-Submit **`context`** callback → **`AnthropicAssistant`** → **`EventKindPromptCacheBreakDetected`** when SSE matches. Optional **`RABBIT_CODE_PROMPT_CACHE_BREAK_SUGGEST_COMPACT`**: after success, if break callback ran, emit reactive **`EventKindCompactSuggest`**. |
| **P5.F.10** | `HISTORY_SNIP` | Each assistant round: if transcript JSON bytes exceed max, drop leading messages (`query.TrimTranscriptPrefixWhileOverBudget`) → **`EventKindHistorySnipApplied`**. Thresholds: **`RABBIT_CODE_HISTORY_SNIP_MAX_BYTES`** (default 32768), **`RABBIT_CODE_HISTORY_SNIP_MAX_ROUNDS`** (default 4). Scrollback / UI: **`messages.StripHistorySnipPieces`** strips `history_snip` content from `[]types.Message`. |
| **P5.2.2** | `SNIP_COMPACT` | Same trim primitive as F.10 but separate env (**`RABBIT_CODE_SNIP_COMPACT`**, **`RABBIT_CODE_SNIP_COMPACT_MAX_BYTES`**, **`RABBIT_CODE_SNIP_COMPACT_MAX_ROUNDS`**) → **`EventKindSnipCompactApplied`**. |
| **Tools** | `BASH_EXEC` | **`RABBIT_CODE_BASH_EXEC`**: **`querydeps.BashExecToolRunner`** runs **`sh -c`** from bash tool JSON **`command`/`cmd`** (otherwise stubs). |

## Follow-on (later phases — not a blocker for Phase 5 checklist)

| Area | Target phase | Notes |
|------|----------------|-------|
| API tokenizer / attachment UX | 5 / 9 | Iter 12: heuristic tokens + memdir raw-byte cap only |
| Full `analyzeContext` / autoCompact | 5+ | H2 增量：**`ProactiveAutoCompactSuggested`**、**`ReactiveCompactByTranscript`** 尊重 **`DISABLE_COMPACT`**；全量仍含 API tokenizer、**`analyzeContext`** 分类 UI、**`autoCompactIfNeeded`** 全语义 |
| Autocompact **`tracking.consecutiveFailures`** 跨 **Submit** 持久化 | 5+ | **Engine** 计数 + **`LoopState.AutoCompactTracking`** 镜像（H3 子集）；会话级恢复仍 defer |
| **`COMPACTABLE_TOOLS`** | 5+ | Go **`IsCompactableToolName`** + **`TestCompactableToolNames_matchMicroCompactTS`**；**`microCompact.ts`** 全量仍 defer |
| Session coordinator / restore tools | 6 / 8 / 9 | Iter 12: **SESSION_RESTORE** text hint only |
| `thinking.ts` / `processUserInput` TUI | 5 / 9 | Iter 12: **UserSubmit** **`PhaseDetail`** mode tags |
| Full REPL `context.ts` | 10 | Iter 12: **`rabbit-code context break-cache`** JSON |
| Job classifier, `stopHooks` from disk | 2 / 5 / 10 | Iter 12: template **`.md`** from dir only |
| F.8 upstream-named cache-editing beta + full `microCompact.ts` / cache field parity | 4 / 6 | Placeholder beta only in headless path |
| F.9 Auto-trim transcript + resend after break | 5 / 6 | Iter 12: compact **suggest** only |
| **P5.2.2** snip metadata / persistence (UUID map, session round-trip) | 5 / 8 | Headless：**H7.6–H7.9**（含转录内 **`rabbit_message_uuid`** + **`ReplaySnipRemovalsAuto`**）；**JSONL Map + parentUuid 重链** 仍 Phase 8 |
| TS **`ToolUseContext`** full object (`AppState`, MCP, full abort lifecycle, `handleStopHooks` tool runner) | 6 / 10 | H6 headless: **`ToolUseContextMirror`** + **`StopHooksAfterSuccessfulTurn`** + defer **`StopHooks`** |

**Full-parity order (headless first, then TUI/REPL)**: see **PHASE05_CONTINUATION.md** §「全量还原推荐顺序」— **H6（headless State/continue 已收口）→ H1 → H2/H3/H4 → H5/H7/H8 → H9**, then **T1→T2→T3** with **T4/T5** interleaved as documented there.

**Review**: When a follow-on row lands, update this file and **PHASE05_SPEC_AND_ACCEPTANCE.md** §6.
