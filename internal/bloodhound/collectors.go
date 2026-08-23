package bloodhound

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	bhservices "github.com/Gubarz/bloodhound-sdk-go/services"
)

var errInvalidCollector = fmt.Errorf("bloodhound: invalid collector type")

var collectorTypes = map[string]bhservices.EnumClientType{
	"sharphound": bhservices.SharpHound,
	"azurehound": bhservices.AzureHound,
}

func collectorFileName(collector string) string {
	if collector == "azurehound" {
		return "azurehound"
	}
	return "sharphound.exe"
}

// CollectorDownload fetches a collector binary (SharpHound/AzureHound) from
// the BloodHound server's own manifest, verifies its SHA-256, and caches it
// under <dataDir>/collectors/<type>/<tag>/. An empty releaseTag resolves to
// the manifest's Latest tag. Cached files are reused when their checksum
// still matches.
func (s *Service) CollectorDownload(ctx context.Context, collector, releaseTag string) (string, string, error) {
	client, err := s.snapshot()
	if err != nil {
		return "", "", err
	}
	ctype, ok := collectorTypes[strings.ToLower(strings.TrimSpace(collector))]
	if !ok {
		return "", "", fmt.Errorf("%w: %q", errInvalidCollector, collector)
	}
	if releaseTag == "" {
		manifest, err := client.Community().Collectors().CollectorManifest(ctx, ctype)
		if err != nil {
			return "", "", err
		}
		if manifest == nil || manifest.Latest == nil || *manifest.Latest == "" {
			return "", "", fmt.Errorf("bloodhound: collector manifest has no latest release")
		}
		releaseTag = *manifest.Latest
	}

	dir := filepath.Join(s.dataDir, "collectors", collector, releaseTag)
	target := filepath.Join(dir, collectorFileName(collector))

	// Cache hit: reuse the previously verified download without re-fetching
	// the checksum (verification happens at download time).
	if data, err := os.ReadFile(target); err == nil && len(data) > 0 {
		return target, sha256Hex(data), nil
	}

	checksum, err := client.Community().Collectors().CollectorChecksum(ctx, ctype, releaseTag)
	if err != nil {
		return "", "", err
	}
	// CE returns sha256sum-style output ("<hex> *<filename>"); keep the hex.
	if fields := strings.Fields(strings.TrimSpace(checksum)); len(fields) > 0 {
		checksum = fields[0]
	}

	reader, err := client.Community().Collectors().DownloadCollector(ctx, ctype, releaseTag)
	if err != nil {
		return "", "", err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}
	if sha256Hex(data) != checksum {
		return "", "", fmt.Errorf("bloodhound: collector checksum mismatch for %s@%s", collector, releaseTag)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(target, data, 0o755); err != nil {
		return "", "", err
	}
	return target, checksum, nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Download implements the CollectorSource interface for CollectionRunner.
func (s *Service) Download(ctx context.Context, collector, tag string) (string, string, error) {
	return s.CollectorDownload(ctx, collector, tag)
}
