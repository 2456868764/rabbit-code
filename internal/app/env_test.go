package app

import (
	"runtime"
	"testing"
)

func TestIsNonInteractive_env(t *testing.T) {
	t.Setenv(envNonInteractive, "1")
	t.Setenv(envCI, "")
	if !IsNonInteractive() {
		t.Fatal("expected non-interactive")
	}
}

func TestIsNonInteractive_ci(t *testing.T) {
	t.Setenv(envNonInteractive, "")
	t.Setenv(envCI, "true")
	if !IsNonInteractive() {
		t.Fatal("expected non-interactive from CI")
	}
}

func TestIsNonInteractive_default(t *testing.T) {
	t.Setenv(envNonInteractive, "")
	t.Setenv(envCI, "")
	if IsNonInteractive() {
		t.Fatal("expected interactive by default when CI unset")
	}
}

func TestIsGlibcIsMusl_mutex(t *testing.T) {
	if IsGlibc() && IsMusl() {
		t.Fatal("glibc and musl should not both be true")
	}
	if runtime.GOOS != "linux" && (IsGlibc() || IsMusl()) {
		t.Fatal("non-linux should not report libc flavor")
	}
}
