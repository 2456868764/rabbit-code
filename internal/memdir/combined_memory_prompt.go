package memdir

import (
	"fmt"
	"strings"
)

// CombinedMemoryPromptOpts configures buildCombinedMemoryPrompt (teamMemPrompts.ts buildCombinedMemoryPrompt).
type CombinedMemoryPromptOpts struct {
	AutoMemDir            string
	TeamMemDir            string
	ProjectDir            string // session / project root for optional "## Searching past context"
	ExtraGuidelines       []string
	SkipIndex             bool
	UseShellGrepForSearch bool
}

// BuildCombinedMemoryPrompt returns the full system prompt block for private + team memory (TEAMMEM + auto memory).
func BuildCombinedMemoryPrompt(opt CombinedMemoryPromptOpts) string {
	autoDir := strings.TrimSpace(opt.AutoMemDir)
	teamDir := strings.TrimSpace(opt.TeamMemDir)
	if autoDir == "" || teamDir == "" {
		return ""
	}
	var howToSave []string
	if opt.SkipIndex {
		howToSave = []string{
			"## How to save memories",
			"",
			"Write each memory to its own file in the chosen directory (private or team, per the type's scope guidance) using this frontmatter format:",
			"",
			rawFrontmatterExample,
			"",
			"- Keep the name, description, and type fields in memory files up-to-date with the content",
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
			fmt.Sprintf("- Both `%s` indexes are loaded into your conversation context — lines after %d will be truncated, so keep them concise", EntrypointName, MaxEntrypointLines),
			"- Keep the name, description, and type fields in memory files up-to-date with the content",
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		}
	}

	lines := []string{
		"# Memory",
		"",
		fmt.Sprintf("You have a persistent, file-based memory system with two directories: a private directory at `%s` and a shared team directory at `%s`. %s", autoDir, teamDir, DirsExistGuidance),
		"",
		"You should build up this memory system over time so that future conversations can have a complete picture of who the user is, how they'd like to collaborate with you, what behaviors to avoid or repeat, and the context behind the work the user gives you.",
		"",
		"If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.",
		"",
		"## Memory scope",
		"",
		"There are two scope levels:",
		"",
		fmt.Sprintf("- private: memories that are private between you and the current user. They persist across conversations with only this specific user and are stored at the root `%s`.", autoDir),
		fmt.Sprintf("- team: memories that are shared with and contributed by all of the users who work within this project directory. Team memories are synced at the beginning of every session and they are stored at `%s`.", teamDir),
		"",
	}
	lines = append(lines, strings.Split(strings.TrimSuffix(rawTypesSectionCombined, "\n"), "\n")...)
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhatNotToSave, "\n"), "\n")...)
	lines = append(lines,
		"- You MUST avoid saving sensitive data within shared team memories. For example, never save API keys or user credentials.",
		"",
	)
	lines = append(lines, howToSave...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhenToAccessCombined, "\n"), "\n")...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(strings.TrimSuffix(rawTrustingRecall, "\n"), "\n")...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(strings.TrimSuffix(rawMemoryPersistence, "\n"), "\n")...)
	for _, g := range opt.ExtraGuidelines {
		g = strings.TrimSpace(g)
		if g != "" {
			lines = append(lines, g)
		}
	}
	lines = append(lines, "")
	lines = append(lines, BuildSearchingPastContextSection(autoDir, opt.ProjectDir, opt.UseShellGrepForSearch)...)
	return strings.Join(lines, "\n")
}
