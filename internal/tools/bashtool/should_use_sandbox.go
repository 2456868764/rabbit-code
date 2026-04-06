package bashtool

import "github.com/2456868764/rabbit-code/internal/features"

// ShouldUseSandbox mirrors shouldUseSandbox.ts shouldUseSandbox (no SandboxManager: env policy only).
func ShouldUseSandbox(command string, dangerouslyDisable *bool) bool {
	if !features.BashSandboxPolicyEnabled() {
		return false
	}
	if command == "" {
		return false
	}
	if dangerouslyDisable != nil && *dangerouslyDisable && features.BashAllowUnsandboxedOverride() {
		return false
	}
	return true
}
