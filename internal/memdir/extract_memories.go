package memdir

// Background memory extraction fork (restored-src extractMemories / prompts parity).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
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
		return false
	}
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

// AutoMemToolRunner wraps a ToolRunner and enforces createAutoMemCanUseTool rules (extractMemories.ts).
type AutoMemToolRunner struct {
	Inner     query.ToolRunner
	MemoryDir string
}

// RunTool implements query.ToolRunner.
func (w *AutoMemToolRunner) RunTool(ctx context.Context, name string, inputJSON []byte) ([]byte, error) {
	if w.Inner == nil {
		return nil, query.ErrNoToolRunner
	}
	memRoot := strings.TrimSpace(w.MemoryDir)
	n := strings.TrimSpace(name)

	switch {
	case strings.EqualFold(n, "Read"), strings.EqualFold(n, "Grep"), strings.EqualFold(n, "Glob"):
		return w.Inner.RunTool(ctx, name, inputJSON)
	case strings.EqualFold(n, "REPL"):
		return w.Inner.RunTool(ctx, name, inputJSON)
	case strings.EqualFold(n, "bash"), strings.EqualFold(n, "Bash"):
		if IsExtractReadOnlyBash(inputJSON) {
			return w.Inner.RunTool(ctx, name, inputJSON)
		}
		return nil, fmt.Errorf("memdir: only read-only shell commands are permitted in this context")
	case strings.EqualFold(n, "Write"), strings.EqualFold(n, "Edit"):
		fp, ok := extractJSONFilePath(inputJSON)
		if ok && memRoot != "" && IsAutoMemPath(fp, memRoot) {
			return w.Inner.RunTool(ctx, name, inputJSON)
		}
		return nil, fmt.Errorf("memdir: Write/Edit only allowed under auto-memory directory")
	default:
		return nil, fmt.Errorf("memdir: tool %q denied in extract context", name)
	}
}

func extractJSONFilePath(inputJSON []byte) (string, bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(inputJSON, &m); err != nil {
		return "", false
	}
	v, ok := m["file_path"]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

// CountModelVisibleMessagesSince counts user + assistant messages after the message with sinceUUID (extractMemories.ts).
func CountModelVisibleMessagesSince(msgs json.RawMessage, sinceUUID, uuidField string) int {
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}
	arr, err := parseTopMessagesArray(msgs)
	if err != nil || len(arr) == 0 {
		return 0
	}
	start := 0
	if sinceUUID != "" {
		found := false
		for i, raw := range arr {
			id := topLevelStringField(raw, uuidField)
			if id == sinceUUID {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			start = 0
		}
	}
	n := 0
	for i := start; i < len(arr); i++ {
		role := topLevelStringField(arr[i], "role")
		if role == "user" || role == "assistant" {
			n++
		}
	}
	return n
}

// HasMemoryWritesSince is true if any assistant message after sinceUUID contains Write/Edit tool_use targeting autoMemDir.
func HasMemoryWritesSince(msgs json.RawMessage, sinceUUID, autoMemDir, uuidField string) bool {
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}
	arr, err := parseTopMessagesArray(msgs)
	if err != nil {
		return false
	}
	foundStart := sinceUUID == ""
	for _, raw := range arr {
		if !foundStart {
			if topLevelStringField(raw, uuidField) == sinceUUID {
				foundStart = true
			}
			continue
		}
		if topLevelStringField(raw, "role") != "assistant" {
			continue
		}
		for _, fp := range toolUseFilePathsFromAssistantMessage(raw, toolNameWrite, toolNameEdit) {
			if IsAutoMemPath(fp, autoMemDir) {
				return true
			}
		}
	}
	return false
}

// WrittenMemoryPathsFromTranscriptSuffix returns Write/Edit file_path values from assistant messages at indices >= startIdx.
func WrittenMemoryPathsFromTranscriptSuffix(msgs json.RawMessage, startIdx int) []string {
	arr, err := parseTopMessagesArray(msgs)
	if err != nil {
		return nil
	}
	if startIdx < 0 {
		startIdx = 0
	}
	var out []string
	seen := make(map[string]struct{})
	for i := startIdx; i < len(arr); i++ {
		if topLevelStringField(arr[i], "role") != "assistant" {
			continue
		}
		for _, fp := range toolUseFilePathsFromAssistantMessage(arr[i], toolNameWrite, toolNameEdit) {
			fp = strings.TrimSpace(fp)
			if fp == "" {
				continue
			}
			if _, ok := seen[fp]; ok {
				continue
			}
			seen[fp] = struct{}{}
			out = append(out, fp)
		}
	}
	return out
}

// LastEmbeddedMessageUUID returns the uuidField value on the last top-level message, if any.
func LastEmbeddedMessageUUID(msgs json.RawMessage, uuidField string) string {
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}
	arr, err := parseTopMessagesArray(msgs)
	if err != nil || len(arr) == 0 {
		return ""
	}
	return topLevelStringField(arr[len(arr)-1], uuidField)
}

