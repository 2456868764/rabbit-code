package app

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"
)

// RunPostBootstrapOnboarding runs P1.5.1 first-run trust and API key prompts when appropriate.
// Skipped when RABBIT_CODE_EXIT_AFTER_INIT=1 (caller should exit before calling) or RABBIT_CODE_SKIP_ONBOARDING=1.
// Non-interactive sessions never block on the TUI; see EnvAcceptTrust and EnvAllowMissingAPIKey.
func RunPostBootstrapOnboarding(ctx context.Context, rt *Runtime) error {
	if rt == nil {
		return nil
	}
	if truthy(os.Getenv(EnvSkipOnboarding)) {
		rt.Log.Debug("onboarding skipped", "env", EnvSkipOnboarding)
		return nil
	}

	globalDir := rt.GlobalConfigDir

	ok, err := TrustAccepted(globalDir)
	if err != nil {
		return fmt.Errorf("trust marker: %w", err)
	}
	if !ok {
		if rt.NonInteractive {
			if truthy(os.Getenv(EnvAcceptTrust)) {
				if err := WriteTrustMarker(globalDir); err != nil {
					return fmt.Errorf("write trust marker: %w", err)
				}
				rt.Log.Debug("trust accepted via env", "env", EnvAcceptTrust)
			} else {
				return fmt.Errorf(
					"trust not accepted (non-interactive): set %s=1 to record trust, or %s=1 to skip onboarding, or run in a TTY",
					EnvAcceptTrust, EnvSkipOnboarding,
				)
			}
		} else {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf(
					"no TTY for trust prompt: set %s=1, or %s=1, or %s=1",
					EnvAcceptTrust, EnvSkipOnboarding, ExitAfterInitEnv,
				)
			}
			accept, err := runTrustTea(ctx)
			if err != nil {
				return fmt.Errorf("trust UI: %w", err)
			}
			if !accept {
				return fmt.Errorf("trust declined")
			}
			if err := WriteTrustMarker(globalDir); err != nil {
				return fmt.Errorf("write trust marker: %w", err)
			}
		}
	}

	if HasAPIKeyConfigured(globalDir) {
		return nil
	}

	if rt.NonInteractive {
		if truthy(os.Getenv(EnvAllowMissingAPIKey)) {
			rt.Log.Warn("missing API key; continuing due to " + EnvAllowMissingAPIKey)
			return nil
		}
		return fmt.Errorf(
			"missing API key (non-interactive): set ANTHROPIC_API_KEY or RABBIT_CODE_API_KEY, or %s=1 for dev/CI",
			EnvAllowMissingAPIKey,
		)
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf(
			"missing API key and no TTY: set ANTHROPIC_API_KEY or RABBIT_CODE_API_KEY, or %s=1",
			EnvAllowMissingAPIKey,
		)
	}

	key, err := runAPIKeyTea(ctx)
	if err != nil {
		return fmt.Errorf("API key UI: %w", err)
	}
	if key == "" {
		return fmt.Errorf("API key not provided (paste a key or set ANTHROPIC_API_KEY / RABBIT_CODE_API_KEY)")
	}
	if err := WriteAPIKeyFile(globalDir, key); err != nil {
		return fmt.Errorf("save API key: %w", err)
	}
	rt.Log.Debug("API key saved to user config file", "file", apiKeyFilePath(globalDir))
	return nil
}
