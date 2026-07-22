package server

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/sliver/rpc"
)

// CertificateAuthorityInfo returns the teamserver's CA descriptor — surfaced
// in Settings so operators can validate they're talking to the CA they think
// they are before onboarding new operators.
func (s *Service) CertificateAuthorityInfo() (*clientpb.CertificateAuthorityInfo, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.GetCertificateAuthorityInfo(context.Background(), &commonpb.Empty{})
}

// Compiler reports the go / cross toolchains the server has available; the
// Generate modal uses it to gate GOOS/GOARCH options rather than blindly
// letting operators pick combinations that would just fail server-side.
func (s *Service) Compiler() (*clientpb.Compiler, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.GetCompiler(context.Background(), &commonpb.Empty{})
}

// Canaries lists every DNS canary the server has issued and whether it has
// been resolved (i.e. an implant was likely analysed).
func (s *Service) Canaries() (*clientpb.Canaries, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.Canaries(context.Background(), &commonpb.Empty{})
}
