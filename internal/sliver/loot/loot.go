package loot

import (
	"context"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
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
	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Save Loot",
		DefaultFilename: fname,
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
