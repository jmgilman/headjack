package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/auth"
	"github.com/jmgilman/headjack/internal/keychain"
	"github.com/jmgilman/headjack/internal/prompt"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Configure authentication for agent CLIs",
	Long: `Configure authentication for supported agent CLIs.

Prompts for authentication method (subscription or API key) and stores
credentials securely in the system keychain.`,
}

var authClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Configure Claude Code authentication",
	Long: `Configure Claude Code authentication for use in Headjack containers.

Choose between:
  1. Subscription: Uses your Claude Pro/Max subscription via OAuth token
  2. API Key: Uses an Anthropic API key for pay-per-use billing`,
	Example: `  # Set up Claude Code authentication
  headjack auth claude`,
	RunE: runAuthClaude,
}

var authGeminiCmd = &cobra.Command{
	Use:   "gemini",
	Short: "Configure Gemini CLI authentication",
	Long: `Configure Gemini CLI authentication for use in Headjack containers.

Choose between:
  1. Subscription: Uses your Google AI Pro/Ultra subscription via OAuth
  2. API Key: Uses a Google AI API key for pay-per-use billing`,
	Example: `  # Set up Gemini CLI authentication
  headjack auth gemini`,
	RunE: runAuthGemini,
}

var authCodexCmd = &cobra.Command{
	Use:   "codex",
	Short: "Configure OpenAI Codex CLI authentication",
	Long: `Configure OpenAI Codex CLI authentication for use in Headjack containers.

Choose between:
  1. Subscription: Uses your ChatGPT Plus/Pro/Team subscription via OAuth
  2. API Key: Uses an OpenAI API key for pay-per-use billing`,
	Example: `  # Set up Codex CLI authentication
  headjack auth codex`,
	RunE: runAuthCodex,
}

var authStatusFlag bool

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authClaudeCmd)
	authCmd.AddCommand(authGeminiCmd)
	authCmd.AddCommand(authCodexCmd)

	// Add --status flag to all auth subcommands
	for _, cmd := range []*cobra.Command{authClaudeCmd, authGeminiCmd, authCodexCmd} {
		cmd.Flags().BoolVar(&authStatusFlag, "status", false, "Show current authentication status")
	}
}

func runAuthClaude(_ *cobra.Command, _ []string) error {
	return runAuth(auth.NewClaudeProvider())
}

func runAuthGemini(_ *cobra.Command, _ []string) error {
	return runAuth(auth.NewGeminiProvider())
}

func runAuthCodex(_ *cobra.Command, _ []string) error {
	return runAuth(auth.NewCodexProvider())
}

// runAuth handles both --status checks and interactive auth flows.
func runAuth(provider auth.Provider) error {
	if authStatusFlag {
		return showAuthStatus(provider)
	}
	return runAuthFlow(provider)
}

// showAuthStatus displays the current authentication status for a provider.
func showAuthStatus(provider auth.Provider) error {
	storage, err := keychain.New()
	if err != nil {
		return fmt.Errorf("initialize credential storage: %w", err)
	}

	info := provider.Info()
	cred, err := provider.Load(storage)
	if errors.Is(err, keychain.ErrNotFound) {
		fmt.Printf("%s: not configured\n", info.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("load credential: %w", err)
	}

	switch cred.Type {
	case auth.CredentialTypeSubscription:
		fmt.Printf("%s: subscription\n", info.Name)
	case auth.CredentialTypeAPIKey:
		fmt.Printf("%s: api key\n", info.Name)
	default:
		fmt.Printf("%s: configured (unknown type)\n", info.Name)
	}

	return nil
}

// runAuthFlow runs the interactive authentication flow for a provider.
func runAuthFlow(provider auth.Provider) error {
	storage, err := keychain.New()
	if err != nil {
		return fmt.Errorf("initialize credential storage: %w", err)
	}

	prompter := prompt.New()
	info := provider.Info()

	prompter.Print(fmt.Sprintf("Configure %s authentication", info.Name))
	prompter.Print("")

	choice, err := prompter.Choice("Authentication method:", []string{
		"Subscription",
		"API Key",
	})
	if err != nil {
		return fmt.Errorf("select auth method: %w", err)
	}

	prompter.Print("")

	var cred auth.Credential

	switch choice {
	case 0: // Subscription
		cred, err = handleSubscriptionAuth(provider, prompter)
	case 1: // API Key
		cred, err = handleAPIKeyAuth(provider, prompter)
	}

	if err != nil {
		return err
	}

	if err := provider.Store(storage, cred); err != nil {
		return fmt.Errorf("store credential: %w", err)
	}

	prompter.Print("")
	prompter.Print("Credentials stored securely.")
	return nil
}

// handleSubscriptionAuth handles subscription-based authentication.
// For Claude, prompts for manual token entry.
// For Gemini/Codex, attempts to read existing credentials from config files.
func handleSubscriptionAuth(provider auth.Provider, prompter prompt.Prompter) (auth.Credential, error) {
	// Try to auto-detect existing credentials
	value, err := provider.CheckSubscription()
	if err == nil {
		// Found existing credentials
		prompter.Print("Found existing subscription credentials.")
		if validateErr := provider.ValidateSubscription(value); validateErr != nil {
			return auth.Credential{}, fmt.Errorf("invalid credentials: %w", validateErr)
		}
		return auth.Credential{
			Type:  auth.CredentialTypeSubscription,
			Value: value,
		}, nil
	}

	// No existing credentials - show instructions and prompt for manual entry
	prompter.Print(err.Error())
	prompter.Print("")

	value, err = prompter.Secret("Paste your credential: ")
	if err != nil {
		return auth.Credential{}, fmt.Errorf("read credential: %w", err)
	}

	if err := provider.ValidateSubscription(value); err != nil {
		return auth.Credential{}, fmt.Errorf("invalid credential: %w", err)
	}

	return auth.Credential{
		Type:  auth.CredentialTypeSubscription,
		Value: value,
	}, nil
}

// handleAPIKeyAuth handles API key authentication.
func handleAPIKeyAuth(provider auth.Provider, prompter prompt.Prompter) (auth.Credential, error) {
	info := provider.Info()

	prompter.Print(fmt.Sprintf("Enter your %s API key.", info.Name))
	prompter.Print("")

	value, err := prompter.Secret("API key: ")
	if err != nil {
		return auth.Credential{}, fmt.Errorf("read API key: %w", err)
	}

	if err := provider.ValidateAPIKey(value); err != nil {
		return auth.Credential{}, fmt.Errorf("invalid API key: %w", err)
	}

	return auth.Credential{
		Type:  auth.CredentialTypeAPIKey,
		Value: value,
	}, nil
}
