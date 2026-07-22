package implants

import (
	"context"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"sliver-gui/internal/sliver/rpc"
)

// isEmptyRecordErr reports whether err is the gRPC NotFound the Sliver server
// returns when a listing table (implant_builds, implant_profiles) has zero
// rows. That's not an error condition to us — the list is just empty.
func isEmptyRecordErr(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		return true
	}
	return false
}

func (s *Service) GetImplantBuilds() (*clientpb.ImplantBuilds, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	builds, err := s.rpc.RPC.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if isEmptyRecordErr(err) {
		return &clientpb.ImplantBuilds{Configs: map[string]*clientpb.ImplantConfig{}}, nil
	}
	return builds, err
}

func (s *Service) DeleteImplantBuild(name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.DeleteImplantBuild(context.Background(), &clientpb.DeleteReq{Name: name})
	return err
}

func (s *Service) GetProfiles() (*clientpb.ImplantProfiles, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	profiles, err := s.rpc.RPC.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if isEmptyRecordErr(err) {
		return &clientpb.ImplantProfiles{Profiles: []*clientpb.ImplantProfile{}}, nil
	}
	return profiles, err
}

func (s *Service) DeleteProfile(name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{Name: name})
	return err
}

func (s *Service) Regenerate(name string) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.Regenerate(context.Background(), &clientpb.RegenerateReq{ImplantName: name})
	if err != nil {
		return "", err
	}
	if resp.File == nil {
		return "", fmt.Errorf("no build artifact found for %q", name)
	}
	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Save Implant",
		DefaultFilename: resp.File.Name,
	})
	if err != nil || localPath == "" {
		return "", err
	}
	if err := os.WriteFile(localPath, resp.File.Data, 0755); err != nil {
		return "", err
	}
	return localPath, nil
}
