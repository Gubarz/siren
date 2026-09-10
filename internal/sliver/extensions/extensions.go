package extensions

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
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
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.RegisterExtension, &sliverpb.RegisterExtensionReq{
		Name:    name,
		Data:    data,
		OS:      os,
		Init:    init,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) RegisterExtensionFromPath(sessionID, name, localPath, targetOS, init string) (*sliverpb.RegisterExtension, error) {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = filepath.Base(localPath)
	}
	return c.RPC().RegisterExtension(context.Background(), &sliverpb.RegisterExtensionReq{
		Name:    name,
		Data:    data,
		OS:      targetOS,
		Init:    init,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) ListExtensions(sessionID string) (*sliverpb.ListExtensions, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.ListExtensions, &sliverpb.ListExtensionsReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) CallExtension(sessionID, name, export string, args []byte, serverStore bool) (*sliverpb.CallExtension, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CallExtension, &sliverpb.CallExtensionReq{
		Name:        name,
		Export:      export,
		Args:        args,
		ServerStore: serverStore,
		Request:     &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) RegisterWasmExtension(sessionID, name string, wasmGz []byte) (*sliverpb.RegisterWasmExtension, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.RegisterWasmExtension, &sliverpb.RegisterWasmExtensionReq{
		Name:    name,
		WasmGz:  wasmGz,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) RegisterWasmExtensionFromPath(sessionID, name, localPath string) (*sliverpb.RegisterWasmExtension, error) {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return nil, err
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
	return c.RPC().RegisterWasmExtension(context.Background(), &sliverpb.RegisterWasmExtensionReq{
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
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.ListWasmExtensions, &sliverpb.ListWasmExtensionsReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) ExecWasmExtension(sessionID, name string, args []string, interactive bool) (*sliverpb.ExecWasmExtension, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.ExecWasmExtension, &sliverpb.ExecWasmExtensionReq{
		Name:        name,
		Args:        args,
		Interactive: interactive,
		Request:     &commonpb.Request{SessionID: sessionID},
	})
}
