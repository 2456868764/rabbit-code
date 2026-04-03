package breakcache

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteBreakCacheCommandJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteBreakCacheCommandJSON(&buf); err != nil {
		t.Fatal(err)
	}
	s := strings.TrimSpace(buf.String())
	var p BreakCacheCommandPayload
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		t.Fatal(err)
	}
	if p.Kind != "break_cache_command" || p.Phase != "submit" {
		t.Fatalf("%+v", p)
	}
}
