package bloodhound

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// This file implements a minimal signed HTTP fetch for entity detail
// endpoints. The SDK's generated EntityInfoQueryResults model declares a
// nested props shape that does not match BloodHound CE's flat
// {"props": {key: scalar}} response,
// so the SDK's typed entity methods fail to unmarshal. We sign requests with
// the same HMAC scheme the SDK uses and decode the raw JSON ourselves.

const (
	headerAuth        = "Authorization"
	headerRequestDate = "RequestDate"
	headerSignature   = "Signature"
	headerUserAgent   = "User-Agent"

	// dateHourFormat is the HMAC date key step granularity.
	dateHourFormat = "2006-01-02T15"
)

var kindEndpoints = map[string]string{
	"User":     "users",
	"Computer": "computers",
	"Group":    "groups",
	"OU":       "ous",
	"GPO":      "gpos",
	"Domain":   "domains",
}

type rawEntityData struct {
	Kinds []string               `json:"kinds"`
	Props map[string]interface{} `json:"props"`
}

type rawEntityEnvelope struct {
	Data rawEntityData `json:"data"`
}

func (s *Service) configSnapshot() Config {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

// signHMAC mirrors the SDK's request signing: three chained HMAC-SHA256
// steps over (method+path), (date hour), and (body).
func signHMAC(tokenID, tokenKey, method, pathAndQuery string, body []byte, now time.Time, req *http.Request) {
	opKey := hmac.New(sha256.New, []byte(tokenKey))
	opKey.Write([]byte(method + pathAndQuery))
	opDigest := opKey.Sum(nil)

	dateKey := hmac.New(sha256.New, opDigest)
	dateKey.Write([]byte(now.UTC().Format(dateHourFormat)))
	dateDigest := dateKey.Sum(nil)

	finalKey := hmac.New(sha256.New, dateDigest)
	finalKey.Write(body)
	signature := base64.StdEncoding.EncodeToString(finalKey.Sum(nil))

	req.Header.Set(headerUserAgent, "siren-bloodhound")
	req.Header.Set(headerAuth, "bhesignature "+tokenID)
	req.Header.Set(headerRequestDate, now.UTC().Format(time.RFC3339))
	req.Header.Set(headerSignature, signature)
}

// fetchEntityDetail GETs the kind-specific entity endpoint and flattens
// scalar props into a string map. Returns nil when the kind has no endpoint.
func (s *Service) fetchEntityDetail(ctx context.Context, kind, objectID string) (map[string]string, error) {
	endpoint, ok := kindEndpoints[kind]
	if !ok {
		return nil, nil
	}
	cfg := s.configSnapshot()
	base, err := url.Parse(cfg.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: invalid server URL: %w", err)
	}
	// Build the target explicitly: url.JoinPath drops the leading slash on
	// an empty-path base, which would desync the HMAC input ("api/v2/..."
	// signed vs "/api/v2/..." sent — the server rejects that).
	target := *base
	target.Path = "/api/v2/" + endpoint + "/" + url.PathEscape(objectID)
	target.RawPath = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	signHMAC(cfg.TokenID, cfg.TokenKey, http.MethodGet, target.RequestURI(), nil, time.Now(), req)

	transport := http.DefaultTransport
	if cfg.InsecureTLS {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // opt-in setting
		}
	}
	httpClient := &http.Client{Timeout: 15 * time.Second, Transport: transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bloodhound: entity detail HTTP %d", resp.StatusCode)
	}
	var envelope rawEntityEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for key, raw := range envelope.Data.Props {
		switch v := raw.(type) {
		case string:
			out[key] = v
		case bool:
			out[key] = strconv.FormatBool(v)
		case float64:
			// Shortest representation: integer-ish timestamps print without
			// scientific notation, decimals keep their precision.
			out[key] = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			// arrays/nested objects are skipped; the UI shows scalars
		}
	}
	if strings.TrimSpace(strings.Join(envelope.Data.Kinds, "")) == "" && len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
