package automationexec

import (
	"context"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/bus"
	"siren/internal/sliver/rpc"
)

const checkinPollInterval = 5 * time.Second

type CheckinPublisher struct {
	rpc *rpc.Client
	bus bus.Bus

	mu     sync.Mutex
	seen   map[string]int64
	primed bool
}

func NewCheckinPublisher(rpcClient *rpc.Client, b bus.Bus) *CheckinPublisher {
	return &CheckinPublisher{rpc: rpcClient, bus: b, seen: map[string]int64{}}
}

func (p *CheckinPublisher) Start(ctx context.Context) {
	go p.loop(ctx)
}

func (p *CheckinPublisher) loop(ctx context.Context) {
	ticker := time.NewTicker(checkinPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

type checkinBeacon struct {
	ID          string
	Name        string
	Hostname    string
	Username    string
	OS          string
	Arch        string
	LastCheckin int64
}

func (p *CheckinPublisher) poll(ctx context.Context) {
	if !p.rpc.Connected() {
		return
	}
	resp, err := p.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return
	}
	current := make([]checkinBeacon, 0, len(resp.Beacons))
	for _, beacon := range resp.Beacons {
		current = append(current, checkinBeacon{
			ID:          beacon.ID,
			Name:        beacon.Name,
			Hostname:    beacon.Hostname,
			Username:    beacon.Username,
			OS:          beacon.OS,
			Arch:        beacon.Arch,
			LastCheckin: beacon.LastCheckin,
		})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fired, next := advancedCheckins(p.seen, current, p.primed)
	p.seen = next
	p.primed = true
	for _, b := range fired {
		p.bus.Publish(bus.Event{
			Type:         "sliver.beacon-checkin",
			Source:       "grpc-stream",
			ConnectionID: p.rpc.ConnectionID(),
			Payload: map[string]any{
				"beaconID": b.ID, "name": b.Name,
				"hostname": b.Hostname, "username": b.Username,
				"os": b.OS, "arch": b.Arch,
				"lastCheckin": b.LastCheckin,
			},
		})
	}
}

func advancedCheckins(seen map[string]int64, current []checkinBeacon, primed bool) (fired []checkinBeacon, next map[string]int64) {
	next = make(map[string]int64, len(current))
	for _, b := range current {
		next[b.ID] = b.LastCheckin
		if primed && seen[b.ID] != 0 && b.LastCheckin > seen[b.ID] {
			fired = append(fired, b)
		}
	}
	return fired, next
}
