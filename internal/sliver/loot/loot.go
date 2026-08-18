package loot

import (
	"context"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
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

func (s *Service) GetLoot() (*clientpb.AllLoot, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.LootAll(context.Background(), &commonpb.Empty{})
}

func (s *Service) DownloadLoot(lootID string) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}
	loot, err := s.rpc.RPC.LootContent(context.Background(), &clientpb.Loot{ID: lootID})
	if err != nil {
		return "", err
	}
	if loot.File == nil {
		return "", fmt.Errorf("loot item has no file content")
	}
	fname := loot.File.Name
	if fname == "" {
		fname = loot.Name
	}
	localPath, err := s.ui.SaveFileDialog(&application.SaveFileDialogOptions{
		Title:    "Save Loot",
		Filename: fname,
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return "", nil
	}
	if err := os.WriteFile(localPath, loot.File.Data, 0644); err != nil {
		return "", err
	}
	return localPath, nil
}

func (s *Service) RemoveLoot(id string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.LootRm(context.Background(), &clientpb.Loot{ID: id})
	return err
}

const maxPreviewBytes = 1 << 20 // 1 MiB

func (s *Service) Add(ctx context.Context, name string, fileType clientpb.FileType, data []byte) (*clientpb.Loot, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.LootAdd(ctx, &clientpb.Loot{
		Name:     name,
		FileType: fileType,
		File:     &commonpb.File{Data: data},
	})
	if err != nil {
		return nil, err
	}
	s.Publish("gui.loot-added", map[string]any{
		"type":     "loot-added",
		"lootID":   resp.ID,
		"name":     resp.Name,
		"fileType": int32(resp.FileType),
		"size":     int64(len(data)),
	})
	return resp, nil
}

func (s *Service) Update(ctx context.Context, id string, name string) (*clientpb.Loot, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.LootUpdate(ctx, &clientpb.Loot{ID: id, Name: name})
}

func (s *Service) Content(ctx context.Context, id string) ([]byte, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.LootContent(ctx, &clientpb.Loot{ID: id})
	if err != nil {
		return nil, err
	}
	if resp.File == nil {
		return nil, fmt.Errorf("loot item has no file content")
	}
	if len(resp.File.Data) > maxPreviewBytes {
		return nil, fmt.Errorf("loot content too large to preview (max 1 MiB)")
	}
	return resp.File.Data, nil
}
