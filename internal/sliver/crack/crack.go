package crack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/klauspost/compress/zstd"

	"siren/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {
}

func (s *Service) Crackstations() (*clientpb.Crackstations, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.Crackstations(context.Background(), &commonpb.Empty{})
}

func (s *Service) SubmitJob(cmd *clientpb.CrackCommand) (*clientpb.CrackResponse, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.Crack(context.Background(), cmd)
}

func (s *Service) TaskByID(task *clientpb.CrackTask) (*clientpb.CrackTask, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.CrackTaskByID(context.Background(), task)
}

func (s *Service) TaskUpdate(task *clientpb.CrackTask) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.CrackTaskUpdate(context.Background(), task)
	return err
}

func (s *Service) CrackstationBenchmark(bench *clientpb.CrackBenchmark) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.CrackstationBenchmark(context.Background(), bench)
	return err
}

func (s *Service) Trigger(ev *clientpb.Event) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.CrackstationTrigger(context.Background(), ev)
	return err
}

func (s *Service) FilesList(filter *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.CrackFilesList(context.Background(), filter)
}

func (s *Service) FileCreate(file *clientpb.CrackFile) (*clientpb.CrackFile, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.CrackFileCreate(context.Background(), file)
}

func (s *Service) FileChunkUpload(chunk *clientpb.CrackFileChunk) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.CrackFileChunkUpload(context.Background(), chunk)
	return err
}

func (s *Service) FileChunkDownload(chunk *clientpb.CrackFileChunk) (*clientpb.CrackFileChunk, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.CrackFileChunkDownload(context.Background(), chunk)
}

func (s *Service) FileComplete(file *clientpb.CrackFile) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.CrackFileComplete(context.Background(), file)
	return err
}

func (s *Service) FileDelete(file *clientpb.CrackFile) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.CrackFileDelete(context.Background(), file)
	return err
}

func (s *Service) UploadFromPath(localPath string, fileType clientpb.CrackFileType) (*clientpb.CrackFile, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	if !validCrackFileType(fileType) {
		return nil, fmt.Errorf("invalid crack file type: %s", fileType)
	}

	source, stat, err := openCrackUpload(localPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = source.Close() }()

	ctx := context.Background()
	name := filepath.Base(localPath)
	created, err := s.rpc.RPC.CrackFileCreate(ctx, &clientpb.CrackFile{
		Name:             name,
		Type:             fileType,
		UncompressedSize: stat.Size(),
		IsCompressed:     true,
	})
	if err != nil {
		return nil, err
	}

	compressed, sha256Sum, err := compressCrackUpload(source)
	if err != nil {
		return nil, err
	}
	defer cleanupTempFile(compressed)

	if err := s.uploadCrackChunks(ctx, created.ID, created.ChunkSize, compressed); err != nil {
		return nil, err
	}
	_, err = s.rpc.RPC.CrackFileComplete(ctx, &clientpb.CrackFile{
		ID:       created.ID,
		Sha2_256: sha256Sum,
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func validCrackFileType(fileType clientpb.CrackFileType) bool {
	switch fileType {
	case clientpb.CrackFileType_WORDLIST,
		clientpb.CrackFileType_RULES,
		clientpb.CrackFileType_MARKOV_HCSTAT2:
		return true
	default:
		return false
	}
}

func openCrackUpload(localPath string) (*os.File, os.FileInfo, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		return nil, nil, err
	}
	if stat.IsDir() {
		return nil, nil, fmt.Errorf("crack file path is a directory: %s", localPath)
	}
	if stat.Size() < 1 {
		return nil, nil, fmt.Errorf("crack file is empty: %s", localPath)
	}
	file, err := os.Open(localPath)
	if err != nil {
		return nil, nil, err
	}
	return file, stat, nil
}

func compressCrackUpload(source *os.File) (*os.File, string, error) {
	tmpFile, err := os.CreateTemp("", "sliver-crack-upload-*")
	if err != nil {
		return nil, "", err
	}
	cleanup := true
	defer func() {
		if cleanup {
			cleanupTempFile(tmpFile)
		}
	}()

	digest := sha256.New()
	compressor, err := zstd.NewWriter(tmpFile, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		return nil, "", err
	}
	if _, err = io.Copy(compressor, io.TeeReader(source, digest)); err != nil {
		_ = compressor.Close()
		return nil, "", err
	}
	if err = compressor.Close(); err != nil {
		return nil, "", err
	}
	if _, err = tmpFile.Seek(0, 0); err != nil {
		return nil, "", err
	}

	cleanup = false
	return tmpFile, hex.EncodeToString(digest.Sum(nil)), nil
}

func (s *Service) uploadCrackChunks(ctx context.Context, fileID string, chunkSize int64, source *os.File) error {
	if chunkSize < 1 || int64(int(chunkSize)) != chunkSize {
		return fmt.Errorf("invalid crack chunk size: %d", chunkSize)
	}

	buf := make([]byte, int(chunkSize))
	for n := uint32(0); ; n++ {
		readN, readErr := io.ReadFull(source, buf)
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return readErr
		}
		if readN > 0 {
			if err := s.uploadCrackChunk(ctx, fileID, n, buf[:readN]); err != nil {
				return err
			}
		}
		if readErr == io.ErrUnexpectedEOF {
			return nil
		}
	}
}

func (s *Service) uploadCrackChunk(ctx context.Context, fileID string, n uint32, data []byte) error {
	_, err := s.rpc.RPC.CrackFileChunkUpload(ctx, &clientpb.CrackFileChunk{
		CrackFileID: fileID,
		N:           n,
		Data:        append([]byte(nil), data...),
	})
	return err
}

func cleanupTempFile(file *os.File) {
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
}
