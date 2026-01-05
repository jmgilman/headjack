package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/auth"
	"github.com/jmgilman/headjack/internal/config"
	"github.com/jmgilman/headjack/internal/instance"
	"github.com/jmgilman/headjack/internal/keychain"
	"github.com/jmgilman/headjack/internal/slogger"
)

var agentCmd = &cobra.Command{
	Use:   "agent <branch> [agent_name] [-- <agent-flags>...]",
	Short: "Start an agent session in an existing instance",
	Long: `Start an agent session within an existing instance for the specified branch.

The instance must already exist (created with 'hjk run'). This command creates
a new session running the specified agent (claude, gemini, or codex) and attaches
to it unless --detached is specified.

If agent_name is not specified, the default agent from configuration is used.
Set the default with 'hjk config default.agent <agent_name>'.

Additional flags can be passed to the agent CLI by placing them after a -- separator.

All session output is captured to a log file regardless of attached/detached mode.`,
	Example: `  # Start default agent on existing instance
  hjk agent feat/auth

  # Start Claude agent explicitly
  hjk agent feat/auth claude

  # Start agent with a prompt
  hjk agent feat/auth --prompt "Implement JWT authentication"
  hjk agent feat/auth claude --prompt "Implement JWT authentication"

  # Start Gemini agent with custom session name
  hjk agent feat/auth gemini --name auth-session

  # Start agent in detached mode (run in background)
  hjk agent feat/auth -d --prompt "Refactor the auth module"

  # Pass additional flags to the agent CLI
  hjk agent feat/auth claude -- --dangerously-skip-permissions`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgentCmd,
}

// agentFlags holds parsed flags for the agent command.
type agentFlags struct {
	sessionName string
	detached    bool
	prompt      string
	agentFlags  []string // flags to pass to the agent CLI (after --)
}

// parseAgentFlags extracts and validates flags from the command.
func parseAgentFlags(cmd *cobra.Command, args []string) (*agentFlags, error) {
	sessionName, err := cmd.Flags().GetString("name")
	if err != nil {
		return nil, fmt.Errorf("get name flag: %w", err)
	}
	detached, err := cmd.Flags().GetBool("detached")
	if err != nil {
		return nil, fmt.Errorf("get detached flag: %w", err)
	}
	prompt, err := cmd.Flags().GetString("prompt")
	if err != nil {
		return nil, fmt.Errorf("get prompt flag: %w", err)
	}

	return &agentFlags{
		sessionName: sessionName,
		detached:    detached,
		prompt:      prompt,
		agentFlags:  parsePassthroughArgs(cmd, args),
	}, nil
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

// buildAgentCommand builds the command for launching an agent.
// The command is: agent [prompt] [flags...]
func buildAgentCommand(agent, prompt string, flags []string) []string {
	cmd := []string{agent}
	if prompt != "" {
		cmd = append(cmd, prompt)
	}
	cmd = append(cmd, flags...)
	return cmd
}

// resolveAgentName determines the agent name from args or config default.
// dashIdx is the index of the -- separator (-1 if not present).
func resolveAgentName(ctx context.Context, args []string, dashIdx int) (string, error) {
	// Check if args[1] exists and is before the -- separator (or no separator)
	hasAgentArg := len(args) > 1 && (dashIdx < 0 || dashIdx > 1)
	if hasAgentArg {
		return args[1], nil
	}

	// Get default agent from config
	loader := LoaderFromContext(ctx)
	if loader == nil {
		return "", errors.New("no default agent configured and none specified\nhint: run 'hjk config default.agent <agent_name>' to set a default")
	}
	cfg, err := loader.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	if cfg.Default.Agent == "" {
		return "", errors.New("no default agent configured and none specified\nhint: run 'hjk config default.agent <agent_name>' to set a default")
	}
	return cfg.Default.Agent, nil
}

func runAgentCmd(cmd *cobra.Command, args []string) error {
	branch := args[0]

	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	flags, err := parseAgentFlags(cmd, args)
	if err != nil {
		return err
	}

	agentName, err := resolveAgentName(cmd.Context(), args, cmd.ArgsLenAtDash())
	if err != nil {
		return err
	}

	// Get existing instance (do NOT create)
	inst, err := getInstanceByBranch(cmd.Context(), mgr, branch)
	if err != nil {
		return err
	}

	// Validate agent name
	if !config.IsValidAgent(agentName) {
		return fmt.Errorf("invalid agent %q (valid: %s)", agentName, formatList(config.ValidAgentNames()))
	}

	// Merge config flags with CLI flags
	var configFlags []string
	if loader := LoaderFromContext(cmd.Context()); loader != nil {
		configFlags = loader.GetAgentFlags(agentName)
	}
	mergedFlags := mergeFlags(configFlags, flags.agentFlags)

	// Build session config
	sessionCfg := &instance.CreateSessionConfig{
		Type:    agentName,
		Name:    flags.sessionName,
		Command: buildAgentCommand(agentName, flags.prompt, mergedFlags),
	}

	// Inject agent-specific environment variables from config
	if loader := LoaderFromContext(cmd.Context()); loader != nil {
		for k, v := range loader.GetAgentEnv(agentName) {
			sessionCfg.Env = append(sessionCfg.Env, k+"="+v)
		}
	}

	// Inject authentication credentials from keychain
	if authErr := injectAuthCredential(agentName, sessionCfg); authErr != nil {
		return authErr
	}

	// Create session
	session, err := mgr.CreateSession(cmd.Context(), inst.ID, sessionCfg)
	if err != nil {
		if errors.Is(err, instance.ErrSessionExists) {
			return fmt.Errorf("session %q already exists in instance %s", flags.sessionName, inst.ID)
		}
		var notRunningErr *instance.NotRunningError
		if errors.As(err, &notRunningErr) {
			hint := formatAgentNotRunningHint(cmd, notRunningErr)
			if hint != "" {
				return fmt.Errorf("create session: %w\nhint: %s", err, hint)
			}
		}
		return fmt.Errorf("create session: %w", err)
	}

	if flags.detached {
		slogger.L(cmd.Context()).Info("created session", "session", session.Name, "instance", inst.ID, "mode", "detached")
		return nil
	}

	return mgr.AttachSession(cmd.Context(), inst.ID, session.Name)
}

func formatAgentNotRunningHint(cmd *cobra.Command, err *instance.NotRunningError) string {
	return formatNotRunningHint(cmd, err)
}

func init() {
	rootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringP("name", "n", "", "override auto-generated session name")
	agentCmd.Flags().BoolP("detached", "d", false, "create session but don't attach (run in background)")
	agentCmd.Flags().StringP("prompt", "p", "", "initial prompt to send to the agent")
}
