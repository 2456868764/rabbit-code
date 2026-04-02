package anthropic

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
)

func TestAttributionSystemPromptLine_DisabledByEnv(t *testing.T) {
	t.Setenv(features.EnvAttributionHeader, "0")
	if s := AttributionSystemPromptLine("fp", "cli", ""); s != "" {
		t.Fatal(s)
	}
}

func TestAttributionSystemPromptLine_DefaultShape(t *testing.T) {
	t.Setenv(features.EnvNativeAttestation, "")
	line := AttributionSystemPromptLine("abc", "cli", "")
	if !strings.HasPrefix(line, "x-anthropic-billing-header: cc_version=") {
		t.Fatal(line)
	}
	if !strings.Contains(line, "cc_entrypoint=cli;") {
		t.Fatal(line)
	}
	if strings.Contains(line, "cch=") {
		t.Fatal("unexpected cch without native attestation")
	}
}

func TestAttributionSystemPromptLine_NativeAttestationCCH(t *testing.T) {
	t.Setenv(features.EnvNativeAttestation, "1")
	t.Setenv(features.EnvAttributionHeader, "1")
	line := AttributionSystemPromptLine("fp", "sdk", "")
	if !strings.Contains(line, "cch=00000") {
		t.Fatal(line)
	}
}

func TestAttributionSystemPromptLine_Workload(t *testing.T) {
	t.Setenv(features.EnvAttributionHeader, "1")
	line := AttributionSystemPromptLine("fp", "cli", "cron")
	if !strings.Contains(line, "cc_workload=cron;") {
		t.Fatal(line)
	}
}
