package readonlycmd

import (
	"regexp"
	"strings"
)

var (
	reHostnameOnlyFlags = regexp.MustCompile(`^hostname(?:\s+(?:-[a-zA-Z]|--[a-zA-Z-]+))*\s*$`)
	rePsBSDE          = regexp.MustCompile(`^[a-zA-Z]*e[a-zA-Z]*$`)
)

// XargsSafeTargets mirrors readOnlyValidation SAFE_TARGET_COMMANDS_FOR_XARGS.
var XargsSafeTargets = []string{"echo", "printf", "wc", "grep", "head", "tail"}

func gitLsRemoteDangerous(tokens []string) bool {
	if len(tokens) < 2 || tokens[0] != "git" || tokens[1] != "ls-remote" {
		return false
	}
	for i := 2; i < len(tokens); i++ {
		t := tokens[i]
		if strings.HasPrefix(t, "-") {
			continue
		}
		if strings.Contains(t, "://") {
			return true
		}
		if strings.ContainsAny(t, "@:") {
			return true
		}
		if strings.Contains(t, "$") {
			return true
		}
	}
	return false
}

func sedHookFails(raw string) bool {
	if SedAllowlistCheck == nil {
		return true
	}
	return !SedAllowlistCheck(raw)
}

func psArgsDangerous(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if rePsBSDE.MatchString(a) {
			return true
		}
	}
	return false
}

func dateArgsDangerous(args []string) bool {
	flagsWithArgs := map[string]struct{}{
		"-d": {}, "--date": {}, "-r": {}, "--reference": {},
		"--iso-8601": {}, "--rfc-3339": {},
	}
	afterDD := false
	i := 0
	for i < len(args) {
		t := args[i]
		if afterDD {
			if !strings.HasPrefix(t, "+") {
				return true
			}
			i++
			continue
		}
		if t == "--" {
			afterDD = true
			i++
			continue
		}
		if strings.HasPrefix(t, "--") && strings.Contains(t, "=") {
			i++
			continue
		}
		if strings.HasPrefix(t, "-") {
			if _, ok := flagsWithArgs[t]; ok {
				i += 2
				continue
			}
			i++
			continue
		}
		if !strings.HasPrefix(t, "+") {
			return true
		}
		i++
	}
	return false
}

func hostnameRawFails(raw string) bool {
	return !reHostnameOnlyFlags.MatchString(strings.TrimSpace(raw))
}

func lsofArgsDangerous(args []string) bool {
	for _, a := range args {
		if a == "+m" || strings.HasPrefix(a, "+m") {
			return true
		}
	}
	return false
}

var tputDangerous = map[string]struct{}{
	"init": {}, "reset": {}, "rs1": {}, "rs2": {}, "rs3": {}, "is1": {}, "is2": {}, "is3": {},
	"iprog": {}, "if": {}, "rf": {}, "clear": {}, "flash": {}, "mc0": {}, "mc4": {}, "mc5": {},
	"mc5i": {}, "mc5p": {}, "pfkey": {}, "pfloc": {}, "pfx": {}, "pfxl": {}, "smcup": {}, "rmcup": {},
}

func tputArgsDangerous(args []string) bool {
	flagsWithArg := map[string]struct{}{"-T": {}}
	afterDD := false
	i := 0
	for i < len(args) {
		t := args[i]
		if t == "--" {
			afterDD = true
			i++
			continue
		}
		if !afterDD && t == "-S" {
			return true
		}
		if !afterDD && strings.HasPrefix(t, "-") && len(t) > 2 && !strings.HasPrefix(t, "--") && strings.ContainsRune(t, 'S') {
			return true
		}
		if !afterDD && strings.HasPrefix(t, "-") {
			if _, ok := flagsWithArg[t]; ok {
				i += 2
				continue
			}
			i++
			continue
		}
		if _, bad := tputDangerous[t]; bad {
			return true
		}
		i++
	}
	return false
}

func runHook(allowlistKey, raw string, tokens []string, startIdx int) bool {
	args := tokens[startIdx:]
	switch allowlistKey {
	case "sed":
		return !sedHookFails(raw)
	case "ps":
		return !psArgsDangerous(args)
	case "date":
		return !dateArgsDangerous(args)
	case "hostname":
		return !hostnameRawFails(raw)
	case "lsof":
		return !lsofArgsDangerous(args)
	case "tput":
		return !tputArgsDangerous(args)
	default:
		return true
	}
}
