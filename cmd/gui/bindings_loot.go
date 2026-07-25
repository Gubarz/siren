package gui

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/sliver/loot"
)

// ---- Loot / Credentials ----

func (a *App) GetLoot() (*clientpb.AllLoot, error) {
	return a.Loot.GetLoot()
}

func (a *App) GetHosts() (*clientpb.AllHosts, error) {
	return a.Hosts.List()
}

func (a *App) GetHost(hostUUID string) (*clientpb.Host, error) {
	return a.Hosts.Get(hostUUID)
}

func (a *App) RemoveHost(hostUUID string) error {
	return a.Hosts.Remove(hostUUID)
}

func (a *App) RemoveHostIOC(iocID string) error {
	return a.Hosts.RemoveIOC(iocID)
}

func (a *App) DownloadLoot(lootID string) (string, error) {
	return a.Loot.DownloadLoot(lootID)
}

func (a *App) GetLootContent(lootID string) (string, error) {
	const maxBytes = 1 << 20 // 1 MiB
	resp, err := a.RPC.RPC.LootContent(a.ctx, &clientpb.Loot{ID: lootID})
	if err != nil {
		return "", err
	}
	if resp.File == nil {
		return "", errors.New("loot item has no file content")
	}
	if len(resp.File.Data) > maxBytes {
		return "", errors.New("loot content too large to preview (max 1 MiB)")
	}
	return base64.StdEncoding.EncodeToString(resp.File.Data), nil
}

func (a *App) RemoveLoot(id string) error {
	return a.Loot.RemoveLoot(id)
}

func (a *App) GetCredentials() (*clientpb.Credentials, error) {
	return a.Loot.GetCredentials()
}

func (a *App) AddCredential(username, plaintext, hash, collection string) error {
	return a.Loot.AddCredential(username, plaintext, hash, collection)
}

func (a *App) RemoveCredential(id string) error {
	return a.Loot.RemoveCredential(id)
}

func (a *App) GetScreenshotData(lootID string) (string, error) {
	return a.Loot.GetScreenshotData(lootID)
}

func (a *App) GetAgentNotes() (map[string]string, error) {
	all := a.Comments.GetAllComments()
	notes := make(map[string]string)
	for key, list := range all {
		if strings.HasPrefix(key, "agent:") && len(list) > 0 {
			agentID := strings.TrimPrefix(key, "agent:")
			notes[agentID] = list[len(list)-1].Text
		}
	}
	return notes, nil
}

func (a *App) SaveAgentNote(agentID, text string) error {
	_, err := a.Comments.SetNote("agent", agentID, "Operator", text)
	if err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "comments-updated", nil)
	runtime.EventsEmit(a.ctx, "agent-notes-updated", agentID)
	return nil
}

// ---- Credentials extended ----

func (a *App) UpdateCredential(req loot.UpdateCredentialRequest) error {
	return a.Loot.UpdateCredential(req)
}

func (a *App) GetCredentialByID(id string) (*clientpb.Credential, error) {
	return a.Loot.GetCredentialByID(id)
}

func (a *App) GetCredentialsByHashType(hashType int32) (*clientpb.Credentials, error) {
	return a.Loot.GetCredentialsByHashType(hashType)
}

func (a *App) GetPlaintextCredentialsByHashType(hashType int32) (*clientpb.Credentials, error) {
	return a.Loot.GetPlaintextCredentialsByHashType(hashType)
}

func (a *App) SniffCredentialHashType(hash string) (*clientpb.Credential, error) {
	return a.Loot.SniffCredentialHashType(hash)
}
