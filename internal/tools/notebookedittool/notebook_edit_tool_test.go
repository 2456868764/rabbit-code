package notebookedittool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/tools/filereadtool"
)

func intPtr(n int) *int { return &n }

func minimalIPYNB() string {
	return `{
  "nbformat": 4,
  "nbformat_minor": 5,
  "metadata": {"language_info": {"name": "python"}},
  "cells": [
    {
      "cell_type": "code",
      "id": "c1",
      "metadata": {},
      "source": "print(1)",
      "outputs": [],
      "execution_count": null
    }
  ]
}`
}

func ctxWithNotebookRead(t *testing.T, abs string) context.Context {
	t.Helper()
	fi, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	st := filereadtool.NewReadFileStateMap()
	st.Set(abs, filereadtool.ReadFileStateEntry{
		Content:       "[]",
		Timestamp:     fi.ModTime().UnixMilli(),
		Offset:        intPtr(1),
		Limit:         nil,
		IsPartialView: false,
	})
	return filereadtool.WithRunContext(context.Background(), &filereadtool.RunContext{ReadFileState: st})
}

func TestNotebookEditReplaceByID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxWithNotebookRead(t, abs)

	in := map[string]any{
		"notebook_path": abs,
		"cell_id":       "c1",
		"new_source":    "print(2)",
		"edit_mode":     "replace",
	}
	body, _ := json.Marshal(in)

	out, err := New().Run(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "" && resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	var nb map[string]any
	if err := json.Unmarshal(data, &nb); err != nil {
		t.Fatal(err)
	}
	cells := nb["cells"].([]any)
	cell0 := cells[0].(map[string]any)
	if cell0["source"] != "print(2)" {
		t.Fatalf("source = %#v", cell0["source"])
	}
}

func TestNotebookEditReplaceByCellIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxWithNotebookRead(t, abs)

	in := map[string]any{
		"notebook_path": abs,
		"cell_id":       "cell-0",
		"new_source":    "print(99)",
	}
	body, _ := json.Marshal(in)

	out, err := New().Run(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if e := resp["error"]; e != nil && e != "" {
		t.Fatalf("error: %v", e)
	}
}

func TestNotebookEditRequiresReadState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	in := map[string]any{
		"notebook_path": abs,
		"cell_id":       "c1",
		"new_source":    "x",
	}
	body, _ := json.Marshal(in)

	_, err = New().Run(ctx, body)
	if err == nil {
		t.Fatal("expected error without read state")
	}
}

func TestMapNotebookEditToolResultForMessagesAPI(t *testing.T) {
	s := MapNotebookEditToolResultForMessagesAPI([]byte(`{"cell_id":"c1","edit_mode":"replace","new_source":"hi","error":""}`))
	if s == "" || s != `Updated cell c1 with hi` {
		t.Fatalf("got %q", s)
	}
	s2 := MapNotebookEditToolResultForMessagesAPI([]byte(`{"cell_id":"c1","edit_mode":"delete","new_source":"","error":""}`))
	if s2 != `Deleted cell c1` {
		t.Fatalf("delete: %q", s2)
	}
	s3 := MapNotebookEditToolResultForMessagesAPI([]byte(`{"error":"bad"}`))
	if s3 != "bad" {
		t.Fatalf("err: %q", s3)
	}
	s4 := MapNotebookEditToolResultForMessagesAPI([]byte(`{"cell_id":"nid","edit_mode":"insert","new_source":"x","error":""}`))
	if s4 != `Inserted cell nid with x` {
		t.Fatalf("insert: %q", s4)
	}
	if MapNotebookEditToolResultForMessagesAPI([]byte(`{"edit_mode":"replace","new_source":"a","error":""}`)) != `Updated cell  with a` {
		t.Fatal("empty cell_id in map")
	}
}

func minimalIPYNB_v44() string {
	return `{
  "nbformat": 4,
  "nbformat_minor": 4,
  "metadata": {"language_info": {"name": "python"}},
  "cells": [
    {
      "cell_type": "code",
      "metadata": {},
      "source": "print(1)",
      "outputs": [],
      "execution_count": null
    }
  ]
}`
}

func TestNotebookEditSuccessOmitsCellIDWhenOldNbformat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB_v44()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxWithNotebookRead(t, abs)

	in := map[string]any{
		"notebook_path": abs,
		"cell_id":       "cell-0",
		"new_source":    "print(2)",
		"edit_mode":     "replace",
	}
	body, _ := json.Marshal(in)

	out, err := New().Run(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["cell_id"]; ok {
		t.Fatalf("nbformat<4.5 success should omit cell_id, got %#v", resp["cell_id"])
	}
}

func TestNotebookEditStrictJSONUnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxWithNotebookRead(t, abs)
	body, _ := json.Marshal(map[string]any{
		"notebook_path": abs,
		"cell_id":       "c1",
		"new_source":    "x",
		"edit_mode":     "replace",
		"extra":         1,
	})
	_, err = New().Run(ctx, body)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("got %v", err)
	}
}

func TestNotebookEditInsertAtBeginning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxWithNotebookRead(t, abs)

	body, _ := json.Marshal(map[string]any{
		"notebook_path": abs,
		"new_source":    "# title",
		"edit_mode":     "insert",
		"cell_type":     "markdown",
	})
	out, err := New().Run(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != nil && resp["error"] != "" {
		t.Fatalf("error: %v", resp["error"])
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	var nb map[string]any
	if err := json.Unmarshal(data, &nb); err != nil {
		t.Fatal(err)
	}
	cells := nb["cells"].([]any)
	if len(cells) != 2 {
		t.Fatalf("want 2 cells, got %d", len(cells))
	}
	first := cells[0].(map[string]any)
	if first["cell_type"] != "markdown" || first["source"] != "# title" {
		t.Fatalf("first cell: %#v", first)
	}
	_ = out
}

func TestNotebookEditInsertInvalidCellType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.ipynb")
	if err := os.WriteFile(path, []byte(minimalIPYNB()), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ctxWithNotebookRead(t, abs)
	body, _ := json.Marshal(map[string]any{
		"notebook_path": abs,
		"new_source":    "x",
		"edit_mode":     "insert",
		"cell_type":     "raw",
	})
	_, err = New().Run(ctx, body)
	if err == nil || !strings.Contains(err.Error(), "cell_type must be code or markdown") {
		t.Fatalf("got %v", err)
	}
}
