package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jmgilman/headjack/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "View and modify configuration",
	Long: `View and modify Headjack configuration.

With no arguments, displays all configuration.
With one argument, displays the value for the specified key.
With two arguments, sets the value for the specified key.`,
	Example: `  # Show all config
  headjack config

  # Show value for a specific key
  headjack config default.agent

  # Set a value
  headjack config default.agent claude

  # Open config file in editor
  headjack config --edit`,
	Args:              cobra.RangeArgs(0, 2),
	PersistentPreRunE: nil, // Override parent - config command doesn't need manager
	RunE: func(cmd *cobra.Command, args []string) error {
		editFlag, err := cmd.Flags().GetBool("edit")
		if err != nil {
			return fmt.Errorf("get edit flag: %w", err)
		}
		if editFlag {
			return runEdit(cmd.Context())
		}

		loader, err := config.NewLoader()
		if err != nil {
			return fmt.Errorf("init config loader: %w", err)
		}

		switch len(args) {
		case 0:
			return runShowAll(loader)
		case 1:
			return runShowKey(loader, args[0])
		case 2:
			return runSetKey(loader, args[0], args[1])
		}

		return nil
	},
}

func runEdit(ctx context.Context) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return config.ErrNoEditor
	}

	loader, err := config.NewLoader()
	if err != nil {
		return fmt.Errorf("init config loader: %w", err)
	}

	// Ensure config exists (Load creates it if missing)
	if _, err := loader.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// G204: Subprocess launched with variable is intentional here - the editor
	// command comes from the user's $EDITOR environment variable, which is the
	// standard Unix convention for specifying the preferred text editor.
	editorCmd := exec.CommandContext(ctx, editor, loader.Path()) //nolint:gosec
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	return editorCmd.Run()
}

func runShowAll(loader *config.Loader) error {
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	fmt.Print(string(out))
	return nil
}

func runShowKey(loader *config.Loader, key string) error {
	if err := config.ValidateKey(key); err != nil {
		return err
	}

	// Load to ensure file exists
	if _, err := loader.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	value, err := loader.Get(key)
	if err != nil {
		return err
	}

	if value == nil {
		fmt.Println("")
		return nil
	}

	switch v := value.(type) {
	case string:
		fmt.Println(v)
	case map[string]any, []any:
		out, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal value: %w", err)
		}
		fmt.Print(string(out))
	default:
		fmt.Println(value)
	}

	return nil
}

func runSetKey(loader *config.Loader, key, value string) error {
	// Load first to ensure file exists
	if _, err := loader.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := loader.Set(key, value); err != nil {
		return err
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().Bool("edit", false, "open config file in $EDITOR")
}
