package gui

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"siren/internal/sliver/loot"
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
	data, err := a.Loot.Content(a.ctx, lootID)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) RemoveLoot(id string) error {
	return a.Loot.RemoveLoot(id)
}

func (a *App) LootAdd(name string, fileType int32, dataBase64 string) (*clientpb.Loot, error) {
	const maxDecodedBytes = 10 << 20 // 10 MiB
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	if len(data) > maxDecodedBytes {
		return nil, fmt.Errorf("data too large (max 10 MiB)")
	}
	return a.Loot.Add(a.ctx, name, clientpb.FileType(fileType), data)
}

func (a *App) LootUpdate(id string, name string) (*clientpb.Loot, error) {
	return a.Loot.Update(a.ctx, id, name)
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