// TranscriptMessageCount returns the number of top-level messages in the API array.
func TranscriptMessageCount(msgs json.RawMessage) int {
	arr, err := parseTopMessagesArray(msgs)
	if err != nil {
		return 0
	}
	return len(arr)
}

const (
	toolNameWrite = "Write"
	toolNameEdit  = "Edit"
)

func parseTopMessagesArray(msgs json.RawMessage) ([]json.RawMessage, error) {
	raw := bytes.TrimSpace(msgs)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func topLevelStringField(msg json.RawMessage, field string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		return ""
	}
	v, ok := m[field]
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(v, &s)
	return s
}

func toolUseFilePathsFromAssistantMessage(msg json.RawMessage, toolNames ...string) []string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		return nil
	}
	rawContent, ok := m["content"]
	if !ok {
		return nil
	}
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(rawContent, &blocks); err != nil {
		return nil
	}
	var paths []string
	for _, b := range blocks {
		t, _ := jsonStringFieldFromMap(b, "type")
		if t != "tool_use" {
			continue
		}
		name, _ := jsonStringFieldFromMap(b, "name")
		if !toolNameMatches(name, toolNames) {
			continue
		}
		inRaw, ok := b["input"]
		if !ok {
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(inRaw, &input); err != nil {
			continue
		}
		if fpRaw, ok := input["file_path"]; ok {
			var fp string
			_ = json.Unmarshal(fpRaw, &fp)
			if fp != "" {
				paths = append(paths, fp)
			}
		}
	}
	return paths
}

func toolNameMatches(name string, allowed []string) bool {
	for _, n := range allowed {
		if strings.EqualFold(strings.TrimSpace(name), n) {
			return true
		}
	}
	return false
}

func jsonStringFieldFromMap(m map[string]json.RawMessage, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", false
	}
	return s, true
}

// Prompt templates for the background memory extraction agent (extractMemories/prompts.ts).

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

// ForkedExtractParams runs a bounded tool loop seeded with parent transcript + extract user prompt (runForkedAgent analogue).
type ForkedExtractParams struct {
	ParentMessagesJSON json.RawMessage
	UserPrompt         string
	MemoryDir          string
	MaxTurns           int
	QuerySource        string
	Merged             map[string]interface{}
	NonInteractive     bool
}

// ForkedExtractDeps supplies model and backends (same Deps as main engine).
type ForkedExtractDeps struct {
	Tools     query.ToolRunner
	Turn      query.TurnAssistant
	Model     string
	MaxTokens int
}

// ForkedExtractResult is the transcript after the fork and paths written under MemoryDir (topic files may include team/).
type ForkedExtractResult struct {
	MessagesJSON     json.RawMessage
	ParentMsgCount   int
	WrittenPaths     []string
	MemoryFilePaths  []string
	TeamMemoryWrites int
}

// RunForkedExtractMemory executes the extract sub-loop with auto-mem tool gating.
func RunForkedExtractMemory(ctx context.Context, dep ForkedExtractDeps, p ForkedExtractParams) (ForkedExtractResult, error) {
	var out ForkedExtractResult
	if dep.Turn == nil || dep.Tools == nil {
		return out, query.ErrNoToolRunner
	}
	memDir := strings.TrimSpace(p.MemoryDir)
	if memDir == "" {
		return out, nil
	}
	parentCount := TranscriptMessageCount(p.ParentMessagesJSON)
	out.ParentMsgCount = parentCount

	seed, err := query.AppendUserTextMessage(p.ParentMessagesJSON, p.UserPrompt)
	if err != nil {
		return out, err
	}

	inner := dep.Tools
	if features.TeamMemoryEnabledFromMerged(p.Merged) && memDir != "" {
		inner = &TeamMemSecretGuardRunner{Inner: inner, AutoMemDir: memDir, Enabled: true}
	}
	wrapped := &AutoMemToolRunner{Inner: inner, MemoryDir: memDir}
	d := query.LoopDriver{
		Deps: query.Deps{
			Tools: wrapped,
			Turn:  dep.Turn,
		},
		Model:       dep.Model,
		MaxTokens:   dep.MaxTokens,
		QuerySource: strings.TrimSpace(p.QuerySource),
	}
	maxTurns := p.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 5
	}
	st := &query.LoopState{
		MaxTurns: maxTurns,
		ToolUseContext: query.ToolUseContextMirror{
			QuerySource: d.QuerySource,
		},
	}

	msgs, _, err := d.RunTurnLoopFromMessages(ctx, st, seed)
	if err != nil {
		return out, err
	}
	out.MessagesJSON = msgs
	out.WrittenPaths = WrittenMemoryPathsFromTranscriptSuffix(msgs, parentCount)
	for _, path := range out.WrittenPaths {
		if filepath.Base(path) == EntrypointName {
			continue
		}
		out.MemoryFilePaths = append(out.MemoryFilePaths, path)
		if features.TeamMemoryEnabledFromMerged(p.Merged) && IsTeamMemPathUnderAutoMem(path, memDir+string(filepath.Separator)) {
			out.TeamMemoryWrites++
		}
	}
	return out, nil
}

