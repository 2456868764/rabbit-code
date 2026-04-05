package memdir

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
)

func TestCountModelVisibleMessagesSince_and_HasMemoryWritesSince(t *testing.T) {
	u := query.RabbitMessageUUIDKey
	raw := []byte(`[
	  {"role":"user","content":[{"type":"text","text":"a"}],"` + u + `":"m0"},
	  {"role":"assistant","content":[{"type":"tool_use","id":"1","name":"Write","input":{"file_path":"/outside/x.md","content":"z"}}],"` + u + `":"m1"},
	  {"role":"user","content":[{"type":"text","text":"b"}],"` + u + `":"m2"}
	]`)
	if n := CountModelVisibleMessagesSince(raw, "m0", u); n != 2 {
		t.Fatalf("after m0 want 2 visible, got %d", n)
	}
	if !HasMemoryWritesSince(raw, "m0", "/outside", u) {
		t.Fatal("expected write detected")
	}
	memRoot := filepath.Clean("/tmp/proj/memory")
	if HasMemoryWritesSince(raw, "m0", memRoot, u) {
		t.Fatal("path outside auto mem should not count")
	}
}

func TestIsExtractReadOnlyBash(t *testing.T) {
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"ls -la"}`)) {
		t.Fatal("ls")
	}
	if IsExtractReadOnlyBash([]byte(`{"cmd":"rm -rf /"}`)) {
		t.Fatal("deny rm")
	}
	if IsExtractReadOnlyBash([]byte(`{"cmd":"ls | wc"}`)) {
		t.Fatal("deny pipe")
	}
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"git log -1 --oneline"}`)) {
		t.Fatal("allow read-only git")
	}
	if IsExtractReadOnlyBash([]byte(`{"cmd":"git push"}`)) {
		t.Fatal("deny git push")
	}
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"git blame README.md"}`)) {
		t.Fatal("allow git blame")
	}
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"git stash list"}`)) {
		t.Fatal("allow git stash list")
	}
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"git remote -v"}`)) {
		t.Fatal("allow git remote -v")
	}
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"git remote show origin"}`)) {
		t.Fatal("allow git remote show")
	}
	if IsExtractReadOnlyBash([]byte(`{"cmd":"git remote add origin u"}`)) {
		t.Fatal("deny git remote add")
	}
	if !IsExtractReadOnlyBash([]byte(`{"cmd":"git config --get core.editor"}`)) {
		t.Fatal("allow git config --get")
	}
	if IsExtractReadOnlyBash([]byte(`{"cmd":"git config --set x y"}`)) {
		t.Fatal("deny git config set-style")
	}
	in, _ := json.Marshal(map[string]string{"cmd": "\x00"})
	if IsExtractReadOnlyBash(in) {
		t.Fatal("deny null in cmd")
	}
}

func TestAutoMemToolRunner(t *testing.T) {
	inner := passthroughTools{}
	mem := filepath.Clean("/mem/root")
	w := &AutoMemToolRunner{Inner: inner, MemoryDir: mem}
	ctx := context.Background()
	_, err := w.RunTool(ctx, "Read", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.RunTool(ctx, "Write", []byte(`{"file_path":"/other/x.md"}`))
	if err == nil {
		t.Fatal("expected deny")
	}
	good := filepath.Join(mem, "a.md")
	_, err = w.RunTool(ctx, "Write", []byte(`{"file_path":"`+good+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.RunTool(ctx, "REPL", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
}

type passthroughTools struct{}

func (passthroughTools) RunTool(context.Context, string, []byte) ([]byte, error) {
	return json.Marshal(map[string]any{"ok": true})
}

func TestRunForkedExtractMemory_writesUnderMemdir(t *testing.T) {
	memDir := filepath.Join(t.TempDir(), "memory")
	if err := os.MkdirAll(memDir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(memDir, "note.md")
	inWrite, _ := json.Marshal(map[string]any{
		"file_path": target,
		"content":   "hello",
	})
	turn := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{ToolUses: []query.ToolUseCall{{ID: "w1", Name: "Write", Input: inWrite}}},
		{Text: "done"},
	}}
	parent, _ := query.InitialUserMessagesJSON("user turn")
	dep := ForkedExtractDeps{
		Tools:     passthroughTools{},
		Turn:      turn,
		Model:     "m",
		MaxTokens: 256,
	}
	res, err := RunForkedExtractMemory(context.Background(), dep, ForkedExtractParams{
		ParentMessagesJSON: parent,
		UserPrompt:         "extract prompt",
		MemoryDir:          memDir,
		MaxTurns:           5,
		QuerySource:        query.QuerySourceExtractMemories,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MemoryFilePaths) != 1 || res.MemoryFilePaths[0] != target {
		t.Fatalf("paths %+v", res.MemoryFilePaths)
	}
}

func TestBuildExtractCombinedPrompt_fallsBackWhenTeamOff(t *testing.T) {
	t.Setenv(features.EnvTeamMem, "")
	t.Setenv(features.EnvDisableAutoMemory, "")
	auto := BuildExtractAutoOnlyPrompt(8, "", false)
	combo := BuildExtractCombinedPrompt(8, "", false, nil)
	if auto != combo {
		t.Fatal("expected same prompt when team memory off")
	}
}

func TestBuildExtractCombinedPrompt_teamSections(t *testing.T) {
	t.Setenv(features.EnvDisableAutoMemory, "")
	t.Setenv(features.EnvTeamMem, "1")
	p := BuildExtractCombinedPrompt(3, "x", false, nil)
	if !strings.Contains(p, "<scope>") || !strings.Contains(p, "team memories") {
		t.Fatalf("missing combined sections: %q…", truncateTestString(p, 120))
	}
}
