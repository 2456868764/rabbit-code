package app

import (
	"os"
	"strconv"
	"strings"
)

const (
	envNonInteractive = "RABBIT_CODE_NONINTERACTIVE"
	envCI             = "CI"
)

// IsNonInteractive reports whether the session should avoid blocking on TUI prompts.
// Aligns with getIsNonInteractiveSession: explicit env and CI. TTY detection may be added later (e.g. x/term) without changing env semantics.
func IsNonInteractive() bool {
	if truthy(os.Getenv(envNonInteractive)) {
		return true
	}
	if truthy(os.Getenv(envCI)) {
		return true
	}
	return false
}

func truthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "1", "true", "yes", "on":
		return true
	}
	if v, err := strconv.ParseBool(s); err == nil {
		return v
	}
	return false
}

// IsMusl reports whether DetectLibc() found musl (Linux only; false elsewhere).
func IsMusl() bool {
	_, m := DetectLibc()
	return m
}

// IsGlibc reports whether DetectLibc() found glibc (Linux only; false elsewhere).
func IsGlibc() bool {
	g, _ := DetectLibc()
	return g
}
