package cmd

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/auth"
	"github.com/jmgilman/headjack/internal/config"
	"github.com/jmgilman/headjack/internal/container"
	"github.com/jmgilman/headjack/internal/devcontainer"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/keychain"
)

// agentDefaultSentinel is the sentinel value used when --agent flag is specified without a value.
const agentDefaultSentinel = "default"

var runCmd = &cobra.Command{
	Use:   "run <branch> [prompt]",
	Short: "Create a new session (and instance if needed), then attach",
	Long: `Create a new session within an instance for the specified branch.

If no instance exists for the branch, one is created first. The container
environment is determined by:

  1. Devcontainer (default): If the repository contains a devcontainer.json,
     it is used to build and run the container environment automatically.
  2. Base image: Use --image to specify a container image directly, bypassing
     devcontainer detection.

A new session is always created within the instance. If --agent is specified,
the agent is started (with an optional prompt). Otherwise, the default shell
is started.

Unless --detached is specified, the terminal attaches to the session.
All session output is captured to a log file regardless of attached/detached mode.`,
	Example: `  # Auto-detect devcontainer.json (recommended)
  headjack run feat/auth

  # Start Claude agent in devcontainer
  headjack run feat/auth --agent claude "Implement JWT authentication"

  # Additional session in existing instance
  headjack run feat/auth --agent gemini --name gemini-experiment

  # Shell session with custom name
  headjack run feat/auth --name debug-shell

  # Detached sessions (run in background)
  headjack run feat/auth --agent claude -d "Refactor the auth module"
  headjack run feat/auth --agent claude -d "Write tests for auth module"

  # Use a specific container image (bypasses devcontainer)
  headjack run feat/auth --image my-registry.io/custom-image:latest`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRunCmd,
}

// runFlags holds parsed flags for the run command.
type runFlags struct {
	image         string
	imageExplicit bool // true if --image was explicitly passed
	agent         string
	sessionName   string
	detached      bool
}

// parseRunFlags extracts and validates flags from the command.
func parseRunFlags(cmd *cobra.Command) (*runFlags, error) {
	image, err := cmd.Flags().GetString("image")
	if err != nil {
		return nil, fmt.Errorf("get image flag: %w", err)
	}
	imageExplicit := cmd.Flags().Changed("image")

	agent, err := cmd.Flags().GetString("agent")
	if err != nil {
		return nil, fmt.Errorf("get agent flag: %w", err)
	}
	sessionName, err := cmd.Flags().GetString("name")
	if err != nil {
		return nil, fmt.Errorf("get name flag: %w", err)
	}
	detached, err := cmd.Flags().GetBool("detached")
	if err != nil {
		return nil, fmt.Errorf("get detached flag: %w", err)
	}

	image = resolveBaseImage(cmd.Context(), image)

	return &runFlags{
		image:         image,
		imageExplicit: imageExplicit,
		agent:         agent,
		sessionName:   sessionName,
		detached:      detached,
	}, nil
}

