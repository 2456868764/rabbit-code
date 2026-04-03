package memdir

import (
	"reflect"
	"testing"
)

func TestParseSelectedMemoriesJSON_plain(t *testing.T) {
	got, err := ParseSelectedMemoriesJSON(`{"selected_memories":["a.md","b.md"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a.md", "b.md"}) {
		t.Fatalf("%v", got)
	}
}

func TestParseSelectedMemoriesJSON_fenced(t *testing.T) {
	raw := "```json\n{\"selected_memories\":[\"x.md\"]}\n```"
	got, err := ParseSelectedMemoriesJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"x.md"}) {
		t.Fatalf("%v", got)
	}
}

func TestParseSelectedMemoriesJSON_emptyArray(t *testing.T) {
	got, err := ParseSelectedMemoriesJSON(`{"selected_memories":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("%v", got)
	}
}
