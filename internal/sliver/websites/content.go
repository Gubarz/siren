package websites

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"sliver-gui/internal/sliver/rpc"
)

// AddContentRequest describes one path we want served under a site. Only
// one of LocalPath / RawContent should be set — RawContent wins if both are.
type AddContentRequest struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ContentType string `json:"contentType"`
	LocalPath   string `json:"localPath,omitempty"`
	RawContent  []byte `json:"rawContent,omitempty"`
}

// AddContent uploads one file into a site under the given URL path. Server
// creates the site if it doesn't exist yet, so this is also our "create
// site" path.
func (s *Service) AddContent(req AddContentRequest) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	entry, err := buildWebContent(req)
	if err != nil {
		return err
	}
	ctx, cancel := ctxWithTimeout()
	defer cancel()
	_, err = s.rpc.RPC.WebsiteAddContent(ctx, &clientpb.WebsiteAddContent{
		Name:     req.Name,
		Contents: map[string]*clientpb.WebContent{req.Path: entry},
	})
	return err
}

// UpdateContent replaces the bytes / content-type at an existing path.
func (s *Service) UpdateContent(req AddContentRequest) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	entry, err := buildWebContent(req)
	if err != nil {
		return err
	}
	ctx, cancel := ctxWithTimeout()
	defer cancel()
	_, err = s.rpc.RPC.WebsiteUpdateContent(ctx, &clientpb.WebsiteAddContent{
		Name:     req.Name,
		Contents: map[string]*clientpb.WebContent{req.Path: entry},
	})
	return err
}

// RemoveContent drops a set of URL paths from a site.
func (s *Service) RemoveContent(name string, paths []string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("website name is required")
	}
	if len(paths) == 0 {
		return fmt.Errorf("at least one path is required")
	}
	ctx, cancel := ctxWithTimeout()
	defer cancel()
	_, err := s.rpc.RPC.WebsiteRemoveContent(ctx, &clientpb.WebsiteRemoveContent{
		Name:  name,
		Paths: paths,
	})
	return err
}

func buildWebContent(req AddContentRequest) (*clientpb.WebContent, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("website name is required")
	}
	if strings.TrimSpace(req.Path) == "" {
		return nil, fmt.Errorf("URL path is required")
	}
	data := req.RawContent
	if len(data) == 0 && req.LocalPath != "" {
		bytes, err := os.ReadFile(req.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", req.LocalPath, err)
		}
		data = bytes
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("content is empty — pick a file or pass raw bytes")
	}
	return &clientpb.WebContent{
		Path:        req.Path,
		ContentType: resolveContentType(req.ContentType, req.LocalPath),
		Content:     data,
		Size:        uint64(len(data)),
	}, nil
}

func resolveContentType(explicit, localPath string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if localPath == "" {
		return "application/octet-stream"
	}
	if guess := mime.TypeByExtension(filepath.Ext(localPath)); guess != "" {
		return guess
	}
	return "application/octet-stream"
}
