package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAPIKey_envPrecedence(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, apiKeyFileName), []byte("from-file\n"), 0o600)

	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("RABBIT_CODE_API_KEY", "from-env")
	if got := ReadAPIKey(dir); got != "from-env" {
		t.Fatalf("env: got %q", got)
	}

	t.Setenv("ANTHROPIC_API_KEY", "anthropic-wins")
	t.Setenv("RABBIT_CODE_API_KEY", "from-env")
	if got := ReadAPIKey(dir); got != "anthropic-wins" {
		t.Fatalf("ANTHROPIC_API_KEY first: got %q", got)
	}
}

func TestReadAPIKey_fileFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("RABBIT_CODE_API_KEY", "")
	if err := os.WriteFile(filepath.Join(dir, apiKeyFileName), []byte("  kfile  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ReadAPIKey(dir); got != "kfile" {
		t.Fatalf("file: got %q", got)
	}
}
