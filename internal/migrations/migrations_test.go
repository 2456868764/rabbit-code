package migrations

import "testing"

func readVersion(m map[string]interface{}) int {
	raw := m["schema_version"]
	switch x := raw.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}

func TestApply_noSchemaNoOp(t *testing.T) {
	m := map[string]interface{}{"k": "v"}
	if err := Apply(m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["schema_version"]; ok {
		t.Fatal("schema_version should not be set when absent before Apply")
	}
}

func TestApply_idempotent(t *testing.T) {
	m := map[string]interface{}{
		"schema_version": float64(CurrentSchemaVersion),
		"k":              "v",
	}
	if err := Apply(m); err != nil {
		t.Fatal(err)
	}
	if readVersion(m) != CurrentSchemaVersion {
		t.Fatalf("version %v", m["schema_version"])
	}
}

func TestApply_from1(t *testing.T) {
	m := map[string]interface{}{
		"schema_version": float64(1),
	}
	if err := Apply(m); err != nil {
		t.Fatal(err)
	}
	if readVersion(m) != CurrentSchemaVersion {
		t.Fatalf("want %d got %v", CurrentSchemaVersion, m["schema_version"])
	}
	if m["migrated_from_v1"] != true {
		t.Fatal("expected migration side effect")
	}
}
