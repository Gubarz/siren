package implants

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v3/pkg/application"

	"siren/internal/sliver/rpc"
	"siren/internal/wailsadapter"
)

type Service struct {
	rpc.Emitter
	rpc *rpc.Client
	ui  *wailsadapter.Bridge
}

func New(rpcClient *rpc.Client) *Service {
	return &Service{
		rpc:     rpcClient,
		Emitter: rpc.NewEmitter(rpcClient),
	}
}

func (s *Service) SetUI(ui *wailsadapter.Bridge) {
	s.ui = ui
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
	return s.saveImplantFile("Save Implant", resp.File.Name, resp.File.Data, req.Name)
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

	return s.saveImplantFile("Save Implant from Profile", resp.File.Name, resp.File.Data, name)
}

func (s *Service) saveImplantFile(title, defaultName string, data []byte, payloadName string) (string, error) {
	if s.ui == nil {
		return "", fmt.Errorf("ui bridge is unavailable")
	}
	localPath, err := s.ui.SaveFileDialog(&application.SaveFileDialogOptions{
		Title:    title,
		Filename: defaultName,
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return "", nil
	}
	if err := os.WriteFile(localPath, data, 0o755); err != nil {
		return "", fmt.Errorf("failed to save implant: %w", err)
	}
	s.Publish("gui.payload-built", map[string]any{"name": payloadName, "builder": "sliver"})
	return localPath, nil
}
