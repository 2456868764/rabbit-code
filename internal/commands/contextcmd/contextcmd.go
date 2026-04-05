package contextcmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/2456868764/rabbit-code/internal/commands/breakcache"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

const usageText = `usage: rabbit-code context <subcommand>

Subcommands:
  break-cache   print one JSON line (break_cache_command / headless parity, P5.F.6)
  report        read Messages API transcript JSON from stdin; print HeadlessContextReport JSON
  report-md     same stdin + flags as report; print Markdown (context-noninteractive.ts subset)
  budget        read resolved submit body from stdin; print submit_token_budget_snapshot JSON (H5.3 / T3 meter feed)
  help          show this text

report / report-md flags:
  -model string                  model id for window/threshold heuristics (default: generic claude)
  -max-output-tokens int         reserved output budget (default 8192)
  -context-window-tokens int     0 = from RABBIT_CODE_CONTEXT_WINDOW_TOKENS / model default
  -query-source string           optional fork source for proactive autocompact preflight

report-md only:
  -microcompact                  run microCompact.ts analogue (MicrocompactMessagesAPIJSON) before analysis;
                                 defaults query-source to repl_main_thread when empty

budget flags:
  -mode string                   bytes4 | structured | api; empty = RABBIT_CODE_TOKEN_SUBMIT_ESTIMATE_MODE
  -inject-raw-bytes int          memdir raw inject bytes to add (ceil(bytes/4) tokens), default 0
`

// Run executes rabbit-code context <subcommand>. stdin is used for report; stdout/stderr for output/errors.
// Returns exit code 0 (ok), 1 (runtime error), 2 (usage / unknown subcommand).
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	case "break-cache":
		if err := breakcache.WriteBreakCacheCommandJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "context break-cache: %v\n", err)
			return 1
		}
		return 0
	case "report":
		return runReport(args[1:], stdin, stdout, stderr)
	case "report-md":
		return runReportMarkdown(args[1:], stdin, stdout, stderr)
	case "budget":
		return runBudget(args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "rabbit-code: unknown context subcommand %q\n\n%s", args[0], usageText)
		return 2
	}
}

func runReport(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("context report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	model := fs.String("model", "", "model id for context window / thresholds")
	maxOut := fs.Int("max-output-tokens", 8192, "max output tokens reserved from context window")
	cwTok := fs.Int("context-window-tokens", 0, "context window tokens (0 = env/model default)")
	qSrc := fs.String("query-source", "", "optional query source for autocompact preflight")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", fs.Args())
		return 2
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "context report: read stdin: %v\n", err)
		return 1
	}
	m := *model
	if m == "" {
		m = "claude-sonnet-4-20250514"
	}
	r := query.BuildHeadlessContextReport(body, m, *maxOut, *cwTok, 0, *qSrc)
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		fmt.Fprintf(stderr, "context report: encode: %v\n", err)
		return 1
	}
	return 0
}

func runReportMarkdown(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("context report-md", flag.ContinueOnError)
	fs.SetOutput(stderr)
	model := fs.String("model", "", "model id for context window / thresholds")
	maxOut := fs.Int("max-output-tokens", 8192, "max output tokens reserved from context window")
	cwTok := fs.Int("context-window-tokens", 0, "context window tokens (0 = env/model default)")
	qSrc := fs.String("query-source", "", "optional query source for autocompact preflight")
	microcompact := fs.Bool("microcompact", false, "run MicrocompactMessagesAPIJSON on stdin before report")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", fs.Args())
		return 2
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "context report-md: read stdin: %v\n", err)
		return 1
	}
	m := *model
	if m == "" {
		m = "claude-sonnet-4-20250514"
	}
	qs := *qSrc
	if *microcompact {
		if qs == "" {
			qs = "repl_main_thread"
		}
		var buf compact.MicrocompactEditBuffer
		out, _, _, _, err := compact.MicrocompactMessagesAPIJSON(body, qs, time.Now(), time.Time{}, m, &buf)
		if err != nil {
			fmt.Fprintf(stderr, "context report-md: microcompact: %v\n", err)
			return 1
		}
		body = out
	}
	r := query.BuildHeadlessContextReport(body, m, *maxOut, *cwTok, 0, *qSrc)
	if _, err := fmt.Fprint(stdout, query.FormatHeadlessContextReportMarkdown(m, r)); err != nil {
		fmt.Fprintf(stderr, "context report-md: write: %v\n", err)
		return 1
	}
	return 0
}

func runBudget(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("context budget", flag.ContinueOnError)
	fs.SetOutput(stderr)
	mode := fs.String("mode", "", "bytes4 | structured | api (empty: env default)")
	inject := fs.Int("inject-raw-bytes", 0, "raw inject bytes (memdir analogue)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", fs.Args())
		return 2
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "context budget: read stdin: %v\n", err)
		return 1
	}
	p := query.BuildSubmitTokenBudgetSnapshotPayload(string(body), *inject, *mode)
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		fmt.Fprintf(stderr, "context budget: encode: %v\n", err)
		return 1
	}
	return 0
}
