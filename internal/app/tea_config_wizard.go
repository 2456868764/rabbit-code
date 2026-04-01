package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/2456868764/rabbit-code/internal/config"
)

type wizardResult struct {
	autoTheme    string
	teamMemPath  string
	shouldWrite  bool
	aborted      bool
}

// steps: 0 intro, 1 overwrite prompt (if needed), 2 theme, 3 team path, 4 confirm
type cfgWizModel struct {
	width, height   int
	step            int
	hasExisting     bool
	overwrite       bool // user agreed to touch existing keys
	autoTheme       string
	teamTI          textinput.Model
	aborted     bool
	shouldWrite bool
}

func newCfgWizModel(globalDir string) (*cfgWizModel, error) {
	userPath := filepath.Join(globalDir, config.UserConfigFileName)
	u, err := config.ReadJSONFile(userPath)
	if err != nil {
		return nil, err
	}
	m := &cfgWizModel{
		step:        0,
		hasExisting: len(u) > 0,
		teamTI:      newTeamPathInput(),
	}
	return m, nil
}

func newTeamPathInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "optional path (Enter to skip)"
	ti.CharLimit = 4096
	ti.Prompt = ""
	s := ti.Styles()
	s.Cursor.Color = lipgloss.Color("220")
	ti.SetStyles(s)
	return ti
}

func (m *cfgWizModel) Init() tea.Cmd {
	return nil
}

func (m *cfgWizModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.step == 3 {
			fieldW := msg.Width - 12
			if fieldW < 24 {
				fieldW = 24
			}
			m.teamTI.SetWidth(fieldW)
		}
		return m, nil
	case tea.KeyPressMsg:
		switch m.step {
		case 0:
			m.step = 1
			if !m.hasExisting {
				m.overwrite = true
				m.step = 2
				return m, nil
			}
			return m, nil
		case 1:
			switch strings.ToLower(msg.String()) {
			case "o", "enter":
				m.overwrite = true
				m.step = 2
				return m, nil
			case "q", "n", "esc", "ctrl+c":
				m.aborted = true
				return m, tea.Quit
			}
			return m, nil
		case 2:
			switch msg.String() {
			case "1":
				m.autoTheme = "auto"
				m.step = 3
				return m, m.teamTI.Focus()
			case "2":
				m.autoTheme = "light"
				m.step = 3
				return m, m.teamTI.Focus()
			case "3":
				m.autoTheme = "dark"
				m.step = 3
				return m, m.teamTI.Focus()
			case "q", "esc", "ctrl+c":
				m.aborted = true
				return m, tea.Quit
			}
			return m, nil
		case 3:
			switch msg.String() {
			case "enter":
				m.step = 4
				return m, nil
			case "esc", "ctrl+c":
				m.aborted = true
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.teamTI, cmd = m.teamTI.Update(msg)
			return m, cmd
		case 4:
			switch strings.ToLower(msg.String()) {
			case "y", "enter":
				m.shouldWrite = true
				return m, tea.Quit
			case "n", "q", "esc", "ctrl+c":
				m.aborted = true
				return m, tea.Quit
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *cfgWizModel) View() tea.View {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	var inner string
	switch m.step {
	case 0:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			onboardingTitleStyle.Render("rabbit-code — config wizard"),
			"",
			onboardingBodyStyle.Render("Set auto_theme and optional team_mem_path in your user config."),
			onboardingBodyStyle.Render("Press any key to continue."),
		)
	case 1:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			onboardingTitleStyle.Render("Existing configuration"),
			"",
			onboardingBodyStyle.Render("Your user config already has keys."),
			onboardingBodyStyle.Render("This wizard will add or overwrite auto_theme and team_mem_path only."),
			"",
			onboardingKeyStyle.Render("[O] Continue   [Q] Quit"),
		)
	case 2:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			onboardingTitleStyle.Render("Theme (auto_theme)"),
			"",
			onboardingBodyStyle.Render("[1] auto   [2] light   [3] dark"),
			"",
			onboardingKeyStyle.Render("[Q] Quit"),
		)
	case 3:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			onboardingTitleStyle.Render("Team / MEM path (optional)"),
			"",
			onboardingBodyStyle.Render("team_mem_path — leave empty to skip."),
			"",
			m.teamTI.View(),
			"",
			onboardingKeyStyle.Render("[Enter] next   [Esc] cancel"),
		)
	case 4:
		inner = lipgloss.JoinVertical(lipgloss.Left,
			onboardingTitleStyle.Render("Confirm"),
			"",
			onboardingBodyStyle.Render(fmt.Sprintf("auto_theme: %s", m.autoTheme)),
			onboardingBodyStyle.Render(fmt.Sprintf("team_mem_path: %q", strings.TrimSpace(m.teamTI.Value()))),
			"",
			onboardingKeyStyle.Render("[Y] Write   [N] Cancel"),
		)
	}
	boxW := min(max(w-6, 40), 72)
	boxed := onboardingBoxStyle.Width(boxW).Render(inner)
	placed := lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, boxed)
	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}

func runConfigWizardTea(ctx context.Context, globalDir string) (*wizardResult, error) {
	m, err := newCfgWizModel(globalDir)
	if err != nil {
		return nil, err
	}
	p := tea.NewProgram(m, tea.WithContext(ctx))
	final, err := p.Run()
	if err != nil {
		if errors.Is(err, tea.ErrProgramKilled) {
			return nil, ctx.Err()
		}
		return nil, err
	}
	wm, ok := final.(*cfgWizModel)
	if !ok {
		return nil, fmt.Errorf("config wizard: unexpected model state")
	}
	return &wizardResult{
		autoTheme:   wm.autoTheme,
		teamMemPath: wm.teamTI.Value(),
		shouldWrite: wm.shouldWrite,
		aborted:     wm.aborted,
	}, nil
}
