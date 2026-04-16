// Package enterplanmodetool implements the EnterPlanMode tool (P6.4).
//
// TS↔Go file mapping (restored-src/src/tools/EnterPlanModeTool/):
//
//	EnterPlanModeTool.ts → enter_plan_mode_tool.go  (EnterPlanMode.Run, Output,
//	                                                  RunContext, MapResultForMessagesAPI)
//	prompt.ts            → prompt.go                (GetEnterPlanModeToolPrompt, promptExternal)
//	constants.ts         → constants.go             (EnterPlanModeToolName, SearchHint,
//	                                                  MaxResultSizeChars, ShouldDefer, etc.)
//
// Input schema: strict empty object {} (z.strictObject({}) in TS).
// Output schema: { message: string } (z.object({ message: z.string() }) in TS).
//
// Plan-mode state management (handlePlanModeTransition + prepareContextForPlanMode +
// applyPermissionUpdate in TS) is delegated to RunContext.OnEnterPlanMode, which
// callers set when wiring the full permission engine. When nil the tool returns the
// stub confirmation message without modifying permission state.
//
// Deferred (Phase 7 / permission context):
//   - handlePlanModeTransition / prepareContextForPlanMode / applyPermissionUpdate
//   - feature('KAIROS') / feature('KAIROS_CHANNELS') channel gate (isEnabled)
//   - UI rendering (renderToolUseMessage / renderToolResultMessage / renderToolUseRejectedMessage)
//   - isPlanModeInterviewPhaseEnabled() branch in mapToolResultToToolResultBlockParam
//   - "ant" user-type variant of prompt
package enterplanmodetool
