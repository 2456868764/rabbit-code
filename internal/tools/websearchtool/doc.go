// Package websearchtool implements the WebSearch tool (P6.3).
//
// TS↔Go file mapping (restored-src/src/tools/WebSearchTool/):
//
//	WebSearchTool.ts → web_search_tool.go  (WebSearch.Run, Output, IsEnabled, Input, SearchHit, SearchResultBlock)
//	prompt.ts        → prompt.go           (WebSearchToolName, PromptBody, Description, localMonthYear)
//	(mapToolResult)  → map_web_search.go   (MapWebSearchToolResultForMessagesAPI, trimToolResultToMaxRunes)
//	(schemas)        → schema.go           (WebSearchToolSchema20250305, WebSearchToolSchemaFromInput)
//	(blocks)         → blocks.go           (MakeOutputFromContentBlocks — mirrors makeOutputFromSearchResponse)
//	(progress)       → progress.go         (WebSearchProgress, WebSearchProgressData, ExtractQueryFromPartialWebSearchInputJSON)
//	(input)          → input_json.go       (DecodeInputStrictJSON)
//	(validate)       → validate.go         (ValidateInput, ValidateInputTS, ZodQueryMinLength, ValidateInputResult)
//	(constants)      → constants.go        (MaxSearchUses, MaxResultSizeChars, SearchHint, UserFacingName, ErrQueryZodMin, …)
//	(upstream)       → upstream_strings.go (QuerySourceWebSearchTool, InnerSearchSystemPrompt, ServerToolSchemaName,
//	                                        InnerSearchUserContent, AutoClassifierInput, ExtractSearchText,
//	                                        DefaultPermissionSuggestions, DefaultCheckPermissions, …)
//	UI.tsx           → ui_headless.go      (ToolUseDescription, ActivityDescription, TruncToolUseSummary,
//	                                        RenderToolUseMessage, FormatToolUseProgressHeadless,
//	                                        SearchCounts, FormatToolResultSummaryLine)
//	                   context.go          (RunContext, WithRunContext, RunContextFrom)
//
// Streaming call loop (WebSearchTool.ts call):
// The inner queryModelWithStreaming + onProgress loop is API-coupled; Go delegates to
// RunContext.ExecuteSearch for the actual network call. MakeOutputFromContentBlocks (blocks.go)
// and ExtractQueryFromPartialWebSearchInputJSON (progress.go) are provided for callers that
// implement streaming adapters (e.g. engine/query loop with Messages API web_search_20250305).
//
// Deferred (Phase 7 / permission context):
//   - WebSearchTool.ts checkPermissions (passthrough — default exposed via DefaultCheckPermissions helper)
//   - getFeatureValue_CACHED_MAY_BE_STALE('tengu_plum_vx3') Haiku model switch
package websearchtool
