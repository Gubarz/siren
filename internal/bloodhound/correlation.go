package bloodhound

import (
	"context"
	"strings"
	"sync"
	"time"
)

// AgentRef identifies a Sliver agent for correlation purposes. It crosses the
// Wails boundary as AgentRefDTO (same JSON shape).
type AgentRef struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname,omitempty"`
	Username      string `json:"username,omitempty"`
	RemoteAddress string `json:"remoteAddress,omitempty"`
}

// Enrichment is the correlated BloodHound context for one agent. Distance is
// the hop count to the nearest Tier-0 principal, or -1 when no path exists
// (including agents that resolve to no entity at all).
type Enrichment struct {
	Entity             Entity    `json:"entity"`
	Owned              bool      `json:"owned"`
	TierZero           bool      `json:"tierZero"`
	DistanceToTierZero int       `json:"distanceToTierZero"`
	Paths              []NodeDTO `json:"paths"`
}

const defaultCorrelationTTL = 60 * time.Second

type cacheEntry struct {
	enrichment Enrichment
	expires    time.Time
}

// Correlator resolves agents to BloodHound entities with a TTL cache. It is
// owned by the Service; all HTTP I/O happens on the caller's client snapshot.
type Correlator struct {
	mu    sync.Mutex
	ttl   time.Duration
	cache map[string]cacheEntry
	refs  []AgentRef // retained for the background sync loop
}

func newCorrelator() *Correlator {
	return &Correlator{ttl: defaultCorrelationTTL, cache: map[string]cacheEntry{}}
}

func (c *Correlator) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = map[string]cacheEntry{}
}

func (c *Correlator) Refs() []AgentRef {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]AgentRef{}, c.refs...)
}

func samAccount(username string) string {
	u := strings.TrimSpace(username)
	if i := strings.IndexByte(u, '\\'); i >= 0 {
		u = u[i+1:]
	}
	if i := strings.IndexByte(u, '@'); i >= 0 {
		u = u[:i]
	}
	return strings.ToLower(strings.TrimSpace(u))
}

// candidatesFor returns the search terms for an agent in priority order: the
// username sam-account first (agents normally enrich to the logged-on user),
// then hostname candidates as fallback so sessions running as machine
// accounts, SYSTEM, or otherwise unresolvable users still resolve to their
// Computer entity.
func candidatesFor(ref AgentRef) []string {
	var cands []string
	if u := samAccount(ref.Username); u != "" {
		cands = append(cands, u)
	}
	h := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(ref.Hostname), ".")))
	if h != "" {
		short := h
		if i := strings.IndexByte(h, '.'); i >= 0 {
			short = h[:i]
		}
		cands = append(cands, short)
		if short != h {
			cands = append(cands, h)
		}
	}
	return cands
}

// pickMatch chooses the search hit that best matches the candidate: exact
// name equality, then (for hostname candidates) a Computer whose short label
// equals the candidate, then sam-account equality (e.g. candidate "jane"
// matching "JANE@CORP.LOCAL"), then the first hit. preferComputer avoids
// resolving a host like DC01 to its Container twin instead of the Computer.
func pickMatch(candidate string, hits []Entity, preferComputer bool) (Entity, bool) {
	candidate = strings.ToLower(candidate)
	for _, h := range hits {
		if strings.ToLower(h.Name) == candidate {
			return h, true
		}
	}
	if preferComputer {
		for _, h := range hits {
			if h.Kind == "Computer" && strings.ToLower(hostShortLabel(h.Name)) == candidate {
				return h, true
			}
		}
	}
	for _, h := range hits {
		if samAccount(h.Name) == candidate {
			return h, true
		}
	}
	if len(hits) > 0 {
		if preferComputer {
			for _, h := range hits {
				if h.Kind == "Computer" {
					return h, true
				}
			}
		}
		return hits[0], true
	}
	return Entity{}, false
}

// hostShortLabel returns the label before the first dot, lowercased
// ("PC01.CORP.LOCAL" → "pc01").
func hostShortLabel(name string) string {
	lower := strings.ToLower(name)
	if i := strings.IndexByte(lower, '.'); i >= 0 {
		return lower[:i]
	}
	return lower
}

// resolveCandidate searches each candidate in order and returns the first
// that yields an entity. Candidates are hostname-derived (short label
// first, full name fallback), so computer hits are preferred.
func (s *Service) resolveCandidate(ctx context.Context, cands []string) (Entity, bool, error) {
	for _, cand := range cands {
		page, err := s.SearchEntities(ctx, cand, 0, 5)
		if err != nil {
			return Entity{}, false, err
		}
		if e, ok := pickMatch(cand, page.Entities, true); ok {
			return e, true, nil
		}
	}
	return Entity{}, false, nil
}

