# messages.ts parity checklist

Reference: `claude-code-sourcemap/restored-src/src/utils/messages.ts` (and linked modules). This is a snapshot, not a live submodule — re-diff when updating TS.

| Item | Status |
|------|--------|
| `normalizeAttachmentForAPI` main switch | Ported; directory `ls` uses shell-quote–compatible arg quoting (`bashQuotePathForLs`). |
| `selected_lines_in_ide` truncation | UTF-16 cap 2000 (JS `String.length`). |
| Bash persisted stdout `generatePreview` | UTF-16 cap `PREVIEW_SIZE_BYTES` (2000), newline bias like TS. |
| `output_style` display | Go **superset** (env, scan dirs, plugins JSON, settings); TS `messages.ts` uses static `OUTPUT_STYLE_CONFIG` only. |
| Plugin output-styles | Go: `RABBIT_OUTPUT_STYLE_PLUGINS_*` manual dirs; TS: `loadPluginOutputStyles` + enabled plugins. |
| `teammate_mailbox` | Always one `createUserMessage` (TS); `formatTeammateMessages` may yield `""`; non-map entries skipped. |
| `file` content `parts` / `file_unchanged` | Present in TS `FileReadTool.ts`; Go implements same branches. |
| `StripToolResultSignature` (json.RawMessage) | No TS counterpart in restored tree; identity function. |
| `normalizeMessagesForAPI` pipeline | Ported; Statsig gates → `RABBIT_*` (see `messages.go` package doc). |
| `HandleMessageFromStream` | Ported; new stream `content_block_*` / delta types require manual diff when TS changes. |
| Analytics (`logEvent`, etc.) | **Not ported** (deliberate). |
| API error strip copy (“double press esc” vs non-interactive) | `RABBIT_NON_INTERACTIVE`. |
