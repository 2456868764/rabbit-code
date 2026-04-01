package config

import (
	"path/filepath"
	"testing"
)

func TestExtraCAPEMPaths_resolve(t *testing.T) {
	absPEM := filepath.Join(t.TempDir(), "root.pem")
	m := map[string]interface{}{
		"extra_ca_paths": []interface{}{absPEM, "rel.pem"},
	}
	proj := filepath.Join(t.TempDir(), "proj")
	out := ExtraCAPEMPaths(m, proj, "/cwd")
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
	if out[0] != absPEM {
		t.Fatalf("got %q want %q", out[0], absPEM)
	}
	wantRel := filepath.Join(proj, "rel.pem")
	if out[1] != wantRel {
		t.Fatalf("got %q want %q", out[1], wantRel)
	}
}

func TestExtraCAPEMPaths_nil(t *testing.T) {
	if ExtraCAPEMPaths(nil, "", "") != nil {
		t.Fatal()
	}
}
