package bashtool

import (
	"fmt"
	"os"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// PromptLead is the opening line of BashTool/prompt.ts getSimplePrompt() (before sandbox/git sections).
const PromptLead = "Executes a given bash command and returns its output."

// DescriptionFallback mirrors BashTool.ts async description() when input.description is empty.
const DescriptionFallback = "Run shell command"

// Tool names aligned with FileReadTool/FileWriteTool/FileEditTool/GlobTool/GrepTool/AgentTool/TodoWriteTool prompts.
const (
	fileReadToolName  = "Read"
	fileWriteToolName = "Write"
	fileEditToolName  = "Edit"
	globToolName = "Glob"
	grepToolName = "Grep"
)

// ToolDescription returns API/catalog description; upstream uses per-invocation description or fallback.
func ToolDescription(inputDescription string) string {
	if s := strings.TrimSpace(inputDescription); s != "" {
		return s
	}
	return DescriptionFallback
}

func envTruthyPrompt(k string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(k)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func backgroundTasksDisabledForPrompt() bool {
	return envTruthyPrompt("CLAUDE_CODE_DISABLE_BACKGROUND_TASKS") || envTruthyPrompt("RABBIT_CODE_DISABLE_BACKGROUND_TASKS")
}

func getBackgroundUsageNote() string {
	if backgroundTasksDisabledForPrompt() {
		return ""
	}
	return "You can use the `run_in_background` parameter to run the command in the background. Only use this if you don't need the result immediately and are OK being notified when the command completes later. You do not need to check the output right away - you'll be notified when it finishes. You do not need to use '&' at the end of the command when using this parameter."
}

func prependBullets(items []any) []string {
	var out []string
	for _, item := range items {
		switch v := item.(type) {
		case string:
			out = append(out, " - "+v)
		case []string:
			for _, sub := range v {
				out = append(out, "  - "+sub)
			}
		}
	}
	return out
}

// GetSimplePrompt mirrors BashTool/prompt.ts getSimplePrompt() except getSimpleSandboxSection and
// getCommitAndPRInstructions (no SandboxManager / git settings in headless — omitted when empty).
func GetSimplePrompt() string {
	embedded := features.HasEmbeddedSearchTools()
	monitor := features.MonitorToolEnabled()

	var toolPreferenceItems []any
	if !embedded {
		toolPreferenceItems = append(toolPreferenceItems,
			fmt.Sprintf("File search: Use %s (NOT find or ls)", globToolName),
			fmt.Sprintf("Content search: Use %s (NOT grep or rg)", grepToolName),
		)
	}
	toolPreferenceItems = append(toolPreferenceItems,
		fmt.Sprintf("Read files: Use %s (NOT cat/head/tail)", fileReadToolName),
		fmt.Sprintf("Edit files: Use %s (NOT sed/awk)", fileEditToolName),
		fmt.Sprintf("Write files: Use %s (NOT echo >/cat <<EOF)", fileWriteToolName),
		"Communication: Output text directly (NOT echo/printf)",
	)

	avoidCommands := "`find`, `grep`, `cat`, `head`, `tail`, `sed`, `awk`, or `echo`"
	if embedded {
		avoidCommands = "`cat`, `head`, `tail`, `sed`, `awk`, or `echo`"
	}

	multipleCommandsSubitems := []string{
		fmt.Sprintf("If the commands are independent and can run in parallel, make multiple %s tool calls in a single message. Example: if you need to run \"git status\" and \"git diff\", send a single message with two %s tool calls in parallel.", BashToolName, BashToolName),
		fmt.Sprintf("If the commands depend on each other and must run sequentially, use a single %s call with '&&' to chain them together.", BashToolName),
		"Use ';' only when you need to run commands sequentially but don't care if earlier commands fail.",
		"DO NOT use newlines to separate commands (newlines are ok in quoted strings).",
	}
	gitSubitems := []string{
		"Prefer to create a new commit rather than amending an existing commit.",
		"Before running destructive operations (e.g., git reset --hard, git push --force, git checkout --), consider whether there is a safer alternative that achieves the same goal. Only use destructive operations when they are truly the best approach.",
		"Never skip hooks (--no-verify) or bypass signing (--no-gpg-sign, -c commit.gpgsign=false) unless the user has explicitly asked for it. If a hook fails, investigate and fix the underlying issue.",
	}
	var sleepSubitems []string
	sleepSubitems = append(sleepSubitems,
		"Do not sleep between commands that can run immediately — just run them.",
	)
	if monitor {
		sleepSubitems = append(sleepSubitems,
			"Use the Monitor tool to stream events from a background process (each stdout line is a notification). For one-shot \"wait until done,\" use Bash with run_in_background instead.",
		)
	}
	sleepSubitems = append(sleepSubitems,
		"If your command is long running and you would like to be notified when it finishes — use `run_in_background`. No sleep needed.",
		"Do not retry failing commands in a sleep loop — diagnose the root cause.",
		"If waiting for a background task you started with `run_in_background`, you will be notified when it completes — do not poll.",
	)
	if monitor {
		sleepSubitems = append(sleepSubitems,
			"`sleep N` as the first command with N ≥ 2 is blocked. If you need a delay (rate limiting, deliberate pacing), keep it under 2 seconds.",
		)
	} else {
		sleepSubitems = append(sleepSubitems,
			"If you must poll an external process, use a check command (e.g. `gh run view`) rather than sleeping first.",
			"If you must sleep, keep the duration short (1-5 seconds) to avoid blocking the user.",
		)
	}

	backgroundNote := getBackgroundUsageNote()
	maxMs := MaxBashTimeoutMs()
	defMs := DefaultBashTimeoutMs()

	instructionItems := []any{
		"If your command will create new directories or files, first use this tool to run `ls` to verify the parent directory exists and is the correct location.",
		"Always quote file paths that contain spaces with double quotes in your command (e.g., cd \"path with spaces/file.txt\")",
		"Try to maintain your current working directory throughout the session by using absolute paths and avoiding usage of `cd`. You may use `cd` if the User explicitly requests it.",
		fmt.Sprintf("You may specify an optional timeout in milliseconds (up to %dms / %d minutes). By default, your command will timeout after %dms (%d minutes).", maxMs, maxMs/60000, defMs, defMs/60000),
	}
	if backgroundNote != "" {
		instructionItems = append(instructionItems, backgroundNote)
	}
	instructionItems = append(instructionItems,
		"When issuing multiple commands:",
		multipleCommandsSubitems,
		"For git commands:",
		gitSubitems,
		"Avoid unnecessary `sleep` commands:",
		sleepSubitems,
	)
	if embedded {
		instructionItems = append(instructionItems,
			"When using `find -regex` with alternation, put the longest alternative first. Example: use `'.*\\.\\(tsx\\|ts\\)'` not `'.*\\.\\(ts\\|tsx\\)'` — the second form silently skips `.tsx` files.",
		)
	}

	var b strings.Builder
	b.WriteString(PromptLead)
	b.WriteString("\n\n")
	b.WriteString("The working directory persists between commands, but shell state does not. The shell environment is initialized from the user's profile (bash or zsh).")
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("IMPORTANT: Avoid using this tool to run %s commands, unless explicitly instructed or after you have verified that a dedicated tool cannot accomplish your task. Instead, use the appropriate dedicated tool as this will provide a much better experience for the user:", avoidCommands))
	b.WriteString("\n\n")
	for _, line := range prependBullets(toolPreferenceItems) {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString(fmt.Sprintf("While the %s tool can do similar things, it's better to use the built-in tools as they provide a better user experience and make it easier to review tool calls and give permission.", BashToolName))
	b.WriteString("\n\n")
	b.WriteString("# Instructions\n")
	for _, line := range prependBullets(instructionItems) {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	// getSimpleSandboxSection / getCommitAndPRInstructions: empty in headless (defer SandboxManager + git settings).
	return strings.TrimRight(b.String(), "\n")
}
