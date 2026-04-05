package globtool

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
)

const orphanedAtFilename = ".orphaned_at"

var (
	pluginExclMu     sync.Mutex
	pluginExclCached []string
	pluginExclReady  bool
)

// ClearPluginCacheExclusions mirrors clearPluginCacheExclusions (orphanedPluginFilter.ts); call on plugin reload.
func ClearPluginCacheExclusions() {
	pluginExclMu.Lock()
	defer pluginExclMu.Unlock()
	pluginExclCached = nil
	pluginExclReady = false
}

func pluginsDirectory() string {
	if e := strings.TrimSpace(os.Getenv("CLAUDE_CODE_PLUGIN_CACHE_DIR")); e != "" {
		return expandTildeClaudePath(e)
	}
	name := "plugins"
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("CLAUDE_CODE_USE_COWORK_PLUGINS"))); v == "1" || v == "true" || v == "yes" || v == "on" {
		name = "cowork_plugins"
	}
	return filepath.Join(features.ConfigHomeDir(), name)
}

func pluginCachePath() string {
	return filepath.Join(pluginsDirectory(), "cache")
}

func expandTildeClaudePath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return filepath.Clean(p)
		}
		return filepath.Clean(filepath.Join(h, strings.TrimPrefix(p, "~/")))
	}
	return filepath.Clean(p)
}

// pathsOverlap mirrors orphanedPluginFilter.ts pathsOverlap.
func pathsOverlap(a, b string) bool {
	na := filepath.Clean(a)
	nb := filepath.Clean(b)
	if runtime.GOOS == "windows" {
		na = strings.ToLower(na)
		nb = strings.ToLower(nb)
	}
	sep := string(filepath.Separator)
	if na == nb {
		return true
	}
	if na == sep || nb == sep {
		return true
	}
	return strings.HasPrefix(na, nb+sep) || strings.HasPrefix(nb, na+sep)
}

// getGlobExclusionsForPluginCache mirrors getGlobExclusionsForPluginCache (orphanedPluginFilter.ts).
func getGlobExclusionsForPluginCache(ctx context.Context, rgPath, searchPath string) []string {
	cachePath := filepath.Clean(pluginCachePath())
	if searchPath != "" && !pathsOverlap(searchPath, cachePath) {
		return nil
	}

	pluginExclMu.Lock()
	if pluginExclReady {
		out := append([]string(nil), pluginExclCached...)
		pluginExclMu.Unlock()
		return out
	}
	pluginExclMu.Unlock()

	// Compute (single flight simplified: lock around full compute)
	pluginExclMu.Lock()
	defer pluginExclMu.Unlock()
	if pluginExclReady {
		return append([]string(nil), pluginExclCached...)
	}

	if _, err := os.Stat(cachePath); err != nil {
		pluginExclCached = nil
		pluginExclReady = true
		return nil
	}

	runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	args := []string{
		"--files",
		"--hidden",
		"--no-ignore",
		"--max-depth", "4",
		"--glob", orphanedAtFilename,
		cachePath,
	}
	cmd := exec.CommandContext(runCtx, rgPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		pluginExclCached = nil
		pluginExclReady = true
		return nil
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var excl []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var markerPath string
		if filepath.IsAbs(line) {
			markerPath = filepath.Clean(line)
		} else {
			markerPath = filepath.Clean(filepath.Join(cachePath, line))
		}
		versionDir := filepath.Dir(markerPath)
		rel, err := filepath.Rel(cachePath, versionDir)
		if err != nil {
			continue
		}
		posixRel := strings.ReplaceAll(filepath.ToSlash(rel), "\\", "/")
		excl = append(excl, "!**/"+posixRel+"/**")
	}
	pluginExclCached = excl
	pluginExclReady = true
	return append([]string(nil), pluginExclCached...)
}
