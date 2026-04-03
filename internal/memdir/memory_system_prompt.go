package memdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

const autoMemDisplayName = "auto memory"

// MemorySystemPromptInput configures LoadMemorySystemPrompt (memdir.ts loadMemoryPrompt).
type MemorySystemPromptInput struct {
	MemoryDir   string // resolved auto-memory root (no trailing sep is OK)
	ProjectRoot string // for "## Searching past context"; empty uses cwd
	Merged      map[string]interface{}
}

// LoadMemorySystemPrompt returns unified memory instructions for the Messages API system field.
func LoadMemorySystemPrompt(in MemorySystemPromptInput) (text string, ok bool) {
	mem := strings.TrimSpace(in.MemoryDir)
	if mem == "" || !features.AutoMemoryEnabledFromMerged(in.Merged) {
		return "", false
	}
	if !features.MemorySystemPromptInjectionEnabled() {
		return "", false
	}
	proj := strings.TrimSpace(in.ProjectRoot)
	if proj == "" {
		wd, err := os.Getwd()
		if err != nil {
			proj = "."
		} else {
			proj = wd
		}
	}
	proj = filepath.Clean(proj)
	skipIdx := features.MemoryPromptSkipIndex()

	_ = EnsureMemoryDirExists(mem)

	memWithSep := mem + string(filepath.Separator)
	var body string
	switch {
	case features.KairosDailyLogMemoryEnabled():
		body = BuildAssistantDailyLogMemoryPrompt(memWithSep, proj, skipIdx)
	case features.TeamMemoryEnabledFromMerged(in.Merged):
		teamDir := TeamMemDirFromAutoMemDir(memWithSep)
		_ = EnsureMemoryDirExists(strings.TrimSuffix(teamDir, string(filepath.Separator)))
		extra := features.CoworkMemoryExtraGuidelineLines()
		body = BuildCombinedMemoryPrompt(CombinedMemoryPromptOpts{
			AutoMemDir:            memWithSep,
			TeamMemDir:            teamDir,
			ProjectDir:            proj,
			ExtraGuidelines:       extra,
			SkipIndex:             skipIdx,
			UseShellGrepForSearch: false,
		})
	default:
		extra := features.CoworkMemoryExtraGuidelineLines()
		body = BuildMemoryLinesAutoOnly(autoMemDisplayName, memWithSep, proj, extra, skipIdx, false)
	}
	if strings.TrimSpace(body) == "" {
		return "", false
	}
	var sb strings.Builder
	sb.WriteString(body)
	AppendClaudeMdStyleMemoryEntrypoints(&sb, mem, features.TeamMemoryEnabledFromMerged(in.Merged))
	return sb.String(), true
}

// BuildMemoryLinesAutoOnly mirrors memdir.ts buildMemoryLines (single auto directory, no team scope section).
func BuildMemoryLinesAutoOnly(displayName, memoryDirWithSep, projectDir string, extraGuidelines []string, skipIndex, useShellGrep bool) string {
	autoDir := strings.TrimSpace(memoryDirWithSep)
	var howToSave []string
	if skipIndex {
		howToSave = []string{
			"## How to save memories",
			"",
			"Write each memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:",
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
			"**Step 1** — write the memory to its own file (e.g., `user_role.md`, `feedback_testing.md`) using this frontmatter format:",
			"",
			rawFrontmatterExample,
			"",
			fmt.Sprintf("**Step 2** — add a pointer to that file in `%s`. `%s` is an index, not a memory — each entry should be one line, under ~150 characters: `- [Title](file.md) — one-line hook`. It has no frontmatter. Never write memory content directly into `%s`.", EntrypointName, EntrypointName, EntrypointName),
			"",
			fmt.Sprintf("- `%s` is always loaded into your conversation context — lines after %d will be truncated, so keep the index concise", EntrypointName, MaxEntrypointLines),
			"- Keep the name, description, and type fields in memory files up-to-date with the content",
			"- Organize memory semantically by topic, not chronologically",
			"- Update or remove memories that turn out to be wrong or outdated",
			"- Do not write duplicate memories. First check if there is an existing memory you can update before writing a new one.",
		}
	}
	lines := []string{
		"# " + displayName,
		"",
		fmt.Sprintf("You have a persistent, file-based memory system at `%s`. %s", autoDir, DirExistsGuidance),
		"",
		"You should build up this memory system over time so that future conversations can have a complete picture of who the user is, how they'd like to collaborate with you, what behaviors to avoid or repeat, and the context behind the work the user gives you.",
		"",
		"If the user explicitly asks you to remember something, save it immediately as whichever type fits best. If they ask you to forget something, find and remove the relevant entry.",
		"",
	}
	lines = append(lines, strings.Split(strings.TrimSuffix(rawTypesSectionIndividual, "\n"), "\n")...)
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhatNotToSave, "\n"), "\n")...)
	lines = append(lines, "")
	lines = append(lines, howToSave...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhenToAccess, "\n"), "\n")...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(strings.TrimSuffix(rawTrustingRecall, "\n"), "\n")...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(strings.TrimSuffix(rawMemoryPersistence, "\n"), "\n")...)
	for _, g := range extraGuidelines {
		g = strings.TrimSpace(g)
		if g != "" {
			lines = append(lines, g)
		}
	}
	lines = append(lines, "")
	lines = append(lines, BuildSearchingPastContextSection(autoDir, projectDir, useShellGrep)...)
	return strings.Join(lines, "\n")
}

