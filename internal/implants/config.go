package implants

import (
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// Config-parsing helpers for newImplantConfigFromRequest — split out to
// keep generate.go under the 350-line file budget.

func parseOutputFormat(format string) (clientpb.OutputFormat, error) {
	formats := map[string]clientpb.OutputFormat{
		"exe":       clientpb.OutputFormat_EXECUTABLE,
		"shared":    clientpb.OutputFormat_SHARED_LIB,
		"shellcode": clientpb.OutputFormat_SHELLCODE,
		"service":   clientpb.OutputFormat_SERVICE,
	}
	outFmt, ok := formats[strings.ToLower(format)]
	if !ok {
		return 0, fmt.Errorf("unknown format %q (use exe, shared, shellcode, or service)", format)
	}
	return outFmt, nil
}

// c2Includes carries the six proto Include* booleans the server sets based
// on which schemes appear in the C2 URL list.
type c2Includes struct {
	MTLS, HTTP, DNS, WG, TCP, NamedPipe bool
}

func parseC2URLs(rawURLs []string) ([]*clientpb.ImplantC2, c2Includes, error) {
	c2Urls := make([]*clientpb.ImplantC2, 0, len(rawURLs))
	includes := c2Includes{}
	for i, raw := range rawURLs {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		// Sliver's server expects the tcppivot:// scheme; the GUI accepts the
		// more readable tcp-pivot:// form and normalizes here (same fixup the
		// stock CLI performs in client/command/generate/generate.go).
		if strings.HasPrefix(strings.ToLower(u), "tcp-pivot://") {
			u = "tcppivot://" + u[len("tcp-pivot://"):]
		}
		c2Urls = append(c2Urls, &clientpb.ImplantC2{Priority: uint32(i), URL: u})
		markC2Scheme(u, &includes)
	}
	if len(c2Urls) == 0 {
		return nil, c2Includes{}, fmt.Errorf("at least one C2 URL is required (e.g. mtls://10.0.0.1:443)")
	}
	return c2Urls, includes, nil
}

func markC2Scheme(url string, includes *c2Includes) {
	switch lower := strings.ToLower(url); {
	case strings.HasPrefix(lower, "mtls://"):
		includes.MTLS = true
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		includes.HTTP = true
	case strings.HasPrefix(lower, "dns://"):
		includes.DNS = true
	case strings.HasPrefix(lower, "wg://"):
		includes.WG = true
	case strings.HasPrefix(lower, "tcppivot://"):
		includes.TCP = true
	case strings.HasPrefix(lower, "namedpipe://"):
		includes.NamedPipe = true
	}
}

func baseImplantConfig(req GenerateRequest, outFmt clientpb.OutputFormat) *clientpb.ImplantConfig {
	return &clientpb.ImplantConfig{
		GOOS:                strings.ToLower(req.GOOS),
		GOARCH:              strings.ToLower(req.GOARCH),
		Format:              outFmt,
		IsBeacon:            req.IsBeacon,
		BeaconInterval:      req.BeaconInterval,
		BeaconJitter:        req.BeaconJitter,
		ReconnectInterval:   req.ReconnectInterval,
		PollTimeout:         req.PollTimeout,
		MaxConnectionErrors: uint32(req.MaxConnectionErrors),
		ConnectionStrategy:  req.ConnectionStrategy,
	}
}

func applyC2Config(config *clientpb.ImplantConfig, c2Urls []*clientpb.ImplantC2, includes c2Includes, httpC2Name string) {
	httpC2 := strings.TrimSpace(httpC2Name)
	if httpC2 == "" {
		httpC2 = "default"
	}
	config.C2 = c2Urls
	config.HTTPC2ConfigName = httpC2
	config.IncludeMTLS = includes.MTLS
	config.IncludeHTTP = includes.HTTP
	config.IncludeDNS = includes.DNS
	config.IncludeWG = includes.WG
	config.IncludeTCP = includes.TCP
	config.IncludeNamePipe = includes.NamedPipe
}

func applyObfuscationConfig(config *clientpb.ImplantConfig, req GenerateRequest) {
	// Mirror the official sliver client: --debug always disables symbol
	// obfuscation (client/command/generate/generate.go:330). Leaving garble
	// on alongside debug symbols pushes some runtime handlers past the
	// Windows nosplit stack budget and the build fails with "exit status 2".
	obfuscateSymbols := req.ObfuscateSymbols
	if req.Debug {
		obfuscateSymbols = false
	}
	config.Debug = req.Debug
	config.Evasion = req.Evasion
	config.ObfuscateSymbols = obfuscateSymbols
	config.SGNEnabled = req.SGNEnabled
	config.NetGoEnabled = req.NetGoEnabled
	config.RunAtLoad = req.RunAtLoad
	config.TrafficEncodersEnabled = req.TrafficEncodersEnabled
	config.TrafficEncoders = req.TrafficEncoders
	config.Assets = trafficEncoderAssets(req.TrafficEncoders)
	config.CanaryDomains = req.CanaryDomains
}

func applyLimitConfig(config *clientpb.ImplantConfig, req GenerateRequest) {
	config.LimitDomainJoined = req.LimitDomainJoined
	config.LimitHostname = req.LimitHostname
	config.LimitUsername = req.LimitUsername
	config.LimitDatetime = req.LimitDatetime
	config.LimitFileExists = req.LimitFileExists
	config.LimitLocale = req.LimitLocale
}

func ConfigFromGenerateRequest(req GenerateRequest) (*clientpb.ImplantConfig, error) {
	return newImplantConfigFromRequest(req)
}
