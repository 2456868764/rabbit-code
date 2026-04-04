# messages.ts parity checklist

Reference snapshot: `claude-code-sourcemap` @ **a8a678c** (`restored-src/src/utils/messages.ts` and linked modules). Re-diff when updating TS.

## normalizeAttachmentForAPI

| Item | Status |
|------|--------|
| Main `switch (attachment.type)` | Ported |
| `directory` / `ls` | `bashQuoteShellArg` — POSIX single-quote style like TS `quote([path])` (npm shell-quote) |
| `selected_lines_in_ide` | 2000 cap = UTF-16 code units (JS `String.length`) |
| Bash persisted stdout preview | `bashGeneratePreview` uses UTF-16 limit + newline bias like `toolResultStorage.generatePreview` (TS compares `content.length`, not byte length) |
| `output_style` display | Go **superset** (env, scan dirs, plugins JSON, settings); TS `messages.ts` uses static `OUTPUT_STYLE_CONFIG` only |
| Plugin output-styles | Go: `RABBIT_OUTPUT_STYLE_PLUGINS_*` manual dirs; TS: `loadPluginOutputStyles` + enabled plugins |
| `teammate_mailbox` | One meta user message; malformed / all-skipped → empty content like TS `formatTeammateMessages` (no debug JSON dump) |
| `file` content `parts` / `file_unchanged` | In TS `FileReadTool.ts`; Go implements same branches (`messages.ts` inner switch may omit them but tool does) |

## Other symbols

| Item | Status |
|------|--------|
| `StripToolResultSignature` (`json.RawMessage`) | No TS counterpart in restored tree; identity function. Assistant path: `StripSignatureBlocks` |
| `normalizeMessagesForAPI` pipeline | Ported; Statsig / `feature()` → `RABBIT_*` (see `messages.go` package doc) |
| `HandleMessageFromStream` | Ported; new stream `content_block_*` / delta types → manual diff when TS changes |
| Analytics (`logEvent`, etc.) | **Not ported** (deliberate) |
| API error strip copy (“double press esc” vs non-interactive) | `RABBIT_NON_INTERACTIVE` |

## Intentionally not matched

- **Analytics** and live **GrowthBook/Statsig** values (use env toggles instead).
- **NODE_ENV=test** branches in TS (e.g. snip `[id:]` injection) — covered by `RABBIT_*` / test envs where ported.
