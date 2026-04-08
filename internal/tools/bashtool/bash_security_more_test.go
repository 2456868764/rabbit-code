package bashtool

import "testing"

func TestBashReadOnlySecurityRejectReason_zshDangerous(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`zmodload zsh/mapfile`) == "" {
		t.Fatal("expected zmodload rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_heredocInSubst(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo $(cat <<EOF
x
EOF
)`) == "" {
		t.Fatal("expected heredoc inside substitution rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_doubleQuotedSubst(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo "$(date)"`) == "" {
		t.Fatal("expected command substitution rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_carriageReturn(t *testing.T) {
	if BashReadOnlySecurityRejectReason("echo hi\rbad") == "" {
		t.Fatal("expected CR outside double quotes rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_IFS(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo $IFS`) == "" {
		t.Fatal("expected IFS rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_procEnviron(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`cat /proc/self/environ`) == "" {
		t.Fatal("expected /proc/*/environ rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_braceExpansion(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo {a,b}`) == "" {
		t.Fatal("expected brace expansion rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_jqDangerousFlag(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`jq -f /tmp/x .`) == "" {
		t.Fatal("expected jq -f rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_fcDashE(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`fc -e vi`) == "" {
		t.Fatal("expected fc -e rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_ansiCQuote(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo $'foo'`) == "" {
		t.Fatal("expected ANSI-C quoting rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_incompleteOperator(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`&& echo hi`) == "" {
		t.Fatal("expected leading operator rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_redirection(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`echo x > /tmp/y`) == "" {
		t.Fatal("expected output redirection rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_backslashOperator(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`cat a.txt \; echo b`) == "" {
		t.Fatal("expected backslash before ; rejected")
	}
}

func TestBashReadOnlySecurityRejectReason_zshPrecmdModifier(t *testing.T) {
	if BashReadOnlySecurityRejectReason(`command zmodload zsh/mapfile`) == "" {
		t.Fatal("expected zmodload after command modifier rejected")
	}
}
