package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/catalog"
	"github.com/jmgilman/headjack/internal/config"
	"github.com/jmgilman/headjack/internal/container"
	hjexec "github.com/jmgilman/headjack/internal/exec"
	"github.com/jmgilman/headjack/internal/git"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/multiplexer"
)

// baseDeps lists the external binaries that must always be available.
var baseDeps = []string{"git"}

// mgr is the instance manager, initialized in PersistentPreRunE.
var mgr *instance.Manager

// appConfig holds the loaded application configuration.
var appConfig *config.Config

// configLoader is used for accessing agent-specific configuration.
var configLoader *config.Loader

var rootCmd = &cobra.Command{
	Use:   "headjack",
	Short: "Spawn isolated LLM coding agents",
	Long: `Headjack is a CLI tool for spawning isolated CLI-based LLM coding agents
in predefined container environments.

Each agent runs in its own VM-isolated container with a dedicated git worktree,
enabling safe parallel development across multiple branches.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkDependencies(); err != nil {
			return err
		}

		// Get multiplexer override from flag
		muxOverride, err := cmd.Flags().GetString("multiplexer")
		if err != nil {
			return fmt.Errorf("get multiplexer flag: %w", err)
		}
		if muxOverride != "" && !config.IsValidMultiplexer(muxOverride) {
			return fmt.Errorf("invalid multiplexer %q (valid: tmux, zellij)", muxOverride)
		}

		if err := initManager(muxOverride); err != nil {
			return err
		}

		// Store dependencies in context for subcommands
		ctx := cmd.Context()
		ctx = WithConfig(ctx, appConfig)
		ctx = WithLoader(ctx, configLoader)
		ctx = WithManager(ctx, mgr)
		cmd.SetContext(ctx)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("multiplexer", "", "terminal multiplexer to use (tmux, zellij)")
}

func initConfig() {
	loader, err := config.NewLoader()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize config: %v\n", err)
		return
	}

	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		return
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config validation failed: %v\n", err)
	}

	appConfig = cfg
	configLoader = loader
}

// checkDependencies verifies that all required external binaries are available.
func checkDependencies() error {
	var missing []string

	// Check base dependencies
	for _, dep := range baseDeps {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, dep)
		}
	}

	// Check runtime-specific dependency
	runtimeBin := getRuntimeBinary()
	if _, err := exec.LookPath(runtimeBin); err != nil {
		missing = append(missing, runtimeBin)
	}

	if len(missing) > 0 {
		return errors.New("missing required dependencies: " + formatList(missing))
	}
	return nil
}

// getRuntimeBinary returns the binary name for the configured runtime.
func getRuntimeBinary() string {
	if appConfig != nil && appConfig.Runtime.Name == "apple" {
		return "container"
	}
	// Default to podman
	return "podman"
}

// initManager initializes the instance manager with all dependencies.
// muxOverride can be used to override the configured multiplexer.
func initManager(muxOverride string) error {
	var worktreesDir string
	var catalogPath string
	var logsDir string

	if appConfig != nil {
		// Use paths from config (already expanded)
		worktreesDir = appConfig.Storage.Worktrees
		catalogPath = appConfig.Storage.Catalog
		logsDir = appConfig.Storage.Logs
	} else {
		// Fallback to defaults
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		worktreesDir = filepath.Join(home, ".local", "share", "headjack", "git")
		catalogPath = filepath.Join(home, ".local", "share", "headjack", "catalog.json")
		logsDir = filepath.Join(home, ".local", "share", "headjack", "logs")
	}

	executor := hjexec.New()
	store := catalog.NewStore(catalogPath)

	// Select runtime: config > default (podman)
	var runtime container.Runtime
	runtimeName := "podman" // default
	if appConfig != nil && appConfig.Runtime.Name != "" {
		runtimeName = appConfig.Runtime.Name
	}
	switch runtimeName {
	case "apple":
		appleCfg := container.AppleConfig{}
		if appConfig != nil {
			appleCfg.Privileged = appConfig.Runtime.Privileged
			appleCfg.Flags = appConfig.Runtime.Flags
		}
		runtime = container.NewAppleRuntime(executor, appleCfg)
	default:
		podmanCfg := container.PodmanConfig{}
		if appConfig != nil {
			podmanCfg.Privileged = appConfig.Runtime.Privileged
			podmanCfg.Flags = appConfig.Runtime.Flags
		}
		runtime = container.NewPodmanRuntime(executor, podmanCfg)
	}

	opener := git.NewOpener(executor)

	// Select multiplexer: CLI flag > config > default (tmux)
	var mux multiplexer.Multiplexer
	muxName := "tmux" // default
	if appConfig != nil && appConfig.Default.Multiplexer != "" {
		muxName = appConfig.Default.Multiplexer
	}
	if muxOverride != "" {
		muxName = muxOverride
	}
	switch muxName {
	case "zellij":
		mux = multiplexer.NewZellij(executor)
	default:
		mux = multiplexer.NewTmux(executor)
	}

	mgr = instance.NewManager(store, runtime, opener, mux, instance.ManagerConfig{
		WorktreesDir: worktreesDir,
		LogsDir:      logsDir,
	})

	return nil
}

// formatList joins strings with commas and "and" before the last item.
func formatList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		var builder strings.Builder
		for i, item := range items {
			if i == len(items)-1 {
				builder.WriteString("and ")
				builder.WriteString(item)
			} else {
				builder.WriteString(item)
				builder.WriteString(", ")
			}
		}
		return builder.String()
	}
}
