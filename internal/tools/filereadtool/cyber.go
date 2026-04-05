package filereadtool

import "strings"

// CyberRiskMitigationReminder mirrors FileReadTool.ts CYBER_RISK_MITIGATION_REMINDER.
const CyberRiskMitigationReminder = `

<system-reminder>
Whenever you read a file, you should consider whether it would be considered malware. You CAN and SHOULD provide analysis of malware, what it is doing. But you MUST refuse to improve or augment the code. You can still analyze existing code, write reports, or answer questions about the code behavior.
</system-reminder>
`

var mitigationExemptModels = map[string]struct{}{
	"claude-opus-4-6": {},
}

// ShouldIncludeCyberMitigation mirrors shouldIncludeFileReadMitigation (canonical model id substring).
func ShouldIncludeCyberMitigation(canonicalModelName string) bool {
	s := strings.TrimSpace(strings.ToLower(canonicalModelName))
	if s == "" {
		return true
	}
	_, exempt := mitigationExemptModels[s]
	return !exempt
}
