package registry

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
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
	return rpcwrap.TargetCall(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RegistryListSubKeys, &sliverpb.RegistrySubKeyListReq{
		Hive: hive,
		Path: path,
	})
}

func (s *Service) ListValues(sessionID, hive, path string) (*sliverpb.RegistryValuesList, error) {
	return rpcwrap.TargetCall(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RegistryListValues, &sliverpb.RegistryListValuesReq{
		Hive: hive,
		Path: path,
	})
}

func (s *Service) ReadValue(sessionID, hive, path, key string) (*Value, error) {
	resp, err := rpcwrap.TargetCall(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RegistryRead, &sliverpb.RegistryReadReq{
		Hive: strings.ToUpper(hive),
		Path: path,
		Key:  key,
	})
	if err != nil {
		return nil, err
	}
	return &Value{Name: key, Type: "Value", Value: resp.Value}, nil
}

func (s *Service) WriteValue(sessionID, hive, path, key, valueType, value string) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
	}
	request, err := c.TargetRequest(sessionID, requestTimeout)
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
	resp, err := c.RPC().RegistryWrite(ctx, req)
	if err != nil {
		return err
	}
	return c.AwaitAsyncResponse(ctx, resp, resp)
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
	return rpcwrap.TargetAction(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RegistryCreateKey, &sliverpb.RegistryCreateKeyReq{
		Hive: strings.ToUpper(hive),
		Path: path,
		Key:  key,
	})
}

func (s *Service) ReadHive(sessionID, rootHive, requestedHive string) (*sliverpb.RegistryReadHive, error) {
	return rpcwrap.TargetCall(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RegistryReadHive, &sliverpb.RegistryReadHiveReq{
		RootHive:      rootHive,
		RequestedHive: requestedHive,
	})
}

func (s *Service) DeleteEntry(sessionID, hive, path, key string) error {
	hive = strings.ToUpper(hive)
	return rpcwrap.TargetAction(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RegistryDeleteKey, &sliverpb.RegistryDeleteKeyReq{
		Hive: hive,
		Path: path,
		Key:  key,
	})
}
