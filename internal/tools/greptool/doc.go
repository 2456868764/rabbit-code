// Package greptool implements the Grep tool (claude-code-sourcemap/restored-src/src/tools/GrepTool/GrepTool.ts).
//
// Parity notes: NODE_ENV=test → files_with_matches sorted by path (GrepTool.ts test branch); negative head_limit → DEFAULT_HEAD_LIMIT like semanticNumber discard; Windows-safe path:line split in content mode (extension over TS indexOf).
package greptool
