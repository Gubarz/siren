package files

import (
	"archive/tar"
	"bytes"
	"strings"
	"testing"
)

// A read error partway through the archive used to break the loop and report
// success, so a truncated download looked like a good one.
func TestAddDirToTarReportsAnUnreadableArchive(t *testing.T) {
	var out bytes.Buffer
	tw := tar.NewWriter(&out)

	err := addDirToTar(tw, "base", bytes.NewReader([]byte("this is not a tar archive")))
	if err == nil {
		t.Fatal("addDirToTar accepted a reader that is not a tar archive")
	}
}

// filepath.Join cleans the entry name, so an archive containing ".." could
// produce entries outside the directory the operator asked for.
func TestAddDirToTarKeepsEntriesUnderTheBaseName(t *testing.T) {
	var src bytes.Buffer
	sw := tar.NewWriter(&src)
	if err := sw.WriteHeader(&tar.Header{Name: "../escape", Mode: 0o600, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if err := sw.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	if err := addDirToTar(tw, "base", bytes.NewReader(src.Bytes())); err != nil {
		t.Fatalf("addDirToTar() = %v, want the entry rewritten rather than rejected", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	hdr, err := tar.NewReader(bytes.NewReader(out.Bytes())).Next()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(hdr.Name, "..") || !strings.HasPrefix(hdr.Name, "base/") {
		t.Fatalf("entry name = %q, want it to stay under base/", hdr.Name)
	}
}
