package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/auth"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/keychain"
)

// agentDefaultSentinel is the sentinel value used when --agent flag is specified without a value.
const agentDefaultSentinel = "default"

var runCmd = &cobra.Command{
	Use:   "run <branch> [prompt]",
	Short: "Create a new session (and instance if needed), then attach",
	Long: `Create a new session within an instance for the specified branch.

If no instance exists for the branch, one is created first:
  - Creates a git worktree at the configured location
  - Spawns a new container with the worktree mounted

A new session is always created within the instance. If --agent is specified,
the agent is started (with an optional prompt). Otherwise, the default shell
is started.

Unless --detached is specified, the terminal attaches to the session.
All session output is captured to a log file regardless of attached/detached mode.`,
	Example: `  # New instance with shell session
  headjack run feat/auth

  # New instance with Claude agent
  headjack run feat/auth --agent claude "Implement JWT authentication"

  # Additional session in existing instance
  headjack run feat/auth --agent gemini --name gemini-experiment

  # Shell session with custom name
  headjack run feat/auth --name debug-shell

  # Detached sessions (run in background)
  headjack run feat/auth --agent claude -d "Refactor the auth module"
  headjack run feat/auth --agent claude -d "Write tests for auth module"

  # Use a custom base image
  headjack run feat/auth --base my-registry.io/custom-image:latest`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRunCmd,
}

// runFlags holds parsed flags for the run command.
type runFlags struct {
	image       string
	agent       string
	sessionName string
	detached    bool
}

// parseRunFlags extracts and validates flags from the command.
func parseRunFlags(cmd *cobra.Command) (*runFlags, error) {
	image, err := cmd.Flags().GetString("base")
	if err != nil {
		return nil, fmt.Errorf("get base flag: %w", err)
	}
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

	// Use default image from config if not specified
	if image == "" {
		if cfg := ConfigFromContext(cmd.Context()); cfg != nil {
			image = cfg.Default.BaseImage
		}
	}

	return &runFlags{
		image:       image,
		agent:       agent,
		sessionName: sessionName,
		detached:    detached,
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

	// Inject authentication tokens from keychain
	if err := injectAuthToken(agent, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// injectAuthToken retrieves the auth token for the agent and adds it to the session config.
func injectAuthToken(agent string, cfg *instance.CreateSessionConfig) error {
	storage := keychain.New()

	switch agent {
	case "claude":
		provider := auth.NewClaudeProvider()
		token, err := provider.Get(storage)
		if err != nil {
			if errors.Is(err, keychain.ErrNotFound) {
				return errors.New("claude auth not configured: run 'headjack auth claude' first")
			}
			return fmt.Errorf("get claude token: %w", err)
		}
		cfg.Env = append(cfg.Env, "CLAUDE_CODE_OAUTH_TOKEN="+token)

	case "gemini":
		provider := auth.NewGeminiProvider()
		creds, err := provider.Get(storage)
		if err != nil {
			if errors.Is(err, keychain.ErrNotFound) {
				return errors.New("gemini auth not configured: run 'headjack auth gemini' first")
			}
			return fmt.Errorf("get gemini credentials: %w", err)
		}
		cfg.Env = append(cfg.Env, "GEMINI_OAUTH_CREDS="+creds)

	case "codex":
		provider := auth.NewCodexProvider()
		creds, err := provider.Get(storage)
		if err != nil {
			if errors.Is(err, keychain.ErrNotFound) {
				return errors.New("codex auth not configured: run 'headjack auth codex' first")
			}
			return fmt.Errorf("get codex credentials: %w", err)
		}
		cfg.Env = append(cfg.Env, "CODEX_AUTH_JSON="+creds)
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

	inst, err := getOrCreateInstance(cmd, mgr, repoPath, branch, flags.image)
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
		return fmt.Errorf("create session: %w", err)
	}

	if flags.detached {
		fmt.Printf("Created session %s in instance %s (detached)\n", session.Name, inst.ID)
		return nil
	}

	return mgr.AttachSession(cmd.Context(), inst.ID, session.Name)
}

// getOrCreateInstance retrieves an existing instance or creates a new one.
// If the instance exists but is stopped, it restarts the container.
func getOrCreateInstance(cmd *cobra.Command, mgr *instance.Manager, repoPath, branch, image string) (*instance.Instance, error) {
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

	// Create new instance
	inst, err = mgr.Create(cmd.Context(), repoPath, instance.CreateConfig{
		Branch: branch,
		Image:  image,
	})
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	fmt.Printf("Created instance %s for branch %s\n", inst.ID, inst.Branch)
	return inst, nil
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
	cfg := ConfigFromContext(cmd.Context())
	if cfg == nil || !cfg.IsValidAgent(agent) {
		return "", fmt.Errorf("invalid agent %q (valid: claude, gemini, codex)", agent)
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
	runCmd.Flags().String("base", "", "override the default base image")
	runCmd.Flags().BoolP("detached", "d", false, "create session but don't attach (run in background)")
}
