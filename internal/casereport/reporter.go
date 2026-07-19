package casereport

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/casefile"
	"sliver-gui/internal/console"
	"sliver-gui/internal/rpc"
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

func (r *reporter) AgentSummary(agentID string) string {
	if r.console == nil {
		return fmt.Sprintf("`%s` \u2014 unresolved", agentID)
	}
	sess, beacon, err := r.console.FindTarget(agentID)
	if err != nil {
		return fmt.Sprintf("`%s` \u2014 unresolved", agentID)
	}
	if sess != nil {
		return fmt.Sprintf("**Session** `%s` \u2014 %s@%s (%s %s)",
			shortID(sess.ID), sess.Username, sess.Hostname, sess.OS, sess.Arch)
	}
	if beacon != nil {
		return fmt.Sprintf("**Beacon** `%s` \u2014 %s@%s (%s %s)",
			shortID(beacon.ID), beacon.Username, beacon.Hostname, beacon.OS, beacon.Arch)
	}
	return fmt.Sprintf("`%s` \u2014 unresolved", agentID)
}

func (r *reporter) LootSummary(lootID string) string {
	if !r.connected() {
		return fmt.Sprintf("`%s` \u2014 server offline", lootID)
	}
	all, err := r.rpc.RPC.LootAll(context.Background(), &commonpb.Empty{})
	if err != nil {
		return fmt.Sprintf("`%s` \u2014 %s", lootID, err.Error())
	}
	for _, loot := range all.Loot {
		if loot.ID == lootID {
			return fmt.Sprintf("**%s** `%s` \u2014 origin=%s (%s)", loot.Name, shortID(loot.ID), loot.OriginHostUUID, loot.FileType)
		}
	}
	return fmt.Sprintf("`%s` \u2014 removed", lootID)
}

func (r *reporter) CredSummary(credID string) string {
	if !r.connected() {
		return fmt.Sprintf("`%s` \u2014 server offline", credID)
	}
	c, err := r.rpc.RPC.GetCredByID(context.Background(), &clientpb.Credential{ID: credID})
	if err != nil {
		return fmt.Sprintf("`%s` \u2014 removed", credID)
	}
	return fmt.Sprintf("`%s` \u2014 user=`%s`, hash-type=%s", shortID(c.ID), c.Username, c.HashType.String())
}

func (r *reporter) HostSummary(hostID string) string {
	if !r.connected() {
		return fmt.Sprintf("`%s` \u2014 server offline", hostID)
	}
	h, err := r.rpc.RPC.Host(context.Background(), &clientpb.Host{HostUUID: hostID})
	if err != nil {
		return fmt.Sprintf("`%s` \u2014 removed", hostID)
	}
	return fmt.Sprintf("**%s** `%s` \u2014 %s (%s)", h.Hostname, shortID(h.HostUUID), h.OSVersion, h.HostUUID)
}

func (r *reporter) CanarySummary(canaryID string) string {
	return fmt.Sprintf("`%s`", canaryID)
}

func (r *reporter) connected() bool {
	return r.rpc != nil && r.rpc.Connected()
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return strings.ToUpper(id[:8])
}
