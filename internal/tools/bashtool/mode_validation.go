package bashtool

import "strings"

// PermissionMode mirrors ToolPermissionContext.mode strings used in modeValidation.ts.
type PermissionMode string

const (
	ModeBypassPermissions PermissionMode = "bypassPermissions"
	ModeDontAsk           PermissionMode = "dontAsk"
	ModeAcceptEdits       PermissionMode = "acceptEdits"
	// ModeDefault corresponds to the standard "default" mode in TS ToolPermissionContext.
	ModeDefault PermissionMode = "default"
)

var acceptEditsFilesystemCommands = map[string]struct{}{
	"mkdir": {}, "touch": {}, "rm": {}, "rmdir": {}, "mv": {}, "cp": {}, "sed": {},
}

// ModeCheckResult mirrors PermissionResult.behavior subset for headless bashtool.
type ModeCheckResult struct {
	Allow       bool
	Passthrough bool
	Reason      string
}

// GetAutoAllowedCommands mirrors modeValidation.ts getAutoAllowedCommands.
// Returns the list of commands auto-allowed in the given mode (only non-empty for acceptEdits).
func GetAutoAllowedCommands(mode PermissionMode) []string {
	if mode != ModeAcceptEdits {
		return nil
	}
	result := make([]string, 0, len(acceptEditsFilesystemCommands))
	for k := range acceptEditsFilesystemCommands {
		result = append(result, k)
	}
	return result
}

// CheckPermissionMode mirrors modeValidation.ts checkPermissionMode (Accept Edits filesystem auto-allow only).
func CheckPermissionMode(command string, mode PermissionMode) ModeCheckResult {
	switch mode {
	case ModeBypassPermissions:
		return ModeCheckResult{Passthrough: true, Reason: "Bypass mode is handled in main permission flow"}
	case ModeDontAsk:
		return ModeCheckResult{Passthrough: true, Reason: "DontAsk mode is handled in main permission flow"}
	case ModeAcceptEdits:
		for _, cmd := range SplitCommandDeprecated(command) {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			base := strings.Fields(cmd)[0]
			if _, ok := acceptEditsFilesystemCommands[base]; ok {
				return ModeCheckResult{Allow: true, Reason: "acceptEdits filesystem command"}
			}
		}
		return ModeCheckResult{Passthrough: true, Reason: "No mode-specific handling in acceptEdits"}
	default:
		return ModeCheckResult{Passthrough: true, Reason: "No mode-specific handling"}
	}
}
