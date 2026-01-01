package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/instance"
)

var resumeCmd = &cobra.Command{
	Use:   "resume <branch> [prompt]",
	Short: "Attach to an existing instance",
	Long: `Attach to an existing instance for the specified branch.

If no container is running for the instance, a new container is started.
Multiple resume calls to the same instance are valid, allowing multiple
shell sessions or agents to run concurrently within a single instance.`,
	Example: `  # Resume into shell
  headjack resume feat/auth

  # Resume and start an agent
  headjack resume feat/auth --agent claude "Continue implementing the login flow"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]

		// Get current working directory as repo path
		repoPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		// Get instance by branch
		mgr := ManagerFromContext(cmd.Context())
		inst, err := mgr.GetByBranch(cmd.Context(), repoPath, branch)
		if err != nil {
			if errors.Is(err, instance.ErrNotFound) {
				return fmt.Errorf("no instance found for branch %q (use 'new' to create one)", branch)
			}
			return fmt.Errorf("get instance: %w", err)
		}

		fmt.Printf("Resuming instance %s for branch %s\n", inst.ID, inst.Branch)

		// Build attach config
		agent, err := cmd.Flags().GetString("agent")
		if err != nil {
			return fmt.Errorf("get agent flag: %w", err)
		}
		attachCfg := instance.AttachConfig{
			Interactive: true,
			Workdir:     "/workspace",
		}

		if agent != "" {
			// Resolve "default" sentinel to config value
			if agent == agentDefaultSentinel {
				if cfg := ConfigFromContext(cmd.Context()); cfg != nil && cfg.Default.Agent != "" {
					agent = cfg.Default.Agent
				} else {
					return errors.New("--agent specified without value but no default.agent configured")
				}
			}

			// Validate agent name
			cfg := ConfigFromContext(cmd.Context())
			if cfg == nil || !cfg.IsValidAgent(agent) {
				return fmt.Errorf("invalid agent %q (valid: claude, gemini, codex)", agent)
			}

			attachCfg.Command = buildAgentCommand(agent, args)

			// Inject agent-specific environment variables from config
			if loader := LoaderFromContext(cmd.Context()); loader != nil {
				for k, v := range loader.GetAgentEnv(agent) {
					attachCfg.Env = append(attachCfg.Env, k+"="+v)
				}
			}
		}

		// Attach to instance
		return mgr.Attach(cmd.Context(), inst.ID, attachCfg)
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)

	resumeCmd.Flags().String("agent", "", "start the specified agent instead of dropping into a shell")
	// Set NoOptDefVal so --agent without a value uses "default" as sentinel
	resumeCmd.Flags().Lookup("agent").NoOptDefVal = agentDefaultSentinel
}
