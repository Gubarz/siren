package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/sliver/beacons"
	"siren/internal/sliver/console"
	"siren/internal/wailsadapter"
)

const discoveryTimeout = 10 * time.Minute

type NetworkDiscovery struct {
	AgentID  string `json:"agentID"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Hostname string `json:"hostname"`
	Vendor   string `json:"vendor"`
	OSHint   string `json:"osHint"`
	TTL      int    `json:"ttl"`
	Method   string `json:"method"`
	LastSeen int64  `json:"lastSeen"`
}

type Service struct {
	console     *console.Service
	beacons     *beacons.Service
	ui          *wailsadapter.Bridge
	mu          sync.RWMutex
	discoveries map[string]map[string]NetworkDiscovery
	path        string
}

func New(con *console.Service, beac *beacons.Service) *Service {
	s := &Service{
		console:     con,
		beacons:     beac,
		discoveries: make(map[string]map[string]NetworkDiscovery),
		path:        filepath.Join(assets.GetRootAppDir(), "gui-discoveries.json"),
	}
	s.load()
	return s
}

func (s *Service) SetUI(ui *wailsadapter.Bridge) {
	s.ui = ui
}

func (s *Service) GetNetworkDiscoveries() []NetworkDiscovery {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var devices []NetworkDiscovery
	for _, byIP := range s.discoveries {
		for _, device := range byIP {
			devices = append(devices, device)
		}
	}
	sort.Slice(devices, func(i, j int) bool {
		if devices[i].AgentID == devices[j].AgentID {
			return devices[i].IP < devices[j].IP
		}
		return devices[i].AgentID < devices[j].AgentID
	})
	return devices
}

func (s *Service) DiscoverNetwork(agentID, method, cidr string) ([]NetworkDiscovery, error) {
	session, beacon, err := s.console.FindTarget(agentID)
	if err != nil {
		return nil, err
	}

	osName := targetOS(session, beacon)
	method = strings.ToLower(strings.TrimSpace(method))
	var commands []string
	switch method {
	case "arp":
		commands = []string{neighborCommand(osName)}
	case "sweep":
		hosts, err := sweepHosts(cidr)
		if err != nil {
			return nil, err
		}
		commands = []string{sweepCommand(osName, hosts), neighborCommand(osName)}
	default:
		return nil, fmt.Errorf("unsupported discovery method: %s", method)
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	var found []NetworkDiscovery
	for _, command := range commands {
		output, taskID, err := s.console.RunAutomationLine(agentID, command)
		if err != nil {
			return nil, err
		}
		if beacon != nil {
			output, _, err = s.beacons.AwaitBeaconTask(ctx, agentID, output, taskID)
			if err != nil {
				return nil, err
			}
		}
		found = mergeDiscoveryResults(found, parseDiscoveryOutput(agentID, method, output))
	}

	s.storeDiscoveries(found)
	return s.GetNetworkDiscoveries(), nil
}

func (s *Service) ClearNetworkDiscoveries(agentID string) {
	s.mu.Lock()
	if strings.TrimSpace(agentID) == "" {
		clear(s.discoveries)
	} else {
		delete(s.discoveries, agentID)
	}
	s.persist()
	s.mu.Unlock()
	s.emitDiscoveryUpdate()
}

func (s *Service) RemoveNetworkDiscoveries(agentID string, ips []string) {
	s.mu.Lock()
	if byIP := s.discoveries[agentID]; byIP != nil {
		for _, ip := range ips {
			delete(byIP, strings.TrimSpace(ip))
		}
		if len(byIP) == 0 {
			delete(s.discoveries, agentID)
		}
	}
	s.persist()
	s.mu.Unlock()
	s.emitDiscoveryUpdate()
}

func (s *Service) storeDiscoveries(devices []NetworkDiscovery) {
	s.mu.Lock()
	for _, device := range devices {
		if s.discoveries[device.AgentID] == nil {
			s.discoveries[device.AgentID] = make(map[string]NetworkDiscovery)
		}
		existing := s.discoveries[device.AgentID][device.IP]
		if device.MAC == "" {
			device.MAC = existing.MAC
		}
		if device.Hostname == "" {
			device.Hostname = existing.Hostname
		}
		if device.Vendor == "" {
			device.Vendor = existing.Vendor
		}
		if device.OSHint == "" {
			device.OSHint = existing.OSHint
			device.TTL = existing.TTL
		}
		s.discoveries[device.AgentID][device.IP] = device
	}
	s.persist()
	s.mu.Unlock()
	s.emitDiscoveryUpdate()
}

func (s *Service) load() {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		log.Printf("discovery: could not load: %v", err)
		return
	}
	var devices []NetworkDiscovery
	if err := json.Unmarshal(data, &devices); err != nil {
		log.Printf("discovery: could not decode: %v", err)
		return
	}
	for _, device := range devices {
		if s.discoveries[device.AgentID] == nil {
			s.discoveries[device.AgentID] = make(map[string]NetworkDiscovery)
		}
		s.discoveries[device.AgentID][device.IP] = device
	}
}

func (s *Service) persist() {
	var devices []NetworkDiscovery
	for _, byIP := range s.discoveries {
		for _, device := range byIP {
			devices = append(devices, device)
		}
	}
	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		log.Printf("discovery: could not encode: %v", err)
		return
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		log.Printf("discovery: could not write: %v", err)
		return
	}
	if err := os.Rename(temp, s.path); err != nil {
		log.Printf("discovery: could not rename: %v", err)
	}
}

func (s *Service) emitDiscoveryUpdate() {
	if s.ui != nil {
		s.ui.Emit("network-discovery-updated", s.GetNetworkDiscoveries())
	}
}

func targetOS(session *clientpb.Session, beacon *clientpb.Beacon) string {
	if session != nil {
		return strings.ToLower(session.OS)
	}
	if beacon != nil {
		return strings.ToLower(beacon.OS)
	}
	return ""
}

func (s *Service) HandlePingOutput(sessionID, output string) {
	if devices := parsePingDiscoveryOutput(sessionID, output); len(devices) > 0 {
		s.storeDiscoveries(devices)
	}
}
