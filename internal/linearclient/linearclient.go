// Package linearclient builds an authenticated linear.Client from config, shared by every entry point that talks to Linear (the poll loop, CLI subcommands).
package linearclient

import (
	"log/slog"

	"github.com/ahmadAlMezaal/noctra/internal/config"
	"github.com/ahmadAlMezaal/noctra/internal/linear"
	"github.com/ahmadAlMezaal/noctra/internal/state"
)

// New picks the strongest credential configured: a self-renewing actor=app token, a static OAuth token, then the personal API key. Both OAuth paths keep the personal key as a fallback so an expired token degrades instead of crash-looping. A nil store means rotated tokens aren't persisted.
func New(cfg *config.Config, store *state.Store) *linear.Client {
	if cfg.OAuthPartiallyConfigured() {
		slog.Warn("linear actor=app config incomplete (need both client id and secret); using personal API key")
	}
	if cfg.ActorAppConfigured() {
		var ts linear.TokenStore
		if store != nil {
			ts = store
		}
		tm := linear.NewTokenManager(linear.TokenManagerConfig{
			ClientID:     cfg.LinearOAuthClientID,
			ClientSecret: cfg.LinearOAuthClientSecret,
			RefreshToken: cfg.LinearOAuthRefreshToken,
			Scope:        cfg.LinearOAuthScope,
			Store:        ts,
		})
		c := linear.New(cfg.LinearAPIKey)
		c.TokenFn = tm.Token
		c.OnAuthError = tm.ForceRefresh
		c.FallbackAPIKey = cfg.LinearAPIKey
		return c
	}
	if cfg.LinearOAuthToken != "" {
		c := linear.NewOAuth(cfg.LinearOAuthToken)
		c.FallbackAPIKey = cfg.LinearAPIKey
		return c
	}
	return linear.New(cfg.LinearAPIKey)
}
