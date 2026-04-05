package app

import (
	"fmt"
	"strings"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

// SubmitChromeState holds T3 status derived from engine events (token budget snapshot + memdir inject).
type SubmitChromeState struct {
	// Budget from EventKindSubmitTokenBudgetSnapshot (H5.3).
	BudgetHasData      bool
	BudgetTotalTokens  int
	BudgetInjectRaw    int
	BudgetModeDetail   string
	// Memdir from EventKindMemdirInject (attachment / memory fragments).
	MemdirFragmentCount int
}

// ApplyEngineEvent updates state for budget snapshot and memdir inject events. Other kinds are ignored.
func (s *SubmitChromeState) ApplyEngineEvent(ev engine.EngineEvent) {
	switch ev.Kind {
	case engine.EventKindSubmitTokenBudgetSnapshot:
		s.BudgetHasData = true
		s.BudgetTotalTokens = ev.PhaseAuxInt
		s.BudgetInjectRaw = ev.PhaseAuxInt2
		s.BudgetModeDetail = ev.PhaseDetail
	case engine.EventKindMemdirInject:
		if ev.MemdirFragmentCount > 0 {
			s.MemdirFragmentCount = ev.MemdirFragmentCount
		}
	default:
	}
}

// FormatSubmitChromeLine returns one status line for footer / status bar, or empty if nothing to show.
func (s SubmitChromeState) FormatSubmitChromeLine() string {
	var parts []string
	if s.BudgetHasData {
		parts = append(parts, fmt.Sprintf("submit ~%d tok", s.BudgetTotalTokens))
		if s.BudgetInjectRaw > 0 {
			parts = append(parts, fmt.Sprintf("inject %d B", s.BudgetInjectRaw))
		}
		if d := strings.TrimSpace(s.BudgetModeDetail); d != "" {
			parts = append(parts, d)
		}
	}
	if s.MemdirFragmentCount > 0 {
		parts = append(parts, fmt.Sprintf("memdir +%d", s.MemdirFragmentCount))
	}
	return strings.Join(parts, " · ")
}
