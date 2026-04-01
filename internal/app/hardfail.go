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

// PrintBootstrapFailure prints err to stderr and exits. Uses exit code 2 when HARD_FAIL is enabled.
func PrintBootstrapFailure(err error) {
	if err == nil {
		return
	}
	if features.HardFailEnabled() {
		fmt.Fprintf(os.Stderr, "rabbit-code: FATAL bootstrap: %v\n", err)
		os.Exit(ExitHardFail)
	}
	fmt.Fprintf(os.Stderr, "rabbit-code: bootstrap: %v\n", err)
	os.Exit(ExitBootstrapFail)
}
