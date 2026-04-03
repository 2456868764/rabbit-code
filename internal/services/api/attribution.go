package anthropic

import (
	"fmt"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/version"
)

// AttributionSystemPromptLine returns the system-prompt line that embeds billing / attribution metadata,
// matching constants/system.ts getAttributionHeader (including optional cch=00000 when NATIVE_CLIENT_ATTESTATION is on).
// When disabled via RABBIT_CODE_ATTRIBUTION_HEADER (falsy when set), returns "".
func AttributionSystemPromptLine(fingerprint, entrypoint, workload string) string {
	if !features.AttributionHeaderPromptEnabled() {
		return ""
	}
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		fp = "unknown"
	}
	ver := strings.TrimSpace(version.Version) + "." + fp
	ep := strings.TrimSpace(entrypoint)
	if ep == "" {
		ep = "unknown"
	}
	var cch string
	if features.NativeClientAttestation() {
		cch = " cch=00000;"
	}
	var wpair string
	if w := strings.TrimSpace(workload); w != "" {
		wpair = fmt.Sprintf(" cc_workload=%s;", w)
	}
	return fmt.Sprintf("x-anthropic-billing-header: cc_version=%s; cc_entrypoint=%s;%s%s", ver, ep, cch, wpair)
}
