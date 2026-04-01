package app

// Environment variables for first-run trust / API key onboarding (P1.5.1 / AC1-8).
const (
	// EnvSkipOnboarding skips trust + API key prompts (CI, automation). Does not write trust marker.
	EnvSkipOnboarding = "RABBIT_CODE_SKIP_ONBOARDING"
	// EnvAcceptTrust in non-interactive mode writes the trust marker without a TUI.
	EnvAcceptTrust = "RABBIT_CODE_ACCEPT_TRUST"
	// EnvAllowMissingAPIKey in non-interactive mode allows proceeding without env or file API key (logs a warning).
	EnvAllowMissingAPIKey = "RABBIT_CODE_ALLOW_MISSING_API_KEY"
)
