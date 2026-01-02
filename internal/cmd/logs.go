package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/jmgilman/headjack/internal/logging"
)

// Default poll interval for following logs.
const defaultLogPollInterval = 100 * time.Millisecond

var logsCmd = &cobra.Command{
	Use:   "logs <branch> <session>",
	Short: "View output from a session",
	Long: `View output from a session without attaching.

Reads from the session's log file, useful for checking on detached agents
without interrupting them.`,
	Example: `  # View recent output (last 100 lines)
  headjack logs feat/auth happy-panda

  # Follow output in real-time
  headjack logs feat/auth happy-panda -f

  # Show last 500 lines
  headjack logs feat/auth happy-panda -n 500

  # Show entire log from session start
  headjack logs feat/auth happy-panda --full`,
	Args: cobra.ExactArgs(2),
	RunE: runLogsCmd,
}

func runLogsCmd(cmd *cobra.Command, args []string) error {
	branch := args[0]
	sessionName := args[1]

	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return fmt.Errorf("get follow flag: %w", err)
	}

	lines, err := cmd.Flags().GetInt("lines")
	if err != nil {
		return fmt.Errorf("get lines flag: %w", err)
	}

	full, err := cmd.Flags().GetBool("full")
	if err != nil {
		return fmt.Errorf("get full flag: %w", err)
	}

	// Get the instance for this branch
	mgr, err := requireManager(cmd.Context())
	if err != nil {
		return err
	}

	inst, err := getInstanceByBranch(cmd.Context(), mgr, branch, "")
	if err != nil {
		return err
	}

	// Get the session to verify it exists and get its ID
	session, err := mgr.GetSession(cmd.Context(), inst.ID, sessionName)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	// Create log reader
	logsDir, err := getLogsDir(cmd.Context())
	if err != nil {
		return fmt.Errorf("get logs directory: %w", err)
	}
	pathMgr := logging.NewPathManager(logsDir)
	reader := logging.NewReader(pathMgr)

	// Check if log file exists
	if !pathMgr.LogExists(inst.ID, session.ID) {
		return fmt.Errorf("no log file found for session %s", sessionName)
	}

	return outputLogs(cmd.Context(), reader, inst.ID, session.ID, follow, lines, full)
}

func outputLogs(ctx context.Context, reader *logging.Reader, instanceID, sessionID string, follow bool, lines int, full bool) error {
	if follow {
		// Follow mode: show last N lines then stream new output
		return reader.FollowWithHistory(ctx, instanceID, sessionID, os.Stdout, lines, defaultLogPollInterval)
	}

	// Read mode: show lines and exit
	var logLines []string
	var err error

	if full {
		logLines, err = reader.ReadAll(instanceID, sessionID)
	} else {
		logLines, err = reader.ReadLastN(instanceID, sessionID, lines)
	}

	if err != nil {
		return fmt.Errorf("read log: %w", err)
	}

	for _, line := range logLines {
		fmt.Println(line)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "follow log output in real-time")
	logsCmd.Flags().IntP("lines", "n", logging.DefaultTailLines, "number of lines to show")
	logsCmd.Flags().Bool("full", false, "show entire log from session start")
}

// getLogsDir returns the logs directory from config, or the default if config is nil.
func getLogsDir(ctx context.Context) (string, error) {
	cfg := ConfigFromContext(ctx)
	if cfg != nil {
		return cfg.Storage.Logs, nil
	}

	// Fallback to default
	dataDir, err := defaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "logs"), nil
}
