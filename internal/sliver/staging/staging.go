package staging

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/wailsapp/wails/v3/pkg/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"siren/internal/sliver/rpc"
	"siren/internal/wailsadapter"
)

const maxStageBytes = 64 * 1024 * 1024

type Service struct {
	rpc stagingRPC
	ui  *wailsadapter.Bridge
}

type GenerateStageRequest struct {
	Profile       string `json:"profile"`
	Name          string `json:"name"`
	AESEncryptKey string `json:"aesEncryptKey"`
	AESEncryptIV  string `json:"aesEncryptIv"`
	RC4EncryptKey string `json:"rc4EncryptKey"`
	Compress      string `json:"compress"`
	PrependSize   bool   `json:"prependSize"`
}

type TCPListenerRequest struct {
	Host        string `json:"host"`
	Port        uint32 `json:"port"`
	Profile     string `json:"profile"`
	StagePath   string `json:"stagePath"`
	Name        string `json:"name"`
	PrependSize bool   `json:"prependSize"`
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: liveStagingRPC{client: rpc}}
}

// stagingRPC is the slice of the Sliver RPC surface the staging service
// needs. The adapter keeps the live client behind an interface so the
// service can be unit tested with a fake (mirrors internal/sliver/server).
type stagingRPC interface {
	Connected() bool
	StageImplantBuild(context.Context, *clientpb.ImplantStageReq) (*commonpb.Empty, error)
	ImplantBuilds(context.Context, *commonpb.Empty) (*clientpb.ImplantBuilds, error)
	GenerateStage(context.Context, *clientpb.GenerateStageReq) (*clientpb.Generate, error)
	StartTCPStagerListener(context.Context, *clientpb.StagerListenerReq) (*clientpb.StagerListener, error)
}

type liveStagingRPC struct {
	client *rpc.Client
}

func (r liveStagingRPC) Connected() bool {
	return r.client != nil && r.client.Connected()
}

func (r liveStagingRPC) StageImplantBuild(
	ctx context.Context,
	req *clientpb.ImplantStageReq,
) (*commonpb.Empty, error) {
	return r.client.RPC.StageImplantBuild(ctx, req)
}

func (r liveStagingRPC) ImplantBuilds(ctx context.Context, req *commonpb.Empty) (*clientpb.ImplantBuilds, error) {
	return r.client.RPC.ImplantBuilds(ctx, req)
}

func (r liveStagingRPC) GenerateStage(ctx context.Context, req *clientpb.GenerateStageReq) (*clientpb.Generate, error) {
	return r.client.RPC.GenerateStage(ctx, req)
}

func (r liveStagingRPC) StartTCPStagerListener(
	ctx context.Context,
	req *clientpb.StagerListenerReq,
) (*clientpb.StagerListener, error) {
	return r.client.RPC.StartTCPStagerListener(ctx, req)
}

func (s *Service) SetUI(ui *wailsadapter.Bridge) {
	s.ui = ui
}

func (s *Service) Close() {}

// isEmptyRecordErr reports whether err is the gRPC NotFound Sliver returns
// for a malformed implant_builds table (e.g. orphaned build rows). Treat it
// as an empty list rather than surfacing it to the UI.
func isEmptyRecordErr(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		return true
	}
	return false
}

// UnstageImplantBuild unstages one build while keeping every other staged
// build staged. Sliver's StageImplantBuild RPC first clears Stage on every
// build, so we re-submit the names that must remain staged. This
// read-modify-write is not atomic; a concurrent operator staging a build
// between the list and the re-submit will have their staging cleared —
// accepted per design (no server-side changes).
func (s *Service) UnstageImplantBuild(name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	target := strings.TrimSpace(name)
	if target == "" {
		return fmt.Errorf("build name is required")
	}
	builds, err := s.rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if isEmptyRecordErr(err) {
		builds = &clientpb.ImplantBuilds{}
		err = nil
	}
	if err != nil {
		return err
	}
	if builds.Staged == nil || !builds.Staged[target] {
		return fmt.Errorf("build %q is not staged", target)
	}
	remaining := make([]string, 0, len(builds.Staged))
	for buildName, staged := range builds.Staged {
		if staged && buildName != target {
			remaining = append(remaining, buildName)
		}
	}
	sort.Strings(remaining)
	_, err = s.rpc.StageImplantBuild(context.Background(), &clientpb.ImplantStageReq{Build: remaining})
	return err
}

