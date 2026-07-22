package implants

import (
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func trafficEncoderAssets(names []string) []*commonpb.File {
	assets := make([]*commonpb.File, 0, len(names))
	for _, raw := range names {
		name := filepath.Base(strings.TrimSpace(raw))
		if name == "." || name == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".wasm") {
			name += ".wasm"
		}
		assets = append(assets, &commonpb.File{Name: name})
	}
	return assets
}
