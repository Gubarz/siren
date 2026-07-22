package files

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func addDirToTar(tw *tar.Writer, baseName string, r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		hdr.Name = filepath.Join(baseName, hdr.Name)
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("failed to write header for %s: %w", hdr.Name, err)
		}
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			if _, err := io.Copy(tw, tr); err != nil {
				return fmt.Errorf("failed to write body for %s: %w", hdr.Name, err)
			}
		}
	}
	return nil
}

func addFileToTar(tw *tar.Writer, baseName string, src *os.File) error {
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	hdr := &tar.Header{
		Name:    baseName,
		Mode:    int64(info.Mode()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write header for %s: %w", baseName, err)
	}
	if _, err := io.Copy(tw, src); err != nil {
		return fmt.Errorf("failed to write data for %s: %w", baseName, err)
	}
	return nil
}
