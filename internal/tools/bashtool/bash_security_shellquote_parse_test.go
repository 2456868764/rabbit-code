package bashtool

import "testing"

func TestTryParseShellQuote_echoBraceSemicolon(t *testing.T) {
	entries, err := tryParseShellQuote(`echo {a;b}`)
	if err != nil {
		t.Fatal(err)
	}
	if !shellQuoteHasCommandSeparator(entries) {
		t.Fatalf("expected command separator in parse: %#v", entries)
	}
	if !shellQuoteHasMalformedTokens(`echo {a;b}`, entries) {
		t.Fatal("expected malformed tokens (unbalanced { in fragment)")
	}
}

func TestTryParseShellQuote_echoAndAnd(t *testing.T) {
	entries, err := tryParseShellQuote(`echo true && false`)
	if err != nil {
		t.Fatal(err)
	}
	if !shellQuoteHasCommandSeparator(entries) {
		t.Fatal("expected && separator")
	}
	if shellQuoteHasMalformedTokens(`echo true && false`, entries) {
		t.Fatal("expected well-formed tokens")
	}
}

func TestBashReadOnlySecurityRejectReason_malformedTokenInjection(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo {a;b}`) == "" {
		t.Fatal("expected rejection for echo {a;b}")
	}
}

func TestBashReadOnlySecurityRejectReason_unmatchedQuoteWithSeparators(t *testing.T) {
	cmd := `echo "hi;evil | cat`
	if BashReadOnlySecurityRejectReason(cmd) == "" {
		t.Fatal("expected rejection for unmatched quote with ; and | parsed as ops")
	}
}
