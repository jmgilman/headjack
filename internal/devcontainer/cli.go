package devcontainer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmgilman/headjack/internal/config"
	"github.com/jmgilman/headjack/internal/exec"
	"github.com/jmgilman/headjack/internal/prompt"
)

const (
	// devcontainerBin is the name of the devcontainer CLI binary.
	devcontainerBin = "devcontainer"

	// npmBin is the name of the npm binary.
	npmBin = "npm"

	// devcontainerPackage is the npm package name for the devcontainer CLI.
	devcontainerPackage = "@devcontainers/cli"

	// cliInstallDir is the subdirectory under XDG data dir for CLI installation.
	cliInstallDir = "devcontainer-cli"
)

// CLIResolver resolves the path to the devcontainer CLI, offering to install
// it locally if not found in PATH or config.
type CLIResolver struct {
	loader   *config.Loader
	prompter prompt.Prompter
	executor exec.Executor
	homeDir  string
}

// NewCLIResolver creates a new CLIResolver.
func NewCLIResolver(loader *config.Loader, prompter prompt.Prompter, executor exec.Executor) *CLIResolver {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fall back to empty string; installDir will fail gracefully
		homeDir = ""
	}
	return &CLIResolver{
		loader:   loader,
		prompter: prompter,
		executor: executor,
		homeDir:  homeDir,
	}
}

// Resolve returns the path to the devcontainer CLI binary.
// It checks in order:
//  1. devcontainer in PATH
//  2. devcontainer.path from config (if set and binary exists)
//  3. Offers to install via npm if available
//
// Returns an error if the CLI cannot be found or installed.
func (r *CLIResolver) Resolve(ctx context.Context) (string, error) {
	// 1. Check if devcontainer is in PATH
	if path, err := r.executor.LookPath(devcontainerBin); err == nil {
		return path, nil
	}

	// 2. Check if devcontainer.path is set in config
	cfg, err := r.loader.Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	if cfg.Devcontainer.Path != "" {
		if _, statErr := os.Stat(cfg.Devcontainer.Path); statErr == nil {
			return cfg.Devcontainer.Path, nil
		}
		// Path is set but binary doesn't exist - fall through to install
	}

	// 3. Check if npm is available
	if _, lookErr := r.executor.LookPath(npmBin); lookErr != nil {
		return "", r.noNpmError()
	}

	// 4. Prompt user to install
	confirmed, err := r.prompter.Confirm(
		"Install devcontainer CLI?",
		"The devcontainer CLI is required to use devcontainer.json configurations.\nWould you like to install it locally via npm?",
	)
	if err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}

	if !confirmed {
		return "", r.declinedError()
	}

	// 5. Install the CLI
	path, err := r.install(ctx)
	if err != nil {
		return "", err
	}

	// 6. Save path to config
	if err := r.loader.Set("devcontainer.path", path); err != nil {
		// Non-fatal: CLI is installed but config wasn't saved
		r.prompter.Print(fmt.Sprintf("Warning: could not save config: %v", err))
	}

	return path, nil
}

// install installs the devcontainer CLI via npm and returns the path to the binary.
func (r *CLIResolver) install(ctx context.Context) (string, error) {
	installDir := r.installDir()
	binPath := filepath.Join(installDir, "node_modules", ".bin", devcontainerBin)

	// Track success for cleanup
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(installDir)
		}
	}()

	// Create install directory
	if err := os.MkdirAll(installDir, 0o750); err != nil {
		return "", fmt.Errorf("create install directory: %w", err)
	}

	r.prompter.Print("Installing devcontainer CLI...")

	// Run npm install
	result, err := r.executor.Run(ctx, &exec.RunOptions{
		Name: npmBin,
		Args: []string{"install", "--prefix", installDir, devcontainerPackage},
	})
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = strings.TrimSpace(string(result.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("npm install failed: %s", stderr)
		}
		return "", fmt.Errorf("npm install failed: %w", err)
	}

	// Validate the installation
	if err := r.validate(ctx, binPath); err != nil {
		return "", fmt.Errorf("validate installation: %w", err)
	}

	r.prompter.Print("devcontainer CLI installed successfully.")

	success = true
	return binPath, nil
}

// validate verifies that the devcontainer CLI is working by running --version.
func (r *CLIResolver) validate(ctx context.Context, cliPath string) error {
	result, err := r.executor.Run(ctx, &exec.RunOptions{
		Name: cliPath,
		Args: []string{"--version"},
	})
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = strings.TrimSpace(string(result.Stderr))
		}
		if stderr != "" {
			return fmt.Errorf("devcontainer --version failed: %s", stderr)
		}
		return fmt.Errorf("devcontainer --version failed: %w", err)
	}
	return nil
}

// installDir returns the directory where the CLI should be installed.
func (r *CLIResolver) installDir() string {
	return filepath.Join(r.homeDir, ".local", "share", "headjack", cliInstallDir)
}

// noNpmError returns an error with instructions when npm is not available.
func (r *CLIResolver) noNpmError() error {
	return errors.New(`devcontainer CLI not found

The devcontainer CLI is required to use devcontainer.json configurations.

To install:
  npm install -g @devcontainers/cli

Or install Node.js first:
  https://nodejs.org/

See: https://github.com/devcontainers/cli`)
}

// declinedError returns an error with instructions when user declines installation.
func (r *CLIResolver) declinedError() error {
	return errors.New(`devcontainer CLI not found

To install manually:
  npm install -g @devcontainers/cli

Or use a container image instead:
  hjk run <branch> --image <image>`)
}
