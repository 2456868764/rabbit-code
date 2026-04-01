package app

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// NoStartupBannerEnv disables the mascot splash (CI / pipes / quiet logs).
const NoStartupBannerEnv = "RABBIT_CODE_NO_STARTUP_BANNER"

// PrintStartupBanner draws the mascot on capable terminals (iTerm2 / WezTerm inline image),
// otherwise a Lip Gloss text header. Writes to w (typically os.Stderr).
func PrintStartupBanner(w io.Writer) error {
	if truthy(os.Getenv(NoStartupBannerEnv)) || truthy(os.Getenv(ExitAfterInitEnv)) {
		return nil
	}
	fd := int(os.Stderr.Fd())
	if !term.IsTerminal(fd) {
		return nil
	}

	pngBytes, err := MascotPNG()
	if err != nil {
		return err
	}

	if tryITermInlineImage(w, pngBytes) {
		fmt.Fprintln(w)
		line := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("220")).
			Render("rabbit-code")
		fmt.Fprintf(w, "%s\n", line)
		return nil
	}

	// Text fallback (no inline image)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("🐰  rabbit-code"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("interactive coding agent"),
		))
	fmt.Fprintln(w, box)
	return nil
}

// tryITermInlineImage uses the iTerm2 / compatible inline image protocol (WezTerm, etc.).
func tryITermInlineImage(w io.Writer, png []byte) bool {
	if len(png) == 0 {
		return false
	}
	if os.Getenv("ITERM_SESSION_ID") == "" &&
		os.Getenv("WEZTERM_EXECUTABLE") == "" &&
		os.Getenv("TERM_PROGRAM") != "iTerm.app" &&
		!strings.EqualFold(os.Getenv("TERM_PROGRAM"), "WezTerm") {
		return false
	}
	enc := base64.StdEncoding.EncodeToString(png)
	// width/height in character cells; preserveAspectRatio keeps mascot recognizable
	fmt.Fprintf(w, "\033]1337;File=inline=1;width=10;height=5;preserveAspectRatio=1:%s\a", enc)
	return true
}
