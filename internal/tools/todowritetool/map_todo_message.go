package todowritetool

import "encoding/json"

const todoWriteResultBase = "Todos have been modified successfully. Ensure that you continue to use the todo list to track your progress. Please proceed with the current tasks if applicable"

// MapTodoWriteToolResultForMessagesAPI mirrors TodoWriteTool.mapToolResultToToolResultBlockParam.
func MapTodoWriteToolResultForMessagesAPI(outJSON []byte) string {
	var m struct {
		VerificationNudge bool `json:"verificationNudgeNeeded"`
	}
	_ = json.Unmarshal(outJSON, &m)
	base := todoWriteResultBase
	if m.VerificationNudge {
		base += "\n\nNOTE: You just closed out 3+ tasks and none of them was a verification step. Before writing your final summary, spawn the verification agent (subagent_type=\"" + VerificationAgentType + "\"). You cannot self-assign PARTIAL by listing caveats in your summary — only the verifier issues a verdict."
	}
	return base
}
