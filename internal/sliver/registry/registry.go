package registry

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

type Value struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) ListSubKeys(sessionID, hive, path string) (*sliverpb.RegistrySubKeyList, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	req := &sliverpb.RegistrySubKeyListReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Hive:    hive,
		Path:    path,
	}

	return s.rpc.RPC.RegistryListSubKeys(context.Background(), req)
}

func (s *Service) ListValues(sessionID, hive, path string) (*sliverpb.RegistryValuesList, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	req := &sliverpb.RegistryListValuesReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Hive:    hive,
		Path:    path,
	}

	return s.rpc.RPC.RegistryListValues(context.Background(), req)
}

func (s *Service) ReadValue(sessionID, hive, path, key string) (*Value, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.RegistryRead(context.Background(), &sliverpb.RegistryReadReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	})
	if err != nil {
		return nil, err
	}
	if err := rpc.CheckResponse(resp); err != nil {
		return nil, err
	}
	return &Value{Name: key, Type: "Value", Value: resp.Value}, nil
}

func (s *Service) WriteValue(sessionID, hive, path, key, valueType, value string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	req := &sliverpb.RegistryWriteReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	}
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "string":
		req.Type = sliverpb.RegistryTypeString
		req.StringValue = value
	case "binary":
		decoded, err := hex.DecodeString(strings.ReplaceAll(value, " ", ""))
		if err != nil {
			return fmt.Errorf("binary values must be hexadecimal: %w", err)
		}
		req.Type = sliverpb.RegistryTypeBinary
		req.ByteValue = decoded
	case "dword":
		parsed, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return fmt.Errorf("invalid DWORD: %w", err)
		}
		req.Type = sliverpb.RegistryTypeDWORD
		req.DWordValue = uint32(parsed)
	case "qword":
		parsed, err := strconv.ParseUint(value, 0, 64)
		if err != nil {
			return fmt.Errorf("invalid QWORD: %w", err)
		}
		req.Type = sliverpb.RegistryTypeQWORD
		req.QWordValue = parsed
	default:
		return fmt.Errorf("unsupported registry value type %q", valueType)
	}
	resp, err := s.rpc.RPC.RegistryWrite(context.Background(), req)
	if err != nil {
		return err
	}
	return rpc.CheckResponse(resp)
}

func (s *Service) CreateKey(sessionID, hive, path, key string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.RegistryCreateKey(context.Background(), &sliverpb.RegistryCreateKeyReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	})
	if err != nil {
		return err
	}
	return rpc.CheckResponse(resp)
}

func (s *Service) ReadHive(sessionID, rootHive, requestedHive string) (*sliverpb.RegistryReadHive, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.RegistryReadHive(context.Background(), &sliverpb.RegistryReadHiveReq{
		Request:       &commonpb.Request{SessionID: sessionID},
		RootHive:      rootHive,
		RequestedHive: requestedHive,
	})
}

func (s *Service) DeleteEntry(sessionID, hive, path, key string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.RegistryDeleteKey(context.Background(), &sliverpb.RegistryDeleteKeyReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	})
	if err != nil {
		return err
	}
	return rpc.CheckResponse(resp)
}
