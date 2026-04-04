package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

func TestEngine_PostCompactAttachmentsForNextTranscript(t *testing.T) {
	e := NewEngine(context.Background())
	e.SetPostCompactWorkspaceDir("/proj")
	// b.txt is not in preserved Read paths from transcript (only a.txt is).
	e.RecordPostCompactFileRead("/proj/src/b.txt", "beta")
	e.SetPostCompactPlan("/proj/plan.md", "# plan")
	e.SetPostCompactPlanMode(true)
	e.AddPostCompactInvokedSkill(compact.PostCompactSkillEntry{Name: "s", Path: "/p", Content: "body"})

	delta, err := compact.CreateAttachmentMessageJSON(map[string]interface{}{"type": "custom_delta", "k": 1})
	if err != nil {
		t.Fatal(err)
	}
	e.AppendPostCompactDeltaAttachment(delta)

	tr := []byte(`[{"role":"assistant","content":[{"type":"tool_use","id":"x","name":"Read","input":{"file_path":"/proj/src/a.txt"}}]}]`)

	atts, err := e.PostCompactAttachmentsForNextTranscript(context.Background(), tr, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) < 3 {
		t.Fatalf("want several attachments, got %d", len(atts))
	}
	s := string(atts[0])
	if !strings.Contains(s, "beta") || !strings.Contains(s, "file") {
		t.Fatalf("file restore: %s", s)
	}
	var joined strings.Builder
	for _, a := range atts {
		joined.WriteString(string(a))
	}
	j := joined.String()
	if !strings.Contains(j, "plan_file_reference") || !strings.Contains(j, "plan_mode") || !strings.Contains(j, "invoked_skills") || !strings.Contains(j, "custom_delta") {
		t.Fatalf("missing types: %s", j)
	}

	// read state cleared
	e.postCompactMu.Lock()
	n := len(e.postCompactReads)
	e.postCompactMu.Unlock()
	if n != 0 {
		t.Fatalf("reads not cleared: %d", n)
	}
}

func TestEngine_PostCompact_skillsPersistAcrossCompact(t *testing.T) {
	e := NewEngine(context.Background())
	e.AddPostCompactInvokedSkill(compact.PostCompactSkillEntry{Name: "a", Path: "/a", Content: "x"})
	_, _ = e.PostCompactAttachmentsForNextTranscript(context.Background(), []byte(`[]`), "")
	e.postCompactMu.Lock()
	n := len(e.postCompactSkills)
	e.postCompactMu.Unlock()
	if n != 1 {
		t.Fatalf("skills cleared unexpectedly: %d", n)
	}
}

func TestEngine_AttachPostCompactToStreamingConfig(t *testing.T) {
	e := NewEngine(context.Background())
	var cfg query.StreamingCompactExecutorConfig
	e.AttachPostCompactToStreamingConfig(&cfg)
	if cfg.PostCompactAttachmentsJSON == nil {
		t.Fatal()
	}
	prev := func(context.Context, []byte, string) ([]json.RawMessage, error) {
		return []json.RawMessage{json.RawMessage(`{"x":1}`)}, nil
	}
	cfg.PostCompactAttachmentsJSON = prev
	e.AttachPostCompactToStreamingConfig(&cfg)
	out, err := cfg.PostCompactAttachmentsJSON(context.Background(), []byte(`[]`), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 1 {
		t.Fatal()
	}
}

func TestPostCompactDisplayPath(t *testing.T) {
	if postCompactDisplayPath("", "/x") != "/x" {
		t.Fatal()
	}
	base := t.TempDir()
	sub := filepath.Join(base, "nest", "f.go")
	if err := os.MkdirAll(filepath.Dir(sub), 0o755); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Rel(base, sub)
	if err != nil {
		t.Fatal(err)
	}
	if g := postCompactDisplayPath(base, sub); g != want {
		t.Fatalf("got %q want %q", g, want)
	}
}
