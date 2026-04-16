// Package webfetchtool implements the WebFetch tool (P6.3).
//
// TS↔Go file mapping (restored-src/src/tools/WebFetchTool/):
//
//	WebFetchTool.ts  → web_fetch_tool.go  (WebFetch.Run, Output, strict JSON input)
//	prompt.ts        → prompt.go           (WebFetchToolName, Description, AuthWarningPrefix, PromptBody, MakeSecondaryModelPrompt→secondary_prompt.go)
//	preapproved.ts   → preapproved.go      (IsPreapprovedHost, IsPreapprovedURL)
//	utils.ts         → validate.go         (ValidateURL, ValidateInput, ValidateInputResult)
//	                   constants.go        (MaxMarkdownLength, MaxResultSizeChars, SearchHint, UserFacingName, ShouldDefer, IsConcurrencySafe, IsReadOnly)
//	                   domain_check.go     (CheckDomainBlocklist, ClearDomainAllowCacheForTest)
//	                   fetch.go            (getWithPermittedRedirects, fetchedRaw)
//	                   redirect.go         (isPermittedRedirect, redirectInfo, redirectCodeText)
//	                   url_cache.go        (urlCacheGet, urlCacheSet, ClearWebFetchCaches)
//	                   html_markdown.go    (htmlToMarkdown)
//	                   html_plain.go       (htmlToPlainText)
//	                   content_type.go     (isBinaryContentType, contentTypeIncludes)
//	                   persist.go          (persistBinaryWebFetch)
//	                   extension.go        (binaryExtension)
//	                   useragent.go        (webFetchUserAgent)
//	UI.tsx           → ui.go              (GetToolUseSummary, ToolUseDescription, ActivityDescription, ToAutoClassifierInput)
//	(mapToolResult)  → map_web_fetch.go   (MapWebFetchToolResultForMessagesAPI)
//	                   context.go          (RunContext, WithRunContext, RunContextFrom)
//	                   secondary_prompt.go (MakeSecondaryModelPrompt, truncateRunes)
//	                   errors.go           (ErrDomainBlocked, ErrDomainCheckFailed, EgressBlockedError)
//
// Deferred (Phase 7 / permission context required):
//   - WebFetchTool.ts checkPermissions / buildSuggestions / webFetchToolInputToPermissionRuleContent
//   - utils.ts logEvent (analytics / ant USER_TYPE telemetry)
//   - UI.tsx renderToolUseMessage / renderToolResultMessage (TUI rendering)
package webfetchtool
