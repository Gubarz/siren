package casereport

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/localstate/casefile"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
)

// NewReporter resolves the IDs stored on a Case back into human-readable
// summaries at report-generation time. Lookups are best effort: if a record
// has been deleted server-side, the report still renders the raw ID.
func NewReporter(con *console.Service, rpcClient *rpc.Client) casefile.Reporter {
	return &reporter{
		console: con,
		rpc:     rpcClient,
	}
}

type reporter struct {
	console *console.Service
	rpc     *rpc.Client
}

const noteLootPrefix = "note-"

func (r *reporter) AgentSummary(agentID string) string {
	if r.console == nil {
		return fmt.Sprintf("`%s` - unresolved", agentID)
	}
	sess, beacon, err := r.console.FindTarget(agentID)
	if err != nil {
		return fmt.Sprintf("`%s` - unresolved", agentID)
	}
	if sess != nil {
		return fmt.Sprintf("**Session** `%s` - %s@%s (%s %s)",
			shortID(sess.ID), sess.Username, sess.Hostname, sess.OS, sess.Arch)
	}
	if beacon != nil {
		return fmt.Sprintf("**Beacon** `%s` - %s@%s (%s %s)",
			shortID(beacon.ID), beacon.Username, beacon.Hostname, beacon.OS, beacon.Arch)
	}
	return fmt.Sprintf("`%s` - unresolved", agentID)
}

func (r *reporter) LootSummary(lootID string) string {
	if !r.connected() {
		return fmt.Sprintf("`%s` - server offline", lootID)
	}
	all, err := r.rpc.RPC.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		return fmt.Sprintf("`%s` - %s", lootID, err.Error())
	}
	for _, loot := range all.Loot {
		if loot.ID == lootID {
			return r.formatLoot(loot)
		}
	}
	return fmt.Sprintf("`%s` - removed", lootID)
}

func (r *reporter) formatLoot(loot *clientpb.Loot) string {
	parts := []string{
		fmt.Sprintf("**%s** `%s`", r.lootTitle(loot), shortID(loot.GetID())),
		strings.ToLower(loot.GetFileType().String()),
	}
	if loot.GetSize() > 0 {
		parts = append(parts, humanSize(loot.GetSize()))
	}
	if origin := r.originHostLabel(loot.GetOriginHostUUID()); origin != "" {
		parts = append(parts, "from "+origin)
	}
	if preview := r.lootPreview(loot); preview != "" {
		parts = append(parts, fmt.Sprintf("preview=`%s`", inlineCode(preview)))
	} else if loot.GetFileType() == clientpb.FileType_TEXT {
		parts = append(parts, "text preview unavailable")
	}
	return strings.Join(parts, " - ")
}

func (r *reporter) lootTitle(loot *clientpb.Loot) string {
	name := textOr(loot.GetName(), "(unnamed loot)")
	if !strings.HasPrefix(name, noteLootPrefix) {
		return name
	}
	targetID := strings.TrimPrefix(name, noteLootPrefix)
	if label := r.agentLabel(targetID); label != "" {
		return "Agent note for " + label
	}
	if short := shortID(targetID); short != "" {
		return "Agent note " + short
	}
	return "Agent note"
}

func (r *reporter) lootPreview(loot *clientpb.Loot) string {
	if loot.GetFileType() != clientpb.FileType_TEXT {
		return ""
	}
	if file := loot.GetFile(); file != nil && len(file.GetData()) > 0 {
		return oneLinePreview(string(file.GetData()), 160)
	}
	if !r.connected() {
		return ""
	}
	full, err := r.rpc.RPC.LootContent(context.Background(), &clientpb.Loot{ID: loot.GetID()})
	if err != nil || full.GetFile() == nil {
		return ""
	}
	return oneLinePreview(string(full.GetFile().GetData()), 160)
}

func (r *reporter) CredSummary(credID string) string {
	if !r.connected() {
		return fmt.Sprintf("`%s` - server offline", credID)
	}
	c, err := r.rpc.RPC.GetCredByID(context.Background(), &clientpb.Credential{ID: credID})
	if err != nil {
		return fmt.Sprintf("`%s` - removed", credID)
	}
	return r.formatCredential(c)
}

