//go:build integration

// Package integration provides integration tests for the Headjack CLI using testscript.
package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain sets up the testscript environment.
func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"hjk": hjkMain,
	}))
}

// hjkMain wraps the hjk binary for testscript execution.
func hjkMain() int {
	binary := os.Getenv("HJK_BINARY")
	if binary == "" {
		// Try to find hjk in PATH
		var err error
		binary, err = exec.LookPath("hjk")
		if err != nil {
			fmt.Fprintf(os.Stderr, "hjk binary not found: set HJK_BINARY or add hjk to PATH\n")
			return 1
		}
	}

	cmd := exec.Command(binary, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}

// TestScripts runs all testscript files in testdata/scripts.
func TestScripts(t *testing.T) {
	// Determine runtime from environment or auto-detect
	runtimeName := os.Getenv("HEADJACK_TEST_RUNTIME")
	if runtimeName == "" {
		runtimeName = detectRuntime()
	}

	t.Logf("Using container runtime: %s", runtimeName)

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts",
		Setup: func(env *testscript.Env) error {
			return setupTestEnv(env, runtimeName)
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"cleanup_containers": cmdCleanupContainers,
			"wait_running":       cmdWaitRunning,
			"sleep":              cmdSleep,
		},
		Condition: func(cond string) (bool, error) {
			return evalCondition(cond, runtimeName)
		},
	})
}

// detectRuntime auto-detects the available container runtime.
func detectRuntime() string {
	if runtime.GOOS == "darwin" {
		// Prefer Apple Containerization on macOS
		if _, err := exec.LookPath("container"); err == nil {
			return "apple"
		}
		// Fall back to Docker
		if _, err := exec.LookPath("docker"); err == nil {
			return "docker"
		}
	}
	// Linux: prefer Docker for CI consistency
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker"
	}
	// Fall back to Podman
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman"
	}
	return "docker" // Default, will fail if not available
}

// setupTestEnv configures the test environment with isolated paths.
func setupTestEnv(env *testscript.Env, runtimeName string) error {
	// Create isolated directory structure
	testHome := filepath.Join(env.WorkDir, "home")
	configDir := filepath.Join(testHome, ".config", "headjack")
	dataDir := filepath.Join(testHome, ".local", "share", "headjack")

	for _, dir := range []string{
		configDir,
		filepath.Join(dataDir, "git"),
		filepath.Join(dataDir, "logs"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Set environment variables for isolation
	env.Setenv("HOME", testHome)
	env.Setenv("XDG_CONFIG_HOME", filepath.Join(testHome, ".config"))
	env.Setenv("XDG_DATA_HOME", filepath.Join(testHome, ".local", "share"))

	// Pass through HJK_BINARY if set, otherwise try to find hjk in PATH
	// This ensures wait_running and other custom commands can find the binary
	if binary := os.Getenv("HJK_BINARY"); binary != "" {
		env.Setenv("HJK_BINARY", binary)
	} else if binary, err := exec.LookPath("hjk"); err == nil {
		env.Setenv("HJK_BINARY", binary)
	}

	// Pass through DOCKER_HOST for rootless Docker (Lima)
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		env.Setenv("DOCKER_HOST", dockerHost)
	}

	// Create config file with test-appropriate settings
	configPath := filepath.Join(configDir, "config.yaml")
	configContent := fmt.Sprintf(`default:
  agent: ""
  base_image: ghcr.io/gilmanlab/headjack:base
storage:
  worktrees: %s/git
  catalog: %s/catalog.json
  logs: %s/logs
runtime:
  name: %s
  flags: {}
agents:
  claude:
    env: {}
  gemini:
    env: {}
  codex:
    env: {}
`, dataDir, dataDir, dataDir, runtimeName)

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Set runtime name in environment for conditions
	env.Setenv("HEADJACK_TEST_RUNTIME", runtimeName)

	return nil
}

// evalCondition evaluates custom conditions for testscript.
func evalCondition(cond string, runtimeName string) (bool, error) {
	switch cond {
	case "podman":
		return runtimeName == "podman", nil
	case "docker":
		return runtimeName == "docker", nil
	case "apple":
		return runtimeName == "apple", nil
	case "linux":
		return runtime.GOOS == "linux", nil
	case "darwin":
		return runtime.GOOS == "darwin", nil
	case "arm64":
		return runtime.GOARCH == "arm64", nil
	case "amd64":
		return runtime.GOARCH == "amd64", nil
	default:
		return false, fmt.Errorf("unknown condition: %s", cond)
	}
}

// cmdCleanupContainers removes any leftover test containers.
func cmdCleanupContainers(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("cleanup_containers does not support negation")
	}

	runtimeName := ts.Getenv("HEADJACK_TEST_RUNTIME")
	var cmd *exec.Cmd

	switch runtimeName {
	case "apple":
		// Apple container CLI cleanup
		cmd = exec.Command("sh", "-c", `container list --format json 2>/dev/null | jq -r '.[].configuration.id // empty' | grep '^hjk-' | while read id; do container stop "$id" 2>/dev/null; container rm "$id" 2>/dev/null; done`)
	case "docker":
		cmd = exec.Command("sh", "-c", `docker ps -a --format '{{.Names}}' 2>/dev/null | grep '^hjk-' | xargs -r docker rm -f 2>/dev/null`)
	default: // podman
		cmd = exec.Command("sh", "-c", `podman ps -a --format '{{.Names}}' 2>/dev/null | grep '^hjk-' | xargs -r podman rm -f 2>/dev/null`)
	}

	// Best-effort cleanup, ignore errors
	cmd.Run()
}

