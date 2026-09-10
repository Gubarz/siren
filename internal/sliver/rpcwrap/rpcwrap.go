// Package rpcwrap provides the connection guard shared by Sliver RPC service
// wrappers. Every method here preserves the wrappers' contract: a
// disconnected client reports rpc.ErrNotConnected before any RPC is issued.
package rpcwrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"siren/internal/sliver/rpc"
)

// Call invokes a unary Sliver RPC method on the connected client. It returns
// ErrNotConnected, without invoking method, when the client is disconnected.
func Call[Req, Resp any](
	client *rpc.Client,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (Resp, error) {
	return CallContext(context.Background(), client, method, req)
}

// CallContext is Call with a caller-supplied context, for RPCs that carry a
// deadline or cancellation.
func CallContext[Req, Resp any](
	ctx context.Context,
	client *rpc.Client,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (Resp, error) {
	var zero Resp
	if !client.Connected() {
		return zero, rpc.ErrNotConnected
	}
	return method(client.RPC, ctx, req)
}

// CallRequired is Call after checking that value is not blank. The
// connectivity check still runs first so a disconnected client reports
// ErrNotConnected even when the input is invalid.
func CallRequired[Req, Resp any](
	client *rpc.Client,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
	value, label string,
) (Resp, error) {
	var zero Resp
	if !client.Connected() {
		return zero, rpc.ErrNotConnected
	}
	if strings.TrimSpace(value) == "" {
		return zero, fmt.Errorf("%s is required", label)
	}
	return method(client.RPC, context.Background(), req)
}

// TargetResponse is a Sliver response that can be awaited and decoded back
// into itself for beacon tasks.
type TargetResponse interface {
	rpc.ResponseWithError
	proto.Message
}

// CallTarget is Call after stamping req with the session request envelope
// for sessionID.
func CallTarget[Req proto.Message, Resp any](
	client *rpc.Client,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
	sessionID string,
) (Resp, error) {
	var zero Resp
	if !client.Connected() {
		return zero, rpc.ErrNotConnected
	}
	SetRequest(req, &commonpb.Request{SessionID: sessionID})
	return method(client.RPC, context.Background(), req)
}

// TargetCall resolves sessionID to a Sliver request envelope, stamps req
// with it, invokes the RPC under timeout, and awaits beacon (async)
// responses. The connectivity check runs before the target lookup.
func TargetCall[Req proto.Message, Resp TargetResponse](
	client *rpc.Client,
	sessionID string,
	timeout time.Duration,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (Resp, error) {
	var zero Resp
	c, err := Client(client)
	if err != nil {
		return zero, err
	}
	request, err := c.TargetRequest(sessionID, timeout)
	if err != nil {
		return zero, err
	}
	SetRequest(req, request)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := method(c.RPC, ctx, req)
	if err != nil {
		return zero, err
	}
	if err := c.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return zero, err
	}
	return resp, nil
}

// TargetAction is TargetCall for RPCs whose response is discarded.
func TargetAction[Req proto.Message, Resp TargetResponse](
	client *rpc.Client,
	sessionID string,
	timeout time.Duration,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) error {
	_, err := TargetCall(client, sessionID, timeout, method, req)
	return err
}

// SetRequest stamps req with the common Sliver request envelope. Every
// Sliver request message carries a singular "Request" field.
func SetRequest(req proto.Message, request *commonpb.Request) {
	msg := req.ProtoReflect()
	field := msg.Descriptor().Fields().ByName("Request")
	if field == nil {
		panic("rpcwrap: request message has no Request field")
	}
	msg.Set(field, protoreflect.ValueOfMessage(request.ProtoReflect()))
}

// Client returns client when it is connected, or ErrNotConnected. Use it for
// multi-step RPC flows that need TargetRequest or AwaitAsyncResponse.
func Client(client *rpc.Client) (*rpc.Client, error) {
	if !client.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return client, nil
}
