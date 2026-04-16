// Package teamcreatetool implements the TeamCreate tool (P6.5).
//
// TS↔Go file mapping (restored-src/src/tools/TeamCreateTool/):
//
//	TeamCreateTool.ts → team_create_tool.go  (TeamCreate.Run, Input, Output,
//	                                          RunContext, MapResultForMessagesAPI)
//	prompt.ts         → prompt.go            (Prompt)
//	constants.ts      → constants.go         (TeamCreateToolName, SearchHint,
//	                                          MaxResultSizeChars, ShouldDefer)
//
// Input schema: { team_name, description?, agent_type? } (strict).
// Output schema: { team_name, team_file_path, lead_agent_id }.
//
// Team creation (readTeamFile/writeTeamFileAsync/getTeamFilePath/resetTaskList/
// ensureTasksDir/setLeaderTeamName/assignTeammateColor/setAppState/logEvent) is
// delegated to RunContext.OnTeamCreate. When nil a stub output is returned.
//
// Deferred (Phase 7 / team swarm infrastructure):
//   - readTeamFile / writeTeamFileAsync / getTeamFilePath (team file I/O)
//   - resetTaskList / ensureTasksDir / setLeaderTeamName (task list setup)
//   - registerTeamForSessionCleanup (session cleanup hook)
//   - assignTeammateColor (UI color manager)
//   - generateUniqueTeamName / generateWordSlug (name collision avoidance)
//   - parseUserSpecifiedModel / getDefaultMainLoopModel (model selection)
//   - logEvent analytics
//   - isAgentSwarmsEnabled() gate (isEnabled)
//   - UI rendering (renderToolUseMessage)
package teamcreatetool
