package extensions

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

const wasmMaxModuleSize = (1 << 30) + (1 << 29)

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) RegisterExtension(sessionID, name string, data []byte, os, init string) (*sliverpb.RegisterExtension, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    name,
		Data:    data,
		OS:      os,
		Init:    init,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) RegisterExtensionFromPath(sessionID, name, localPath, targetOS, init string) (*sliverpb.RegisterExtension, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = filepath.Base(localPath)
	}
	return s.rpc.RPC.RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    name,
		Data:    data,
		OS:      targetOS,
		Init:    init,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) ListExtensions(sessionID string) (*sliverpb.ListExtensions, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) CallExtension(sessionID, name, export string, args []byte, serverStore bool) (*sliverpb.CallExtension, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.CallExtension(context.Background(), &sliverpb.CallExtensionReq{
		Name:        name,
		Export:      export,
		Args:        args,
		ServerStore: serverStore,
		Request:     &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) RegisterWasmExtension(sessionID, name string, wasmGz []byte) (*sliverpb.RegisterWasmExtension, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.RegisterWasmExtension(context.Background(), &sliverpb.RegisterWasmExtensionReq{
		Name:    name,
		WasmGz:  wasmGz,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) RegisterWasmExtensionFromPath(sessionID, name, localPath string) (*sliverpb.RegisterWasmExtension, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	wasmGz, err := gzipWasmExtension(data)
	if err != nil {
		return nil, err
	}
	if len(wasmGz) > wasmMaxModuleSize {
		return nil, fmt.Errorf("wasm module is too big: %d bytes", len(wasmGz))
	}
	if name == "" {
		name = filepath.Base(localPath)
	}
	return s.rpc.RPC.RegisterWasmExtension(context.Background(), &sliverpb.RegisterWasmExtensionReq{
		Name:    name,
		WasmGz:  wasmGz,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func gzipWasmExtension(data []byte) ([]byte, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return data, nil
	}

	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Service) ListWasmExtensions(sessionID string) (*sliverpb.ListWasmExtensions, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.ListWasmExtensions(context.Background(), &sliverpb.ListWasmExtensionsReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) ExecWasmExtension(sessionID, name string, args []string, interactive bool) (*sliverpb.ExecWasmExtension, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.ExecWasmExtension(context.Background(), &sliverpb.ExecWasmExtensionReq{
		Name:        name,
		Args:        args,
		Interactive: interactive,
		Request:     &commonpb.Request{SessionID: sessionID},
	})
}
