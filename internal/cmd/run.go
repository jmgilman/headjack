package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/container"
	"github.com/jmgilman/headjack/internal/devcontainer"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/prompt"
)

var runCmd = &cobra.Command{
	Use:   "run <branch> [-- <runtime-flags>...]",
	Short: "Create a new instance for the specified branch",
	Long: `Create a new instance (worktree + container) for the specified branch.

If an instance already exists for the branch, it is reused (and restarted if
stopped). The container environment is determined by:

  1. Devcontainer (default): If the repository contains a devcontainer.json,
     it is used to build and run the container environment automatically.
  2. Base image: Use --image to specify a container image directly, bypassing
     devcontainer detection.

Additional flags can be passed to the container runtime (or devcontainer CLI)
by placing them after a -- separator.

This command only creates the instance. To start a session, use:
  - 'hjk agent <branch> <agent>' to start an agent session
  - 'hjk exec <branch>' to start a shell session`,
	Example: `  # Auto-detect devcontainer.json (recommended)
  hjk run feat/auth

  # Use a specific container image (bypasses devcontainer)
  hjk run feat/auth --image my-registry.io/custom-image:latest

  # Pass additional flags to the container runtime
  hjk run feat/auth -- --memory=4g --privileged

  # Typical workflow: create instance, then start agent
  hjk run feat/auth
  hjk agent feat/auth claude --prompt "Implement JWT authentication"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRunCmd,
}

// runFlags holds parsed flags for the run command.
type runFlags struct {
	image         string
	imageExplicit bool     // true if --image was explicitly passed
	runtimeFlags  []string // flags to pass to the container runtime (after --)
}

// parseRunFlags extracts and validates flags from the command.
func parseRunFlags(cmd *cobra.Command, args []string) (*runFlags, error) {
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		return nil, fmt.Errorf("get image flag: %w", err)
	}
	imageExplicit := cmd.Flags().Changed("image")

	image = resolveBaseImage(cmd.Context(), image)

	return &runFlags{
		image:         image,
		imageExplicit: imageExplicit,
		runtimeFlags:  parsePassthroughArgs(cmd, args),
	}, nil
}

func runRunCmd(cmd *cobra.Command, args []string) error {
	branch := args[0]

	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	flags, err := parseRunFlags(cmd, args)
	if err != nil {
		return err
	}

	repoPath, err := repoPath()
	if err != nil {
		return err
	}

	inst, err := getOrCreateInstance(cmd, mgr, repoPath, branch, flags)
	if err != nil {
		return err
	}

	fmt.Printf("Instance %s ready for branch %s\n", inst.ID, inst.Branch)
	return nil
}

// getOrCreateInstance retrieves an existing instance or creates a new one.
// If the instance exists but is stopped, it restarts the container.
// If imageExplicit is false and a devcontainer.json exists, devcontainer mode is used.
func getOrCreateInstance(cmd *cobra.Command, mgr *instance.Manager, repoPath, branch string, flags *runFlags) (*instance.Instance, error) {
	// Try to get existing instance
	inst, err := mgr.GetByBranch(cmd.Context(), repoPath, branch)
	if err == nil {
		// Instance exists - check if we need to restart it
		if inst.Status == instance.StatusStopped {
			if startErr := mgr.Start(cmd.Context(), inst.ID); startErr != nil {
				return nil, fmt.Errorf("start stopped instance: %w", startErr)
			}
			fmt.Printf("Restarted instance %s for branch %s\n", inst.ID, inst.Branch)
			// Refresh the instance to get updated status
			inst, err = mgr.GetByBranch(cmd.Context(), repoPath, branch)
			if err != nil {
				return nil, fmt.Errorf("get restarted instance: %w", err)
			}
		}
		return inst, nil
	}
	if !errors.Is(err, instance.ErrNotFound) {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	// Build create config - detect devcontainer mode if applicable
	createCfg, err := buildCreateConfig(cmd, repoPath, branch, flags)
	if err != nil {
		return nil, err
	}

	// Create new instance
	inst, err = mgr.Create(cmd.Context(), repoPath, &createCfg)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	fmt.Printf("Created instance %s for branch %s\n", inst.ID, inst.Branch)
	return inst, nil
}

// buildCreateConfig builds the instance creation config, detecting devcontainer mode if applicable.
// Devcontainer mode is used when:
//   - No --image flag was explicitly passed (imageExplicit is false)
//   - A devcontainer.json exists in the repo
//   - The devcontainer CLI is available (or can be installed)
//
// Returns an error if no devcontainer.json is found and no image is configured.
func buildCreateConfig(cmd *cobra.Command, repoPath, branch string, flags *runFlags) (instance.CreateConfig, error) {
	cfg := instance.CreateConfig{
		Branch:       branch,
		Image:        flags.image,
		RuntimeFlags: flags.runtimeFlags,
	}

	// If image was explicitly passed, use vanilla mode
	if flags.imageExplicit {
		return cfg, nil
	}

	// Check for devcontainer.json
	hasDevcontainer := devcontainer.HasConfig(repoPath)

	if hasDevcontainer {
		// Resolve devcontainer CLI (may prompt for installation)
		mgr := ManagerFromContext(cmd.Context())
		if mgr == nil {
			return cfg, errors.New("manager not available")
		}

		loader := LoaderFromContext(cmd.Context())
		if loader == nil {
			return cfg, errors.New("config loader not available")
		}

		resolver := devcontainer.NewCLIResolver(loader, prompt.New(), mgr.Executor())
		cliPath, err := resolver.Resolve(cmd.Context())
		if err != nil {
			return cfg, err
		}

		// Create devcontainer runtime wrapping the underlying runtime
		runtimeName := runtimeNameDocker
		if appCfg := ConfigFromContext(cmd.Context()); appCfg != nil && appCfg.Runtime.Name != "" {
			runtimeName = appCfg.Runtime.Name
		}
		dcRuntime := createDevcontainerRuntime(cmd, runtimeName, cliPath)
		if dcRuntime == nil {
			return cfg, errors.New("failed to create devcontainer runtime")
		}

		fmt.Println("Detected devcontainer.json, using devcontainer mode")

		cfg.WorkspaceFolder = repoPath
		cfg.Runtime = dcRuntime
		cfg.Image = "" // Not needed in devcontainer mode

		return cfg, nil
	}

	// No devcontainer.json - need an image
	if flags.image == "" {
		return cfg, errors.New("no devcontainer.json found and no image configured\n\nTo fix this, either:\n  1. Add a devcontainer.json to your repository\n  2. Use --image to specify a container image\n  3. Set default.base_image in your config")
	}

	return cfg, nil
}

// createDevcontainerRuntime creates a DevcontainerRuntime wrapping the appropriate underlying runtime.
func createDevcontainerRuntime(cmd *cobra.Command, runtimeName, cliPath string) container.Runtime {
	// Get the underlying runtime from the manager
	mgr := ManagerFromContext(cmd.Context())
	if mgr == nil {
		return nil
	}

	// Determine the docker path based on runtime
	var dockerPath string
	switch runtimeName {
	case runtimeNameDocker:
		dockerPath = "docker"
	default:
		dockerPath = "podman"
	}

	// Create devcontainer runtime
	// Note: We use the manager's runtime as the underlying runtime
	return devcontainer.NewRuntime(
		mgr.Runtime(),
		mgr.Executor(),
		cliPath,
		dockerPath,
	)
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String("image", "", "use a container image instead of devcontainer")
}
