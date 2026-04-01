package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/2456868764/rabbit-code/internal/config"
)

// RunConfigWizard runs the Phase 2 settings wizard (P2.4.1); does not replace Phase 1 trust/onboarding.
func RunConfigWizard(ctx context.Context) error {
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	res, err := runConfigWizardTea(ctx, globalDir)
	if err != nil {
		return err
	}
	if res == nil || res.aborted || !res.shouldWrite {
		return nil
	}
	path := filepath.Join(globalDir, config.UserConfigFileName)
	out, err := config.ReadJSONFile(path)
	if err != nil {
		return err
	}
	if res.autoTheme != "" {
		out["auto_theme"] = res.autoTheme
	}
	if strings.TrimSpace(res.teamMemPath) != "" {
		out["team_mem_path"] = strings.TrimSpace(res.teamMemPath)
	}
	if verrs := config.Validate(out); len(verrs) > 0 {
		return fmt.Errorf("validation: %v", verrs)
	}
	return config.AtomicWriteJSON(path, out)
}
