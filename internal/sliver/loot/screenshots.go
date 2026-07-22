package loot

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) GetScreenshotData(lootID string) (string, error) {
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

	encoded := base64.StdEncoding.EncodeToString(loot.File.Data)
	ext := "png"
	if strings.HasSuffix(strings.ToLower(loot.Name), ".jpg") || strings.HasSuffix(strings.ToLower(loot.Name), ".jpeg") {
		ext = "jpeg"
	}

	return fmt.Sprintf("data:image/%s;base64,%s", ext, encoded), nil
}
