package triggers

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"siren/internal/automation"
	"siren/internal/bus"
)

type keyword struct{ b bus.Bus }

func Keyword(b bus.Bus) automation.Trigger { return &keyword{b: b} }

func (keyword) Type() string { return "keyword" }

func (keyword) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "pattern", Label: "Pattern", Type: "string", Required: true},
		{Key: "match", Label: "Match type", Type: "select", Options: []string{"glob", "regex"}, Default: "glob"},
		{Key: "targetID", Label: "Target ID (optional)", Type: "string"},
	}
}

func (keyword) Validate(cfg map[string]any) error {
	_, err := keywordMatcher(cfg)
	return err
}

func (k *keyword) Arm(ctx context.Context, cfg map[string]any, fire func(automation.FireEvent)) error {
	matcher, err := keywordMatcher(cfg)
	if err != nil {
		return err
	}
	targetFilter, _ := cfg["targetID"].(string)
	unsub := k.b.Subscribe([]string{"gui.console-output"}, func(ev bus.Event) {
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			return
		}
		if targetFilter != "" && str(payload, "targetID") != targetFilter {
			return
		}
		if !matcher(str(payload, "tail")) {
			return
		}
		fire(automation.FireEvent{
			Target: &automation.Target{ID: str(payload, "targetID"), Kind: str(payload, "targetKind")},
			Data:   payload,
		})
	})
	defer unsub()
	<-ctx.Done()
	return ctx.Err()
}

func keywordMatcher(cfg map[string]any) (func(string) bool, error) {
	pattern, _ := cfg["pattern"].(string)
	if pattern == "" {
		return nil, fmt.Errorf("keyword pattern is required")
	}
	match, _ := cfg["match"].(string)
	if match == "regex" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("keyword regex: %w", err)
		}
		return re.MatchString, nil
	}
	if _, err := filepath.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("keyword glob: %w", err)
	}
	return func(s string) bool {
		ok, _ := filepath.Match(pattern, s)
		if ok {
			return true
		}
		for _, line := range splitLines(s) {
			if ok, _ := filepath.Match(pattern, line); ok {
				return true
			}
		}
		return false
	}, nil
}

func splitLines(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '\n' })
}
