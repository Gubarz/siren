package console

import (
	"context"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) CompleteCommand(sessionID, line string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return nil, err
	}

	if sessionID != s.menuSession {
		if sessionID != "" {
			s.sliverCon.App.SwitchMenu(consts.ImplantMenu)
		} else {
			s.sliverCon.App.SwitchMenu(consts.ServerMenu)
		}
		s.menuSession = sessionID
	}

	menu := s.sliverCon.App.ActiveMenu()
	if menu == nil || menu.Command == nil {
		return nil, nil
	}

	var trie *completionTrie
	if sessionID != "" {
		trie = s.sliverCmpl
	} else {
		trie = s.serverCmpl
	}

	parts := strings.Split(line, " ")
	return trie.lookup(parts), nil
}

func (s *Service) CompletePath(sessionID, partial string) ([]string, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	var dirText, prefix, sep string
	if i := strings.LastIndexAny(partial, "/\\"); i >= 0 {
		sep = string(partial[i])
		dirText = partial[:i+1]
		prefix = partial[i+1:]
	} else {
		sep = s.sessionSep(sessionID)
		prefix = partial
	}

	listPath := dirText
	if listPath == "" {
		listPath = "."
	}

	resp, err := s.rpc.RPC.Ls(context.Background(), &sliverpb.LsReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Path:    listPath,
	})
	if err != nil {
		return nil, err
	}

	lowerPrefix := strings.ToLower(prefix)
	var out []string
	for _, f := range resp.Files {
		if f.Name == "." || f.Name == ".." {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(f.Name), lowerPrefix) {
			continue
		}
		cand := dirText + f.Name
		if f.IsDir {
			cand += sep
		}
		out = append(out, cand)
	}
	return out, nil
}

func (s *Service) sessionSep(sessionID string) string {
	if sess := s.rpc.LookupSession(sessionID); sess != nil {
		if strings.EqualFold(sess.OS, "windows") {
			return "\\"
		}
	}
	return "/"
}
