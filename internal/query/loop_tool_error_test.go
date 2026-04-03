package query

import (
	"context"
	"errors"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

type errToolRunner struct{}

func (errToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	return nil, errors.New("tool boom")
}

func TestLoopDriver_RunToolStep_toolErrorUndoesSchedule(t *testing.T) {
	d := LoopDriver{Deps: querydeps.Deps{Tools: errToolRunner{}}}
	st := LoopState{}
	_, err := d.RunToolStep(context.Background(), &st, "bash", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if st.PendingTools != 0 {
		t.Fatalf("pending=%d", st.PendingTools)
	}
}
