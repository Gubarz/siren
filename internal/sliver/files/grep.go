package files

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"sliver-gui/internal/sliver/rpc"
)

const maxGrepResults = 5000

func (s *Service) GrepFiles(sessionID, pattern, path string, recursive bool, beforeLines, afterLines int32) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	req := &commonpb.Request{
		Timeout: int64((120 * time.Second) / time.Second),
	}
	if sess := s.rpc.LookupSession(sessionID); sess != nil {
		req.SessionID = sessionID
	} else if beacon := s.rpc.LookupBeacon(sessionID); beacon != nil {
		req.BeaconID = sessionID
		req.Async = true
	} else {
		req.SessionID = sessionID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	grep, err := s.rpc.RPC.Grep(ctx, &sliverpb.GrepReq{
		Request:       req,
		SearchPattern: pattern,
		Path:          path,
		Recursive:     recursive,
		LinesBefore:   beforeLines,
		LinesAfter:    afterLines,
	})
	if err != nil {
		return "", err
	}
	if grep.Response != nil && grep.Response.Err != "" {
		return "", fmt.Errorf("implant error: %s", grep.Response.Err)
	}

	if grep.Response != nil && grep.Response.Async && grep.Response.TaskID != "" {
		return s.awaitGrepTask(ctx, grep, grep.Response.TaskID)
	}

	return formatGrepResults(grep, pattern), nil
}

func (s *Service) awaitGrepTask(ctx context.Context, grep *sliverpb.Grep, taskID string) (string, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		task, err := s.rpc.RPC.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return "", err
		}
		switch task.State {
		case "completed":
			if len(task.Response) > 0 {
				if err := proto.Unmarshal(task.Response, grep); err != nil {
					return "", fmt.Errorf("decode grep result: %w", err)
				}
			}
			return formatGrepResults(grep, ""), nil
		case "failed", "canceled":
			return "", fmt.Errorf("grep task %s", task.State)
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func formatGrepResults(grep *sliverpb.Grep, pattern string) string {
	if grep == nil || grep.Results == nil || len(grep.Results) == 0 {
		return ""
	}
	var out string
	total := 0
	for filePath, fileResults := range grep.Results {
		if fileResults.IsBinary {
			continue
		}
		for _, match := range fileResults.FileResults {
			if total >= maxGrepResults {
				out += fmt.Sprintf("... %d more results truncated", maxGrepResults-total)
				return out
			}
			total++
			line := fmt.Sprintf("%s:%d", filePath, match.LineNumber)

			contextLines := append([]string{}, match.LinesBefore...)
			contextLines = append(contextLines, match.Line)
			contextLines = append(contextLines, match.LinesAfter...)

			for _, cl := range contextLines {
				out += line + "\t" + cl + "\n"
				line = ""
			}
		}
	}
	return out
}
