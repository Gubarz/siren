// Package rpctest runs an in-memory Sliver RPC server so tests can exercise the
// capture decorator and the domain wrappers over real gRPC.
//
// The alternative used elsewhere in this repo is a stub that embeds the client
// interface, which means every method the test does not stub panics with a nil
// dereference. Serving a real interface gets grpc status codes, streaming and
// metadata for free, and leaves the untouched methods answering Unimplemented.
package rpctest

import (
	"context"
	"net"
	"testing"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Harness is an in-memory teamserver and a client attached to it.
type Harness struct {
	Client rpcpb.SliverRPCClient
	Conn   *grpc.ClientConn
}

// Start serves impl on an in-memory listener and returns a client for it.
//
// Embed rpcpb.UnimplementedSliverRPCServer in impl and override only the methods
// the test cares about. The server is stopped and the connection closed when the
// test ends.
func Start(t *testing.T, impl rpcpb.SliverRPCServer) *Harness {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	rpcpb.RegisterSliverRPCServer(server, impl)
	go func() { _ = server.Serve(listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		server.Stop()
		t.Fatalf("rpctest: dial: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		server.Stop()
	})
	return &Harness{Client: rpcpb.NewSliverRPCClient(conn), Conn: conn}
}
