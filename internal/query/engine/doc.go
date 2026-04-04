// Package engine implements QueryEngine-style orchestration: user submit, assistant stream events, cancel (Phase 5).
// Import path: …/internal/query/engine (mirrors src/QueryEngine.ts next to query/).
// Does not import TUI (ARCHITECTURE_BOUNDARIES).
// Production wiring: internal/app.ApplyEngineCompactIntegration + (*Engine).InstallAnthropicStreamingCompact (see compact_install.go).
package engine