// UnstageAllImplantBuilds clears the Stage flag on every implant build.
func (s *Service) UnstageAllImplantBuilds() error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.StageImplantBuild(context.Background(), &clientpb.ImplantStageReq{})
	return err
}

func (s *Service) GenerateStage(req GenerateStageRequest) (string, error) {
	resp, err := s.generateStage(req)
	if err != nil {
		return "", err
	}
	return saveGeneratedFile(s.ui, resp)
}

// StageImplantBuilds marks the named builds as staged without unstaging any
// builds already staged. Sliver's StageImplantBuild RPC replaces the entire
// staged set (it clears Stage on every build first), so we merge the new
// names with the currently staged builds before submitting.
func (s *Service) StageImplantBuilds(builds []string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	names := cleanNames(builds)
	if len(names) == 0 {
		return fmt.Errorf("at least one build is required")
	}
	current, err := s.rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if isEmptyRecordErr(err) {
		current = &clientpb.ImplantBuilds{}
		err = nil
	}
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(names)+len(current.Staged))
	for name, staged := range current.Staged {
		if staged {
			seen[name] = true
		}
	}
	for _, name := range names {
		seen[name] = true
	}
	merged := make([]string, 0, len(seen))
	for name := range seen {
		merged = append(merged, name)
	}
	sort.Strings(merged)
	_, err = s.rpc.StageImplantBuild(context.Background(), &clientpb.ImplantStageReq{Build: merged})
	return err
}

func (s *Service) StartTCPStagerListener(req TCPListenerRequest) (*clientpb.StagerListener, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	data, profileName, err := s.stageData(req)
	if err != nil {
		return nil, err
	}
	return s.rpc.StartTCPStagerListener(context.Background(), &clientpb.StagerListenerReq{
		Protocol:    clientpb.StageProtocol_TCP,
		Host:        strings.TrimSpace(req.Host),
		Port:        req.Port,
		Data:        data,
		ProfileName: profileName,
	})
}

func (s *Service) stageData(req TCPListenerRequest) ([]byte, string, error) {
	if strings.TrimSpace(req.Host) == "" || req.Port == 0 {
		return nil, "", fmt.Errorf("host and port are required")
	}
	if strings.TrimSpace(req.StagePath) != "" {
		return stageFileData(req)
	}
	if strings.TrimSpace(req.Profile) == "" {
		return nil, "", fmt.Errorf("profile or stage file is required")
	}
	resp, err := s.generateStage(GenerateStageRequest{
		Profile: req.Profile,
	})
	if err != nil {
		return nil, "", err
	}
	if resp.File == nil || len(resp.File.Data) == 0 {
		return nil, "", fmt.Errorf("server returned no stage data")
	}
	return maybePrepend(resp.File.Data, req.PrependSize), req.Profile, nil
}

func (s *Service) generateStage(req GenerateStageRequest) (*clientpb.Generate, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	profile := strings.TrimSpace(req.Profile)
	if profile == "" {
		return nil, fmt.Errorf("profile is required")
	}
	return s.rpc.GenerateStage(context.Background(), &clientpb.GenerateStageReq{
		Profile:       profile,
		Name:          strings.TrimSpace(req.Name),
		AESEncryptKey: req.AESEncryptKey,
		AESEncryptIv:  req.AESEncryptIV,
		RC4EncryptKey: req.RC4EncryptKey,
		PrependSize:   req.PrependSize,
		Compress:      strings.ToLower(strings.TrimSpace(req.Compress)),
	})
}

func stageFileData(req TCPListenerRequest) ([]byte, string, error) {
	path := strings.TrimSpace(req.StagePath)
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	if info.Size() > maxStageBytes {
		return nil, "", fmt.Errorf("stage file is larger than 64 MiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return maybePrepend(data, req.PrependSize), filepath.Base(path), nil
}

func saveGeneratedFile(ui *wailsadapter.Bridge, resp *clientpb.Generate) (string, error) {
	if resp.File == nil {
		return "", fmt.Errorf("server returned no stage file")
	}
	localPath, err := ui.SaveFileDialog(&application.SaveFileDialogOptions{
		Title:    "Save Stage",
		Filename: resp.File.Name,
	})
	if err != nil || localPath == "" {
		return localPath, err
	}
	return localPath, os.WriteFile(localPath, resp.File.Data, 0755)
}

func maybePrepend(data []byte, prepend bool) []byte {
	if !prepend {
		return data
	}
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(len(data)))
	return append(lenBuf, data...)
}

func cleanNames(names []string) []string {
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
