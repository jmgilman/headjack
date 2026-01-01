package cmd

import (
	"context"

	"github.com/jmgilman/headjack/internal/config"
	"github.com/jmgilman/headjack/internal/instance"
)

type contextKey string

const (
	configKey  contextKey = "config"
	loaderKey  contextKey = "loader"
	managerKey contextKey = "manager"
)

// WithConfig adds the config to the context.
func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

// ConfigFromContext retrieves the config from context.
func ConfigFromContext(ctx context.Context) *config.Config {
	cfg, ok := ctx.Value(configKey).(*config.Config)
	if !ok {
		return nil
	}
	return cfg
}

// WithLoader adds the config loader to the context.
func WithLoader(ctx context.Context, loader *config.Loader) context.Context {
	return context.WithValue(ctx, loaderKey, loader)
}

// LoaderFromContext retrieves the config loader from context.
func LoaderFromContext(ctx context.Context) *config.Loader {
	loader, ok := ctx.Value(loaderKey).(*config.Loader)
	if !ok {
		return nil
	}
	return loader
}

// WithManager adds the instance manager to the context.
func WithManager(ctx context.Context, mgr *instance.Manager) context.Context {
	return context.WithValue(ctx, managerKey, mgr)
}

// ManagerFromContext retrieves the instance manager from context.
func ManagerFromContext(ctx context.Context) *instance.Manager {
	mgr, ok := ctx.Value(managerKey).(*instance.Manager)
	if !ok {
		return nil
	}
	return mgr
}
