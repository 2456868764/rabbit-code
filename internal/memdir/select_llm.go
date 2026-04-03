package memdir

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// SelectMemoriesSystemPrompt mirrors findRelevantMemories.ts SELECT_MEMORIES_SYSTEM_PROMPT (H8.2).
const SelectMemoriesSystemPrompt = `You are selecting memories that will be useful to Claude Code as it processes a user's query. You will be given the user's query and a list of available memory files with their filenames and descriptions.

Return a JSON object with key "selected_memories" whose value is an array of filenames (strings) for the memories that will clearly be useful (up to 5). Only include memories that you are certain will be helpful based on their name and description.
- If you are unsure if a memory will be useful in processing the user's query, then do not include it in your list. Be selective and discerning.
- If there are no memories in the list that would clearly be useful, return {"selected_memories":[]}.
- If a list of recently-used tools is provided, do not select memories that are usage reference or API documentation for those tools. DO still select memories containing warnings, gotchas, or known issues about those tools.`

// TextCompleteFunc performs one assistant-style completion (H8.2 side-query); engine wires Anthropic streaming read.
type TextCompleteFunc func(ctx context.Context, systemPrompt, userMessage string) (assistantText string, err error)

// ParseSelectedMemoriesJSON extracts selected_memories from model output (tolerates markdown fences).
func ParseSelectedMemoriesJSON(assistantText string) ([]string, error) {
	s := strings.TrimSpace(assistantText)
	if s == "" {
		return nil, errors.New("memdir: empty model output")
	}
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i+3:]
		if j := strings.Index(rest, "```"); j >= 0 {
			inner := strings.TrimSpace(rest[:j])
			if strings.HasPrefix(inner, "json") {
				inner = strings.TrimSpace(strings.TrimPrefix(inner, "json"))
			}
			s = inner
		}
	}
	var obj struct {
		SelectedMemories []string `json:"selected_memories"`
	}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		// try to find a JSON object substring
		re := regexp.MustCompile(`\{[\s\S]*"selected_memories"[\s\S]*\}`)
		if m := re.FindString(s); m != "" {
			if err2 := json.Unmarshal([]byte(m), &obj); err2 == nil {
				return obj.SelectedMemories, nil
			}
		}
		return nil, err
	}
	return obj.SelectedMemories, nil
}

func buildMemdirUserPayload(query, manifest string, recentTools []string) string {
	var b strings.Builder
	b.WriteString("Query: ")
	b.WriteString(query)
	b.WriteString("\n\nAvailable memories:\n")
	b.WriteString(manifest)
	if len(recentTools) > 0 {
		b.WriteString("\n\nRecently used tools: ")
		b.WriteString(strings.Join(recentTools, ", "))
	}
	b.WriteString("\n\nRespond with ONLY valid JSON: {\"selected_memories\":[\"filename.md\",...]}")
	return b.String()
}
