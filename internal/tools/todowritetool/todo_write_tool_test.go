package todowritetool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func item(c, st, af string) TodoItem {
	return TodoItem{Content: c, Status: st, ActiveForm: af}
}

func TestTodoWrite_validation(t *testing.T) {
	_, err := New().Run(context.Background(), []byte(`{"todos":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = New().Run(context.Background(), []byte(`{"todos":[{"content":"x","status":"pending","activeForm":"X"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = New().Run(context.Background(), []byte(`{"todos":[{"content":"","status":"pending","activeForm":"X"}]}`))
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	_, err = New().Run(context.Background(), []byte(`{"todos":[{"content":"x","status":"nope","activeForm":"X"}]}`))
	if err == nil {
		t.Fatal("expected error for bad status")
	}
}

func TestTodoWrite_storeAndClear(t *testing.T) {
	store := NewStore()
	ctx := WithRunContext(context.Background(), &RunContext{
		SessionID: "s1",
		Store:     store,
	})
	tool := New()
	in1, _ := json.Marshal(map[string]any{
		"todos": []TodoItem{item("a", "in_progress", "Doing a")},
	})
	out1, err := tool.Run(ctx, in1)
	if err != nil {
		t.Fatal(err)
	}
	var r1 map[string]any
	if err := json.Unmarshal(out1, &r1); err != nil {
		t.Fatal(err)
	}
	if len(store.Get("s1")) != 1 {
		t.Fatalf("store %v", store.Get("s1"))
	}

	done := []TodoItem{
		item("a", "completed", "Did a"),
		item("b", "completed", "Did b"),
		item("c", "completed", "Did c"),
	}
	in2, _ := json.Marshal(map[string]any{"todos": done})
	out2, err := tool.Run(ctx, in2)
	if err != nil {
		t.Fatal(err)
	}
	var r2 struct {
		NewTodos []TodoItem `json:"newTodos"`
	}
	if err := json.Unmarshal(out2, &r2); err != nil {
		t.Fatal(err)
	}
	if len(r2.NewTodos) != 3 {
		t.Fatalf("newTodos should echo input: %d", len(r2.NewTodos))
	}
	if len(store.Get("s1")) != 0 {
		t.Fatalf("all completed clears store, got %v", store.Get("s1"))
	}
}

func TestTodoWrite_agentKey(t *testing.T) {
	store := NewStore()
	ctx := WithRunContext(context.Background(), &RunContext{
		SessionID: "s1",
		AgentID:   "sub-1",
		Store:     store,
	})
	tool := New()
	in1, _ := json.Marshal(map[string]any{
		"todos": []TodoItem{item("x", "pending", "Xing")},
	})
	if _, err := tool.Run(ctx, in1); err != nil {
		t.Fatal(err)
	}
	if len(store.Get("sub-1")) != 1 || len(store.Get("s1")) != 0 {
		t.Fatalf("expected agent key sub-1")
	}
}

func TestTodoWrite_disabledNonInteractive(t *testing.T) {
	store := NewStore()
	ctx := WithRunContext(context.Background(), &RunContext{
		SessionID:      "s",
		NonInteractive: true,
		Store:          store,
	})
	_, err := New().Run(ctx, []byte(`{"todos":[{"content":"x","status":"pending","activeForm":"X"}]}`))
	if err == nil {
		t.Fatal("expected disabled")
	}
}

func TestTodoWrite_disabledEnableTasksEnv(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENABLE_TASKS", "1")
	_, err := New().Run(context.Background(), []byte(`{"todos":[{"content":"x","status":"pending","activeForm":"X"}]}`))
	if err == nil {
		t.Fatal("expected disabled")
	}
}

func TestTodoWrite_verificationNudgeFromRun(t *testing.T) {
	t.Setenv("RABBIT_CODE_TODO_VERIFICATION_NUDGE", "1")
	store := NewStore()
	ctx := WithRunContext(context.Background(), &RunContext{
		SessionID: "main",
		Store:     store,
	})
	tool := New()
	done := []TodoItem{
		item("one", "completed", "One"),
		item("two", "completed", "Two"),
		item("three", "completed", "Three"),
	}
	raw, _ := json.Marshal(map[string]any{"todos": done})
	out, err := tool.Run(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["verificationNudgeNeeded"] != true {
		t.Fatalf("expected nudge: %v", m)
	}
}

func TestMapTodoWriteToolResultForMessagesAPI(t *testing.T) {
	s := MapTodoWriteToolResultForMessagesAPI([]byte(`{"oldTodos":[],"newTodos":[]}`))
	if s == "" || !strings.Contains(s, "Todos have been modified") {
		t.Fatalf("%q", s)
	}
	raw, _ := json.Marshal(map[string]any{
		"oldTodos":                []TodoItem{},
		"newTodos":                []TodoItem{item("a", "completed", "A"), item("b", "completed", "B"), item("c", "completed", "C")},
		"verificationNudgeNeeded": true,
	})
	s2 := MapTodoWriteToolResultForMessagesAPI(raw)
	if !strings.Contains(s2, "verification agent") {
		t.Fatalf("%q", s2)
	}
}
