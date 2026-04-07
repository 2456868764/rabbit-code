package readonlycmd

import (
	"runtime"
	"strings"
)

// IsCommandSafeViaFlagParsing mirrors readOnlyValidation isCommandSafeViaFlagParsing (allowlist + validateFlags + hooks; no READONLY_COMMAND_REGEXES).
func IsCommandSafeViaFlagParsing(raw string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	al := mergedAllowlist()
	var bestPattern string
	var bestN int
	var cfg CommandConfig
	found := false
	for pattern, c := range al {
		parts := strings.Fields(pattern)
		if len(parts) > len(tokens) {
			continue
		}
		ok := true
		for i := range parts {
			if tokens[i] != parts[i] {
				ok = false
				break
			}
		}
		if ok && len(parts) > bestN {
			bestN = len(parts)
			bestPattern = pattern
			cfg = c
			found = true
		}
	}
	if !found {
		return false
	}
	if bestPattern == "git ls-remote" && gitLsRemoteDangerous(tokens) {
		return false
	}
	for i := bestN; i < len(tokens); i++ {
		t := tokens[i]
		if strings.Contains(t, "$") {
			return false
		}
		if strings.Contains(t, "{") && (strings.Contains(t, ",") || strings.Contains(t, "..")) {
			return false
		}
	}
	commandName := tokens[0]
	if runtime.GOOS == "windows" && commandName == "xargs" {
		return false
	}
	vo := &ValidateFlagsOptions{
		CommandName:         commandName,
		RawCommand:          raw,
		XargsTargetCommands: XargsSafeTargets,
	}
	if !ValidateFlags(tokens, bestN, &cfg, vo) {
		return false
	}
	if !runHook(bestPattern, raw, tokens, bestN) {
		return false
	}
	return true
}