func (r *reporter) formatCredential(c *clientpb.Credential) string {
	parts := []string{fmt.Sprintf("`%s`", shortID(c.GetID()))}
	if c.GetUsername() != "" {
		parts = append(parts, fmt.Sprintf("user=`%s`", inlineCode(c.GetUsername())))
	}
	if c.GetPlaintext() != "" {
		parts = append(parts, fmt.Sprintf("plaintext=`%s`", inlineCode(c.GetPlaintext())))
	}
	if c.GetHash() != "" {
		parts = append(parts, fmt.Sprintf("hash=`%s`", inlineCode(c.GetHash())))
	}
	parts = append(parts, "hash-type="+c.GetHashType().String())
	if c.GetCollection() != "" {
		parts = append(parts, fmt.Sprintf("collection=`%s`", inlineCode(c.GetCollection())))
	}
	if c.GetIsCracked() {
		parts = append(parts, "cracked")
	}
	if origin := r.originHostLabel(c.GetOriginHostUUID()); origin != "" {
		parts = append(parts, "from "+origin)
	}
	return strings.Join(parts, " - ")
}

func (r *reporter) HostSummary(hostID string) string {
	if !r.connected() {
		return fmt.Sprintf("`%s` - server offline", hostID)
	}
	h, err := r.rpc.RPC.Host(context.Background(), &clientpb.Host{HostUUID: hostID})
	if err != nil {
		return fmt.Sprintf("`%s` - removed", hostID)
	}
	return fmt.Sprintf("**%s** `%s` - %s (%s)", h.Hostname, shortID(h.HostUUID), h.OSVersion, h.HostUUID)
}

func (r *reporter) CanarySummary(canaryID string) string {
	return fmt.Sprintf("`%s`", canaryID)
}

func (r *reporter) connected() bool {
	return r.rpc != nil && r.rpc.Connected()
}

func (r *reporter) originHostLabel(hostID string) string {
	if hostID == "" || hostID == "00000000-0000-0000-0000-000000000000" || !r.connected() {
		return ""
	}
	host, err := r.rpc.RPC.Host(context.Background(), &clientpb.Host{HostUUID: hostID})
	if err == nil && host.GetHostname() != "" {
		return fmt.Sprintf("**%s** `%s`", host.GetHostname(), shortID(hostID))
	}
	return fmt.Sprintf("`%s`", shortID(hostID))
}

func (r *reporter) agentLabel(agentID string) string {
	if agentID == "" {
		return ""
	}
	if r.console != nil {
		sess, beacon, err := r.console.FindTarget(agentID)
		if err == nil {
			if sess != nil {
				return agentDisplayName(sess.GetUsername(), sess.GetHostname(), sess.GetID())
			}
			if beacon != nil {
				return agentDisplayName(beacon.GetUsername(), beacon.GetHostname(), beacon.GetID())
			}
		}
	}
	if !r.connected() {
		return ""
	}
	ctx := context.Background()
	if sessions, err := r.rpc.RPC.GetSessions(ctx, &commonpb.Empty{}); err == nil {
		for _, sess := range sessions.GetSessions() {
			if sess.GetID() == agentID || sess.GetUUID() == agentID {
				return agentDisplayName(sess.GetUsername(), sess.GetHostname(), sess.GetID())
			}
		}
	}
	if beacons, err := r.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{}); err == nil {
		for _, beacon := range beacons.GetBeacons() {
			if beacon.GetID() == agentID || beacon.GetUUID() == agentID {
				return agentDisplayName(beacon.GetUsername(), beacon.GetHostname(), beacon.GetID())
			}
		}
	}
	return ""
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return strings.ToUpper(id[:8])
}

func textOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func oneLinePreview(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func inlineCode(value string) string {
	return strings.ReplaceAll(value, "`", "'")
}

func agentDisplayName(username, hostname, id string) string {
	switch {
	case username != "" && hostname != "":
		return username + "@" + hostname
	case hostname != "":
		return hostname
	case username != "":
		return username
	default:
		return shortID(id)
	}
}

func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
