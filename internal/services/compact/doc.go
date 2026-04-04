// Package compact implements headless subsets of restored-src/src/services/compact (Phase 4–5).
//
// Upstream directory: src/services/compact/*.ts. Target PARITY layout is one Go basename per TS module
// (snake_case.go); see table below.
//
// TS module (restored-src/src/services/compact/)     Go file (this package)
// -------------------------------------------------------------------------------------------------
// apiMicrocompact.ts                                 api_microcompact.go — entire module (getAPIContextManagement, defaults, tool lists); MicroCompactRequested / PromptCacheBreakActive colocated; HTTP merge in internal/services/api (client.go, betas_merge.go)
// autoCompact.ts                                     auto_compact.go + auto_compact_if_needed.go（AutoCompactIfNeeded, SnipTokensFreed）；ProactiveAutocompactFromUsage、AfterTurn*；engine auto_compact_chain 传入 LoopState.SnipTokensFreedAccum 与 Config.AfterSessionMemoryCompactSuccess（→ AfterSessionMemorySuccess）
// compact.ts                                         compact*.go + compact_context.go + post_compact_attachments.go — TS 同名镜像：StripImagesFromMessages, StripReinjectedAttachments（→ 既有 *JSON 实现）；其余见 querydeps / engine。normalizeMessagesForAPI 全量未在包内。
// microCompact.ts                                    micro_compact.go — TS 同名包级函数：ConsumePendingCacheEdits/GetPinnedCacheEdits/PinCacheEdits/MarkToolsSentToAPIState/ResetMicrocompactState(buf)（模块状态在 Go 中由 MicrocompactEditBuffer 承载）；cached_microcompact.go, time_based_trigger.go, time_based_microcompact.go
// postCompactCleanup.ts                              post_compact_cleanup.go — RunPostCompactCleanup (+ PostCompactCleanupHooks), IsMainThreadPostCompactSource; engine wires optional hooks + legacy PostCompactCleanup callback
// prompt.ts                                          prompt_compact.go — FormatCompactSummary, GetCompactPrompt, GetPartialCompactPrompt, GetCompactUserSummaryMessage (proactive extra gated by features.KairosDailyLogMemoryEnabled)
// grouping.ts                                        grouping.go — GroupMessagesByApiRound, ApiRoundMessage
// compactWarningState.ts + compactWarningHook.ts     compact_warning.go — suppress/clear store + SubscribeCompactWarningSuppression（useSyncExternalStore analogue）；CalculateTokenWarningState（数值）仍在 auto_compact.go
// sessionMemoryCompact.ts                            session_memory_compact.go — TS 同名/镜像：DEFAULT_SM_COMPACT_CONFIG, HasTextBlocks, AdjustIndexToPreserveAPIInvariants, CalculateMessagesToKeepIndex；TrySessionMemoryCompactionTranscriptJSON, NewSessionMemoryCompactExecutor；engine wires SessionMemoryCompact
// timeBasedMCConfig.ts                               time_based_trigger.go — GetTimeBasedMCConfig（features env，与 evaluate 同文件）
//
// Related TS outside services/compact/:
//   - services/api/promptCacheBreakDetection.ts      api_microcompact.go — PromptCacheBreakActive (via features); notifyCacheDeletion (cacheDeletionsPending) not ported — no Go phase-2 cache-read comparison yet
//   - internal/query — blocking ladder / strip cache_control — query/loop.go, query/transcript.go (prompt cache break path)
//   - internal/query/engine — time-based MC on API transcript; LastAssistantAtForPersistence / RestoredSessionLastAssistantAt for session sidecars
//
// Package query imports compact for LoopState.AutoCompactTracking and BuildHeadlessContextReport; compact must not import query.
//
// Tool names for COMPACTABLE_TOOLS: internal/tools/* (mirrors src/tools/*/prompt.ts|constants.ts) and
// internal/utils/shell.ShellToolNames (mirrors src/utils/shell/shellToolUtils.ts).
//
// Parity checklist: COMPACT_TS_PARITY.md (diff matrix, completed execution list, appendices).
// Streaming compact / partial: internal/query/querydeps (anthropic_compact.go, compact_executor.go).
package compact
