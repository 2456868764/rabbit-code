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
