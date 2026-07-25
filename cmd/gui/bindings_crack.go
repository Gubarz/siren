package gui

import (
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// ---- Password Cracking ----

func (a *App) Crackstations() (*clientpb.Crackstations, error) {
	return a.Crack.Crackstations()
}

func (a *App) CrackSubmitJob(attackMode int32, hashType int32, hashes []string, rulesFile []byte) (*clientpb.CrackResponse, error) {
	return a.Crack.SubmitJob(&clientpb.CrackCommand{
		AttackMode: clientpb.CrackAttackMode(attackMode),
		HashType:   clientpb.HashType(hashType),
		Hashes:     hashes,
		RulesFile:  rulesFile,
	})
}

func (a *App) CrackTaskByID(id, hostUUID string) (*clientpb.CrackTask, error) {
	return a.Crack.TaskByID(&clientpb.CrackTask{ID: id, HostUUID: hostUUID})
}

func (a *App) CrackTaskCancel(id, hostUUID string) error {
	task, err := a.Crack.TaskByID(&clientpb.CrackTask{ID: id})
	if err != nil {
		return err
	}
	if hostUUID != "" {
		task.HostUUID = hostUUID
	}
	task.CompletedAt = time.Now().Unix()
	task.Err = "cancelled by GUI operator"
	return a.Crack.TaskUpdate(task)
}

func (a *App) CrackFilesList() (*clientpb.CrackFiles, error) {
	return a.Crack.FilesList(&clientpb.CrackFile{})
}

func (a *App) CrackFileCreate(name string, fileType int32, isCompressed bool, uncompressedSize int64, maxFileSize, chunkSize int64) (*clientpb.CrackFile, error) {
	return a.Crack.FileCreate(&clientpb.CrackFile{
		Name:             name,
		Type:             clientpb.CrackFileType(fileType),
		IsCompressed:     isCompressed,
		UncompressedSize: uncompressedSize,
		MaxFileSize:      maxFileSize,
		ChunkSize:        chunkSize,
	})
}

func (a *App) CrackFileChunkUpload(crackFileID string, n uint32, data []byte) error {
	return a.Crack.FileChunkUpload(&clientpb.CrackFileChunk{
		CrackFileID: crackFileID,
		N:           n,
		Data:        data,
	})
}

func (a *App) CrackFileChunkDownload(crackFileID string, n uint32) (*clientpb.CrackFileChunk, error) {
	return a.Crack.FileChunkDownload(&clientpb.CrackFileChunk{
		CrackFileID: crackFileID,
		N:           n,
	})
}

func (a *App) CrackFileComplete(fileID string) error {
	return a.Crack.FileComplete(&clientpb.CrackFile{ID: fileID})
}

func (a *App) CrackFileDelete(fileID string) error {
	return a.Crack.FileDelete(&clientpb.CrackFile{ID: fileID})
}

func (a *App) CrackFileUploadFromPath(localPath string, fileType int32) (*clientpb.CrackFile, error) {
	return a.Crack.UploadFromPath(localPath, clientpb.CrackFileType(fileType))
}

func (a *App) CrackstationTrigger(eventType string, data []byte) error {
	return a.Crack.Trigger(&clientpb.Event{EventType: eventType, Data: data})
}
