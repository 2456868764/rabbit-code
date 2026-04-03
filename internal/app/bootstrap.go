package app

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/2456868764/rabbit-code/internal/bootstrap"
	"github.com/2456868764/rabbit-code/internal/features"
)

// ExitAfterInitEnv mirrors PHASE01 E2E: exit 0 after successful Bootstrap.
const ExitAfterInitEnv = "RABBIT_CODE_EXIT_AFTER_INIT"

// Runtime holds live process state after Bootstrap.
type Runtime struct {
	State             *bootstrap.State
	Log               *slog.Logger
	SlowOpLog         *slog.Logger
	LibcGlibc         bool
	LibcMusl          bool
	Proxy             ProxyConfig
	RootCAs           *x509.CertPool
	NonInteractive    bool
	GlobalConfigDir   string
	ProjectConfigPath string
	// MergedSettings snapshot after successful LoadAndApplyMergedConfig (trust path); used for engine memdir H8.
	MergedSettings map[string]interface{}
	Cleanup        *CleanupRegistry
}

// Close runs registered cleanups (LIFO). Safe to call multiple times.
func (r *Runtime) Close() {
	if r == nil || r.Cleanup == nil {
		return
	}
	r.Cleanup.Run()
}

// Bootstrap executes Phase 1 startup: env, logging, TLS trust store, proxy, discovery, parallel prefetch hooks.
// ctx cancellation must stop prefetch goroutines without leaking them.
func Bootstrap(ctx context.Context) (*Runtime, error) {
	start := time.Now()
	ApplySafeManagedEnv()

	log, logClose, err := NewLogger()
	if err != nil {
		return nil, err
	}
	reg := &CleanupRegistry{}
	reg.Register(logClose)

	slowLog, slowClose, err := NewSlowLogger()
	if err != nil {
		reg.Run()
		return nil, err
	}
	if slowLog != nil {
		reg.Register(slowClose)
	}

	pool, err := SystemCertPool()
	if err != nil {
		reg.Run()
		return nil, fmt.Errorf("cert pool: %w", err)
	}

	// Phase 4 / AC4-6: same outbound resolution as Messages (ResolveAPIOutboundTransport).
	runAPIPreconnect(ctx, pool, log)

	cwd, err := os.Getwd()
	if err != nil {
		reg.Run()
		return nil, err
	}

	globalDir, err := GlobalConfigDir()
	if err != nil {
		reg.Run()
		return nil, err
	}

	st := bootstrap.NewState()
	st.SetCwd(cwd)
	root := FindProjectRoot(cwd)
	st.SetProjectRoot(root)
	st.SetSessionID(newSessionID())

	LogUndercoverMode(log)
	glibc, musl := DetectLibc()
	log.Debug("libc detection", "glibc", glibc, "musl", musl)

	if features.FilePersistenceEnabled() {
		reg.Register(func() {
			log.Debug("FILE_PERSISTENCE shutdown hook (no-op until Phase 8 session I/O)")
		})
	}

	// Phase 4: API key path warmed here; Keychain + OAuth slots still NoopPrefetch unless AC requires them.
	np := NoopPrefetch{}
	prefetchStart := time.Now()
	if err := ParallelPrefetch(ctx, np, np, APIKeyFilePrefetch{GlobalConfigDir: globalDir}); err != nil {
		reg.Run()
		return nil, err
	}
	LogBootstrapPrefetchSlow(slowLog, time.Since(prefetchStart))

	if d := time.Since(start); d > time.Second {
		log.Warn("bootstrap exceeded 1s", "duration", d)
	}

	ni := IsNonInteractive()
	rt := &Runtime{
		State:             st,
		Log:               log,
		SlowOpLog:         slowLog,
		LibcGlibc:         glibc,
		LibcMusl:          musl,
		Proxy:             LoadProxyFromEnv(),
		RootCAs:           pool,
		NonInteractive:    ni,
		GlobalConfigDir:   globalDir,
		ProjectConfigPath: ProjectConfigCandidate(root),
		Cleanup:           reg,
	}
	runLodestonePhase1Hook(log, ni)
	log.Debug("bootstrap complete",
		"session_id", st.SessionID(),
		"project_root", root,
		"non_interactive", rt.NonInteractive,
		"hard_fail_env", features.HardFailEnabled(),
		"slow_op_log", features.SlowOperationLoggingEnabled(),
		"file_persistence_env", features.FilePersistenceEnabled(),
		"lodestone_env", features.LodestoneEnabled(),
	)
	return rt, nil
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
