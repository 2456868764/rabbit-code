package memdir

import (
	"fmt"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Prompt templates for the background memory extraction agent (extractMemories/prompts.ts).
// The engine wires ExtractController + RunForkedExtractMemory for the full forked loop; these builders supply the user message text.

func buildExtractOpener(newMessageCount int, existingMemories string) string {
	existingMemories = strings.TrimSpace(existingMemories)
	manifest := ""
	if existingMemories != "" {
		manifest = "\n\n## Existing memory files\n\n" + existingMemories + "\n\nCheck this list before writing — update an existing file rather than creating a duplicate."
	}
	return strings.Join([]string{
		fmt.Sprintf("You are now acting as the memory extraction subagent. Analyze the most recent ~%d messages above and use them to update your persistent memory systems.", newMessageCount),
		"",
		"Available tools: Read, Grep, Glob, read-only Bash (ls/find/cat/stat/wc/head/tail and similar), and Edit/Write for paths inside the memory directory only. Bash rm is not permitted. All other tools — MCP, Agent, write-capable Bash, etc — will be denied.",
		"",
		"You have a limited turn budget. Edit requires a prior Read of the same file, so the efficient strategy is: turn 1 — issue all Read calls in parallel for every file you might update; turn 2 — issue all Write/Edit calls in parallel. Do not interleave reads and writes across multiple turns.",
		"",
		fmt.Sprintf("You MUST only use content from the last ~%d messages to update your persistent memories. Do not waste any turns attempting to investigate or verify that content further — no grepping source files, no reading code to confirm a pattern exists, no git commands.%s", newMessageCount, manifest),
	}, "\n")
}

// BuildExtractAutoOnlyPrompt builds the extraction prompt for private auto memory only (no team directory).
func BuildExtractAutoOnlyPrompt(newMessageCount int, existingMemories string, skipIndex bool) string {
	var howToSave []string
	if skipIndex {
		howToSave = []string{
			"## How to save memories",
			"",
			"Write each memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:",
			"",
			rawFrontmatterExample,
			"",
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		}
	} else {
		howToSave = []string{
			"## How to save memories",
			"",
			"Saving a memory is a two-step process:",
			"",
			"**Step 1** — write the memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:",
			"",
			rawFrontmatterExample,
			"",
			fmt.Sprintf("**Step 2** — add a pointer to that file in `%s`. `%s` is an index, not a memory — each entry should be one line, under ~150 characters: `- [Title](file.md) — one-line hook`. It has no frontmatter. Never write memory content directly into `%s`.", EntrypointName, EntrypointName, EntrypointName),
			"",
			fmt.Sprintf("- `%s` is always loaded into your system prompt — lines after %d will be truncated, so keep the index concise", EntrypointName, MaxEntrypointLines),
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		}
	}
	lines := []string{
		buildExtractOpener(newMessageCount, existingMemories),
		"",
		"If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.",
		"",
	}
	lines = append(lines, strings.Split(strings.TrimSuffix(rawTypesSectionIndividual, "\n"), "\n")...)
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhatNotToSave, "\n"), "\n")...)
	lines = append(lines, "")
	lines = append(lines, howToSave...)
	return strings.Join(lines, "\n")
}

// BuildExtractCombinedPrompt builds the extraction prompt when team memory is enabled; otherwise it returns BuildExtractAutoOnlyPrompt.
// Pass merged settings (e.g. config.LoadMerged) so teamMemoryEnabled can be read; nil uses env only.
func BuildExtractCombinedPrompt(newMessageCount int, existingMemories string, skipIndex bool, merged map[string]interface{}) string {
	if !features.TeamMemoryEnabledFromMerged(merged) {
		return BuildExtractAutoOnlyPrompt(newMessageCount, existingMemories, skipIndex)
	}
	var howToSave []string
	if skipIndex {
		howToSave = []string{
			"## How to save memories",
			"",
			"Write each memory to its own file in the chosen directory (private or team, per the type's scope guidance) using this frontmatter format:",
			"",
			rawFrontmatterExample,
			"",
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		}
	} else {
		howToSave = []string{
			"## How to save memories",
			"",
			"Saving a memory is a two-step process:",
			"",
			"**Step 1** — write the memory to its own file in the chosen directory (private or team, per the type's scope guidance) using this frontmatter format:",
			"",
			rawFrontmatterExample,
			"",
			fmt.Sprintf("**Step 2** — add a pointer to that file in the same directory's `%s`. Each directory (private and team) has its own `%s` index — each entry should be one line, under ~150 characters: `- [Title](file.md) — one-line hook`. They have no frontmatter. Never write memory content directly into a `%s`.", EntrypointName, EntrypointName, EntrypointName),
			"",
			fmt.Sprintf("- Both `%s` indexes are loaded into your system prompt — lines after %d will be truncated, so keep them concise", EntrypointName, MaxEntrypointLines),
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		}
	}
	lines := []string{
		buildExtractOpener(newMessageCount, existingMemories),
		"",
		"If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.",
		"",
	}
	lines = append(lines, strings.Split(strings.TrimSuffix(rawTypesSectionCombined, "\n"), "\n")...)
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhatNotToSave, "\n"), "\n")...)
	lines = append(lines, "- You MUST avoid saving sensitive data within shared team memories. For example, never save API keys or user credentials.", "")
	lines = append(lines, howToSave...)
	return strings.Join(lines, "\n")
}
