// Package todowritetool implements the TodoWrite tool (P6.4).
//
// TS↔Go file mapping (restored-src/src/tools/TodoWriteTool/):
//
//	TodoWriteTool.ts → todo_write_tool.go  (TodoWrite.Run, TodoItem, Output, validateTodos,
//	                                        allCompleted, anyVerificationMention, verif nudge)
//	prompt.ts        → prompt.go           (Prompt, Description, ToAutoClassifierInput)
//	constants.ts     → constants.go        (TodoWriteToolName, VerificationAgentType,
//	                                        SearchHint, MaxResultSizeChars, ShouldDefer, Strict)
//	(mapToolResult)  → map_todo_message.go (MapTodoWriteToolResultForMessagesAPI)
//	(context/store)  → context.go          (RunContext, Store, NewStore, TodoKey,
//	                                        WithRunContext, RunContextFrom)
//
// Enablement: TodoWriteTool.ts isEnabled() = !isTodoV2Enabled() is mirrored by
// features.TodoWriteToolEnabled (CLAUDE_CODE_ENABLE_TASKS env / nonInteractive gate).
//
// Verification nudge: feature('VERIFICATION_AGENT') + tengu_hive_evidence growthbook flag
// → features.TodoWriteVerificationNudgeEnabled (RABBIT_CODE_VERIFICATION_AGENT + RABBIT_CODE_TENGU_HIVE_EVIDENCE).
//
// Deferred (Phase 7 / task service layer):
//   - isTodoV2Enabled() full dynamic growthbook check (currently env-gate only)
//   - UI rendering (renderToolUseMessage always returns null in TS — no Go equivalent needed)
package todowritetool
