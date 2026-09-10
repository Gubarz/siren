package implants

import (
	"fmt"
	"os"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/wailsapp/wails/v3/pkg/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"siren/internal/sliver/rpcwrap"
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
	builds, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.ImplantBuilds, &commonpb.Empty{})
	if isEmptyRecordErr(err) {
		return &clientpb.ImplantBuilds{Configs: map[string]*clientpb.ImplantConfig{}}, nil
	}
	return builds, err
}

func (s *Service) DeleteImplantBuild(name string) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.DeleteImplantBuild, &clientpb.DeleteReq{Name: name})
	return err
}

func (s *Service) GetProfiles() (*clientpb.ImplantProfiles, error) {
	profiles, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.ImplantProfiles, &commonpb.Empty{})
	if isEmptyRecordErr(err) {
		return &clientpb.ImplantProfiles{Profiles: []*clientpb.ImplantProfile{}}, nil
	}
	return profiles, err
}

func (s *Service) DeleteProfile(name string) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.DeleteImplantProfile, &clientpb.DeleteReq{Name: name})
	return err
}

func (s *Service) Regenerate(name string) (string, error) {
	resp, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.Regenerate, &clientpb.RegenerateReq{ImplantName: name})
	if err != nil {
		return "", err
	}
	if resp.File == nil {
		return "", fmt.Errorf("no build artifact found for %q", name)
	}
	localPath, err := s.ui.SaveFileDialog(&application.SaveFileDialogOptions{
		Title:    "Save Implant",
		Filename: resp.File.Name,
	})
	if err != nil || localPath == "" {
		return "", err
	}
	if err := os.WriteFile(localPath, resp.File.Data, 0755); err != nil {
		return "", err
	}
	s.Publish("gui.payload-built", map[string]any{"name": name, "builder": "sliver"})
	return localPath, nil
}
