package server

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/sliver/rpc"
)

func (s *Service) GetCertificates() (*clientpb.CertificateInfo, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.GetCertificateInfo(context.Background(), &clientpb.CertificatesReq{})
}
