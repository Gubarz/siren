package automation

import (
	"strings"
	"testing"
)

// Webhook headers normally carry an Authorization or API key, so exporting
// without secrets has to drop them alongside the URL.
func TestExportRulesWithoutSecretsDropsWebhookHeaders(t *testing.T) {
	e := newTestEngine(t)
	_ = e.RegisterTrigger(&fakeTrigger{typ: "manual"})
	e.RegisterAction(&fakeAction{typ: "webhook"})

	const (
		secretURL    = "https://hooks.example.test/T000/B000/abcdef"
		secretHeader = "Bearer super-secret-token"
	)
	if _, err := e.SaveRule(AutomationRule{
		Name: "notify", Trigger: "manual",
		Actions: []ActionSpec{{
			Type: "webhook",
			Config: map[string]any{
				"url":          secretURL,
				"method":       "POST",
				"headers":      map[string]string{"Authorization": secretHeader},
				"bodyTemplate": "deployed",
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	scrubbed, err := e.ExportRules(false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(scrubbed, secretHeader) {
		t.Fatalf("ExportRules(false) leaked a webhook header:\n%s", scrubbed)
	}
	if strings.Contains(scrubbed, secretURL) {
		t.Fatalf("ExportRules(false) leaked the webhook URL:\n%s", scrubbed)
	}
	if !strings.Contains(scrubbed, "bodyTemplate") {
		t.Fatalf("ExportRules(false) dropped config that is not a secret:\n%s", scrubbed)
	}

	withSecrets, err := e.ExportRules(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withSecrets, secretHeader) {
		t.Fatalf("ExportRules(true) dropped the headers, want them kept:\n%s", withSecrets)
	}
}
