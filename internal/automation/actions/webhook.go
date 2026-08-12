package actions

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"siren/internal/automation"
)

const (
	webhookMaxResponseBody = 64 * 1024
	webhookOutputTail      = 2 * 1024
)

type webhook struct{}

func Webhook() automation.Action { return webhook{} }

func (webhook) Type() string { return "webhook" }

func (webhook) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "url", Label: "Webhook URL", Type: "string", Required: true},
		{Key: "method", Label: "Method", Type: "select", Options: []string{"POST", "GET"}, Default: "POST"},
		{Key: "preset", Label: "Preset", Type: "select", Options: []string{"generic", "slack", "discord"}, Default: "generic"},
		{Key: "bodyTemplate", Label: "Body template", Type: "string"},
		{Key: "timeoutSeconds", Label: "Timeout (s)", Type: "number", Default: 10},
		{Key: "retries", Label: "Retries", Type: "number", Default: 1},
	}
}

func (webhook) Execute(rc *automation.RunContext) error {
	cfg := rc.Action.Config
	url := cfgString("url", cfg)
	if url == "" {
		return fmt.Errorf("webhook url is required")
	}
	method := cfgString("method", cfg)
	if method == "" {
		method = http.MethodPost
	}
	body := renderWebhookBody(rc, cfg)
	timeout := time.Duration(cfgFloat("timeoutSeconds", cfg, 10)) * time.Second
	retries := int(cfgFloat("retries", cfg, 1))
	var lastErr error
	backoff := time.Second
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			select {
			case <-rc.Ctx.Done():
				return rc.Ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 4
		}
		lastErr = webhookOnce(rc, method, url, body, cfg, timeout)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func webhookOnce(rc *automation.RunContext, method, url, body string, cfg map[string]any, timeout time.Duration) error {
	if rc.Deps.HTTP == nil {
		return fmt.Errorf("webhook: no HTTP client configured")
	}
	ctx, cancel := context.WithTimeout(rc.Ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: request failed")
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range cfgStringMap(cfg, "headers") {
		req.Header.Set(key, value)
	}
	resp, err := rc.Deps.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: request failed")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, webhookMaxResponseBody))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: HTTP %d", resp.StatusCode)
	}
	rc.Log(fmt.Sprintf("webhook: %s %s → %d", method, url, resp.StatusCode))
	return nil
}

func renderWebhookBody(rc *automation.RunContext, cfg map[string]any) string {
	body := cfgString("bodyTemplate", cfg)
	if body == "" {
		switch cfgString("preset", cfg) {
		case "slack":
			body = `{"text": "[{{rule.name}}] fired on {{target.name}} ({{trigger}})"}`
		case "discord":
			body = `{"content": "[{{rule.name}}] fired on {{target.name}} ({{trigger}})"}`
		default:
			return ""
		}
	}
	vars := webhookVars(rc)
	for key, value := range vars {
		body = strings.ReplaceAll(body, "{{"+key+"}}", value)
	}
	return body
}

func webhookVars(rc *automation.RunContext) map[string]string {
	return map[string]string{
		"target.id":       rc.Target.ID,
		"target.name":     rc.Target.Name,
		"target.hostname": rc.Target.Hostname,
		"target.username": rc.Target.Username,
		"target.os":       rc.Target.OS,
		"target.arch":     rc.Target.Arch,
		"target.kind":     rc.Target.Kind,
		"rule.name":       rc.Rule.Name,
		"trigger":         rc.Trigger,
		"run.status":      "running",
		"run.output_tail": tailString(rc, webhookOutputTail),
	}
}