// cmdWaitRunning waits for an instance to be running.
func cmdWaitRunning(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) < 1 {
		ts.Fatalf("usage: wait_running <branch> [timeout_seconds]")
	}

	branch := args[0]
	timeout := 30 * time.Second
	if len(args) >= 2 {
		var secs int
		if _, err := fmt.Sscanf(args[1], "%d", &secs); err == nil {
			timeout = time.Duration(secs) * time.Second
		}
	}

	binary := ts.Getenv("HJK_BINARY")
	if binary == "" {
		// HJK_BINARY should be set by setupTestEnv, but fall back to PATH lookup
		var err error
		binary, err = exec.LookPath("hjk")
		if err != nil {
			ts.Fatalf("hjk binary not found: set HJK_BINARY or add hjk to PATH")
		}
	}

	// Get current working directory (tracks cd commands in script)
	workDir := ts.MkAbs(".")

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command(binary, "ps")
		// Build environment from key testscript variables
		cmd.Env = []string{
			"HOME=" + ts.Getenv("HOME"),
			"PATH=" + ts.Getenv("PATH"),
			"XDG_CONFIG_HOME=" + ts.Getenv("XDG_CONFIG_HOME"),
			"XDG_DATA_HOME=" + ts.Getenv("XDG_DATA_HOME"),
			"HJK_BINARY=" + ts.Getenv("HJK_BINARY"),
			"DOCKER_HOST=" + ts.Getenv("DOCKER_HOST"),
		}
		cmd.Dir = workDir
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), branch) && strings.Contains(string(output), "running") {
			if !neg {
				return // Found running instance
			}
			// neg: should NOT be running, keep waiting
		} else if neg {
			return // Not running, which is what we want with negation
		}
		time.Sleep(500 * time.Millisecond)
	}

	if neg {
		ts.Fatalf("instance %s is still running after %v", branch, timeout)
	} else {
		ts.Fatalf("instance %s not running after %v", branch, timeout)
	}
}

// cmdSleep pauses execution for the specified number of seconds.
func cmdSleep(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("sleep does not support negation")
	}
	if len(args) < 1 {
		ts.Fatalf("usage: sleep <seconds>")
	}

	var secs float64
	if _, err := fmt.Sscanf(args[0], "%f", &secs); err != nil {
		ts.Fatalf("invalid sleep duration: %s", args[0])
	}

	time.Sleep(time.Duration(secs * float64(time.Second)))
}
