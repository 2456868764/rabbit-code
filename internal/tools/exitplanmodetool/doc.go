// Package exitplanmodetool implements the ExitPlanMode (V2) tool (P6.4).
//
// TS↔Go file mapping (restored-src/src/tools/ExitPlanModeTool/):
//
//	ExitPlanModeV2Tool.ts → exit_plan_mode_v2_tool.go  (ExitPlanModeV2.Run, AllowedPrompt,
//	                                                     Output, RunContext, MapResultForMessagesAPI)
//	prompt.ts             → prompt.go                  (Prompt / EXIT_PLAN_MODE_V2_TOOL_PROMPT)
//	constants.ts          → constants.go               (ExitPlanModeToolName, SearchHint,
//	                                                     MaxResultSizeChars, ShouldDefer, etc.)
//
// Input schema: { allowedPrompts?: AllowedPrompt[] } with passthrough (extra fields allowed).
// Output schema: { plan: string|null, isAgent: bool, filePath?, hasTaskTool?, planWasEdited?,
//                  awaitingLeaderApproval?, requestId? }.
//
// Plan-exit state management (getPlan, getPlanFilePath, setHasExitedPlanMode,
// setNeedsPlanModeExitAttachment, permission mode restoration, transcript classifier
// gating, teammate mailbox) is delegated to RunContext.OnExitPlanMode. When nil the
// tool returns a stub approved output.
//
// MapResultForMessagesAPI implements mapToolResultToToolResultBlockParam handling the
// four main branches: awaitingLeaderApproval, isAgent, empty plan, normal plan.
// The hasTaskTool TeamCreate hint branch is included; AGENT_TOOL_NAME / TEAM_CREATE_TOOL_NAME
// are referenced by their string values ("Agent" / "TeamCreate").
//
// Deferred (Phase 7 / permission + plan-file infrastructure):
//   - getPlan / getPlanFilePath / persistFileSnapshotIfRemote (plan file I/O)
//   - setHasExitedPlanMode / setNeedsPlanModeExitAttachment (session flags)
//   - autoModeState / permissionSetup module gate (transcript classifier)
//   - writeToMailbox / setAwaitingPlanApproval (teammate mailbox)
//   - logEvent analytics
//   - feature('KAIROS') / feature('KAIROS_CHANNELS') channel gate (isEnabled)
//   - UI rendering (renderToolUseMessage / renderToolResultMessage / renderToolUseRejectedMessage)
package exitplanmodetool
