package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/jmgilman/headjack/internal/instance"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <branch> [prompt]",
	Short: "Create a new instance from the current repository",
	Long: `Create a new instance with a git worktree for the specified branch.

If the branch already has a managed worktree, an error is returned (use 'resume' instead).
If the branch exists in the repo, creates a worktree from the existing branch.
If the branch does not exist, creates a worktree with a new branch from HEAD.`,
	Example: `  # Drop into shell in new instance
  headjack new feat/auth

  # Start Claude Code with a prompt
  headjack new feat/auth --agent claude "Implement JWT authentication"

  # Start agent without initial prompt
  headjack new feat/auth --agent claude

  # Use a custom base image
  headjack new feat/auth --base my-registry.io/custom-image:latest`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]

		// Get current working directory as repo path
		repoPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		// Determine image to use (precedence: flag > config)
		// Config already has defaults set via Viper, so just use it
		image, _ := cmd.Flags().GetString("base")
		if image == "" {
			if cfg := ConfigFromContext(cmd.Context()); cfg != nil {
				image = cfg.Default.BaseImage
			}
		}

		// Create instance
		mgr := ManagerFromContext(cmd.Context())
		inst, err := mgr.Create(cmd.Context(), repoPath, instance.CreateConfig{
			Branch: branch,
			Image:  image,
		})
		if err != nil {
			if errors.Is(err, instance.ErrAlreadyExists) {
				return fmt.Errorf("instance already exists for branch %q (use 'resume' instead)", branch)
			}
			return fmt.Errorf("create instance: %w", err)
		}

		fmt.Printf("Created instance %s for branch %s\n", inst.ID, inst.Branch)

		// Build attach config
		agent, _ := cmd.Flags().GetString("agent")
		attachCfg := instance.AttachConfig{
			Interactive: true,
			Workdir:     "/workspace",
		}

		if agent != "" {
			// Resolve "default" sentinel to config value
			if agent == "default" {
				if cfg := ConfigFromContext(cmd.Context()); cfg != nil && cfg.Default.Agent != "" {
					agent = cfg.Default.Agent
				} else {
					return fmt.Errorf("--agent specified without value but no default.agent configured")
				}
			}

			// Validate agent name
			cfg := ConfigFromContext(cmd.Context())
			if cfg == nil || !cfg.IsValidAgent(agent) {
				return fmt.Errorf("invalid agent %q (valid: claude, gemini, codex)", agent)
			}

			// Build agent command
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

// buildAgentCommand builds the command for launching an agent.
func buildAgentCommand(agent string, args []string) []string {
	var cmd []string

	switch agent {
	case "claude":
		cmd = []string{"claude"}
		if len(args) > 1 {
			cmd = append(cmd, args[1])
		}
	case "gemini":
		cmd = []string{"gemini"}
		if len(args) > 1 {
			cmd = append(cmd, args[1])
		}
	case "codex":
		cmd = []string{"codex"}
		if len(args) > 1 {
			cmd = append(cmd, args[1])
		}
	default:
		// Unknown agent, just use as command name
		cmd = []string{agent}
		if len(args) > 1 {
			cmd = append(cmd, args[1])
		}
	}

	return cmd
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().String("agent", "", "start the specified agent instead of dropping into a shell")
	// Set NoOptDefVal so --agent without a value uses "default" as sentinel
	newCmd.Flags().Lookup("agent").NoOptDefVal = "default"
	newCmd.Flags().String("base", "", "override the default base image")
}
