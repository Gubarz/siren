package loot

import (
	"context"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

const notePrefix = "note-"

func (s *Service) GetAgentNotes() (map[string]string, error) {
	all, err := s.rpc.RPC.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	notes := make(map[string]string)
	for _, item := range all.Loot {
		if strings.HasPrefix(item.Name, notePrefix) && item.File != nil {
			notes[strings.TrimPrefix(item.Name, notePrefix)] = string(item.File.Data)
		}
	}
	return notes, nil
}

func (s *Service) SaveAgentNote(agentID, text string) error {
	name := notePrefix + agentID
	entry := &clientpb.Loot{
		Name:     name,
		FileType: clientpb.FileType_TEXT,
		File:     &commonpb.File{Name: agentID + "-note.txt", Data: []byte(text)},
	}

	all, err := s.rpc.RPC.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		return err
	}
	for _, item := range all.Loot {
		if item.Name == name {
			entry.ID = item.ID
			_, err := s.rpc.RPC.LootUpdate(context.Background(), entry)
			return err
		}
	}
	_, err = s.rpc.RPC.LootAdd(context.Background(), entry)
	return err
}
