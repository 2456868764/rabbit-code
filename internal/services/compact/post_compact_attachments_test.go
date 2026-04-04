package compact

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTruncateSkillContentForPostCompact(t *testing.T) {
	short := "hello"
	if TruncateSkillContentForPostCompact(short, 100) != short {
		t.Fatal()
	}
	long := strings.Repeat("a", PostCompactMaxTokensPerSkill*4+200)
	out := TruncateSkillContentForPostCompact(long, PostCompactMaxTokensPerSkill)
	if !strings.HasSuffix(out, skillTruncationMarkerPostCompact) {
		t.Fatalf("expected marker suffix, got len %d", len(out))
	}
}

func TestCreateAttachmentMessageJSON_shape(t *testing.T) {
	raw, err := CreateAttachmentMessageJSON(map[string]interface{}{
		"type": "plan_file_reference",
		"path": "/p",
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "attachment" {
		t.Fatalf("type=%v", m["type"])
	}
	if _, ok := m["uuid"].(string); !ok {
		t.Fatal("uuid")
	}
	am, _ := m["attachment"].(map[string]interface{})
	if am["type"] != "plan_file_reference" {
		t.Fatal()
	}
}

func TestBuildPlanFileReferenceAttachmentMessageJSON_empty(t *testing.T) {
	raw, err := BuildPlanFileReferenceAttachmentMessageJSON("/x", "  ")
	if err != nil || len(raw) != 0 {
		t.Fatalf("err=%v len=%d", err, len(raw))
	}
}

func TestBuildInvokedSkillsAttachmentMessageJSON_truncates(t *testing.T) {
	huge := strings.Repeat("b", PostCompactMaxTokensPerSkill*4+5000)
	raw, err := BuildInvokedSkillsAttachmentMessageJSON([]PostCompactSkillEntry{
		{Name: "a", Path: "/a", Content: huge},
	})
	if err != nil || len(raw) == 0 {
		t.Fatal(err)
	}
	var top map[string]interface{}
	_ = json.Unmarshal(raw, &top)
	att, _ := top["attachment"].(map[string]interface{})
	sk, _ := att["skills"].([]interface{})
	if len(sk) != 1 {
		t.Fatalf("skills len=%d", len(sk))
	}
	row := sk[0].(map[string]interface{})
	c, _ := row["content"].(string)
	if !strings.HasSuffix(c, skillTruncationMarkerPostCompact) {
		t.Fatal("expected per-skill truncation marker")
	}
}

func TestFilterAttachmentMessagesByRoughTokenBudget(t *testing.T) {
	a, errA := CreateAttachmentMessageJSON(map[string]interface{}{"type": "x", "k": strings.Repeat("z", 400)})
	b, errB := CreateAttachmentMessageJSON(map[string]interface{}{"type": "y", "k": strings.Repeat("z", 400)})
	if errA != nil || errB != nil {
		t.Fatal(errA, errB)
	}
	out := FilterAttachmentMessagesByRoughTokenBudget([]json.RawMessage{a, b}, 150)
	if len(out) != 1 {
		t.Fatalf("got %d", len(out))
	}
}
