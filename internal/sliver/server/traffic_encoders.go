package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/google/uuid"

	"sliver-gui/internal/sliver/rpc"
)

const maxTrafficEncoderBytes = 8 * 1024 * 1024

func (s *Service) TrafficEncoderMap() (*clientpb.TrafficEncoderMap, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.TrafficEncoderMap(context.Background(), &commonpb.Empty{})
}

func (s *Service) AddTrafficEncoder(localPath string, skipTests bool) (*clientpb.TrafficEncoderTests, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	encoder, err := trafficEncoderFromPath(localPath, skipTests)
	if err != nil {
		return nil, err
	}
	return s.rpc.TrafficEncoderAdd(context.Background(), encoder)
}

func (s *Service) RemoveTrafficEncoder(name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	encoderName, err := normalizeTrafficEncoderName(name)
	if err != nil {
		return err
	}
	_, err = s.rpc.TrafficEncoderRm(context.Background(), &clientpb.TrafficEncoder{
		Wasm: &commonpb.File{Name: encoderName},
	})
	return err
}

func trafficEncoderFromPath(localPath string, skipTests bool) (*clientpb.TrafficEncoder, error) {
	name, err := normalizeTrafficEncoderName(localPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(strings.TrimSpace(localPath))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s is empty", name)
	}
	if len(data) > maxTrafficEncoderBytes {
		return nil, fmt.Errorf("%s is larger than 8 MiB", name)
	}
	return &clientpb.TrafficEncoder{
		Wasm:      &commonpb.File{Name: name, Data: data},
		SkipTests: skipTests,
		TestID:    uuid.NewString(),
	}, nil
}

func normalizeTrafficEncoderName(name string) (string, error) {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == "" {
		return "", fmt.Errorf("traffic encoder name is required")
	}
	if !strings.HasSuffix(strings.ToLower(base), ".wasm") {
		return "", fmt.Errorf("traffic encoder must be a .wasm file")
	}
	return base, nil
}
