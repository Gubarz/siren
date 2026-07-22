package implants

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
	ctx context.Context
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) SetCtx(ctx context.Context) {
	s.ctx = ctx
}

func newImplantConfig(goos, goarch, format, c2url string, isBeacon bool, beaconInterval int64) (*clientpb.ImplantConfig, error) {
	formats := map[string]clientpb.OutputFormat{
		"exe":       clientpb.OutputFormat_EXECUTABLE,
		"shared":    clientpb.OutputFormat_SHARED_LIB,
		"shellcode": clientpb.OutputFormat_SHELLCODE,
		"service":   clientpb.OutputFormat_SERVICE,
	}
	outFmt, ok := formats[strings.ToLower(format)]
	if !ok {
		return nil, fmt.Errorf("unknown format %q (use exe, shared, shellcode, or service)", format)
	}

	c2url = strings.TrimSpace(c2url)
	if c2url == "" {
		return nil, fmt.Errorf("a C2 URL is required, e.g. mtls://10.0.0.1:443")
	}

	return &clientpb.ImplantConfig{
		GOOS:             strings.ToLower(goos),
		GOARCH:           strings.ToLower(goarch),
		Format:           outFmt,
		IsBeacon:         isBeacon,
		BeaconInterval:   beaconInterval,
		C2:               []*clientpb.ImplantC2{{Priority: 0, URL: c2url}},
		HTTPC2ConfigName: "default",
	}, nil
}

// GenerateRequest is the full option set exposed by the advanced generate
// modal. Fields left at their zero value are omitted from the ImplantConfig
// so the server picks its own default.
type GenerateRequest struct {
	// Target
	Name   string `json:"name"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	Format string `json:"format"` // exe|shared|shellcode|service

	// C2 — multiple URLs in priority order
	C2URLs           []string `json:"c2Urls"`
	HTTPC2ConfigName string   `json:"httpC2ConfigName"`

	// Behavior
	IsBeacon            bool   `json:"isBeacon"`
	BeaconInterval      int64  `json:"beaconInterval"` // seconds
	BeaconJitter        int64  `json:"beaconJitter"`   // seconds
	ReconnectInterval   int64  `json:"reconnectInterval"`
	PollTimeout         int64  `json:"pollTimeout"`
	MaxConnectionErrors int    `json:"maxConnectionErrors"`
	ConnectionStrategy  string `json:"connectionStrategy"` // "r"|"s"|""

	// Evasion / build
	Debug            bool `json:"debug"`
	Evasion          bool `json:"evasion"`
	ObfuscateSymbols bool `json:"obfuscateSymbols"`
	SGNEnabled       bool `json:"sgnEnabled"`
	NetGoEnabled     bool `json:"netGoEnabled"`
	RunAtLoad        bool `json:"runAtLoad"`

	// Traffic encoders / canaries
	TrafficEncodersEnabled bool     `json:"trafficEncodersEnabled"`
	TrafficEncoders        []string `json:"trafficEncoders"`
	CanaryDomains          []string `json:"canaryDomains"`

	// Constraints
	LimitDomainJoined bool   `json:"limitDomainJoined"`
	LimitHostname     string `json:"limitHostname"`
	LimitUsername     string `json:"limitUsername"`
	LimitDatetime     string `json:"limitDatetime"`
	LimitFileExists   string `json:"limitFileExists"`
	LimitLocale       string `json:"limitLocale"`
}

func newImplantConfigFromRequest(req GenerateRequest) (*clientpb.ImplantConfig, error) {
	outFmt, err := parseOutputFormat(req.Format)
	if err != nil {
		return nil, err
	}
	c2Urls, includes, err := parseC2URLs(req.C2URLs)
	if err != nil {
		return nil, err
	}

	config := baseImplantConfig(req, outFmt)
	applyC2Config(config, c2Urls, includes, req.HTTPC2ConfigName)
	applyObfuscationConfig(config, req)
	applyLimitConfig(config, req)
	return config, nil
}

// GenerateAdvanced builds an implant using the full option set from the modal
// (multiple C2 URLs, jitter, kill date, evasion toggles, constraints, etc.).
func (s *Service) GenerateAdvanced(req GenerateRequest) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}
	cfg, err := newImplantConfigFromRequest(req)
	if err != nil {
		return "", err
	}
	resp, err := s.rpc.RPC.Generate(context.Background(), &clientpb.GenerateReq{
		Name:   req.Name,
		Config: cfg,
	})
	if err != nil {
		return "", err
	}
	if resp.File == nil {
		return "", fmt.Errorf("server returned no implant file")
	}
	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Save Implant",
		DefaultFilename: resp.File.Name,
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return "", nil
	}
	if err := os.WriteFile(localPath, resp.File.Data, 0755); err != nil {
		return "", fmt.Errorf("failed to save implant: %w", err)
	}
	return localPath, nil
}

// SaveProfileAdvanced saves an implant profile with the full option set.
func (s *Service) SaveProfileAdvanced(req GenerateRequest) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("profile name is required")
	}
	cfg, err := newImplantConfigFromRequest(req)
	if err != nil {
		return err
	}
	_, err = s.rpc.RPC.SaveImplantProfile(context.Background(), &clientpb.ImplantProfile{
		Name:   req.Name,
		Config: cfg,
	})
	return err
}

func (s *Service) Generate(goos, goarch, format, c2url, name string, isBeacon bool, beaconInterval int64) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	cfg, err := newImplantConfig(goos, goarch, format, c2url, isBeacon, beaconInterval)
	if err != nil {
		return "", err
	}

	resp, err := s.rpc.RPC.Generate(context.Background(), &clientpb.GenerateReq{
		Name:   name,
		Config: cfg,
	})
	if err != nil {
		return "", err
	}
	if resp.File == nil {
		return "", fmt.Errorf("server returned no implant file")
	}

	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Save Implant",
		DefaultFilename: resp.File.Name,
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return "", nil
	}
	if err := os.WriteFile(localPath, resp.File.Data, 0755); err != nil {
		return "", fmt.Errorf("failed to save implant: %w", err)
	}
	return localPath, nil
}

func (s *Service) SaveProfile(name, goos, goarch, format, c2url string, isBeacon bool, beaconInterval int64) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	cfg, err := newImplantConfig(goos, goarch, format, c2url, isBeacon, beaconInterval)
	if err != nil {
		return err
	}

	profile := &clientpb.ImplantProfile{
		Name:   name,
		Config: cfg,
	}

	_, err = s.rpc.RPC.SaveImplantProfile(context.Background(), profile)
	return err
}

func (s *Service) GenerateFromProfile(profileConfigID string, name string, format int) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	cfg := &clientpb.ImplantConfig{
		ID:               profileConfigID,
		HTTPC2ConfigName: "default",
		Format:           clientpb.OutputFormat(format),
	}

	resp, err := s.rpc.RPC.Generate(context.Background(), &clientpb.GenerateReq{
		Name:   name,
		Config: cfg,
	})
	if err != nil {
		return "", err
	}
	if resp.File == nil {
		return "", fmt.Errorf("server returned no implant file")
	}

	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Save Implant from Profile",
		DefaultFilename: resp.File.Name,
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return "", nil
	}
	if err := os.WriteFile(localPath, resp.File.Data, 0755); err != nil {
		return "", fmt.Errorf("failed to save implant: %w", err)
	}
	return localPath, nil
}
