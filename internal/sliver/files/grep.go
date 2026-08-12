package files

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
)

const maxGrepResults = 5000

const grepTimeout = 120 * time.Second

func (s *Service) GrepFiles(sessionID, pattern, path string, recursive bool, beforeLines, afterLines int32) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	request, err := s.rpc.TargetRequest(sessionID, grepTimeout)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grepTimeout)
	defer cancel()

	grep, err := s.rpc.RPC.Grep(ctx, &sliverpb.GrepReq{
		Request:       request,
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
	if err := s.rpc.AwaitAsyncResponse(ctx, grep, grep); err != nil {
		return "", err
	}

	return formatGrepResults(grep, pattern), nil
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
