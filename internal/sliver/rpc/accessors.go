// Package rpc's connection accessors. The fields themselves are unexported so
// that a reader cannot race Connect swapping them.
package rpc

import (
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

// Config returns the active client config, or nil when not connected.
func (c *Client) Config() *assets.ClientConfig {
	config, _ := c.cfgVal.Load().(*assets.ClientConfig)
	return config
}

// RPC returns the active Sliver RPC client, or nil when not connected.
func (c *Client) RPC() rpcpb.SliverRPCClient {
	client, _ := c.rpcVal.Load().(rpcpb.SliverRPCClient)
	return client
}

// Conn returns the active gRPC connection, or nil when not connected.
func (c *Client) Conn() *grpc.ClientConn {
	conn, _ := c.connVal.Load().(*grpc.ClientConn)
	return conn
}

// NewForTest builds a Client already marked connected to rpcClient. Tests in
// other packages need it because the connection fields are unexported, which is
// what stops a reader racing a reconnect.
func NewForTest(rpcClient rpcpb.SliverRPCClient) *Client {
	c := NewClient()
	c.rpcVal.Store(rpcClient)
	c.connected.Store(true)
	return c
}
