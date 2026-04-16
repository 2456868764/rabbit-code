// Package agenttool implements the Agent tool (P6.5).
//
// TS↔Go file mapping (restored-src/src/tools/AgentTool/):
//
//	AgentTool.tsx      → agent_tool.go  (Agent.Run, Input, SyncOutput, AsyncOutput,
//	                                     ContentBlock, Usage, RunContext,
//	                                     MapResultForMessagesAPI)
//	constants.ts       → constants.go   (AgentToolName, LegacyAgentToolName,
//	                                     VerificationAgentType, OneShotBuiltInAgentTypes,
//	                                     SearchHint, MaxResultSizeChars, ShouldDefer)
//
// Input schema: { description, prompt, subagent_type?, model?, run_in_background?,
//                 name?, team_name?, mode?, isolation?, cwd? }
// The schema is feature-gated in TS (KAIROS gates cwd; FORK_SUBAGENT gates
// run_in_background); the Go Input struct always includes all optional fields.
//
// Output schema: SyncOutput (status:"completed") | AsyncOutput (status:"async_launched").
//
// Execution is fully delegated to RunContext.RunAgent. The full TS implementation
// (runAgent.ts, LocalAgentTask, RemoteAgentTask, worktree helpers, MCP assembly,
// analytics) requires the query loop and task registry — delegated via the callback.
//
// Aliases: ["Task"] (LegacyAgentToolName for backward compat).
//
// Deferred (Phase 7+ / subagent infrastructure):
//   - runAgent.ts / query loop execution (must be wired via RunContext.RunAgent)
//   - LocalAgentTask / RemoteAgentTask lifecycle management
//   - Worktree creation/removal (isolation:"worktree")
//   - Remote agent / teleport (isolation:"remote")
//   - Teammate spawn / spawnTeammate + team context
//   - Fork subagent (isForkSubagentEnabled)
//   - Permission filtering (filterDeniedAgents / getDenyRuleForAgent)
//   - Background task auto-bg (CLAUDE_AUTO_BACKGROUND_TASKS / tengu_auto_background_agents)
//   - Analytics (logEvent)
//   - UI rendering (renderGroupedAgentToolUse / renderToolResultMessage / etc.)
//   - Built-in agents (Explore, Plan, general-purpose definitions)
//   - agentMemory / agentMemorySnapshot
//   - proactiveModule (PROACTIVE / KAIROS feature gates)
package agenttool
