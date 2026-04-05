package query

import (
	"encoding/json"
	"testing"
)

func TestExtractTodosFromTranscriptJSON_lastTodoWrite(t *testing.T) {
	raw := []byte(`[
		{"role":"user","content":"hi"},
		{"role":"assistant","content":[{"type":"text","text":"ok"}]},
		{"role":"assistant","content":[
			{"type":"tool_use","id":"x","name":"Read","input":{}},
			{"type":"tool_use","id":"y","name":"TodoWrite","input":{"todos":[
				{"content":"one","status":"pending","activeForm":"doing one"},
				{"content":"two","status":"completed","activeForm":"doing two"}
			]}}
		]}
	]`)
	got, err := ExtractTodosFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Content != "one" || got[1].Status != "completed" {
		t.Fatalf("%+v", got)
	}
}

func TestExtractTodosFromTranscriptJSON_skipsInvalidItems(t *testing.T) {
	raw := []byte(`[{"role":"assistant","content":[
		{"type":"tool_use","name":"TodoWrite","input":{"todos":[
			{"content":"","status":"pending","activeForm":"x"},
			{"content":"ok","status":"pending","activeForm":"ok"}
		]}}
	]}]`)
	got, err := ExtractTodosFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "ok" {
		t.Fatalf("%+v", got)
	}
}

func TestExtractTodosFromTranscriptJSON_none(t *testing.T) {
	got, err := ExtractTodosFromTranscriptJSON([]byte(`[]`))
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %+v", err, got)
	}
	_, err = ExtractTodosFromTranscriptJSON([]byte(`not-json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractTodosFromTranscriptJSON_jsonRoundTrip(t *testing.T) {
	raw := []byte(`[{"role":"assistant","content":[
		{"type":"tool_use","name":"TodoWrite","input":{"todos":[
			{"content":"a","status":"in_progress","activeForm":"working on a"}
		]}}
	]}]`)
	got, err := ExtractTodosFromTranscriptJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatal(string(b))
	}
}
