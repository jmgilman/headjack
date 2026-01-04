// Package cmd implements the Headjack CLI commands using Cobra.
// It provides commands for managing isolated LLM agent instances,
// including create, attach, stop, remove, and session management.
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
	"github.com/jmgilman/headjack/internal/flags"
	"github.com/jmgilman/headjack/internal/git"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/multiplexer"
	"github.com/jmgilman/headjack/internal/registry"
)

// baseDeps lists the external binaries that must always be available.
var baseDeps = []string{"git"}

// runtimeNameDocker is the runtime name for Docker.
const runtimeNameDocker = "docker"

// runtimeBinaryDocker is the binary name for Docker.
const runtimeBinaryDocker = "docker"

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
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := checkDependencies(); err != nil {
			return err
		}

		if err := initManager(); err != nil {
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
	if appConfig != nil && appConfig.Runtime.Name != "" {
		return appConfig.Runtime.Name // Runtime name matches binary name (docker, podman)
	}
	// Default to docker
	return runtimeBinaryDocker
}

// initManager initializes the instance manager with all dependencies.
func initManager() error {
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
		dataDir, err := defaultDataDir()
		if err != nil {
			return err
		}
		worktreesDir = filepath.Join(dataDir, "git")
		catalogPath = filepath.Join(dataDir, "catalog.json")
		logsDir = filepath.Join(dataDir, "logs")
	}

	executor := hjexec.New()
	store := catalog.NewStore(catalogPath)

	// Select runtime: config > default (docker)
	var runtime container.Runtime
	runtimeName := runtimeNameDocker // default
	if appConfig != nil && appConfig.Runtime.Name != "" {
		runtimeName = appConfig.Runtime.Name
	}
	switch runtimeName {
	case runtimeNameDocker:
		runtime = container.NewDockerRuntime(executor, container.DockerConfig{})
	default:
		runtime = container.NewPodmanRuntime(executor, container.PodmanConfig{})
	}

	opener := git.NewOpener(executor)

	// Use tmux as the terminal multiplexer
	mux := multiplexer.NewTmux(executor)

	// Create registry client for fetching image metadata
	regClient := registry.NewClient(registry.ClientConfig{})

	// Map runtime name to RuntimeType
	runtimeType := runtimeNameToType(runtimeName)

	// Parse config flags for merging with image label flags
	configFlags, err := getConfigFlags()
	if err != nil {
		return err
	}

	mgr = instance.NewManager(store, runtime, opener, mux, regClient, instance.ManagerConfig{
		WorktreesDir: worktreesDir,
		LogsDir:      logsDir,
		RuntimeType:  runtimeType,
		ConfigFlags:  configFlags,
		Executor:     executor,
	})

	return nil
}

// runtimeNameToType converts a runtime name string to RuntimeType.
func runtimeNameToType(name string) instance.RuntimeType {
	switch name {
	case runtimeNameDocker:
		return instance.RuntimeDocker
	default:
		return instance.RuntimePodman
	}
}

// getConfigFlags parses runtime flags from config.
func getConfigFlags() (flags.Flags, error) {
	if appConfig == nil || appConfig.Runtime.Flags == nil {
		return make(flags.Flags), nil
	}
	configFlags, err := flags.FromConfig(appConfig.Runtime.Flags)
	if err != nil {
		return nil, fmt.Errorf("parse runtime flags: %w", err)
	}
	return configFlags, nil
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
