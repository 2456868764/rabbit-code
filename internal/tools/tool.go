package tools

import (
	"context"
)

// Tool is the minimal headless execution surface aligned with Tool.ts (name, aliases, call → Run here).
// Full Zod schemas, permission hooks, and progress live in per-tool packages as Phase 6 expands.
type Tool interface {
	Name() string
	// Aliases returns optional extra lookup names (toolMatchesName in Tool.ts).
	Aliases() []string
	Run(ctx context.Context, inputJSON []byte) (resultJSON []byte, err error)
}

// MatchesName returns true if name equals the tool's primary name or any alias (findToolByName / toolMatchesName).
func MatchesName(t Tool, name string) bool {
	if t == nil {
		return false
	}
	if t.Name() == name {
		return true
	}
	for _, a := range t.Aliases() {
		if a == name {
			return true
		}
	}
	return false
}

// Find returns the first tool in list matching name or alias, or nil.
func Find(list []Tool, name string) Tool {
	for _, x := range list {
		if MatchesName(x, name) {
			return x
		}
	}
	return nil
}
