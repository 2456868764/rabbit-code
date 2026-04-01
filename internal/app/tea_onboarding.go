package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Onboarding lipgloss v2 theme (first-run screens).
var (
	onboardingTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	onboardingBodyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	onboardingKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	onboardingBoxStyle   = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2).
				Margin(0, 1)
)

type trustModel struct {
	width, height int
	decided       bool
	accept        bool
}

func (m *trustModel) Init() tea.Cmd { return nil }

func (m *trustModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "y", "enter":
			m.decided, m.accept = true, true
			return m, tea.Quit
		case "n", "q", "esc":
			m.decided, m.accept = true, false
			return m, tea.Quit
		case "ctrl+c":
			m.decided, m.accept = true, false
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *trustModel) View() tea.View {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	inner := lipgloss.JoinVertical(lipgloss.Left,
		onboardingTitleStyle.Render("rabbit-code — first-run trust"),
		"",
		onboardingBodyStyle.Render("This tool may run commands and access files in your project."),
		onboardingBodyStyle.Render("By continuing you accept responsibility for how you use it."),
		"",
		onboardingKeyStyle.Render("[Y] Accept   [N] Decline"),
	)
	boxW := min(max(w-6, 40), 72)
	boxed := onboardingBoxStyle.Width(boxW).Render(inner)
	placed := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, boxed)
	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}

func runTrustTea(ctx context.Context) (accept bool, err error) {
	m := &trustModel{}
	p := tea.NewProgram(m, tea.WithContext(ctx))
	final, err := p.Run()
	if err != nil {
		if errors.Is(err, tea.ErrProgramKilled) {
			return false, ctx.Err()
		}
		return false, err
	}
	tm, ok := final.(*trustModel)
	if !ok || !tm.decided {
		return false, fmt.Errorf("trust UI: unexpected model state")
	}
	return tm.accept, nil
}

type apiKeyModel struct {
	width, height int
	ti            textinput.Model
	cancelled     bool
}

func newAPIKeyModel() *apiKeyModel {
	ti := textinput.New()
	ti.Placeholder = "paste API key (hidden)"
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 8192
	ti.Prompt = ""
	s := ti.Styles()
	s.Cursor.Color = lipgloss.Color("220")
	ti.SetStyles(s)
	return &apiKeyModel{ti: ti}
}

func (m *apiKeyModel) Init() tea.Cmd {
	return m.ti.Focus()
}

func (m *apiKeyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		fieldW := msg.Width - 12
		if fieldW < 24 {
			fieldW = 24
		}
		m.ti.SetWidth(fieldW)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *apiKeyModel) View() tea.View {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	field := m.ti.View()
	inner := lipgloss.JoinVertical(lipgloss.Left,
		onboardingTitleStyle.Render("API key"),
		"",
		onboardingBodyStyle.Render("Set ANTHROPIC_API_KEY or RABBIT_CODE_API_KEY in your environment,"),
		onboardingBodyStyle.Render("or paste a key below. Empty line + Enter cancels."),
		"",
		field,
		"",
		onboardingKeyStyle.Render("[Enter] submit   [Esc] cancel"),
	)
	boxW := min(max(w-6, 40), 72)
	boxed := onboardingBoxStyle.Width(boxW).Render(inner)
	placed := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, boxed)
	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}

func runAPIKeyTea(ctx context.Context) (key string, err error) {
	m := newAPIKeyModel()
	p := tea.NewProgram(m, tea.WithContext(ctx))
	final, err := p.Run()
	if err != nil {
		if errors.Is(err, tea.ErrProgramKilled) {
			return "", ctx.Err()
		}
		return "", err
	}
	am, ok := final.(*apiKeyModel)
	if !ok {
		return "", fmt.Errorf("API key UI: unexpected model state")
	}
	if am.cancelled {
		return "", nil
	}
	return strings.TrimSpace(am.ti.Value()), nil
}
