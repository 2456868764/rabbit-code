package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/api/services"
	"github.com/2456868764/rabbit-code/internal/config"
)

// RunProbe issues ProbeServiceAPI without full Bootstrap (no TUI). When trust is accepted, loads merged config
// so extra_ca_paths and managed_env match main startup. tsFile is a services/api module name (e.g. emptyUsage.ts).
func RunProbe(ctx context.Context, w io.Writer, tsFile string) error {
	if !services.HasTSModule(tsFile) {
		return fmt.Errorf("unknown services/api module %q", tsFile)
	}
	ApplySafeManagedEnv()
	pool, err := SystemCertPool()
	if err != nil {
		return err
	}
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root := FindProjectRoot(cwd)
	if ok, err := TrustAccepted(globalDir); err != nil {
		return err
	} else if ok {
		m, err := config.LoadMerged(config.Paths{
			GlobalConfigDir: globalDir,
			ProjectRoot:     root,
		})
		if err != nil {
			return err
		}
		extraPaths := config.ExtraCAPEMPaths(m, root, cwd)
		if len(extraPaths) > 0 {
			if err := AppendPEMFiles(pool, extraPaths); err != nil {
				return fmt.Errorf("extra_ca_paths: %w", err)
			}
		}
		_ = ApplyManagedEnvFromMerged(m)
	}
	rt := &Runtime{RootCAs: pool, GlobalConfigDir: globalDir}
	resp, err := ProbeServiceAPI(ctx, rt, tsFile, anthropic.DefaultPolicy())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if w == nil {
		w = io.Discard
	}
	_, err = fmt.Fprintf(w, "probe %s: %s\n", tsFile, resp.Status)
	return err
}
