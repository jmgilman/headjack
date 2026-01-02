package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/auth"
	"github.com/jmgilman/headjack/internal/keychain"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Configure authentication for agent CLIs",
	Long: `Configure authentication for supported agent CLIs.

Runs the agent-specific authentication flow and stores credentials
securely in the macOS Keychain.`,
}

var authClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Configure Claude Code authentication",
	Long: `Configure Claude Code authentication for use in Headjack containers.

This command runs the Claude setup-token flow which:
1. Displays a URL to open in your browser
2. Prompts you to log in with your Anthropic account
3. Presents a code to enter back in the terminal
4. Stores the resulting OAuth token securely in macOS Keychain

The stored token uses your Claude Pro/Max subscription rather than API billing.`,
	Example: `  # Set up Claude Code authentication
  headjack auth claude`,
	RunE: runAuthClaude,
}

var authGeminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Configure Gemini CLI authentication",
	Long: `Configure Gemini CLI authentication for use in Headjack containers.

This command reads existing Gemini CLI credentials and stores them securely
in the macOS Keychain. You must first authenticate with Gemini CLI by running
'gemini' and completing the Google OAuth login flow.

The stored credentials use your Google AI Pro/Ultra subscription rather than API billing.`,
	Example: `  # First, authenticate with Gemini CLI (if not already done)
  gemini
  # Then, store credentials in Headjack
  headjack auth gemini`,
	RunE: runAuthGemini,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authClaudeCmd)
	authCmd.AddCommand(authGeminiCmd)
}

func runAuthClaude(cmd *cobra.Command, _ []string) error {
	fmt.Println("Starting Claude authentication flow...")
	fmt.Println()

	provider := auth.NewClaudeProvider()
	storage := keychain.New()

	if err := provider.Authenticate(cmd.Context(), storage); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println()
	fmt.Println("Authentication successful! Token stored in macOS Keychain.")
	return nil
}

func runAuthGemini(cmd *cobra.Command, _ []string) error {
	fmt.Println("Reading Gemini CLI credentials...")

	provider := auth.NewGeminiProvider()
	storage := keychain.New()

	if err := provider.Authenticate(cmd.Context(), storage); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	fmt.Println("Credentials stored in macOS Keychain.")
	return nil
}
