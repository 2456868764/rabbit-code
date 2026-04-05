package toolsearchtool

import (
	"github.com/2456868764/rabbit-code/internal/tools/bashtool"
	"github.com/2456868764/rabbit-code/internal/tools/fileedittool"
	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
	"github.com/2456868764/rabbit-code/internal/tools/filewritetool"
	"github.com/2456868764/rabbit-code/internal/tools/globtool"
	"github.com/2456868764/rabbit-code/internal/tools/greptool"
	"github.com/2456868764/rabbit-code/internal/tools/notebookedittool"
	"github.com/2456868764/rabbit-code/internal/tools/powershelltool"
	"github.com/2456868764/rabbit-code/internal/tools/todowritetool"
	"github.com/2456868764/rabbit-code/internal/tools/webfetchtool"
	"github.com/2456868764/rabbit-code/internal/tools/websearchtool"
)

// ToolEntry is one tool for keyword scoring (name + searchHint + prompt/description body).
// Aliases optional; select: resolution matches toolMatchesName upstream (name or alias, exact case).
type ToolEntry struct {
	Name        string
	Aliases     []string
	SearchHint  string
	Description string
}

// DefaultCatalog lists Phase-6 builtins + deferred tools (prompt bodies for scoring).
func DefaultCatalog() []ToolEntry {
	return []ToolEntry{
		{Name: filereadtool.FileReadToolName, SearchHint: "read files, images, PDFs, notebooks", Description: filereadtool.Description},
		{Name: filewritetool.FileWriteToolName, SearchHint: "create or overwrite files", Description: filewritetool.GetWriteToolDescription()},
		{Name: fileedittool.FileEditToolName, SearchHint: "modify file contents in place", Description: fileedittool.GetEditToolPrompt()},
		{Name: globtool.GlobToolName, SearchHint: "find files by name pattern or wildcard", Description: globtool.Description},
		{Name: greptool.GrepToolName, SearchHint: "search file contents with regex (ripgrep)", Description: greptool.GetDescription()},
		{Name: notebookedittool.NotebookEditToolName, SearchHint: "edit Jupyter notebook cells (.ipynb)", Description: notebookedittool.ToolDescription},
		{Name: todowritetool.TodoWriteToolName, SearchHint: "manage the session task checklist", Description: "TodoWrite: create and update structured todo items for the session."},
		{Name: webfetchtool.WebFetchToolName, SearchHint: "fetch and extract content from a URL", Description: webfetchtool.Description},
		{Name: websearchtool.WebSearchToolName, SearchHint: websearchtool.SearchHint, Description: websearchtool.Description},
		{Name: bashtool.BashToolName, SearchHint: "run shell commands", Description: "Bash: execute shell commands in the project environment."},
		{Name: powershelltool.PowerShellToolName, SearchHint: "run PowerShell commands", Description: "PowerShell: execute PowerShell commands on Windows."},
	}
}

// DefaultDeferredToolNames mirrors shouldDefer:true builtins in default preset (NotebookEdit, TodoWrite, Web*, ToolSearch excluded at search time).
func DefaultDeferredToolNames() []string {
	return []string{
		notebookedittool.NotebookEditToolName,
		todowritetool.TodoWriteToolName,
		webfetchtool.WebFetchToolName,
		websearchtool.WebSearchToolName,
	}
}

func catalogByName(entries []ToolEntry) map[string]ToolEntry {
	m := make(map[string]ToolEntry, len(entries))
	for _, e := range entries {
		m[e.Name] = e
	}
	return m
}

func filterDeferred(all []ToolEntry, deferredNames []string) []ToolEntry {
	want := make(map[string]struct{}, len(deferredNames))
	for _, n := range deferredNames {
		want[n] = struct{}{}
	}
	var out []ToolEntry
	for _, e := range all {
		if _, ok := want[e.Name]; ok {
			out = append(out, e)
		}
	}
	return out
}
