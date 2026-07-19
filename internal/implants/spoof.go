package implants

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"sliver-gui/internal/rpc"
)

// SpoofMetadataRequest is the flat form the GUI passes for priming a
// PE-spoof-metadata source before a Windows Generate is dispatched.
// Only PE is exposed today — sliver keeps the message polymorphic on the
// wire so other executable formats can be added later without a break.
type SpoofMetadataRequest struct {
	ImplantName    string `json:"implantName"`
	ImplantBuildID string `json:"implantBuildId"`
	ResourceID     uint64 `json:"resourceId"`

	// Reference PE whose metadata / resources should be cloned onto the
	// implant binary. Server reads bytes off disk itself if only the path
	// is set; PEBytes lets the GUI upload directly.
	PESourceName  string `json:"peSourceName,omitempty"`
	PESourceBytes []byte `json:"peSourceBytes,omitempty"`
}

// PrimeSpoofMetadataFromPath is the operator-facing convenience: pick a PE
// on the operator's disk, we read it here, and forward to
// GenerateSpoofMetadata. The path is only ever touched on the GUI host —
// nothing is left on the teamserver besides the metadata payload.
func (s *Service) PrimeSpoofMetadataFromPath(implantName, sourcePath string) error {
	implantName = strings.TrimSpace(implantName)
	sourcePath = strings.TrimSpace(sourcePath)
	if implantName == "" {
		return fmt.Errorf("implant name is required")
	}
	if sourcePath == "" {
		return fmt.Errorf("source PE path is required")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", sourcePath, err)
	}
	return s.GenerateSpoofMetadata(SpoofMetadataRequest{
		ImplantName:   implantName,
		PESourceName:  filepath.Base(sourcePath),
		PESourceBytes: data,
	})
}

// GenerateSpoofMetadata primes the server-side spoof metadata store for the
// implant identified by name / build / resource. The Generate call that
// follows must reference the same identifier so it can pick up the metadata.
func (s *Service) GenerateSpoofMetadata(req SpoofMetadataRequest) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if strings.TrimSpace(req.ImplantName) == "" && strings.TrimSpace(req.ImplantBuildID) == "" {
		return fmt.Errorf("implant name or build ID is required")
	}
	config := spoofConfigFromRequest(req)
	_, err := s.rpc.RPC.GenerateSpoofMetadata(context.Background(), &clientpb.GenerateSpoofMetadataReq{
		ImplantName:    req.ImplantName,
		ImplantBuildID: req.ImplantBuildID,
		ResourceID:     req.ResourceID,
		SpoofMetadata:  config,
	})
	return err
}

func spoofConfigFromRequest(req SpoofMetadataRequest) *clientpb.SpoofMetadataConfig {
	if len(req.PESourceBytes) == 0 {
		return nil
	}
	return &clientpb.SpoofMetadataConfig{
		PE: &clientpb.PESpoofMetadataConfig{
			Source: &clientpb.SpoofMetadataFile{
				Name: req.PESourceName,
				Data: req.PESourceBytes,
			},
		},
	}
}
