package automationexec

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/sliver/rpc"
)

type LootWriter struct {
	rpc *rpc.Client
}

func NewLootWriter(rpcClient *rpc.Client) *LootWriter {
	return &LootWriter{rpc: rpcClient}
}

func (w *LootWriter) Add(ctx context.Context, name, lootType string, data []byte) error {
	if !w.rpc.Connected() {
		return fmt.Errorf("not connected")
	}
	ft := clientpb.FileType_TEXT
	switch lootType {
	case "binary":
		ft = clientpb.FileType_BINARY
	case "credential":
	default:
	}
	_, err := w.rpc.RPC.LootAdd(ctx, &clientpb.Loot{
		Name:     name,
		FileType: ft,
		File:     &commonpb.File{Data: data},
	})
	return err
}

func (w *LootWriter) List(ctx context.Context) ([]automation.LootItem, error) {
	if !w.rpc.Connected() {
		return nil, fmt.Errorf("not connected")
	}
	all, err := w.rpc.RPC.LootAll(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	items := make([]automation.LootItem, 0, len(all.Loot))
	for _, l := range all.Loot {
		items = append(items, automation.LootItem{Name: l.Name, Type: l.FileType.String()})
	}
	return items, nil
}
