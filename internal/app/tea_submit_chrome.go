package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

// EngineEventMsg carries one engine event into the Bubble Tea loop (T3 / future REPL multiplexer).
type EngineEventMsg struct {
	Event engine.EngineEvent
}

// engineEventsClosedMsg signals the engine event channel was closed.
type engineEventsClosedMsg struct{}

var submitChromeBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// SubmitChromeTeaModel subscribes to engine.EngineEvent values and renders budget + memdir hints (T3 §3.0 序 2).
type SubmitChromeTeaModel struct {
	ch    <-chan engine.EngineEvent
	state SubmitChromeState
}

// NewSubmitChromeTeaModel returns a model that listens on ch; each event is read via sequential tea.Cmd callbacks.
func NewSubmitChromeTeaModel(ch <-chan engine.EngineEvent) *SubmitChromeTeaModel {
	return &SubmitChromeTeaModel{ch: ch}
}

func listenOneEngineEvent(ch <-chan engine.EngineEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return engineEventsClosedMsg{}
		}
		return EngineEventMsg{Event: ev}
	}
}

// Init starts listening for the first engine event.
func (m *SubmitChromeTeaModel) Init() tea.Cmd {
	if m == nil || m.ch == nil {
		return nil
	}
	return listenOneEngineEvent(m.ch)
}

// Update applies engine events and re-arms the listener until the channel closes.
func (m *SubmitChromeTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case EngineEventMsg:
		m.state.ApplyEngineEvent(msg.Event)
		if m.ch != nil {
			return m, listenOneEngineEvent(m.ch)
		}
		return m, nil
	case engineEventsClosedMsg:
		return m, nil
	}
	return m, nil
}

// View renders the chrome line (empty when no data yet).
func (m *SubmitChromeTeaModel) View() tea.View {
	if m == nil {
		return tea.NewView("")
	}
	line := m.state.FormatSubmitChromeLine()
	if line == "" {
		return tea.NewView("")
	}
	return tea.NewView(submitChromeBarStyle.Render(line))
}
