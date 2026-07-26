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
	rpc     *rpc.Client
	service interface {
		Add(ctx context.Context, name string, fileType clientpb.FileType, data []byte) (*clientpb.Loot, error)
	}
}

func NewLootWriter(rpcClient *rpc.Client) *LootWriter {
	return &LootWriter{rpc: rpcClient}
}

func (w *LootWriter) SetService(svc interface {
	Add(ctx context.Context, name string, fileType clientpb.FileType, data []byte) (*clientpb.Loot, error)
}) {
	w.service = svc
}

func (w *LootWriter) Add(ctx context.Context, name, lootType string, data []byte) error {
	if !w.rpc.Connected() {
		return fmt.Errorf("not connected")
	}
	ft := clientpb.FileType_TEXT
	if lootType == "binary" {
		ft = clientpb.FileType_BINARY
	}
	if w.service != nil {
		_, err := w.service.Add(ctx, name, ft, data)
		return err
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
