package tunneling

// Sliver's bundled SOCKS driver (client/core/socks.go) hardcodes a 10-op/sec
// rate limiter and a 4 KB read buffer per connection, which caps every TCP
// stream through the tunnel at ~40 KB/s. This file replaces that driver with
// a straight read → send/recv → write loop using a 64 KB buffer and no rate
// limiting. It talks to the same server-side SocksProxy stream, so the
// implant/server side is unchanged.

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const socksBufSize = 64 * 1024

type socksDriver struct {
	rpc       rpcpb.SliverRPCClient
	sessionID string
	username  string
	password  string
	listener  net.Listener

	ctx    context.Context
	cancel context.CancelFunc

	stream rpcpb.SliverRPC_SocksProxyClient

	mu    sync.Mutex
	conns map[uint64]net.Conn

	sendMu sync.Mutex
}

func newSocksDriver(rpc rpcpb.SliverRPCClient, sessionID, username, password string, listener net.Listener) *socksDriver {
	ctx, cancel := context.WithCancel(context.Background())
	return &socksDriver{
		rpc:       rpc,
		sessionID: sessionID,
		username:  username,
		password:  password,
		listener:  listener,
		ctx:       ctx,
		cancel:    cancel,
		conns:     map[uint64]net.Conn{},
	}
}

func (d *socksDriver) start() error {
	stream, err := d.rpc.SocksProxy(d.ctx)
	if err != nil {
		return err
	}
	d.stream = stream

	go d.recvLoop()
	go d.acceptLoop()
	return nil
}

// stop closes the local listener, cancels the gRPC stream, and drops every
// open TCP connection. Safe to call more than once.
func (d *socksDriver) stop() {
	d.cancel()
	if d.listener != nil {
		_ = d.listener.Close()
	}
	if d.stream != nil {
		_ = d.stream.CloseSend()
	}
	d.mu.Lock()
	conns := d.conns
	d.conns = map[uint64]net.Conn{}
	d.mu.Unlock()
	for _, c := range conns {
		_ = c.Close()
	}
}

// acceptLoop takes new local SOCKS connections, asks the server for a fresh
// tunnel id, and hands each off to its own goroutine that pumps data from the
// local socket into the gRPC stream.
func (d *socksDriver) acceptLoop() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			return
		}
		created, err := d.rpc.CreateSocks(d.ctx, &sliverpb.Socks{SessionID: d.sessionID})
		if err != nil {
			_ = conn.Close()
			continue
		}
		d.mu.Lock()
		d.conns[created.TunnelID] = conn
		d.mu.Unlock()
		go d.pumpToImplant(conn, created.TunnelID, created.SessionID)
	}
}

// pumpToImplant reads from the local browser/client socket and forwards each
// chunk into the gRPC stream — no rate limiting, one send per read.
func (d *socksDriver) pumpToImplant(conn net.Conn, tunnelID uint64, sessionID string) {
	defer func() {
		d.mu.Lock()
		delete(d.conns, tunnelID)
		d.mu.Unlock()
		_ = conn.Close()
	}()

	buf := make([]byte, socksBufSize)
	var seq uint64

	// The first frame carries auth material and creates the association on
	// the server. Subsequent frames only need the payload.
	firstFrame := &sliverpb.SocksData{
		Username: d.username,
		Password: d.password,
		TunnelID: tunnelID,
		Request:  &commonpb.Request{SessionID: sessionID},
	}

	for {
		n, err := conn.Read(buf)
		if n > 0 {
			if !d.sendDataFrame(firstFrame, buf[:n], seq) {
				return
			}
			seq++
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return
			}
			d.sendCloseFrame(firstFrame, seq)
			return
		}
	}
}

// sendDataFrame forwards one chunk as a SocksData frame. It reports false
// when the stream is broken and the pump should stop.
func (d *socksDriver) sendDataFrame(firstFrame *sliverpb.SocksData, chunk []byte, seq uint64) bool {
	payload := make([]byte, len(chunk))
	copy(payload, chunk)
	frame := firstFrame
	if seq > 0 {
		frame = &sliverpb.SocksData{
			TunnelID: firstFrame.TunnelID,
			Request:  firstFrame.Request,
		}
	}
	frame.Data = payload
	frame.Sequence = seq
	d.sendMu.Lock()
	err := d.stream.Send(frame)
	d.sendMu.Unlock()
	return err == nil
}

// sendCloseFrame signals the far side to close so the implant releases the
// upstream socket instead of waiting on us.
func (d *socksDriver) sendCloseFrame(firstFrame *sliverpb.SocksData, seq uint64) {
	d.sendMu.Lock()
	_ = d.stream.Send(&sliverpb.SocksData{
		TunnelID:  firstFrame.TunnelID,
		Request:   firstFrame.Request,
		Sequence:  seq,
		CloseConn: true,
	})
	d.sendMu.Unlock()
}

// recvLoop pulls SocksData frames off the gRPC stream and writes them into
// the matching local socket. Sliver multiplexes every tunnel over one stream,
// so we demux by TunnelID.
func (d *socksDriver) recvLoop() {
	for {
		data, err := d.stream.Recv()
		if err != nil {
			return
		}
		d.mu.Lock()
		conn := d.conns[data.TunnelID]
		if data.CloseConn {
			delete(d.conns, data.TunnelID)
		}
		d.mu.Unlock()

		if conn == nil {
			continue
		}
		if data.CloseConn {
			_ = conn.Close()
			continue
		}
		if _, err := conn.Write(data.Data); err != nil {
			d.mu.Lock()
			delete(d.conns, data.TunnelID)
			d.mu.Unlock()
			_ = conn.Close()
		}
	}
}
