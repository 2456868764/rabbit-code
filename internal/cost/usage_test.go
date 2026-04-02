package cost

import (
	"testing"

	"github.com/2456868764/rabbit-code/internal/bootstrap"
)

func TestEmptyUsage(t *testing.T) {
	u := EmptyUsage()
	if u.InputTokens != 0 || u.OutputTokens != 0 {
		t.Fatal(u)
	}
	if u.ServiceTier != "standard" || u.Speed != "standard" {
		t.Fatal(u)
	}
}

func TestMerge(t *testing.T) {
	a := EmptyUsage()
	b := Usage{InputTokens: 10, OutputTokens: 3, ServiceTier: "priority"}
	m := Merge(a, b)
	if m.InputTokens != 10 || m.OutputTokens != 3 || m.ServiceTier != "priority" {
		t.Fatal(m)
	}
}

func TestApplyUsageToBootstrap(t *testing.T) {
	st := bootstrap.NewState()
	ApplyUsageToBootstrap(st, Usage{InputTokens: 5000, OutputTokens: 100})
	u := st.LastTokenUsage()
	if u.InputTokens != 5000 || u.OutputTokens != 100 {
		t.Fatal(u)
	}
	if st.TotalCost() == 0 {
		t.Fatal("expected coarse cost bump")
	}
}