// ExtractController coordinates background memory extraction (initExtractMemories closure in extractMemories.ts).
type ExtractController struct {
	mu sync.Mutex
	wg sync.WaitGroup

	lastMessageUUID string
	inProgress      bool
	pending         *extractPending
	turnsSince      int
}

type extractPending struct {
	MessagesJSON json.RawMessage
	Merged       map[string]interface{}
}

// ExtractHookArgs is passed from the engine stop hook into the controller.
type ExtractHookArgs struct {
	LoopErr        error
	MessagesJSON   json.RawMessage
	MemoryDir      string
	Merged         map[string]interface{}
	NonInteractive bool
	AgentID        string
	Deps           query.Deps
	Model          string
	MaxTokens      int
	OnMemorySaved  func(memoryPaths []string, teamCount int)
	UUIDField      string
	IsTrailingRun  bool
}

// HandleStopHook mirrors executeExtractMemories (fire-and-forget from the engine).
func (c *ExtractController) HandleStopHook(ctx context.Context, a ExtractHookArgs) {
	if c == nil {
		return
	}
	if a.LoopErr != nil {
		return
	}
	if strings.TrimSpace(a.AgentID) != "" {
		return
	}
	if !features.ExtractMemoriesAllowed(a.NonInteractive) {
		return
	}
	if features.RemoteModeWithoutMemoryDir() {
		return
	}
	memDir := strings.TrimSpace(a.MemoryDir)
	if memDir == "" {
		return
	}
	if !features.AutoMemoryEnabledFromMerged(a.Merged) {
		return
	}

	uuidField := strings.TrimSpace(a.UUIDField)
	if uuidField == "" {
		uuidField = query.RabbitMessageUUIDKey
	}

	c.mu.Lock()
	if c.inProgress {
		c.pending = &extractPending{MessagesJSON: cloneRawJSON(a.MessagesJSON), Merged: a.Merged}
		c.mu.Unlock()
		return
	}
	if !a.IsTrailingRun {
		c.turnsSince++
		if c.turnsSince < features.ExtractMemoriesInterval() {
			c.mu.Unlock()
			return
		}
	}
	c.turnsSince = 0
	sinceUUID := c.lastMessageUUID
	c.mu.Unlock()

	if HasMemoryWritesSince(a.MessagesJSON, sinceUUID, memDir, uuidField) {
		c.mu.Lock()
		if u := LastEmbeddedMessageUUID(a.MessagesJSON, uuidField); u != "" {
			c.lastMessageUUID = u
		}
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.inProgress = true
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.execAndMaybeTrail(ctx, a, uuidField, memDir)
	}()
}

func cloneRawJSON(r json.RawMessage) json.RawMessage {
	if len(r) == 0 {
		return nil
	}
	return json.RawMessage(append([]byte(nil), r...))
}

func (c *ExtractController) execAndMaybeTrail(ctx context.Context, a ExtractHookArgs, uuidField, memDir string) {
	defer func() {
		c.mu.Lock()
		c.inProgress = false
		p := c.pending
		c.pending = nil
		c.mu.Unlock()
		if p != nil {
			next := a
			next.MessagesJSON = p.MessagesJSON
			next.Merged = p.Merged
			next.IsTrailingRun = true
			c.execAndMaybeTrail(ctx, next, uuidField, memDir)
		}
	}()

	c.mu.Lock()
	since := c.lastMessageUUID
	c.mu.Unlock()
	newCount := CountModelVisibleMessagesSince(a.MessagesJSON, since, uuidField)

	headers, _ := ScanMemoryFiles(ctx, memDir)
	manifest := FormatMemoryManifest(headers)

	skipIdx := features.ExtractMemoriesSkipIndex()
	var userPrompt string
	if features.TeamMemoryEnabledFromMerged(a.Merged) {
		userPrompt = BuildExtractCombinedPrompt(newCount, manifest, skipIdx, a.Merged)
	} else {
		userPrompt = BuildExtractAutoOnlyPrompt(newCount, manifest, skipIdx)
	}

	dep := ForkedExtractDeps{
		Tools:     a.Deps.Tools,
		Turn:      a.Deps.Turn,
		Model:     a.Model,
		MaxTokens: a.MaxTokens,
	}
	res, err := RunForkedExtractMemory(ctx, dep, ForkedExtractParams{
		ParentMessagesJSON: a.MessagesJSON,
		UserPrompt:         userPrompt,
		MemoryDir:          memDir,
		MaxTurns:           5,
		QuerySource:        query.QuerySourceExtractMemories,
		NonInteractive:     a.NonInteractive,
		Merged:             a.Merged,
	})
	if err != nil {
		return
	}

	c.mu.Lock()
	if u := LastEmbeddedMessageUUID(a.MessagesJSON, uuidField); u != "" {
		c.lastMessageUUID = u
	}
	c.mu.Unlock()

	if len(res.MemoryFilePaths) > 0 && a.OnMemorySaved != nil {
		a.OnMemorySaved(res.MemoryFilePaths, res.TeamMemoryWrites)
	}
}

// Wait blocks until in-flight extractions finish or ctx is done.
func (c *ExtractController) Wait(ctx context.Context) {
	if c == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
}
