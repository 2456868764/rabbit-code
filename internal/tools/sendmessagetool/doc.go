// Package sendmessagetool implements the SendMessage tool (P6.5).
//
// TS↔Go file mapping (restored-src/src/tools/SendMessageTool/):
//
//	SendMessageTool.ts → send_message_tool.go  (SendMessage.Run, Input, StructuredMessage,
//	                                            MessageOutput, BroadcastOutput, RequestOutput,
//	                                            ResponseOutput, MessageRouting, RunContext,
//	                                            MapResultForMessagesAPI)
//	prompt.ts          → prompt.go             (Prompt, Description)
//	constants.ts       → constants.go          (SendMessageToolName, TeamLeadName,
//	                                            Description, MaxResultSizeChars)
//
// Input schema: { to, summary?, message: string | StructuredMessage }.
// StructuredMessage is a discriminated union on "type":
//   - shutdown_request: { type, reason? }
//   - shutdown_response: { type, request_id, approve, reason? }
//   - plan_approval_response: { type, request_id, approve, feedback? }
//
// Output schema: union of MessageOutput | BroadcastOutput | RequestOutput | ResponseOutput.
// All variants include { success: bool, message: string }.
//
// Message delivery (writeToMailbox, readTeamFileAsync, queuePendingMessage,
// resumeAgentBackground, bridge/UDS cross-session) is delegated to RunContext.SendMsg.
// When nil a stub success response is returned.
//
// Deferred (Phase 7 / swarm + mailbox infrastructure):
//   - writeToMailbox (mailbox write)
//   - readTeamFileAsync / getAgentName / getTeamName / getTeammateColor (team helpers)
//   - queuePendingMessage / resumeAgentBackground (in-process agent lifecycle)
//   - gracefulShutdown (shutdown_request / shutdown_response handling)
//   - bridge (postInterClaudeMessage — REPL bridge)
//   - UDS socket (sendToUdsSocket — UDS_INBOX feature gate)
//   - isAgentSwarmsEnabled() gate
//   - logEvent analytics
//   - UI rendering (renderToolUseMessage / renderToolResultMessage)
package sendmessagetool