// buildEnrichment computes distance-to-tier-zero and the ordered path for a
// resolved entity. Path queries are best-effort: failure degrades to -1.
func (s *Service) buildEnrichment(ctx context.Context, entity Entity) Enrichment {
	dist, paths := -1, []NodeDTO{}
	if graph, err := s.EntityAttackPaths(ctx, entity.ObjectID, 1); err == nil {
		ordered := orderedPathNodes(graph)
		if len(ordered) > 0 {
			dist = len(ordered) - 1
			paths = ordered
		}
	}
	return Enrichment{
		Entity:             entity,
		Owned:              entity.Owned,
		TierZero:           entity.TierZero,
		DistanceToTierZero: dist,
		Paths:              paths,
	}
}

// Correlate resolves each agent to a BloodHound entity and computes its
// distance to Tier-0. Agents sharing a canonical candidate (same host short
// label or sam-account) are batched into one search and one path query per
// distinct entity. Results are cached per agent key for the TTL.
func (s *Service) Correlate(ctx context.Context, refs []AgentRef) (map[string]Enrichment, error) {
	if _, err := s.snapshot(); err != nil {
		return nil, err
	}

	now := time.Now()
	out := map[string]Enrichment{}
	var stale []AgentRef

	s.corr.mu.Lock()
	for _, ref := range refs {
		key := cacheKey(ref)
		if entry, ok := s.corr.cache[key]; ok && entry.expires.After(now) {
			out[ref.ID] = entry.enrichment
			continue
		}
		stale = append(stale, ref)
	}
	s.corr.refs = append([]AgentRef{}, refs...)
	s.corr.mu.Unlock()

	if len(stale) == 0 {
		s.publish(EventEnrichment, out)
		return out, nil
	}

	// Group stale agents by canonical (first) candidate.
	groups := map[string][]AgentRef{}
	var order []string
	for _, ref := range stale {
		cands := candidatesFor(ref)
		if len(cands) == 0 {
			out[ref.ID] = Enrichment{DistanceToTierZero: -1}
			continue
		}
		primary := cands[0]
		if _, ok := groups[primary]; !ok {
			order = append(order, primary)
		}
		groups[primary] = append(groups[primary], ref)
	}

	resolved := map[string]Enrichment{} // objectID → enrichment (path dedupe)
	for _, primary := range order {
		refs := groups[primary]
		entity, ok, err := s.resolveCandidate(ctx, candidatesFor(refs[0]))
		if err != nil {
			return nil, err
		}
		if !ok {
			for _, ref := range refs {
				out[ref.ID] = Enrichment{DistanceToTierZero: -1}
			}
			continue
		}
		enr, ok := resolved[entity.ObjectID]
		if !ok {
			enr = s.buildEnrichment(ctx, entity)
			resolved[entity.ObjectID] = enr
		}
		for _, ref := range refs {
			out[ref.ID] = enr
		}
	}

	// Cache per agent key.
	s.corr.mu.Lock()
	for _, ref := range stale {
		s.corr.cache[cacheKey(ref)] = cacheEntry{enrichment: out[ref.ID], expires: now.Add(s.corr.ttl)}
	}
	s.corr.mu.Unlock()

	s.publish(EventEnrichment, out)
	return out, nil
}

func cacheKey(ref AgentRef) string {
	return strings.ToLower(ref.ID) + "|" + strings.ToLower(ref.Hostname) + "|" + strings.ToLower(ref.Username)
}

// InvalidateCorrelation clears the correlation cache; called after ingest or
// sync so enrichment reflects fresh data.
func (s *Service) InvalidateCorrelation() {
	s.corr.Invalidate()
}

// orderedPathNodes walks the edge list to restore path order from the
// unordered graph projection. Falls back to the given order when the walk is
// ambiguous (multiple sources) — ordering is best-effort for tooltips.
func orderedPathNodes(g GraphDTO) []NodeDTO {
	if len(g.Nodes) == 0 {
		return []NodeDTO{}
	}
	if len(g.Edges) == 0 {
		return g.Nodes
	}
	inbound := map[string]int{}
	for _, e := range g.Edges {
		inbound[e.Target]++
	}
	var start string
	for _, n := range g.Nodes {
		if inbound[n.ID] == 0 {
			start = n.ID
			break
		}
	}
	if start == "" {
		return g.Nodes
	}
	outbound := map[string]string{}
	for _, e := range g.Edges {
		outbound[e.Source] = e.Target
	}
	nodesByID := map[string]NodeDTO{}
	for _, n := range g.Nodes {
		nodesByID[n.ID] = n
	}
	ordered := make([]NodeDTO, 0, len(g.Nodes))
	seen := map[string]bool{}
	for cur := start; cur != "" && !seen[cur]; cur = outbound[cur] {
		seen[cur] = true
		if n, ok := nodesByID[cur]; ok {
			ordered = append(ordered, n)
		}
	}
	if len(ordered) != len(g.Nodes) {
		return g.Nodes
	}
	return ordered
}
