package casefile

import (
	"fmt"
	"strings"
	"time"
)

// Reporter is anything that can hand back structured payloads for the
// ids we're referencing. The Service doesn't know about sliver's loot/
// creds/hosts services — callers wire this up at report-generation time.
type Reporter interface {
	AgentSummary(agentID string) string
	LootSummary(lootID string) string
	CredSummary(credID string) string
	HostSummary(hostID string) string
	CanarySummary(canaryID string) string
}

// GenerateMarkdown builds a self-contained case report. Empty sections
// are dropped so a lightly-tagged case produces a short document.
func (s *Service) GenerateMarkdown(caseID string, r Reporter) (string, error) {
	c := s.Get(caseID)
	if c == nil {
		return "", fmt.Errorf("case %s not found", caseID)
	}
	var b strings.Builder
	writeHeader(&b, c)
	writeSection(&b, "Agents", c.AgentIDs, r.AgentSummary)
	writeSection(&b, "Loot", c.LootIDs, r.LootSummary)
	writeSection(&b, "Credentials", c.CredIDs, r.CredSummary)
	writeSection(&b, "Hosts", c.HostIDs, r.HostSummary)
	writeSection(&b, "Canaries", c.CanaryIDs, r.CanarySummary)
	if strings.TrimSpace(c.Notes) != "" {
		fmt.Fprintf(&b, "\n## Notes\n\n%s\n", c.Notes)
	}
	return b.String(), nil
}

func writeHeader(b *strings.Builder, c *Record) {
	fmt.Fprintf(b, "# %s\n\n", c.Name)
	if c.Description != "" {
		fmt.Fprintf(b, "%s\n\n", c.Description)
	}
	fmt.Fprintf(b, "- **Created:** %s\n", formatTS(c.CreatedAt))
	fmt.Fprintf(b, "- **Updated:** %s\n", formatTS(c.UpdatedAt))
	fmt.Fprintf(b, "- **Agents:** %d · **Loot:** %d · **Credentials:** %d · **Hosts:** %d · **Canaries:** %d\n",
		len(c.AgentIDs), len(c.LootIDs), len(c.CredIDs), len(c.HostIDs), len(c.CanaryIDs))
}

func writeSection(b *strings.Builder, title string, ids []string, summarize func(string) string) {
	if len(ids) == 0 {
		return
	}
	fmt.Fprintf(b, "\n## %s\n\n", title)
	for _, id := range ids {
		summary := summarize(id)
		if summary == "" {
			summary = fmt.Sprintf("`%s` — (record not resolvable)", id)
		}
		fmt.Fprintf(b, "- %s\n", summary)
	}
}

func formatTS(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return time.UnixMilli(ms).Format(time.RFC3339)
}
