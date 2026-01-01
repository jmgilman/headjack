// Package config provides configuration management for Headjack.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Default configuration values.
const (
	DefaultConfigDir  = ".config/headjack"
	DefaultConfigFile = "config.yaml"
	DefaultDataDir    = ".local/share/headjack"
)

// defaultBaseImage is the default container image (unexported).
// Available variants: :base (minimal), :systemd (+ init), :dind (+ Docker)
const defaultBaseImage = "ghcr.io/jmgilman/headjack:base"

// Sentinel errors for configuration operations.
var (
	ErrInvalidKey         = errors.New("invalid configuration key")
	ErrInvalidAgent       = errors.New("invalid agent name")
	ErrInvalidMultiplexer = errors.New("invalid multiplexer name")
	ErrInvalidRuntime     = errors.New("invalid runtime name")
	ErrNoEditor           = errors.New("$EDITOR environment variable not set")
)

// validAgents contains the allowed agent names (unexported).
var validAgents = map[string]bool{
	"claude": true,
	"gemini": true,
	"codex":  true,
}

// validMultiplexers contains the allowed multiplexer names (unexported).
var validMultiplexers = map[string]bool{
	"tmux":   true,
	"zellij": true,
}

// validRuntimes contains the allowed runtime names (unexported).
var validRuntimes = map[string]bool{
	"podman": true,
	"apple":  true,
}

// validKeys is built once from Config struct reflection.
var validKeys = buildValidKeys()

// validate is the shared validator instance.
var validate = validator.New()

// Config represents the full Headjack configuration.
type Config struct {
	Default DefaultConfig          `mapstructure:"default" validate:"required"`
	Agents  map[string]AgentConfig `mapstructure:"agents" validate:"dive,keys,oneof=claude gemini codex,endkeys"`
	Storage StorageConfig          `mapstructure:"storage" validate:"required"`
	Runtime RuntimeConfig          `mapstructure:"runtime"`
}

// DefaultConfig holds default values for new instances.
type DefaultConfig struct {
	Agent       string `mapstructure:"agent" validate:"omitempty,oneof=claude gemini codex"`
	BaseImage   string `mapstructure:"base_image" validate:"required"`
	Multiplexer string `mapstructure:"multiplexer" validate:"omitempty,oneof=tmux zellij"`
}

// AgentConfig holds agent-specific configuration.
type AgentConfig struct {
	Env map[string]string `mapstructure:"env"`
}

// StorageConfig holds storage location configuration.
type StorageConfig struct {
	Worktrees string `mapstructure:"worktrees" validate:"required"`
	Catalog   string `mapstructure:"catalog" validate:"required"`
	Logs      string `mapstructure:"logs" validate:"required"`
}

// RuntimeConfig holds container runtime configuration.
type RuntimeConfig struct {
	Name       string   `mapstructure:"name" validate:"omitempty,oneof=podman apple"`
	Privileged bool     `mapstructure:"privileged"`
	Flags      []string `mapstructure:"flags"`
}

// Validate checks the configuration for errors using struct tags.
func (c *Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	return nil
}

// IsValidAgent returns true if the agent name is valid.
func (c *Config) IsValidAgent(name string) bool {
	return validAgents[name]
}

// ValidAgentNames returns the list of valid agent names.
func (c *Config) ValidAgentNames() []string {
	return []string{"claude", "gemini", "codex"}
}

// IsValidMultiplexer returns true if the multiplexer name is valid.
func (c *Config) IsValidMultiplexer(name string) bool {
	return validMultiplexers[name]
}

// ValidMultiplexerNames returns the list of valid multiplexer names.
func (c *Config) ValidMultiplexerNames() []string {
	return []string{"tmux", "zellij"}
}

// IsValidRuntime returns true if the runtime name is valid.
func (c *Config) IsValidRuntime(name string) bool {
	return validRuntimes[name]
}

// ValidRuntimeNames returns the list of valid runtime names.
func (c *Config) ValidRuntimeNames() []string {
	return []string{"podman", "apple"}
}

// Loader provides configuration loading and saving.
type Loader struct {
	v       *viper.Viper
	path    string
	homeDir string
}

// NewLoader creates a new configuration loader.
func NewLoader() (*Loader, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	configPath := filepath.Join(home, DefaultConfigDir, DefaultConfigFile)

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Environment variable binding
	v.SetEnvPrefix("HEADJACK")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind specific env vars to config keys.
	// We intentionally ignore errors here as BindEnv only fails if called with zero arguments.
	//nolint:errcheck // BindEnv only fails with zero arguments
	v.BindEnv("default.agent", "HEADJACK_DEFAULT_AGENT")
	//nolint:errcheck // BindEnv only fails with zero arguments
	v.BindEnv("default.base_image", "HEADJACK_BASE_IMAGE")
	//nolint:errcheck // BindEnv only fails with zero arguments
	v.BindEnv("default.multiplexer", "HEADJACK_MULTIPLEXER")
	//nolint:errcheck // BindEnv only fails with zero arguments
	v.BindEnv("storage.worktrees", "HEADJACK_WORKTREE_DIR")

	l := &Loader{
		v:       v,
		path:    configPath,
		homeDir: home,
	}

	// Set defaults before any config reading
	l.setDefaults()

	return l, nil
}

