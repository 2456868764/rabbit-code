package memdir

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

// ForkedExtractParams runs a bounded tool loop seeded with parent transcript + extract user prompt (runForkedAgent analogue).
type ForkedExtractParams struct {
	ParentMessagesJSON json.RawMessage
	UserPrompt         string
	MemoryDir          string
	MaxTurns           int
	QuerySource        string
	Merged             map[string]interface{}
	// NonInteractive forwarded for feature gates outside this call.
	NonInteractive bool
}

// ForkedExtractDeps supplies model and backends (same Deps as main engine).
type ForkedExtractDeps struct {
	Tools     querydeps.ToolRunner
	Turn      querydeps.TurnAssistant
	Model     string
	MaxTokens int
}

// ForkedExtractResult is the transcript after the fork and paths written under MemoryDir (topic files may include team/).
type ForkedExtractResult struct {
	MessagesJSON     json.RawMessage
	ParentMsgCount   int
	WrittenPaths     []string
	MemoryFilePaths  []string // WrittenPaths excluding MEMORY.md index only
	TeamMemoryWrites int
}

// RunForkedExtractMemory executes the extract sub-loop with auto-mem tool gating.
func RunForkedExtractMemory(ctx context.Context, dep ForkedExtractDeps, p ForkedExtractParams) (ForkedExtractResult, error) {
	var out ForkedExtractResult
	if dep.Turn == nil || dep.Tools == nil {
		return out, querydeps.ErrNoToolRunner
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
		Deps: querydeps.Deps{
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
