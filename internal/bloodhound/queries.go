package bloodhound

import (
	"fmt"
	"regexp"
	"strings"

	bh "github.com/Gubarz/bloodhound-sdk-go"
)

// CommunityKind names the built-in BloodHound community quick queries.
type CommunityKind string

const (
	CommunityKerberoastable CommunityKind = "kerberoastable"
	CommunityASREP          CommunityKind = "asrep"
	CommunityDCSync         CommunityKind = "dcsync"
	CommunityUnconstrained  CommunityKind = "unconstrained_delegation"
)

var communityCypher = map[CommunityKind]func() string{
	CommunityKerberoastable: bh.CommunityQueries.KerberoastableAccounts,
	CommunityASREP:          bh.CommunityQueries.ASREPRoastableAccounts,
	CommunityDCSync:         bh.CommunityQueries.DCSyncPrincipals,
	CommunityUnconstrained:  bh.CommunityQueries.UnconstrainedDelegation,
}

func CommunityCypher(kind CommunityKind) (string, bool) {
	fn, ok := communityCypher[kind]
	if !ok {
		return "", false
	}
	return fn(), true
}

// BloodHound object IDs are SIDs or GUIDs. Anything else is rejected outright
// so no user input ever reaches the query string unvalidated.
var (
	sidPattern  = regexp.MustCompile(`^S-1-5-[0-9-]+$`)
	guidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

func validObjectID(id string) bool {
	return sidPattern.MatchString(id) || guidPattern.MatchString(id)
}

const defaultMaxPaths = 5
const maxMaxPaths = 20

func clampMaxPaths(n int) int {
	if n <= 0 {
		return defaultMaxPaths
	}
	if n > maxMaxPaths {
		return maxMaxPaths
	}
	return n
}

// attackPathEdgeKinds is the allow-list of exploitable relationship kinds.
// Paths are only meaningful when every hop is an edge an attacker can use.
var attackPathEdgeKinds = []string{
	"AdminTo", "HasSession", "CanRDP", "CanPSRemote", "ExecuteDCOM",
	"AllowedToDelegate", "AllowedToAct", "GenericAll", "GenericWrite",
	"WriteDacl", "WriteOwner", "Owns", "MemberOf", "AddMember",
	"ForceChangePassword", "DCSync", "GetChanges", "GetChangesAll",
	"GPLink", "TrustedBy",
}

// AttackPathsCypher returns up to maxPaths shortest paths from the entity to
// any Tier-0 principal, restricted to exploitable edge kinds via the
// relationship-type alternation. The object ID must be a valid SID/GUID: it
// is substituted after validation, and the edge list is a fixed literal, so
// the string cannot be injected.
//
// Shape notes verified against BloodHound CE:
//   - the tier-zero target is matched in its own MATCH clause so shortestPath
//     never sees start == end rows (Neo4j rejects those with an execution
//     error);
//   - edge restriction happens in the relationship pattern — the server
//     rejects all(r IN relationships(p) ...) predicates.
func AttackPathsCypher(objectID string, maxPaths int) (string, error) {
	if !validObjectID(objectID) {
		return "", fmt.Errorf("bloodhound: invalid object id %q", objectID)
	}
	return fmt.Sprintf(
		"MATCH (s) WHERE s.objectid = %q "+
			"MATCH (t) WHERE t <> s AND any(tag IN coalesce(t.system_tags, []) WHERE tag = 'admin_tier_0') "+
			"MATCH p = shortestPath((s)-[:%s*1..15]->(t)) "+
			"RETURN p LIMIT %d",
		objectID, strings.Join(attackPathEdgeKinds, "|"), clampMaxPaths(maxPaths),
	), nil
}
