package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

type Client struct {
	Config *assets.ClientConfig
	RPC    rpcpb.SliverRPCClient
	Conn   *grpc.ClientConn

	connected atomic.Bool
	connectMu sync.Mutex

	cacheMu        sync.RWMutex
	cachedSessions []*clientpb.Session
	cachedBeacons  []*clientpb.Beacon

	streamMu     sync.Mutex
	streamCancel context.CancelFunc
}

const connectTimeout = 12 * time.Second

func NewClient() *Client {
	return &Client{}
}

func selectClientConfig(profileName string) (*assets.ClientConfig, error) {
	configs := assets.GetConfigs()
	if len(configs) == 0 {
		return nil, fmt.Errorf("no sliver configs found in ~/.sliver-client/configs")
	}

	if profileName != "" {
		cfg, ok := configs[profileName]
		if !ok {
			return nil, fmt.Errorf("profile not found: %s", profileName)
		}
		return cfg, nil
	}

	for _, c := range configs {
		return c, nil
	}
	return nil, fmt.Errorf("no configs found")
}

func (c *Client) Connect(profileName string) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	config, err := selectClientConfig(profileName)
	if err != nil {
		return err
	}

	log.Printf("Connecting to %s:%d as %s", config.LHost, config.LPort, config.Operator)

	type connectResult struct {
		rpcClient rpcpb.SliverRPCClient
		grpcConn  *grpc.ClientConn
		err       error
	}

	resultCh := make(chan connectResult)
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	go func() {
		rpcClient, grpcConn, err := transport.MTLSConnect(config)
		result := connectResult{rpcClient: rpcClient, grpcConn: grpcConn, err: err}
		select {
		case resultCh <- result:
		case <-ctx.Done():
			if grpcConn != nil {
				_ = grpcConn.Close()
			}
		}
	}()

	var result connectResult
	select {
	case result = <-resultCh:
	case <-ctx.Done():
		return fmt.Errorf("connection timed out after %s", connectTimeout)
	}
	if result.err != nil {
		return fmt.Errorf("failed to connect: %w", result.err)
	}

	oldConn := c.Conn
	c.Config = config
	c.RPC = result.rpcClient
	c.Conn = result.grpcConn
	c.connected.Store(true)
	if oldConn != nil && oldConn != result.grpcConn {
		_ = oldConn.Close()
	}
	return nil
}

func (c *Client) Disconnect() {
	c.streamMu.Lock()
	if c.streamCancel != nil {
		c.streamCancel()
		c.streamCancel = nil
	}
	c.streamMu.Unlock()
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.connected.Store(false)
}

func (c *Client) Connected() bool {
	return c.connected.Load()
}

func (c *Client) GetClientConfigs() ([]string, error) {
	configs := assets.GetConfigs()
	var names []string
	for name := range configs {
		names = append(names, name)
	}
	return names, nil
}

type ClientConfigSummary struct {
	Name     string `json:"name"`
	Operator string `json:"operator"`
	LHost    string `json:"lhost"`
	LPort    int    `json:"lport"`
}

func (c *Client) GetClientConfigDetails() ([]ClientConfigSummary, error) {
	configs := assets.GetConfigs()
	out := make([]ClientConfigSummary, 0, len(configs))
	for name, cfg := range configs {
		out = append(out, ClientConfigSummary{
			Name: name, Operator: cfg.Operator, LHost: cfg.LHost, LPort: cfg.LPort,
		})
	}
	return out, nil
}

func (c *Client) ImportClientConfig(payload string) (string, error) {
	cfg := &assets.ClientConfig{}
	if err := json.Unmarshal([]byte(payload), cfg); err != nil {
		return "", fmt.Errorf("invalid config JSON: %w", err)
	}
	if cfg.LHost == "" || cfg.Operator == "" || cfg.Certificate == "" {
		return "", fmt.Errorf("config missing required fields (operator, lhost, certificate)")
	}
	if err := assets.SaveConfig(cfg); err != nil {
		return "", err
	}
	for name, existing := range assets.GetConfigs() {
		if existing.Operator == cfg.Operator && existing.LHost == cfg.LHost && existing.Certificate == cfg.Certificate {
			return name, nil
		}
	}
	return fmt.Sprintf("%s@%s", cfg.Operator, cfg.LHost), nil
}

func (c *Client) ExportClientConfig(name string) (string, error) {
	cfg, ok := assets.GetConfigs()[name]
	if !ok {
		return "", fmt.Errorf("profile not found: %s", name)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) DeleteClientConfig(name string) error {
	configs := assets.GetConfigs()
	target, ok := configs[name]
	if !ok {
		return fmt.Errorf("profile not found: %s", name)
	}
	dir := assets.GetConfigDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		cfg, err := assets.ReadConfig(path)
		if err != nil {
			continue
		}
		if cfg.Operator == target.Operator && cfg.LHost == target.LHost && cfg.Certificate == target.Certificate {
			return os.Remove(path)
		}
	}
	return fmt.Errorf("profile file not found on disk")
}

func (c *Client) PopulateSessions(s *clientpb.Sessions) {
	c.cacheMu.Lock()
	c.cachedSessions = s.Sessions
	c.cacheMu.Unlock()
}

func (c *Client) PopulateBeacons(b *clientpb.Beacons) {
	c.cacheMu.Lock()
	c.cachedBeacons = b.Beacons
	c.cacheMu.Unlock()
}

func (c *Client) LookupSession(id string) *clientpb.Session {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	for _, s := range c.cachedSessions {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (c *Client) LookupBeacon(id string) *clientpb.Beacon {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	for _, b := range c.cachedBeacons {
		if b.ID == id {
			return b
		}
	}
	return nil
}

func (c *Client) InvalidateAgentCache() {
	c.cacheMu.Lock()
	c.cachedSessions = nil
	c.cachedBeacons = nil
	c.cacheMu.Unlock()
}
