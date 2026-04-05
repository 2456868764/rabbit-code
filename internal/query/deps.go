package query

import (
	"context"
	"errors"
)

// Deps and interfaces below extend the narrow src/query/deps.ts QueryDeps pattern for headless I/O injection.

// ToolRunner runs one tool invocation (Phase 6 wires real tools).
// Use *registry.Registry from internal/tools/registry when dispatching named tools + dynamic MCP registration.
// NewDefaultToolRunner provides Read/Write/Edit/Glob/Grep/NotebookEdit/TodoWrite plus bash fallback; engine.New sets it when Tools is nil and Turn/Assistant is set.
type ToolRunner interface {
	RunTool(ctx context.Context, name string, inputJSON []byte) (resultJSON []byte, err error)
}

// StreamAssistant performs one assistant turn from serialized messages JSON (wraps Messages API + stream consumption).
type StreamAssistant interface {
	StreamAssistant(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (assistantText string, err error)
}

// Deps bundles dependencies for query.Loop / engine (all injectable for tests).
type Deps struct {
	Tools     ToolRunner
	Assistant StreamAssistant
	// Turn, if set, drives tool rounds in query.LoopDriver.RunTurnLoop; otherwise Assistant is wrapped as text-only.
	Turn TurnAssistant
}

// NoopToolRunner returns ErrNoToolRunner from RunTool.
type NoopToolRunner struct{}

func (NoopToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	return nil, ErrNoToolRunner
}

// ErrNoToolRunner means no tool layer is wired yet.
var ErrNoToolRunner = errors.New("query: no tool runner configured")

// NoopStreamAssistant returns empty assistant text (tests / bootstrap).
type NoopStreamAssistant struct{}

func (NoopStreamAssistant) StreamAssistant(context.Context, string, int, []byte) (string, error) {
	return "", nil
}