// BuildAssistantDailyLogMemoryPrompt mirrors memdir.ts buildAssistantDailyLogPrompt (KAIROS).
func BuildAssistantDailyLogMemoryPrompt(memoryDirWithSep, projectDir string, skipIndex bool) string {
	mem := strings.TrimSuffix(strings.TrimSpace(memoryDirWithSep), string(filepath.Separator))
	logPattern := filepath.Join(mem, "logs", "YYYY", "MM", "YYYY-MM-DD.md")
	logPattern = filepath.ToSlash(logPattern)
	autoShow := mem + string(filepath.Separator)
	lines := []string{
		"# auto memory",
		"",
		fmt.Sprintf("You have a persistent, file-based memory system found at: `%s`", autoShow),
		"",
		"This session is long-lived. As you work, record anything worth remembering by **appending** to today's daily log file:",
		"",
		fmt.Sprintf("`%s`", logPattern),
		"",
		"Substitute today's date (from `currentDate` in your context) for `YYYY-MM-DD`. When the date rolls over mid-session, start appending to the new day's file.",
		"",
		"Write each entry as a short timestamped bullet. Create the file (and parent directories) on first write if it does not exist. Do not rewrite or reorganize the log — it is append-only. A separate nightly process distills these logs into `MEMORY.md` and topic files.",
		"",
		"## What to log",
		"- User corrections and preferences (\"use bun, not npm\"; \"stop summarizing diffs\")",
		"- Facts about the user, their role, or their goals",
		"- Project context that is not derivable from the code (deadlines, incidents, decisions and their rationale)",
		"- Pointers to external systems (dashboards, Linear projects, Slack channels)",
		"- Anything the user explicitly asks you to remember",
		"",
	}
	lines = append(lines, strings.Split(strings.TrimSuffix(rawWhatNotToSave, "\n"), "\n")...)
	lines = append(lines, "")
	if !skipIndex {
		lines = append(lines,
			fmt.Sprintf("## %s", EntrypointName),
			fmt.Sprintf("`%s` is the distilled index (maintained nightly from your logs) and is loaded into your context automatically. Read it for orientation, but do not edit it directly — record new information in today's log instead.", EntrypointName),
			"",
		)
	}
	lines = append(lines, BuildSearchingPastContextSection(autoShow, projectDir, false)...)
	return strings.Join(lines, "\n")
}

// AppendClaudeMdStyleMemoryEntrypoints appends truncated MEMORY.md bodies (claudemd getMemoryFiles analogue).
func AppendClaudeMdStyleMemoryEntrypoints(sb *strings.Builder, autoMemRoot string, includeTeam bool) {
	autoMemRoot = filepath.Clean(strings.TrimSpace(autoMemRoot))
	if autoMemRoot == "" {
		return
	}
	appendOneEntrypoint(sb, "Private "+EntrypointName, filepath.Join(autoMemRoot, EntrypointName))
	if includeTeam {
		appendOneEntrypoint(sb, "Team "+EntrypointName, filepath.Join(autoMemRoot, TeamMemSubdir, EntrypointName))
	}
}

func appendOneEntrypoint(sb *strings.Builder, title, path string) {
	b, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(b)) == "" {
		sb.WriteString("\n\n## ")
		sb.WriteString(title)
		sb.WriteString("\n\n")
		sb.WriteString("_(empty or unreadable — new memories will appear here.)_")
		return
	}
	t := TruncateEntrypointContent(string(b))
	sb.WriteString("\n\n## ")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	sb.WriteString(t.Content)
}
