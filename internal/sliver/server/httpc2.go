package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/rpc"
)

func (s *Service) HTTPC2Profiles() (*clientpb.HTTPC2Configs, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.GetHTTPC2Profiles(context.Background(), &commonpb.Empty{})
}

func (s *Service) HTTPC2ProfileByName(name string) (*clientpb.HTTPC2Config, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.GetHTTPC2ProfileByName(context.Background(), &clientpb.C2ProfileReq{
		Name: strings.TrimSpace(name),
	})
}

func (s *Service) SaveHTTPC2Profile(config *clientpb.HTTPC2Config, overwrite bool) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if config == nil || strings.TrimSpace(config.Name) == "" {
		return fmt.Errorf("HTTP C2 profile name is required")
	}
	_, err := s.rpc.SaveHTTPC2Profile(context.Background(), &clientpb.HTTPC2ConfigReq{
		Overwrite: overwrite,
		C2Config:  config,
	})
	return err
}