// buildSessionConfig builds a session configuration from flags and args.
func buildSessionConfig(cmd *cobra.Command, flags *runFlags, args []string) (*instance.CreateSessionConfig, error) {
	cfg := &instance.CreateSessionConfig{
		Type: "shell",
		Name: flags.sessionName,
	}

	if flags.agent == "" {
		return cfg, nil
	}

	agent, err := resolveAgent(cmd, flags.agent)
	if err != nil {
		return nil, err
	}

	cfg.Type = agent
	cfg.Command = buildAgentCommand(agent, args)

	// Inject agent-specific environment variables from config
	if loader := LoaderFromContext(cmd.Context()); loader != nil {
		for k, v := range loader.GetAgentEnv(agent) {
			cfg.Env = append(cfg.Env, k+"="+v)
		}
	}

	// Inject authentication credentials from keychain
	if err := injectAuthCredential(agent, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// agentAuthSpec maps agent names to their providers.
type agentAuthSpec struct {
	provider         func() auth.Provider
	notConfiguredMsg string
}

var agentAuthSpecs = map[string]agentAuthSpec{
	"claude": {
		provider:         func() auth.Provider { return auth.NewClaudeProvider() },
		notConfiguredMsg: "claude auth not configured: run 'hjk auth claude' first",
	},
	"gemini": {
		provider:         func() auth.Provider { return auth.NewGeminiProvider() },
		notConfiguredMsg: "gemini auth not configured: run 'hjk auth gemini' first",
	},
	"codex": {
		provider:         func() auth.Provider { return auth.NewCodexProvider() },
		notConfiguredMsg: "codex auth not configured: run 'hjk auth codex' first",
	},
}

// injectAuthCredential retrieves the credential for the agent and configures the session.
func injectAuthCredential(agent string, cfg *instance.CreateSessionConfig) error {
	spec, ok := agentAuthSpecs[agent]
	if !ok {
		return nil
	}

	storage, err := keychain.New()
	if err != nil {
		return fmt.Errorf("initialize credential storage: %w", err)
	}

	provider := spec.provider()
	cred, err := provider.Load(storage)
	if err != nil {
		if errors.Is(err, keychain.ErrNotFound) {
			return errors.New(spec.notConfiguredMsg)
		}
		return fmt.Errorf("load %s credential: %w", agent, err)
	}

	info := provider.Info()

	// Set environment variable based on credential type
	switch cred.Type {
	case auth.CredentialTypeSubscription:
		cfg.Env = append(cfg.Env, info.SubscriptionEnvVar+"="+cred.Value)
		cfg.CredentialType = string(auth.CredentialTypeSubscription)
		cfg.RequiresAgentSetup = info.RequiresContainerSetup
	case auth.CredentialTypeAPIKey:
		cfg.Env = append(cfg.Env, info.APIKeyEnvVar+"="+cred.Value)
		cfg.CredentialType = string(auth.CredentialTypeAPIKey)
		cfg.RequiresAgentSetup = false // API keys don't need file setup
	default:
		return fmt.Errorf("unknown credential type: %s", cred.Type)
	}

	return nil
}

func runRunCmd(cmd *cobra.Command, args []string) error {
	branch := args[0]

	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	flags, err := parseRunFlags(cmd)
	if err != nil {
		return err
	}

	repoPath, err := repoPath()
	if err != nil {
		return err
	}

	inst, err := getOrCreateInstance(cmd, mgr, repoPath, branch, flags.image, flags.imageExplicit)
	if err != nil {
		return err
	}

	sessionCfg, err := buildSessionConfig(cmd, flags, args)
	if err != nil {
		return err
	}

	session, err := mgr.CreateSession(cmd.Context(), inst.ID, sessionCfg)
	if err != nil {
		if errors.Is(err, instance.ErrSessionExists) {
			return fmt.Errorf("session %q already exists in instance %s", flags.sessionName, inst.ID)
		}
		var notRunningErr *instance.NotRunningError
		if errors.As(err, &notRunningErr) {
			hint := formatInstanceNotRunningHint(cmd, notRunningErr)
			if hint != "" {
				return fmt.Errorf("create session: %w\nhint: %s", err, hint)
			}
		}
		return fmt.Errorf("create session: %w", err)
	}

	if flags.detached {
		fmt.Printf("Created session %s in instance %s (detached)\n", session.Name, inst.ID)
		return nil
	}

	return mgr.AttachSession(cmd.Context(), inst.ID, session.Name)
}

func formatInstanceNotRunningHint(cmd *cobra.Command, err *instance.NotRunningError) string {
	if err == nil || err.ContainerID == "" {
		return ""
	}
	runtimeName := runtimeNameDocker
	if cfg := ConfigFromContext(cmd.Context()); cfg != nil && cfg.Runtime.Name != "" {
		runtimeName = cfg.Runtime.Name
	}
	logsCmd := runtimeLogsCommand(runtimeName, err.ContainerID)
	if logsCmd == "" {
		return fmt.Sprintf("container %s is %s", err.ContainerID, err.Status)
	}
	return fmt.Sprintf("container %s is %s; check logs with `%s`", err.ContainerID, err.Status, logsCmd)
}

func runtimeLogsCommand(runtimeName, containerID string) string {
	switch runtimeName {
	case runtimeNameDocker:
		return "docker logs " + containerID
	default:
		return "podman logs " + containerID
	}
}

// getOrCreateInstance retrieves an existing instance or creates a new one.
// If the instance exists but is stopped, it restarts the container.
// If imageExplicit is false and a devcontainer.json exists, devcontainer mode is used.
func getOrCreateInstance(cmd *cobra.Command, mgr *instance.Manager, repoPath, branch, image string, imageExplicit bool) (*instance.Instance, error) {
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
	createCfg, err := buildCreateConfig(cmd, repoPath, branch, image, imageExplicit)
	if err != nil {
		return nil, err
	}

	// Create new instance
	inst, err = mgr.Create(cmd.Context(), repoPath, createCfg)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	fmt.Printf("Created instance %s for branch %s\n", inst.ID, inst.Branch)
	return inst, nil
}

// devcontainerCLI is the name of the devcontainer CLI binary.
const devcontainerCLI = "devcontainer"

// buildCreateConfig builds the instance creation config, detecting devcontainer mode if applicable.
// Devcontainer mode is used when:
//   - No --image flag was explicitly passed (imageExplicit is false)
//   - A devcontainer.json exists in the repo
//   - The devcontainer CLI is available
//
// Returns an error if no devcontainer.json is found and no image is configured.
func buildCreateConfig(cmd *cobra.Command, repoPath, branch, image string, imageExplicit bool) (instance.CreateConfig, error) {
	cfg := instance.CreateConfig{
		Branch: branch,
		Image:  image,
	}

	// Always check if devcontainer CLI is available and warn if not
	devcontainerAvailable := isDevcontainerCLIAvailable()
	if !devcontainerAvailable {
		fmt.Println("Warning: devcontainer CLI not found in PATH")
		fmt.Println("  Install with: npm install -g @devcontainers/cli")
		fmt.Println("  See: https://github.com/devcontainers/cli")
	}

	// If image was explicitly passed, use vanilla mode
	if imageExplicit {
		return cfg, nil
	}

	// Check for devcontainer.json
	hasDevcontainer := devcontainer.HasConfig(repoPath)

	if hasDevcontainer {
		if !devcontainerAvailable {
			// Devcontainer exists but CLI not available - error
			return cfg, errors.New("devcontainer.json found but devcontainer CLI is not installed")
		}

		// Create devcontainer runtime wrapping the underlying runtime
		runtimeName := runtimeNameDocker
		if appCfg := ConfigFromContext(cmd.Context()); appCfg != nil && appCfg.Runtime.Name != "" {
			runtimeName = appCfg.Runtime.Name
		}
		dcRuntime := createDevcontainerRuntime(cmd, runtimeName)
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
	if image == "" {
		return cfg, errors.New("no devcontainer.json found and no image configured\n\nTo fix this, either:\n  1. Add a devcontainer.json to your repository\n  2. Use --image to specify a container image\n  3. Set default.base_image in your config")
	}

	return cfg, nil
}

// isDevcontainerCLIAvailable checks if the devcontainer CLI is in PATH.
func isDevcontainerCLIAvailable() bool {
	_, err := exec.LookPath(devcontainerCLI)
	return err == nil
}

// createDevcontainerRuntime creates a DevcontainerRuntime wrapping the appropriate underlying runtime.
func createDevcontainerRuntime(cmd *cobra.Command, runtimeName string) container.Runtime {
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
		"devcontainer", // CLI path - assumes it's in PATH
		dockerPath,
	)
}

// resolveAgent resolves the agent name, handling the default sentinel.
func resolveAgent(cmd *cobra.Command, agent string) (string, error) {
	if agent == agentDefaultSentinel {
		cfg := ConfigFromContext(cmd.Context())
		if cfg != nil && cfg.Default.Agent != "" {
			return cfg.Default.Agent, nil
		}
		return "", errors.New("--agent specified without value but no default.agent configured")
	}

	// Validate agent name
	if !config.IsValidAgent(agent) {
		return "", fmt.Errorf("invalid agent %q (valid: %s)", agent, formatList(config.ValidAgentNames()))
	}

	return agent, nil
}

// buildAgentCommand builds the command for launching an agent.
func buildAgentCommand(agent string, args []string) []string {
	cmd := []string{agent}
	if len(args) > 1 {
		cmd = append(cmd, args[1])
	}
	return cmd
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String("agent", "", "start an agent (claude, gemini, codex, or 'default' for configured default)")
	runCmd.Flags().String("name", "", "override auto-generated session name")
	runCmd.Flags().String("image", "", "use a container image instead of devcontainer")
	runCmd.Flags().BoolP("detached", "d", false, "create session but don't attach (run in background)")

	agentFlag := runCmd.Flags().Lookup("agent")
	if agentFlag != nil {
		agentFlag.NoOptDefVal = agentDefaultSentinel
	}
}
