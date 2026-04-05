package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/2456868764/rabbit-code/internal/app"
	"github.com/2456868764/rabbit-code/internal/commands/breakcache"
	"github.com/2456868764/rabbit-code/internal/services/api/services"
	"github.com/2456868764/rabbit-code/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("rabbit-code %s (%s)\n", version.Version, version.Commit)
			return
		case "config":
			if err := handleConfigSubcommand(); err != nil {
				fmt.Fprintf(os.Stderr, "rabbit-code: %v\n", err)
				os.Exit(1)
			}
			return
		case "probe":
			tsFile := services.EmptyUsage
			if len(os.Args) > 2 {
				tsFile = os.Args[2]
			}
			if err := app.RunProbe(context.Background(), os.Stdout, tsFile); err != nil {
				fmt.Fprintf(os.Stderr, "rabbit-code: probe: %v\n", err)
				os.Exit(1)
			}
			return
		case "context":
			if len(os.Args) < 3 || os.Args[2] != "break-cache" {
				fmt.Fprintf(os.Stderr, "usage: rabbit-code context break-cache\n")
				os.Exit(1)
			}
			if err := breakcache.WriteBreakCacheCommandJSON(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "rabbit-code: context: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_ = app.PrintStartupBanner(os.Stderr)

	rt, err := app.Bootstrap(ctx)
	if err != nil {
		app.PrintBootstrapFailure(err)
	}

	if os.Getenv(app.ExitAfterInitEnv) == "1" {
		// AC1-7: full Bootstrap, skip first-run TUI (no hang in CI).
		app.QuitRuntime(rt, 0)
	}

	if err := app.RunPostBootstrapOnboarding(ctx, rt); err != nil {
		app.FailBootstrap(rt, err)
	}

	// Phase 2: after trust, load merged config and apply managed_env (SPEC §1.5, AC2-5).
	if ok, err := app.TrustAccepted(rt.GlobalConfigDir); err != nil {
		fmt.Fprintf(os.Stderr, "rabbit-code: config: trust check: %v\n", err)
		app.QuitRuntime(rt, 1)
	} else if ok {
		if err := app.LoadAndApplyMergedConfig(rt); err != nil {
			fmt.Fprintf(os.Stderr, "rabbit-code: config: %v\n", err)
			app.QuitRuntime(rt, 1)
		}
		app.RunAPIPreconnect(ctx, rt)
	}

	fmt.Fprintf(os.Stderr, "rabbit-code — Phase 1 bootstrap OK. Commands: version | config dump | probe | context break-cache | set | wizard | sync | %s=1\n", app.ExitAfterInitEnv)
	app.QuitRuntime(rt, 0)
}

func handleConfigSubcommand() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: rabbit-code config dump | set <key> <value> | wizard | sync pull [url] | sync push [url]")
	}
	switch os.Args[2] {
	case "dump":
		return app.RunConfigDump()
	case "set":
		if len(os.Args) != 5 {
			return fmt.Errorf("usage: rabbit-code config set <key> <value>")
		}
		return app.RunConfigSet(os.Args[3], os.Args[4])
	case "wizard":
		return app.RunConfigWizard(context.Background())
	case "sync":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: rabbit-code config sync pull [url] | sync push [url]")
		}
		urlArg := ""
		if len(os.Args) >= 5 {
			urlArg = os.Args[4]
		}
		ctx := context.Background()
		switch os.Args[3] {
		case "pull":
			return app.RunConfigSyncPull(ctx, urlArg)
		case "push":
			return app.RunConfigSyncPush(ctx, urlArg)
		default:
			return fmt.Errorf("unknown config sync %q (want pull or push)", os.Args[3])
		}
	default:
		return fmt.Errorf("unknown config subcommand %q", os.Args[2])
	}
}
