package staging

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v3/pkg/application"

	"siren/internal/sliver/rpc"
	"siren/internal/wailsadapter"
)

const maxStageBytes = 64 * 1024 * 1024

type Service struct {
	rpc *rpc.Client
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
	return &Service{rpc: rpc}
}

func (s *Service) SetUI(ui *wailsadapter.Bridge) {
	s.ui = ui
}

func (s *Service) Close() {}

func (s *Service) GenerateStage(req GenerateStageRequest) (string, error) {
	resp, err := s.generateStage(req)
	if err != nil {
		return "", err
	}
	return saveGeneratedFile(s.ui, resp)
}

func (s *Service) StageImplantBuilds(builds []string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	names := cleanNames(builds)
	if len(names) == 0 {
		return fmt.Errorf("at least one build is required")
	}
	_, err := s.rpc.RPC.StageImplantBuild(context.Background(), &clientpb.ImplantStageReq{
		Build: names,
	})
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
	return s.rpc.RPC.StartTCPStagerListener(context.Background(), &clientpb.StagerListenerReq{
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
	return s.rpc.RPC.GenerateStage(context.Background(), &clientpb.GenerateStageReq{
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
