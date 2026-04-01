package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestTrustMarker_roundTrip(t *testing.T) {
	dir := t.TempDir()
	ok, err := TrustAccepted(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no marker")
	}
	if err := WriteTrustMarker(dir); err != nil {
		t.Fatal(err)
	}
	ok, err = TrustAccepted(dir)
	if err != nil || !ok {
		t.Fatalf("TrustAccepted = %v, %v", ok, err)
	}
}

func TestHasAPIKeyConfigured_envAndFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("RABBIT_CODE_API_KEY", "")
	if HasAPIKeyConfigured(dir) {
		t.Fatal("expected false")
	}
	t.Setenv("RABBIT_CODE_API_KEY", "sk-test")
	if !HasAPIKeyConfigured(dir) {
		t.Fatal("expected true from env")
	}
	t.Setenv("RABBIT_CODE_API_KEY", "")
	path := filepath.Join(dir, apiKeyFileName)
	if err := os.WriteFile(path, []byte("k-from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !HasAPIKeyConfigured(dir) {
		t.Fatal("expected true from file")
	}
}

func TestRunPostBootstrapOnboarding_skip(t *testing.T) {
	dir := t.TempDir()
	rt := &Runtime{
		Log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		NonInteractive:  true,
		GlobalConfigDir: dir,
	}
	t.Setenv(EnvSkipOnboarding, "1")
	defer t.Setenv(EnvSkipOnboarding, "")
	if err := RunPostBootstrapOnboarding(context.Background(), rt); err != nil {
		t.Fatal(err)
	}
	ok, _ := TrustAccepted(dir)
	if ok {
		t.Fatal("skip should not write trust marker")
	}
}

func TestRunPostBootstrapOnboarding_nonInteractive_trustAndKey(t *testing.T) {
	dir := t.TempDir()
	rt := &Runtime{
		Log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		NonInteractive:  true,
		GlobalConfigDir: dir,
	}
	t.Setenv("RABBIT_CODE_NONINTERACTIVE", "1")
	t.Setenv(EnvAcceptTrust, "1")
	t.Setenv(EnvAllowMissingAPIKey, "1")
	defer func() {
		_ = os.Unsetenv("RABBIT_CODE_NONINTERACTIVE")
		_ = os.Unsetenv(EnvAcceptTrust)
		_ = os.Unsetenv(EnvAllowMissingAPIKey)
	}()
	if err := RunPostBootstrapOnboarding(context.Background(), rt); err != nil {
		t.Fatal(err)
	}
	ok, err := TrustAccepted(dir)
	if err != nil || !ok {
		t.Fatalf("trust: %v %v", ok, err)
	}
}

func TestRunPostBootstrapOnboarding_nonInteractive_trustRequired(t *testing.T) {
	dir := t.TempDir()
	rt := &Runtime{
		Log:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		NonInteractive:  true,
		GlobalConfigDir: dir,
	}
	t.Setenv("RABBIT_CODE_NONINTERACTIVE", "1")
	defer func() { _ = os.Unsetenv("RABBIT_CODE_NONINTERACTIVE") }()
	_ = os.Unsetenv(EnvAcceptTrust)
	_ = os.Unsetenv(EnvSkipOnboarding)
	if err := RunPostBootstrapOnboarding(context.Background(), rt); err == nil {
		t.Fatal("expected error without trust")
	}
}

func TestWriteAPIKeyFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteAPIKeyFile(dir, "  secret  "); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, apiKeyFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "secret\n" {
		t.Fatalf("got %q", b)
	}
}
