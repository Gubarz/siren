package actions

import (
	"strings"

	"siren/internal/automation"
)

func cfgString(key string, cfg map[string]any) string {
	if v, ok := cfg[key].(string); ok {
		return v
	}
	return ""
}

func cfgFloat(key string, cfg map[string]any, fallback float64) float64 {
	switch v := cfg[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return fallback
}

func cfgBool(cfg map[string]any, key string) bool {
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	return false
}

func cfgStringList(cfg map[string]any, key string) []string {
	switch raw := cfg[key].(type) {
	case []string:
		return raw
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func cfgStringMap(cfg map[string]any, key string) map[string]string {
	raw, ok := cfg[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func renderTemplate(text string, target automation.Target) string {
	replacer := strings.NewReplacer(
		"{{id}}", target.ID,
		"{{name}}", target.Name,
		"{{hostname}}", target.Hostname,
		"{{username}}", target.Username,
		"{{os}}", target.OS,
		"{{arch}}", target.Arch,
		"{{kind}}", target.Kind,
	)
	return replacer.Replace(text)
}

func tailString(rc *automation.RunContext, n int) string {
	s := rc.OutputSoFar()
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
