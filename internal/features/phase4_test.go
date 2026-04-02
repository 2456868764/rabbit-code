package features

import "testing"

func TestOAuthBetaAppendNames(t *testing.T) {
	t.Setenv(EnvOAuthBetaAppend, " a , b ")
	got := OAuthBetaAppendNames()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%v", got)
	}
}

func TestNativeAttestationRequestHeader(t *testing.T) {
	t.Setenv(EnvNativeAttestation, "1")
	t.Setenv(EnvNativeAttestationHeader, "X-Test-Attest")
	t.Setenv(EnvNativeAttestationValue, "token")
	n, v, ok := NativeAttestationRequestHeader()
	if !ok || n != "X-Test-Attest" || v != "token" {
		t.Fatalf("%q %q %v", n, v, ok)
	}
}

func TestStrictForeground529Enabled(t *testing.T) {
	t.Setenv(EnvStrictForeground529, "1")
	if !StrictForeground529Enabled() {
		t.Fatal()
	}
	t.Setenv(EnvStrictForeground529, "")
	if StrictForeground529Enabled() {
		t.Fatal()
	}
}

func TestAttributionHeaderPromptEnabled(t *testing.T) {
	t.Setenv(EnvAttributionHeader, "0")
	if AttributionHeaderPromptEnabled() {
		t.Fatal()
	}
	t.Setenv(EnvAttributionHeader, "1")
	if !AttributionHeaderPromptEnabled() {
		t.Fatal()
	}
}

func TestAntiDistillationFakeToolsInBody(t *testing.T) {
	t.Setenv(EnvAntiDistillation, "")
	t.Setenv(EnvAntiDistillationFakeTools, "1")
	if AntiDistillationFakeToolsInBody() {
		t.Fatal("CC off")
	}
	t.Setenv(EnvAntiDistillation, "1")
	if !AntiDistillationFakeToolsInBody() {
		t.Fatal("both on")
	}
}

func TestDisableKeepAliveOnECONNRESETEnabled(t *testing.T) {
	t.Setenv(EnvDisableKeepAliveOnECONNRESET, "1")
	if !DisableKeepAliveOnECONNRESETEnabled() {
		t.Fatal()
	}
	t.Setenv(EnvDisableKeepAliveOnECONNRESET, "")
	if DisableKeepAliveOnECONNRESETEnabled() {
		t.Fatal()
	}
}

func TestAntiDistillationRequestHeader(t *testing.T) {
	t.Setenv(EnvAntiDistillation, "1")
	t.Setenv(EnvAntiDistillationHeader, "X-Custom-AD")
	t.Setenv(EnvAntiDistillationValue, "yes")
	n, v, ok := AntiDistillationRequestHeader()
	if !ok || n != "X-Custom-AD" || v != "yes" {
		t.Fatalf("%q %q %v", n, v, ok)
	}
}
