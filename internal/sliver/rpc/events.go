package rpc

import (
	"context"
	"log"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func (c *Client) StartEventStream(ctx context.Context, onEvent func(*clientpb.Event)) {
	c.streamMu.Lock()
	if c.streamCancel != nil {
		c.streamCancel()
	}
	streamCtx, streamCancel := context.WithCancel(ctx)
	c.streamCancel = streamCancel
	c.streamMu.Unlock()

	go func() {
		defer streamCancel()

		stream, err := c.RPC.Events(streamCtx, &commonpb.Empty{})
		if err != nil {
			if streamCtx.Err() != nil {
				return
			}
			c.connected.Store(false)
			log.Printf("failed to open event stream: %v", err)
			if onEvent != nil {
				onEvent(&clientpb.Event{EventType: "stream-closed"})
			}
			return
		}

		for {
			ev, err := stream.Recv()
			if err != nil {
				if streamCtx.Err() != nil {
					return
				}
				c.connected.Store(false)
				if onEvent != nil {
					onEvent(&clientpb.Event{EventType: "stream-closed"})
				}
				return
			}
			if onEvent != nil {
				onEvent(ev)
			}
		}
	}()
}
