package cmd

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth <agent>",
	Short: "Configure authentication for an agent CLI",
	Long: `Configure authentication for a supported agent CLI.

Runs the agent-specific authentication flow and stores credentials
securely in the macOS Keychain.

Supported agents:
- claude   Claude Code (Anthropic)
- gemini   Gemini CLI (Google)
- codex    Codex CLI (OpenAI)`,
	Example: `  # Set up Claude Code authentication
  headjack auth claude

  # Set up Gemini CLI authentication
  headjack auth gemini`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"claude", "gemini", "codex"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Placeholder - no implementation yet
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
