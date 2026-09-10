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
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/klauspost/compress/zstd"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
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
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.Crackstations, &commonpb.Empty{})
}

func (s *Service) SubmitJob(cmd *clientpb.CrackCommand) (*clientpb.CrackResponse, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.Crack, cmd)
}

func (s *Service) TaskByID(task *clientpb.CrackTask) (*clientpb.CrackTask, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackTaskByID, task)
}

func (s *Service) TaskUpdate(task *clientpb.CrackTask) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackTaskUpdate, task)
	return err
}

func (s *Service) CrackstationBenchmark(bench *clientpb.CrackBenchmark) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackstationBenchmark, bench)
	return err
}

func (s *Service) Trigger(ev *clientpb.Event) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackstationTrigger, ev)
	return err
}

func (s *Service) FilesList(filter *clientpb.CrackFile) (*clientpb.CrackFiles, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackFilesList, filter)
}

func (s *Service) FileCreate(file *clientpb.CrackFile) (*clientpb.CrackFile, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackFileCreate, file)
}

func (s *Service) FileChunkUpload(chunk *clientpb.CrackFileChunk) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackFileChunkUpload, chunk)
	return err
}

func (s *Service) FileChunkDownload(chunk *clientpb.CrackFileChunk) (*clientpb.CrackFileChunk, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackFileChunkDownload, chunk)
}

func (s *Service) FileComplete(file *clientpb.CrackFile) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackFileComplete, file)
	return err
}

func (s *Service) FileDelete(file *clientpb.CrackFile) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.CrackFileDelete, file)
	return err
}

func (s *Service) UploadFromPath(localPath string, fileType clientpb.CrackFileType) (*clientpb.CrackFile, error) {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return nil, err
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
	created, err := c.RPC().CrackFileCreate(ctx, &clientpb.CrackFile{
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
	_, err = c.RPC().CrackFileComplete(ctx, &clientpb.CrackFile{
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
	_, err := s.rpc.RPC().CrackFileChunkUpload(ctx, &clientpb.CrackFileChunk{
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
