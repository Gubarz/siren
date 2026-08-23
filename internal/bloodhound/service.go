package bloodhound

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	bh "github.com/Gubarz/bloodhound-sdk-go"

	"siren/internal/bus"
)

var ErrNotConnected = errors.New("bloodhound: not connected")

// Status is what the UI connection bar renders.
type Status struct {
	Configured bool   `json:"configured"`
	Connected  bool   `json:"connected"`
	ServerURL  string `json:"serverUrl"`
	Error      string `json:"error,omitempty"`
}

// clientFactory builds the SDK client; overridden in tests to point at a
// fake BloodHound HTTP server.
var clientFactory = func(cfg Config) (*bh.Client, error) {
	opts := []bh.Option{
		bh.WithTimeout(15 * time.Second),
		bh.WithRetry(3, 500*time.Millisecond),
		bh.WithUserAgent("siren-bloodhound"),
	}
	if cfg.InsecureTLS {
		opts = append(opts, bh.WithInsecureSkipVerify())
	}
	return bh.NewClient(cfg.ServerURL, cfg.TokenID, cfg.TokenKey, opts...)
}

// Service owns the BloodHound connection lifecycle. It is deliberately
// independent of Sliver RPC state: operators can enrich findings while the
// teamserver is disconnected.
//
// Locking: cfgMu guards the config, connMu guards the live client. Query
// methods take a client snapshot and release connMu before any network I/O,
// so a slow query never blocks Connect/Disconnect/Status.
type Service struct {
	cfgMu sync.RWMutex
	cfg   Config
	store *ConfigStore

	connMu sync.Mutex
	client *bh.Client
	lastEr error

	bus     bus.Bus
	corr    *Correlator
	dataDir string
}

// New builds the service, loading a previously saved config when present.
// b may be nil: event publishing then becomes a no-op.
func New(rootDir string, b bus.Bus) *Service {
	s := &Service{
		store:   NewConfigStore(rootDir),
		bus:     b,
		corr:    newCorrelator(),
		dataDir: rootDir,
	}
	if cfg, ok, err := s.store.Load(); err == nil && ok {
		s.cfg = cfg // invalid/partial files simply leave the service unconfigured
	}
	return s
}

func (s *Service) GetConfig() ConfigView {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return Masked(s.cfg)
}

// SaveConfig validates and persists cfg, then rebuilds the connection if we
// were connected. A failed reconnect keeps the saved config and reports the error.
func (s *Service) SaveConfig(cfg Config) error {
	return s.save(cfg)
}

// MergeSaveConfig persists cfg, keeping the previously stored token key when
// the incoming one is blank (the UI never receives the saved secret back).
// Reconnect semantics match SaveConfig.
func (s *Service) MergeSaveConfig(cfg Config) error {
	if cfg.TokenKey == "" {
		s.cfgMu.RLock()
		cfg.TokenKey = s.cfg.TokenKey
		s.cfgMu.RUnlock()
	}
	return s.save(cfg)
}

// TestConnection probes credentials against the server without persisting
// anything or disturbing the live connection/config. A blank TokenKey falls
// back to the stored key, mirroring MergeSaveConfig.
func (s *Service) TestConnection(cfg Config) error {
	if cfg.TokenKey == "" {
		s.cfgMu.RLock()
		cfg.TokenKey = s.cfg.TokenKey
		s.cfgMu.RUnlock()
	}
	client, err := clientFactory(cfg)
	if err != nil {
		return err
	}
	if err := client.Ping(context.Background()); err != nil {
		return fmt.Errorf("bloodhound unreachable: %w", err)
	}
	return nil
}

func (s *Service) save(cfg Config) error {
	if err := s.store.Save(cfg); err != nil {
		return err
	}
	s.connMu.Lock()
	wasConnected := s.client != nil
	s.disconnectLocked()
	s.connMu.Unlock()
	s.cfgMu.Lock()
	s.cfg = cfg
	s.cfgMu.Unlock()
	if wasConnected {
		if err := s.Connect(context.Background()); err != nil {
			s.publishStatus()
			return fmt.Errorf("saved, but reconnect failed: %w", err)
		}
	}
	s.publishStatus()
	return nil
}

func (s *Service) Connect(ctx context.Context) error {
	s.connMu.Lock()
	err := s.connectLocked()
	s.connMu.Unlock()
	s.publishStatus()
	return err
}

func (s *Service) connectLocked() error {
	s.cfgMu.RLock()
	cfg := s.cfg
	s.cfgMu.RUnlock()
	if !cfg.configured() {
		s.lastEr = ErrNotConfigured
		return ErrNotConfigured
	}
	client, err := clientFactory(cfg)
	if err != nil {
		s.lastEr = err
		return err
	}
	if err := client.Ping(context.Background()); err != nil {
		s.lastEr = fmt.Errorf("bloodhound unreachable: %w", err)
		return s.lastEr
	}
	s.client = client
	s.lastEr = nil
	return nil
}

func (s *Service) Disconnect() {
	s.connMu.Lock()
	s.disconnectLocked()
	s.connMu.Unlock()
	s.publishStatus()
}

func (s *Service) disconnectLocked() {
	s.client = nil
}

func (s *Service) Close() { s.Disconnect() }

func (s *Service) Status() Status {
	s.cfgMu.RLock()
	configured := s.cfg.configured()
	serverURL := s.cfg.ServerURL
	s.cfgMu.RUnlock()

	s.connMu.Lock()
	st := Status{
		Configured: configured,
		Connected:  s.client != nil,
		ServerURL:  serverURL,
	}
	if s.lastEr != nil {
		st.Error = s.lastEr.Error()
	}
	s.connMu.Unlock()
	return st
}

// snapshot returns the live client without holding connMu past the return,
// so callers can run HTTP requests without blocking connection operations.
func (s *Service) snapshot() (*bh.Client, error) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.client == nil {
		return nil, ErrNotConnected
	}
	return s.client, nil
}
