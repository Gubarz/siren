package websites

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"

	"siren/internal/sliver/rpcwrap"
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
	return s.submitContent(req, rpcpb.SliverRPCClient.WebsiteAddContent)
}

// UpdateContent replaces the bytes / content-type at an existing path.
func (s *Service) UpdateContent(req AddContentRequest) error {
	return s.submitContent(req, rpcpb.SliverRPCClient.WebsiteUpdateContent)
}

func (s *Service) submitContent(
	req AddContentRequest,
	method func(rpcpb.SliverRPCClient, context.Context, *clientpb.WebsiteAddContent, ...grpc.CallOption) (*clientpb.Website, error),
) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
	}
	entry, err := buildWebContent(req)
	if err != nil {
		return err
	}
	ctx, cancel := ctxWithTimeout()
	defer cancel()
	_, err = method(c.RPC, ctx, &clientpb.WebsiteAddContent{
		Name:     req.Name,
		Contents: map[string]*clientpb.WebContent{req.Path: entry},
	})
	return err
}

// RemoveContent drops a set of URL paths from a site.
func (s *Service) RemoveContent(name string, paths []string) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
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
	_, err = c.RPC.WebsiteRemoveContent(ctx, &clientpb.WebsiteRemoveContent{
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
