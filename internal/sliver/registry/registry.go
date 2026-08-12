package registry

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
)

type Value struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Service struct {
	rpc *rpc.Client
}

const requestTimeout = 5 * time.Minute

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) ListSubKeys(sessionID, hive, path string) (*sliverpb.RegistrySubKeyList, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return nil, err
	}
	req := &sliverpb.RegistrySubKeyListReq{
		Request: request,
		Hive:    hive,
		Path:    path,
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryListSubKeys(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) ListValues(sessionID, hive, path string) (*sliverpb.RegistryValuesList, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return nil, err
	}
	req := &sliverpb.RegistryListValuesReq{
		Request: request,
		Hive:    hive,
		Path:    path,
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryListValues(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) ReadValue(sessionID, hive, path, key string) (*Value, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryRead(ctx, &sliverpb.RegistryReadReq{
		Request: request,
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	})
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return &Value{Name: key, Type: "Value", Value: resp.Value}, nil
}

func (s *Service) WriteValue(sessionID, hive, path, key, valueType, value string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return err
	}
	req := &sliverpb.RegistryWriteReq{
		Request: request,
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	}
	if err := applyTypedValue(req, valueType, value); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryWrite(ctx, req)
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}

// applyTypedValue sets the type and typed-value fields on req from the
// operator-facing valueType ("string", "binary", "dword", "qword").
func applyTypedValue(req *sliverpb.RegistryWriteReq, valueType, value string) error {
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
	return nil
}

func (s *Service) CreateKey(sessionID, hive, path, key string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryCreateKey(ctx, &sliverpb.RegistryCreateKeyReq{
		Request: request,
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	})
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}

func (s *Service) ReadHive(sessionID, rootHive, requestedHive string) (*sliverpb.RegistryReadHive, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryReadHive(ctx, &sliverpb.RegistryReadHiveReq{
		Request:       request,
		RootHive:      rootHive,
		RequestedHive: requestedHive,
	})
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) DeleteEntry(sessionID, hive, path, key string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.RegistryDeleteKey(ctx, &sliverpb.RegistryDeleteKeyReq{
		Request: request,
		Hive:    strings.ToUpper(hive),
		Path:    path,
		Key:     key,
	})
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}
