package discovery

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strings"
	"sync"
)

//go:embed data/oui.tsv.gz
var ouiRegistryData []byte

var (
	ouiRegistryOnce sync.Once
	ouiRegistry     map[string]string
)

func lookupOUI(mac string) string {
	prefix := strings.ToUpper(strings.ReplaceAll(mac, ":", ""))
	if len(prefix) < 9 {
		return ""
	}

	ouiRegistryOnce.Do(loadOUIRegistry)
	for _, length := range [...]int{9, 7, 6} {
		if vendor := ouiRegistry[prefix[:length]]; vendor != "" {
			return vendor
		}
	}
	return ""
}

func loadOUIRegistry() {
	ouiRegistry = make(map[string]string)
	reader, err := gzip.NewReader(bytes.NewReader(ouiRegistryData))
	if err != nil {
		return
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		prefix, vendor, ok := strings.Cut(scanner.Text(), "\t")
		if ok && (len(prefix) == 6 || len(prefix) == 7 || len(prefix) == 9) && vendor != "" {
			ouiRegistry[prefix] = vendor
		}
	}
}
