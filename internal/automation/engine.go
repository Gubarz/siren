package automation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/protobuf/proto"

	"sliver-gui/internal/beacons"
	"sliver-gui/internal/console"
	"sliver-gui/internal/rpc"
)

const (
	automationHistoryLimit = 500
	beaconPollInterval     = 5 * time.Second
)

type Engine struct {
	rpc     *rpc.Client
	console *console.Service
	beacons *beacons.Service
	tags    AgentTagStore
	ctx     context.Context
	path    string

	mu             sync.RWMutex
	rules          []AutomationRule
	history        []AutomationRun
	running        map[string]bool
	activeByRule   map[string]int
	lastRun        map[string]time.Time
	lastInterval   map[string]time.Time
	beaconCheckins map[string]int64
	beaconsPrimed  bool
}

type AgentTagStore interface {
	GetAgentTags(agentID string) []string
	SetAgentTags(agentID string, tags []string) error
}

func New(rpc *rpc.Client, con *console.Service, beac *beacons.Service, tagStore AgentTagStore) *Engine {
	e := &Engine{
		rpc:            rpc,
		console:        con,
		beacons:        beac,
		tags:           tagStore,
		path:           filepath.Join(assets.GetRootAppDir(), "gui-automation.json"),
		running:        map[string]bool{},
		activeByRule:   map[string]int{},
		lastRun:        map[string]time.Time{},
		lastInterval:   map[string]time.Time{},
		beaconCheckins: map[string]int64{},
	}
	if err := e.load(); err != nil {
		log.Printf("automation: could not load state: %v", err)
	}
	return e
}

func (e *Engine) Start(ctx context.Context) {
	e.ctx = ctx
	go func() {
		ticker := time.NewTicker(beaconPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				e.tick(now)
			}
		}
	}()
}

func (e *Engine) load() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.loadLocked()
}

func (e *Engine) loadLocked() error {
	data, err := os.ReadFile(e.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var state automationState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	e.rules = state.Rules
	e.history = state.History
	return nil
}

func (e *Engine) SetServer(host string, port uint32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	newPath := filepath.Join(assets.GetRootAppDir(), fmt.Sprintf("gui-automation-%s_%d.json", host, port))
	if newPath == e.path {
		return
	}
	e.path = newPath
	e.rules = nil
	e.history = nil
	if err := e.loadLocked(); err != nil {
		log.Printf("automation: could not load state for %s:%d: %v", host, port, err)
	}
}

func (e *Engine) persistLocked() error {
	state := automationState{Rules: e.rules, History: e.history}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	temp := e.path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, e.path)
}

func (e *Engine) tick(now time.Time) {
	if !e.rpc.Connected() {
		return
	}
	e.pollBeaconCheckins()

	e.mu.RLock()
	rules := append([]AutomationRule(nil), e.rules...)
	e.mu.RUnlock()
	for _, rule := range rules {
		if !rule.Enabled || rule.Trigger != "interval" {
			continue
		}
		interval := time.Duration(rule.IntervalSeconds) * time.Second
		if interval < 10*time.Second {
			interval = 10 * time.Second
		}
		e.mu.Lock()
		last := e.lastInterval[rule.ID]
		if last.IsZero() {
			e.lastInterval[rule.ID] = now
			e.mu.Unlock()
			continue
		}
		if now.Sub(last) < interval {
			e.mu.Unlock()
			continue
		}
		e.lastInterval[rule.ID] = now
		e.mu.Unlock()
		e.dispatchRule(rule, "interval", nil)
	}
}

func (e *Engine) pollBeaconCheckins() {
	beaconsResp, err := e.rpc.RPC.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return
	}
	e.mu.RLock()
	previous := make(map[string]int64, len(e.beaconCheckins))
	for id, checkin := range e.beaconCheckins {
		previous[id] = checkin
	}
	primed := e.beaconsPrimed
	e.mu.RUnlock()

	current := make(map[string]int64, len(beaconsResp.Beacons))
	for _, beacon := range beaconsResp.Beacons {
		current[beacon.ID] = beacon.LastCheckin
		if primed && previous[beacon.ID] != 0 && beacon.LastCheckin > previous[beacon.ID] {
			e.dispatchTrigger("beacon-checkin", targetFromBeacon(beacon))
		}
	}
	e.mu.Lock()
	e.beaconCheckins = current
	e.beaconsPrimed = true
	e.mu.Unlock()
}

func (e *Engine) HandleSliverEvent(event *clientpb.Event) {
	switch event.EventType {
	case consts.SessionOpenedEvent:
		if event.Session != nil {
			e.dispatchTrigger("session-connected", targetFromSession(event.Session))
		}
	case consts.BeaconRegisteredEvent:
		beacon := &clientpb.Beacon{}
		if len(event.Data) > 0 && proto.Unmarshal(event.Data, beacon) == nil && beacon.ID != "" {
			e.mu.Lock()
			e.beaconCheckins[beacon.ID] = beacon.LastCheckin
			e.mu.Unlock()
			e.dispatchTrigger("beacon-registered", targetFromBeacon(beacon))
		}
	}
}

func (e *Engine) emit(name string, payload interface{}) {
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, name, payload)
	}
}

func (e *Engine) ruleByIDLocked(id string) *AutomationRule {
	for index := range e.rules {
		if e.rules[index].ID == id {
			return &e.rules[index]
		}
	}
	return nil
}
