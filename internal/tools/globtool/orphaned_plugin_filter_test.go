package globtool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPathsOverlap(t *testing.T) {
	if !pathsOverlap("/a/b", "/a") {
		t.Fatal()
	}
	if pathsOverlap("/a/b", "/c") {
		t.Fatal()
	}
}

func TestGetGlobExclusionsForPluginCache_markers(t *testing.T) {
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not in PATH")
	}
	pluginRoot := t.TempDir()
	t.Setenv("CLAUDE_CODE_PLUGIN_CACHE_DIR", pluginRoot)
	ClearPluginCacheExclusions()

	cache := filepath.Join(pluginRoot, "cache")
	ver := filepath.Join(cache, "m", "pl", "v1")
	if err := os.MkdirAll(ver, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(ver, orphanedAtFilename)
	if err := os.WriteFile(marker, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		t.Fatal(err)
	}
	excl := getGlobExclusionsForPluginCache(context.Background(), rgPath, cache)
	if len(excl) != 1 || excl[0] != "!**/m/pl/v1/**" {
		t.Fatalf("%v", excl)
	}

	ClearPluginCacheExclusions()
	if ex2 := getGlobExclusionsForPluginCache(context.Background(), rgPath, cache); len(ex2) != 1 {
		t.Fatalf("%v", ex2)
	}
}

func TestGetGlobExclusionsForPluginCache_noOverlap(t *testing.T) {
	ClearPluginCacheExclusions()
	t.Setenv("CLAUDE_CODE_PLUGIN_CACHE_DIR", t.TempDir())
	rgPath, _ := exec.LookPath("rg")
	if rgPath == "" {
		t.Skip()
	}
	if ex := getGlobExclusionsForPluginCache(context.Background(), rgPath, "/unrelated/path"); len(ex) != 0 {
		t.Fatalf("%v", ex)
	}
}
