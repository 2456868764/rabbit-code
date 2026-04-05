package memdir

// Corresponds to restored-src/src/memdir/memdir.ts (guidance, entrypoint, ensure dir, searching-past-context, system prompt).

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// DirExistsGuidance and DirsExistGuidance are appended to memory directory prompts (memdir.ts).
const DirExistsGuidance = "This directory already exists — write to it directly with the Write tool (do not run mkdir or check for its existence)."
const DirsExistGuidance = "Both directories already exist — write to them directly with the Write tool (do not run mkdir or check for their existence)."

// EnsureMemoryDirExists creates memoryDir recursively (memdir.ts ensureMemoryDirExists); ignores EEXIST.
func EnsureMemoryDirExists(memoryDir string) error {
	memoryDir = strings.TrimSpace(memoryDir)
	if memoryDir == "" {
		return nil
	}
	return os.MkdirAll(memoryDir, 0o700)
}

// Entrypoint file name for memory index (memdir.ts ENTRYPOINT_NAME).
const EntrypointName = "MEMORY.md"

// MaxEntrypointLines caps MEMORY.md line count before truncation (memdir.ts).
const MaxEntrypointLines = 200

// MaxEntrypointBytes caps MEMORY.md UTF-8 size before truncation (memdir.ts).
const MaxEntrypointBytes = 25_000

// EntrypointTruncation is the result of TruncateEntrypointContent (memdir.ts).
type EntrypointTruncation struct {
	Content          string
	LineCount        int
	ByteCount        int
	WasLineTruncated bool
	WasByteTruncated bool
}

// formatFileSizeBytes matches utils/format.ts formatFileSize for MEMORY.md truncation warnings (memdir.ts).
func formatFileSizeBytes(sizeInBytes int) string {
	kb := float64(sizeInBytes) / 1024
	if kb < 1 {
		return strconv.Itoa(sizeInBytes) + " bytes"
	}
	if kb < 1024 {
		return trimOneDecimalFileSize(kb) + "KB"
	}
	mb := kb / 1024
	if mb < 1024 {
		return trimOneDecimalFileSize(mb) + "MB"
	}
	gb := mb / 1024
	return trimOneDecimalFileSize(gb) + "GB"
}

func trimOneDecimalFileSize(x float64) string {
	s := strconv.FormatFloat(x, 'f', 1, 64)
	s = strings.TrimSuffix(strings.TrimSuffix(s, "0"), ".")
	return s
}

// TruncateEntrypointContent applies line then byte caps and appends a warning (memdir.ts truncateEntrypointContent).
func TruncateEntrypointContent(raw string) EntrypointTruncation {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return EntrypointTruncation{Content: "", LineCount: 0, ByteCount: 0}
	}
	contentLines := strings.Split(trimmed, "\n")
	lineCount := len(contentLines)
	byteCount := len(trimmed)

	wasLineTruncated := lineCount > MaxEntrypointLines
	wasByteTruncated := byteCount > MaxEntrypointBytes

	if !wasLineTruncated && !wasByteTruncated {
		return EntrypointTruncation{
			Content:          trimmed,
			LineCount:        lineCount,
			ByteCount:        byteCount,
			WasLineTruncated: false,
			WasByteTruncated: false,
		}
	}

	truncated := trimmed
	if wasLineTruncated {
		truncated = strings.Join(contentLines[:MaxEntrypointLines], "\n")
	}

	if len(truncated) > MaxEntrypointBytes {
		search := truncated
		if len(search) > MaxEntrypointBytes {
			search = search[:MaxEntrypointBytes]
		}
		cutAt := strings.LastIndex(search, "\n")
		if cutAt <= 0 {
			truncated = truncated[:MaxEntrypointBytes]
		} else {
			truncated = truncated[:cutAt]
		}
	}

	var reason string
	switch {
	case wasByteTruncated && !wasLineTruncated:
		reason = fmt.Sprintf("%s (limit: %s) — index entries are too long", formatFileSizeBytes(byteCount), formatFileSizeBytes(MaxEntrypointBytes))
	case wasLineTruncated && !wasByteTruncated:
		reason = fmt.Sprintf("%d lines (limit: %d)", lineCount, MaxEntrypointLines)
	default:
		reason = fmt.Sprintf("%d lines and %s", lineCount, formatFileSizeBytes(byteCount))
	}

	warning := fmt.Sprintf(
		"\n\n> WARNING: %s is %s. Only part of it was loaded. Keep index entries to one line under ~200 chars; move detail into topic files.",
		EntrypointName, reason,
	)

	return EntrypointTruncation{
		Content:          truncated + warning,
		LineCount:        lineCount,
		ByteCount:        byteCount,
		WasLineTruncated: wasLineTruncated,
		WasByteTruncated: wasByteTruncated,
	}
}

