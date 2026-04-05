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
type ToolEntry struct {
	Name        string
	SearchHint  string
	Description string
}

// DefaultCatalog lists Phase-6 builtins + deferred tools (prompt bodies for scoring).
func DefaultCatalog() []ToolEntry {
	return []ToolEntry{
		{filereadtool.FileReadToolName, "read files, images, PDFs, notebooks", "Read tool: read text files, images, PDFs, and Jupyter notebooks from disk."},
		{filewritetool.FileWriteToolName, "create or overwrite files", "Write tool: create new files or overwrite existing file content."},
		{fileedittool.FileEditToolName, "modify file contents in place", fileedittool.PromptDescription},
		{globtool.GlobToolName, "find files by name pattern or wildcard", globtool.Description},
		{greptool.GrepToolName, "search file contents with regex (ripgrep)", greptool.GetDescription()},
		{notebookedittool.NotebookEditToolName, "edit Jupyter notebook cells (.ipynb)", "NotebookEdit: replace, insert, or delete cells in .ipynb notebooks."},
		{todowritetool.TodoWriteToolName, "manage the session task checklist", "TodoWrite: create and update structured todo items for the session."},
		{webfetchtool.WebFetchToolName, "fetch and extract content from a URL", "WebFetch: fetch a URL and return extracted readable content."},
		{websearchtool.WebSearchToolName, "search the web for current information", "WebSearch: run a web search and return summarized results."},
		{bashtool.BashToolName, "run shell commands", "Bash: execute shell commands in the project environment."},
		{powershelltool.PowerShellToolName, "run PowerShell commands", "PowerShell: execute PowerShell commands on Windows."},
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
