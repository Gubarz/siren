// Package bloodhound integrates the BloodHound Community Edition API via
// github.com/Gubarz/bloodhound-sdk-go. It is independent of Sliver RPC state.
package bloodhound

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"siren/internal/localstate/jsonstore"
)

var ErrNotConfigured = errors.New("bloodhound: not configured")

// Config holds BloodHound connection settings. TokenKey is a secret persisted
// only to local disk (0600) and never returned to the UI after save.
type Config struct {
	ServerURL   string `json:"serverUrl"`
	TokenID     string `json:"tokenId"`
	TokenKey    string `json:"tokenKey"`
	InsecureTLS bool   `json:"insecureTls"`
}

func (c Config) configured() bool {
	return c.ServerURL != "" && c.TokenID != "" && c.TokenKey != ""
}

// ConfigView is the UI-safe projection of Config; the secret is replaced by
// HasTokenKey so saved credentials are never echoed back to the frontend.
type ConfigView struct {
	ServerURL   string `json:"serverUrl"`
	TokenID     string `json:"tokenId"`
	HasTokenKey bool   `json:"hasTokenKey"`
	InsecureTLS bool   `json:"insecureTls"`
}

func Masked(cfg Config) ConfigView {
	return ConfigView{
		ServerURL:   cfg.ServerURL,
		TokenID:     cfg.TokenID,
		HasTokenKey: cfg.TokenKey != "",
		InsecureTLS: cfg.InsecureTLS,
	}
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.ServerURL) == "" {
		return fmt.Errorf("%w: server URL is required", ErrNotConfigured)
	}
	u, err := url.Parse(cfg.ServerURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: invalid server URL %q", ErrNotConfigured, cfg.ServerURL)
	}
	if strings.TrimSpace(cfg.TokenID) == "" {
		return fmt.Errorf("%w: API token ID is required", ErrNotConfigured)
	}
	if strings.TrimSpace(cfg.TokenKey) == "" {
		return fmt.Errorf("%w: API token key is required", ErrNotConfigured)
	}
	return nil
}

const configPrefix = "bloodhound"

type ConfigStore struct {
	store *jsonstore.ScopedStore[Config]
}

func NewConfigStore(rootDir string) *ConfigStore {
	return &ConfigStore{store: jsonstore.New[Config](rootDir, configPrefix)}
}

func (s *ConfigStore) Load() (Config, bool, error) { return s.store.Load() }

func (s *ConfigStore) Save(cfg Config) error {
	if err := validate(cfg); err != nil {
		return err
	}
	return s.store.Save(cfg)
}
