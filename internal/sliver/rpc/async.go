package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/protobuf/proto"
)

const asyncTaskPollInterval = 500 * time.Millisecond

// TargetRequest resolves an agent ID to the request shape expected by Sliver.
// Session calls are synchronous; beacon calls are queued and completed on the
// beacon's next callback.
func (c *Client) TargetRequest(targetID string, timeout time.Duration) (*commonpb.Request, error) {
	if strings.TrimSpace(targetID) == "" {
		return nil, fmt.Errorf("target ID is required")
	}
	if timeout <= 0 {
		timeout = time.Minute
	}

	req := &commonpb.Request{Timeout: int64(timeout - time.Nanosecond)}
	if session := c.LookupSession(targetID); session != nil {
		req.SessionID = session.ID
		return req, nil
	}
	if beacon := c.LookupBeacon(targetID); beacon != nil {
		req.Async = true
		req.BeaconID = beacon.ID
		return req, nil
	}
	return nil, fmt.Errorf("agent not found: %s", targetID)
}

// AwaitAsyncResponse leaves synchronous responses untouched and, for beacon
// responses, waits for the queued task then unmarshals its payload back into
// the same response message used by session callers.
func (c *Client) AwaitAsyncResponse(
	ctx context.Context,
	response ResponseWithError,
	target proto.Message,
) error {
	if err := CheckResponse(response); err != nil {
		return err
	}
	meta := response.GetResponse()
	if meta == nil || !meta.Async {
		return nil
	}
	if strings.TrimSpace(meta.TaskID) == "" {
		return fmt.Errorf("beacon response did not include a task ID")
	}

	ticker := time.NewTicker(asyncTaskPollInterval)
	defer ticker.Stop()

	for {
		task, err := c.RPC.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: meta.TaskID})
		if err != nil {
			return err
		}
		switch strings.ToLower(task.State) {
		case "completed":
			if len(task.Response) > 0 {
				if err := proto.Unmarshal(task.Response, target); err != nil {
					return fmt.Errorf("decode beacon task %s: %w", shortTaskID(task.ID), err)
				}
			}
			if decoded, ok := target.(ResponseWithError); ok {
				return CheckResponse(decoded)
			}
			return nil
		case "failed", "canceled":
			return fmt.Errorf("beacon task %s %s", shortTaskID(task.ID), strings.ToLower(task.State))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func shortTaskID(taskID string) string {
	if index := strings.IndexByte(taskID, '-'); index >= 0 {
		return taskID[:index]
	}
	if len(taskID) > 8 {
		return taskID[:8]
	}
	return taskID
}
