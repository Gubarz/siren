// Package health surfaces GUI-visible connection quality metrics — RPC
// round-trip time, active counts, tunnel throughput. Everything here is a
// snapshot; the panel polls on its own cadence so the wire format stays
// stateless.
package health

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/tunneling"
)

const rpcTimeout = 3 * time.Second

type Service struct {
	rpc       *rpc.Client
	tunneling *tunneling.Service
}

// Snapshot is the flat wire format the frontend consumes — every field is
// a scalar so the health panel doesn't need extra deserialization logic.
type Snapshot struct {
	Connected     bool   `json:"connected"`
	RPCLatencyMs  int64  `json:"rpcLatencyMs"`
	RPCError      string `json:"rpcError,omitempty"`
	SessionCount  int    `json:"sessionCount"`
	BeaconCount   int    `json:"beaconCount"`
	JobCount      int    `json:"jobCount"`
	TunnelCount   int    `json:"tunnelCount"`
	SocksCount    int    `json:"socksCount"`
	PortfwdCount  int    `json:"portfwdCount"`
	RportfwdCount int    `json:"rportfwdCount"`
	CheckedAt     int64  `json:"checkedAt"`
}

func New(rpc *rpc.Client, tun *tunneling.Service) *Service {
	return &Service{rpc: rpc, tunneling: tun}
}

func (s *Service) Close() {}

// Snapshot pings the server (measures RPC round-trip via GetVersion) and
// collects local tunnel + agent counts. Every RPC has a 3s hard timeout —
// a hanging server shouldn't lock the health panel.
func (s *Service) Snapshot() Snapshot {
	out := Snapshot{CheckedAt: time.Now().UnixMilli()}
	if !s.rpc.Connected() {
		return out
	}
	out.Connected = true
	out.RPCLatencyMs, out.RPCError = s.pingLatency()
	s.fillAgentCounts(&out)
	s.fillTunnelCounts(&out)
	return out
}

func (s *Service) pingLatency() (int64, string) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	start := time.Now()
	_, err := s.rpc.RPC.GetVersion(ctx, &commonpb.Empty{})
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return elapsed, err.Error()
	}
	return elapsed, ""
}

func (s *Service) fillAgentCounts(out *Snapshot) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	if sessions, err := s.rpc.RPC.GetSessions(ctx, &commonpb.Empty{}); err == nil {
		out.SessionCount = len(sessions.Sessions)
	}
	if beacons, err := s.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{}); err == nil {
		out.BeaconCount = len(beacons.Beacons)
	}
	if jobs, err := s.rpc.RPC.GetJobs(ctx, &commonpb.Empty{}); err == nil {
		out.JobCount = len(jobs.Active)
	}
}

func (s *Service) fillTunnelCounts(out *Snapshot) {
	for _, proxy := range s.tunneling.List() {
		out.TunnelCount++
		switch proxy.Kind {
		case "socks":
			out.SocksCount++
		case "portfwd":
			out.PortfwdCount++
		}
	}
	if rp, err := s.tunneling.ListRportfwds(); err == nil {
		out.RportfwdCount = len(rp)
		out.TunnelCount += len(rp)
	}
}
