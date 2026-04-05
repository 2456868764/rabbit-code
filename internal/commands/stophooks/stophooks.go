package stophooks

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

const usageText = `usage: rabbit-code stop-hooks <subcommand>

Subcommands:
  list    print JSON manifest of *.md basenames in a hooks directory

list flags:
  -dir string   directory to scan (default: RABBIT_CODE_STOP_HOOKS_DIR; required if unset)
`

// ListMarkdownBasenames returns sorted basenames of regular files ending in .md (case-sensitive).
func ListMarkdownBasenames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// Run executes rabbit-code stop-hooks <subcommand>. Returns exit 0 / 1 / 2.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	case "list":
		return runList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "rabbit-code stop-hooks: unknown subcommand %q\n", args[0])
		fmt.Fprint(stderr, usageText)
		return 2
	}
}

func runList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stop-hooks list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "hooks directory (default from RABBIT_CODE_STOP_HOOKS_DIR)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	d := strings.TrimSpace(*dir)
	if d == "" {
		d = features.StopHooksDir()
	}
	if d == "" {
		fmt.Fprint(stderr, "rabbit-code stop-hooks list: set -dir or RABBIT_CODE_STOP_HOOKS_DIR\n")
		return 1
	}
	d = filepath.Clean(d)
	files, err := ListMarkdownBasenames(d)
	if err != nil {
		fmt.Fprintf(stderr, "rabbit-code stop-hooks list: %v\n", err)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string][]string{"markdown": files}); err != nil {
		fmt.Fprintf(stderr, "rabbit-code stop-hooks list: %v\n", err)
		return 1
	}
	return 0
}
