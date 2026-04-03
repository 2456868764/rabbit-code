package memdir

import (
	"encoding/json"
	"strings"
)

// IsExtractReadOnlyBash is a conservative subset of BashTool.isReadOnly for the memory-extraction fork (extractMemories createAutoMemCanUseTool).
func IsExtractReadOnlyBash(inputJSON []byte) bool {
	var in struct {
		Command string `json:"command"`
		Cmd     string `json:"cmd"`
	}
	_ = json.Unmarshal(inputJSON, &in)
	cmd := strings.TrimSpace(in.Command)
	if cmd == "" {
		cmd = strings.TrimSpace(in.Cmd)
	}
	if cmd == "" {
		return true
	}
	return isExtractReadOnlyShellCommand(cmd)
}

func isExtractReadOnlyShellCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return true
	}
	lower := strings.ToLower(cmd)
	if strings.ContainsAny(lower, "><&`|;$(){}") {
		// Redirections, pipelines, subshells — deny (too easy to hide writes).
		return false
	}
	// Split on && || ; newlines — each segment must be read-only.
	for _, part := range splitShellCompound(cmd) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !singleSegmentReadOnly(part) {
			return false
		}
	}
	return true
}

func splitShellCompound(cmd string) []string {
	var out []string
	cur := cmd
	for {
		idx := minIndexOp(cur)
		if idx < 0 {
			out = append(out, strings.TrimSpace(cur))
			break
		}
		out = append(out, strings.TrimSpace(cur[:idx]))
		rest := strings.TrimSpace(cur[idx:])
		if strings.HasPrefix(rest, "&&") {
			cur = strings.TrimSpace(rest[2:])
			continue
		}
		if strings.HasPrefix(rest, "||") {
			cur = strings.TrimSpace(rest[2:])
			continue
		}
		if strings.HasPrefix(rest, ";") {
			cur = strings.TrimSpace(rest[1:])
			continue
		}
		break
	}
	return out
}

func minIndexOp(s string) int {
	best := -1
	for _, sep := range []string{"&&", "||", ";", "\n"} {
		i := strings.Index(s, sep)
		if i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

func singleSegmentReadOnly(seg string) bool {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return true
	}
	// Strip leading env assignments FOO=bar
	for {
		eq := strings.Index(seg, "=")
		if eq <= 0 {
			break
		}
		pre := strings.TrimSpace(seg[:eq])
		if pre == "" || strings.ContainsAny(pre, " \t") {
			break
		}
		rest := strings.TrimSpace(seg[eq+1:])
		if rest == "" {
			return false
		}
		if rest[0] == '\'' || rest[0] == '"' {
			break
		}
		nextSp := strings.IndexAny(rest, " \t")
		if nextSp < 0 {
			seg = ""
			break
		}
		seg = strings.TrimSpace(rest[nextSp:])
	}
	if seg == "" {
		return true
	}
	fields := strings.Fields(seg)
	if len(fields) == 0 {
		return true
	}
	base := strings.ToLower(strings.TrimPrefix(fields[0], "./"))
	switch base {
	case "ls", "find", "grep", "egrep", "fgrep", "cat", "head", "tail", "wc", "stat", "file",
		"pwd", "echo", "true", "false", "sort", "uniq", "cut", "dirname", "basename", "realpath",
		"readlink", "which", "whereis", "date", "uname", "id", "whoami", "env", "printenv":
		return true
	case "git":
		if len(fields) < 2 {
			return true
		}
		switch strings.ToLower(fields[1]) {
		case "log", "show", "diff", "status", "branch", "rev-parse", "ls-files", "ls-tree",
			"grep", "describe", "tag":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
