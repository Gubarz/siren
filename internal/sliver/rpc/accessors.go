// Package rpc's connection accessors. The active connection is published as a
// single atomic pointer so that Config, RPC and Conn can never disagree about
// which session they describe.
package rpc

import (
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

// connState is the active connection. Connect publishes it in one store and
// Disconnect clears it in one store, so a reader sees a whole session or none
// of it, never a mix of the outgoing and incoming ones.
type connState struct {
	config *assets.ClientConfig
	rpc    rpcpb.SliverRPCClient
	conn   *grpc.ClientConn
}

// Config returns the active client config, or nil when there is no connection.
func (c *Client) Config() *assets.ClientConfig {
	state := c.state.Load()
	if state == nil {
		return nil
	}
	return state.config
}

// RPC returns the active Sliver RPC client, or nil when there is no connection.
func (c *Client) RPC() rpcpb.SliverRPCClient {
	state := c.state.Load()
	if state == nil {
		return nil
	}
	return state.rpc
}

// Conn returns the active gRPC connection, or nil when there is no connection.
func (c *Client) Conn() *grpc.ClientConn {
	state := c.state.Load()
	if state == nil {
		return nil
	}
	return state.conn
}

// NewForTest builds a Client already marked connected to rpcClient. Tests in
// other packages need it because the connection state is unexported, which is
// what stops a reader racing a reconnect.
func NewForTest(rpcClient rpcpb.SliverRPCClient) *Client {
	c := NewClient()
	c.state.Store(&connState{rpc: rpcClient})
	return c
}
