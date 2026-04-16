// Package teamdeletetool implements the TeamDelete tool (P6.5).
//
// TS↔Go file mapping (restored-src/src/tools/TeamDeleteTool/):
//
//	TeamDeleteTool.ts → team_delete_tool.go  (TeamDelete.Run, Output, RunContext,
//	                                          MapResultForMessagesAPI)
//	prompt.ts         → prompt.go            (Prompt)
//	constants.ts      → constants.go         (TeamDeleteToolName, SearchHint,
//	                                          MaxResultSizeChars, ShouldDefer, ToolDescription)
//
// Input schema: {} (strict empty object, z.strictObject({})).
// Output schema: { success, message, team_name? }.
//
// Team cleanup (readTeamFile/cleanupTeamDirectories/unregisterTeamForSessionCleanup/
// clearTeammateColors/clearLeaderTeamName/setAppState/logEvent) is delegated to
// RunContext.OnTeamDelete. The team name is injected via RunContext.TeamName (mirrors
// appState.teamContext.teamName). When no RunContext/callback, a stub success is returned.
//
// TS enforces "no active non-lead members" before deletion (TeamDeleteTool.ts ~71-98);
// this guard must be implemented in OnTeamDelete when wired.
//
// Deferred (Phase 7 / team swarm infrastructure):
//   - readTeamFile / cleanupTeamDirectories (team file I/O)
//   - unregisterTeamForSessionCleanup (session cleanup hook)
//   - clearTeammateColors / clearLeaderTeamName (app state cleanup)
//   - Active-members guard (TEAM_LEAD_NAME filtering)
//   - logEvent analytics
//   - isAgentSwarmsEnabled() gate (isEnabled)
//   - UI rendering (renderToolUseMessage / renderToolResultMessage)
package teamdeletetool