// BuildSearchingPastContextSection returns the "## Searching past context" block (memdir.ts buildSearchingPastContextSection).
// TS gates on GrowthBook tengu_coral_fern; Go uses features.MemorySearchPastContextEnabled (RABBIT_CODE_MEMORY_SEARCH_PAST_CONTEXT).
// useShellGrep selects shell grep vs Grep-tool wording (TS: hasEmbeddedSearchTools || isReplModeEnabled).
func BuildSearchingPastContextSection(autoMemDir, projectDir string, useShellGrep bool) []string {
	if !features.MemorySearchPastContextEnabled() {
		return nil
	}
	autoMemDir = strings.TrimSpace(autoMemDir)
	projectDir = filepath.Clean(strings.TrimSpace(projectDir))
	if autoMemDir == "" || projectDir == "" {
		return nil
	}
	projWithSep := projectDir + string(filepath.Separator)
	var memSearch, transcriptSearch string
	if useShellGrep {
		memSearch = fmt.Sprintf(`grep -rn "<search term>" %s --include="*.md"`, autoMemDir)
		transcriptSearch = fmt.Sprintf(`grep -rn "<search term>" %s --include="*.jsonl"`, projWithSep)
	} else {
		memSearch = fmt.Sprintf(`Grep with pattern="<search term>" path="%s" glob="*.md"`, autoMemDir)
		transcriptSearch = fmt.Sprintf(`Grep with pattern="<search term>" path="%s" glob="*.jsonl"`, projWithSep)
	}
	return []string{
		"## Searching past context",
		"",
		"When looking for past context:",
		"1. Search topic files in your memory directory:",
		"```",
		memSearch,
		"```",
		"2. Session transcript logs (last resort — large files, slow):",
		"```",
		transcriptSearch,
		"```",
		"Use narrow search terms (error messages, file paths, function names) rather than broad keywords.",
		"",
	}
}

const autoMemDisplayName = "auto memory"

// MemorySystemPromptInput configures LoadMemorySystemPrompt (memdir.ts loadMemoryPrompt).
type MemorySystemPromptInput struct {
	MemoryDir   string // resolved auto-memory root (no trailing sep is OK)
	ProjectRoot string // for "## Searching past context"; empty uses cwd
	Merged      map[string]interface{}
}

// BuildMemoryPromptInput mirrors memdir.ts buildMemoryPrompt parameter object.
type BuildMemoryPromptInput struct {
	DisplayName     string
	MemoryDir       string // shown in prompt and used to locate MEMORY.md; trailing separator optional
	ExtraGuidelines []string
	// SkipIndex maps to buildMemoryLines(skipIndex); memdir.ts buildMemoryPrompt calls buildMemoryLines without skip → false.
	SkipIndex bool
	// ProjectRoot is for BuildSearchingPastContextSection transcript path; empty uses os.Getwd (TS: getProjectDir(getOriginalCwd())).
	ProjectRoot string
	// UseShellGrep selects shell grep vs Grep-tool wording (TS embedded-search / REPL branch).
	UseShellGrep bool
}

// BuildMemoryPrompt mirrors memdir.ts buildMemoryPrompt: buildMemoryLines plus an inline "## MEMORY.md" body.
// It does not mkdir (TS: callers ensure the directory when needed).
//
// LoadMemorySystemPrompt implements memdir.ts loadMemoryPrompt for the main agent: it gates on auto-memory
// and memory-prompt injection features, calls EnsureMemoryDirExists, dispatches KAIROS / combined team / auto-only
// bodies, then AppendClaudeMdStyleMemoryEntrypoints (Private + optional Team MEMORY.md). Use BuildMemoryPrompt when
// you need the agent-memory shape that inlines a single entrypoint without those gates.
func BuildMemoryPrompt(p BuildMemoryPromptInput) string {
	mem := strings.TrimSpace(p.MemoryDir)
	if mem == "" {
		return ""
	}
	root := filepath.Clean(mem)
	memWithSep := root + string(filepath.Separator)

	proj := strings.TrimSpace(p.ProjectRoot)
	if proj == "" {
		wd, err := os.Getwd()
		if err != nil {
			proj = "."
		} else {
			proj = filepath.Clean(wd)
		}
	} else {
		proj = filepath.Clean(proj)
	}

	body := BuildMemoryLinesAutoOnly(p.DisplayName, memWithSep, proj, p.ExtraGuidelines, p.SkipIndex, p.UseShellGrep)
	ep := filepath.Join(root, EntrypointName)
	var sb strings.Builder
	sb.WriteString(body)
	b, err := os.ReadFile(ep)
	if err != nil || strings.TrimSpace(string(b)) == "" {
		sb.WriteString("\n\n## ")
		sb.WriteString(EntrypointName)
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf("Your %s is currently empty. When you save new memories, they will appear here.", EntrypointName))
		return sb.String()
	}
	t := TruncateEntrypointContent(string(b))
	sb.WriteString("\n\n## ")
	sb.WriteString(EntrypointName)
	sb.WriteString("\n\n")
	sb.WriteString(t.Content)
	return sb.String()
}

// LoadMemorySystemPrompt returns unified memory instructions for the Messages API system field (memdir.ts loadMemoryPrompt).
// Extra Go gates: features.MemorySystemPromptInjectionEnabled (RABBIT_CODE_MEMORY_SYSTEM_PROMPT). TS telemetry (logMemoryDirCounts, tengu_* events) is omitted in headless.
// Auto-only TS returns buildMemoryLines only and relies on claudemd for MEMORY.md; Go appends Private/Team MEMORY.md via AppendClaudeMdStyleMemoryEntrypoints for a single system string.
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
			MemoryFrontmatterExampleBlock(),
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
			MemoryFrontmatterExampleBlock(),
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
	lines = append(lines, TypesSectionIndividual()...)
	lines = append(lines, WhatNotToSaveSection()...)
	lines = append(lines, "")
	lines = append(lines, howToSave...)
	lines = append(lines, "")
	lines = append(lines, WhenToAccessSection()...)
	lines = append(lines, "")
	lines = append(lines, TrustingRecallSection()...)
	lines = append(lines, "")
	lines = append(lines, MemoryAndPersistenceSection()...)
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
	lines = append(lines, WhatNotToSaveSection()...)
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

// SessionFragments returns optional extra system prompt lines injected at session start.
// Stub returns nil until full parity with upstream session injection.
func SessionFragments() []string {
	return nil
}
