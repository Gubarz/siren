package shellcode

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/sliver/rpc"
)

const maxInputBytes = 64 * 1024 * 1024

type RDIRequest struct {
	LocalPath    string `json:"localPath"`
	FunctionName string `json:"functionName"`
	Arguments    string `json:"arguments"`
}

type EncodeRequest struct {
	LocalPath    string `json:"localPath"`
	Encoder      int32  `json:"encoder"`
	Architecture string `json:"architecture"`
	Iterations   uint32 `json:"iterations"`
	BadCharsHex  string `json:"badCharsHex"`
}

func EncoderMap(client *rpc.Client) (*clientpb.ShellcodeEncoderMap, error) {
	if !client.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return client.RPC.ShellcodeEncoderMap(context.Background(), &commonpb.Empty{})
}

func GenerateRDI(ctx context.Context, client *rpc.Client, req RDIRequest) (string, error) {
	if !client.Connected() {
		return "", rpc.ErrNotConnected
	}
	data, name, err := readInput(req.LocalPath)
	if err != nil {
		return "", err
	}
	resp, err := client.RPC.ShellcodeRDI(context.Background(), &clientpb.ShellcodeRDIReq{
		Data: data, FunctionName: req.FunctionName, Arguments: req.Arguments,
	})
	if err != nil {
		return "", err
	}
	return saveBytes(ctx, resp.GetData(), "Save RDI Shellcode", outputName(name, "rdi"))
}

func Encode(ctx context.Context, client *rpc.Client, req EncodeRequest) (string, error) {
	if !client.Connected() {
		return "", rpc.ErrNotConnected
	}
	data, name, err := readInput(req.LocalPath)
	if err != nil {
		return "", err
	}
	badChars, err := decodeHexBytes(req.BadCharsHex)
	if err != nil {
		return "", err
	}
	resp, err := encodeBytes(client, req, data, badChars)
	if err != nil {
		return "", err
	}
	return saveBytes(ctx, resp.GetData(), "Save Encoded Shellcode", outputName(name, "encoded"))
}

func encodeBytes(client *rpc.Client, req EncodeRequest, data []byte, badChars []byte) (*clientpb.ShellcodeEncode, error) {
	resp, err := client.RPC.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
		Encoder:      clientpb.ShellcodeEncoder(req.Encoder),
		Architecture: req.Architecture,
		Iterations:   req.Iterations,
		BadChars:     badChars,
		Data:         data,
	})
	if err != nil {
		return nil, err
	}
	if resp.GetResponse().GetErr() != "" {
		return nil, errors.New(resp.GetResponse().GetErr())
	}
	return resp, nil
}

func readInput(localPath string) ([]byte, string, error) {
	path := strings.TrimSpace(localPath)
	if path == "" {
		return nil, "", fmt.Errorf("input file is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("%s is empty", filepath.Base(path))
	}
	if len(data) > maxInputBytes {
		return nil, "", fmt.Errorf("%s is larger than 64 MiB", filepath.Base(path))
	}
	return data, filepath.Base(path), nil
}

func saveBytes(ctx context.Context, data []byte, title string, name string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("server returned no shellcode bytes")
	}
	localPath, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title: title, DefaultFilename: name,
	})
	if err != nil || localPath == "" {
		return localPath, err
	}
	return localPath, os.WriteFile(localPath, data, 0644)
}

func decodeHexBytes(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "0x")
	trimmed = strings.TrimPrefix(trimmed, "0X")
	trimmed = strings.NewReplacer(" ", "", ",", "", "\\x", "").Replace(trimmed)
	if trimmed == "" {
		return nil, nil
	}
	return hex.DecodeString(trimmed)
}

func outputName(inputName string, suffix string) string {
	ext := filepath.Ext(inputName)
	base := strings.TrimSuffix(inputName, ext)
	if base == "" {
		base = "shellcode"
	}
	return base + "." + suffix + ".bin"
}
