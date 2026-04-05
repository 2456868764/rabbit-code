package app

import (
	"fmt"
	"os"

	"github.com/2456868764/rabbit-code/internal/features"
)

const (
	// ExitBootstrapFail is the default non-zero exit for bootstrap/onboarding errors.
	ExitBootstrapFail = 1
	// ExitHardFail is used when HARD_FAIL is enabled (aligns with stricter fatal-style exits).
	ExitHardFail = 2
)

// QuitRuntime runs r.Close when r != nil, then os.Exit(code).
// Use after successful Bootstrap instead of bare os.Exit so CleanupRegistry runs
// (log flush, RegisterEngineShutdown / extract drain, etc.).
func QuitRuntime(r *Runtime, code int) {
	if r != nil {
		r.Close()
	}
	os.Exit(code)
}

// FailBootstrap prints err to stderr and exits with bootstrap fail codes after closing rt when non-nil.
func FailBootstrap(rt *Runtime, err error) {
	if err == nil {
		return
	}
	if rt != nil {
		rt.Close()
	}
	if features.HardFailEnabled() {
		fmt.Fprintf(os.Stderr, "rabbit-code: FATAL bootstrap: %v\n", err)
		os.Exit(ExitHardFail)
	}
	fmt.Fprintf(os.Stderr, "rabbit-code: bootstrap: %v\n", err)
	os.Exit(ExitBootstrapFail)
}

// PrintBootstrapFailure prints err to stderr and exits. Uses exit code 2 when HARD_FAIL is enabled.
func PrintBootstrapFailure(err error) {
	FailBootstrap(nil, err)
}