// setDefaults sets all default configuration values using Viper.
func (l *Loader) setDefaults() {
	l.v.SetDefault("default.agent", "")
	l.v.SetDefault("default.base_image", defaultBaseImage)
	l.v.SetDefault("default.multiplexer", "tmux")
	l.v.SetDefault("storage.worktrees", "~/.local/share/headjack/git")
	l.v.SetDefault("storage.catalog", "~/.local/share/headjack/catalog.json")
	l.v.SetDefault("storage.logs", "~/.local/share/headjack/logs")
	l.v.SetDefault("agents.claude.env", map[string]string{"CLAUDE_CODE_MAX_TURNS": "100"})
	l.v.SetDefault("agents.gemini.env", map[string]string{})
	l.v.SetDefault("agents.codex.env", map[string]string{})
	l.v.SetDefault("runtime.name", "podman")
	l.v.SetDefault("runtime.privileged", false)
	l.v.SetDefault("runtime.flags", []string{})
}

// Load reads the configuration file, creating defaults if it doesn't exist.
func (l *Loader) Load() (*Config, error) {
	if _, err := os.Stat(l.path); os.IsNotExist(err) {
		if err := l.createDefault(); err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
	}

	if err := l.v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := l.v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.WeaklyTypedInput = true
	}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Expand paths
	cfg.Storage.Worktrees = l.expandPath(cfg.Storage.Worktrees)
	cfg.Storage.Catalog = l.expandPath(cfg.Storage.Catalog)
	cfg.Storage.Logs = l.expandPath(cfg.Storage.Logs)

	return &cfg, nil
}

// Path returns the configuration file path.
func (l *Loader) Path() string {
	return l.path
}

// Get returns a configuration value by dot-notation key.
func (l *Loader) Get(key string) (any, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	return l.v.Get(key), nil
}

// GetAgentEnv returns the environment variables for a specific agent.
// Returns an empty map if the agent has no env configuration.
func (l *Loader) GetAgentEnv(agent string) map[string]string {
	key := fmt.Sprintf("agents.%s.env", agent)
	raw := l.v.GetStringMapString(key)
	if raw == nil {
		return make(map[string]string)
	}
	return raw
}

// Set sets a configuration value by dot-notation key.
func (l *Loader) Set(key, value string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	// Validate agent name if setting default.agent
	if key == "default.agent" && value != "" {
		if !validAgents[value] {
			return fmt.Errorf("%w: %s (valid: claude, gemini, codex)", ErrInvalidAgent, value)
		}
	}

	// Validate multiplexer name if setting default.multiplexer
	if key == "default.multiplexer" && value != "" {
		if !validMultiplexers[value] {
			return fmt.Errorf("%w: %s (valid: tmux, zellij)", ErrInvalidMultiplexer, value)
		}
	}

	// Validate runtime name if setting runtime.name
	if key == "runtime.name" && value != "" {
		if !validRuntimes[value] {
			return fmt.Errorf("%w: %s (valid: podman, apple)", ErrInvalidRuntime, value)
		}
	}

	l.v.Set(key, value)
	return l.v.WriteConfig()
}

// createDefault writes the default configuration file using Viper.
func (l *Loader) createDefault() error {
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	return l.v.SafeWriteConfigAs(l.path)
}

// expandPath replaces ~ with the home directory.
func (l *Loader) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(l.homeDir, path[2:])
	}
	if path == "~" {
		return l.homeDir
	}
	return path
}

// ValidateKey checks if a key is a valid configuration key.
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: empty key", ErrInvalidKey)
	}

	// Check for exact match in derived valid keys
	if validKeys[key] {
		return nil
	}

	// Check for agents.<name> pattern (map type needs special handling)
	if strings.HasPrefix(key, "agents.") {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) >= 2 {
			agentName := parts[1]
			if validAgents[agentName] {
				// Valid patterns: agents.claude, agents.claude.env
				return nil
			}
			return fmt.Errorf("%w: %s (valid: claude, gemini, codex)", ErrInvalidAgent, agentName)
		}
	}

	return fmt.Errorf("%w: %s", ErrInvalidKey, key)
}

// buildValidKeys builds the set of valid keys from Config struct using reflection.
func buildValidKeys() map[string]bool {
	keys := make(map[string]bool)
	addKeysFromType(reflect.TypeOf(Config{}), "", keys)
	return keys
}

// addKeysFromType recursively adds keys from a struct type.
func addKeysFromType(t reflect.Type, prefix string, keys map[string]bool) {
	for i := range t.NumField() {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			continue
		}

		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}
		keys[key] = true

		// Recurse into nested structs (but not maps)
		if field.Type.Kind() == reflect.Struct {
			addKeysFromType(field.Type, key, keys)
		}
	}
}

// IsValidAgent is a package-level helper for checking agent validity.
func IsValidAgent(name string) bool {
	return validAgents[name]
}

// ValidAgentNames returns the list of valid agent names.
func ValidAgentNames() []string {
	return []string{"claude", "gemini", "codex"}
}

// IsValidMultiplexer is a package-level helper for checking multiplexer validity.
func IsValidMultiplexer(name string) bool {
	return validMultiplexers[name]
}

// ValidMultiplexerNames returns the list of valid multiplexer names.
func ValidMultiplexerNames() []string {
	return []string{"tmux", "zellij"}
}

// IsValidRuntime is a package-level helper for checking runtime validity.
func IsValidRuntime(name string) bool {
	return validRuntimes[name]
}

// ValidRuntimeNames returns the list of valid runtime names.
func ValidRuntimeNames() []string {
	return []string{"podman", "apple"}
}
