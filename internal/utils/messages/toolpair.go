package messages

import (
	"fmt"

	"github.com/2456868764/rabbit-code/internal/types"
)

// ValidateToolPairing checks assistant tool_use blocks are followed by matching user tool_results.
// When strict is false, missing trailing tool_result does not error (streaming / incomplete transcript).
func ValidateToolPairing(msgs []types.Message, strict bool) error {
	for i := range msgs {
		if msgs[i].Role != types.RoleAssistant {
			continue
		}
		need := toolUseIDs(msgs[i].Content)
		if len(need) == 0 {
			continue
		}
		if i+1 >= len(msgs) {
			if strict {
				return fmt.Errorf("message %d (assistant): tool_use ids %v but no following message for tool_result", i, need)
			}
			continue
		}
		next := msgs[i+1]
		if next.Role != types.RoleUser {
			if strict {
				return fmt.Errorf("message %d (assistant): expected next role user for tool_result, got %q", i, next.Role)
			}
			continue
		}
		got := toolResultIDs(next.Content)
		missing, extra := diffToolIDCounts(need, got)
		if len(missing) > 0 {
			return fmt.Errorf("message %d (assistant) / message %d (user) tool_result mismatch: missing tool_result for id(s) %v",
				i, i+1, missing)
		}
		if strict && len(extra) > 0 {
			return fmt.Errorf("message %d (assistant) / message %d (user) tool_result mismatch: unexpected tool_result id(s) %v",
				i, i+1, extra)
		}
	}
	return nil
}

func toolUseIDs(c []types.ContentPiece) []string {
	var ids []string
	for _, p := range c {
		if p.Type == types.BlockTypeToolUse && p.ID != "" {
			ids = append(ids, p.ID)
		}
	}
	return ids
}

func toolResultIDs(c []types.ContentPiece) []string {
	var ids []string
	for _, p := range c {
		if p.Type == types.BlockTypeToolResult && p.ToolUseID != "" {
			ids = append(ids, p.ToolUseID)
		}
	}
	return ids
}

func diffToolIDCounts(need, got []string) (missing, extra []string) {
	nc := countIDs(need)
	gc := countIDs(got)
	for id, n := range nc {
		g := gc[id]
		for i := 0; i < n-g; i++ {
			missing = append(missing, id)
		}
	}
	for id, g := range gc {
		n := nc[id]
		for i := 0; i < g-n; i++ {
			extra = append(extra, id)
		}
	}
	return missing, extra
}

func countIDs(ids []string) map[string]int {
	m := make(map[string]int)
	for _, id := range ids {
		m[id]++
	}
	return m
}
