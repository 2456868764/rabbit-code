package bashtool

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// bashShellParseSecurityRejectReason uses mvdan.cc/sh (bash grammar) to detect command and process
// substitution, complementing regex gates (readOnlyValidation / bashSecurity.ts tree-sitter path).
func bashShellParseSecurityRejectReason(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	p := syntax.NewParser(syntax.Variant(syntax.LangBash))
	f, err := p.Parse(strings.NewReader(command), "")
	if err != nil {
		return ""
	}
	var reason string
	syntax.Walk(f, func(node syntax.Node) bool {
		if reason != "" {
			return false
		}
		switch node.(type) {
		case *syntax.CmdSubst:
			reason = "command substitution $(...) detected by shell parser"
			return false
		case *syntax.ProcSubst:
			reason = "process substitution detected by shell parser"
			return false
		}
		return true
	})
	return reason
}
