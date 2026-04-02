package query

import (
	"context"
	"testing"

	"github.com/2456868764/rabbit-code/internal/querydeps"
)

func TestLoopDriver_RunAssistantChain_sequence(t *testing.T) {
	seq := &querydeps.SequenceAssistant{Replies: []string{"first", "second"}}
	d := LoopDriver{
		Deps:      querydeps.Deps{Assistant: seq},
		Model:     "m",
		MaxTokens: 16,
	}
	final, texts, err := d.RunAssistantChain(context.Background(), "hi", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(texts) != 2 || texts[0] != "first" || texts[1] != "second" {
		t.Fatalf("%#v", texts)
	}
	if final == nil {
		t.Fatal("nil final")
	}
}

func TestLoopDriver_RunToolStep_state(t *testing.T) {
	var tools mockToolRunner
	d := LoopDriver{Deps: querydeps.Deps{Tools: tools}}
	st := LoopState{}
	out, err := d.RunToolStep(context.Background(), &st, "bash", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"ok":true}` {
		t.Fatalf("%s", out)
	}
	if st.PendingTools != 0 || st.TurnCount != 0 {
		t.Fatalf("%+v", st)
	}
}

type mockToolRunner struct{}

func (mockToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	return []byte(`{"ok":true}`), nil
}
