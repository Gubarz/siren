package bloodhound

import (
	"context"
	"errors"
	"fmt"
	"strings"

	bhservices "github.com/Gubarz/bloodhound-sdk-go/services"
)

var ErrEntityNotFound = errors.New("bloodhound: entity not found")

// Entity is a typed BloodHound node crossing the Wails boundary.
type Entity struct {
	ObjectID   string            `json:"objectId"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"` // User | Group | Computer | Domain | OU | GPO | ...
	Owned      bool              `json:"owned"`
	TierZero   bool              `json:"tierZero"`
	Properties map[string]string `json:"properties,omitempty"`
}

// SearchPage is one page of entity search results. The BloodHound search API
// returns no total count, so pagination is client-driven via Offset/Limit.
type SearchPage struct {
	Entities []Entity `json:"entities"`
	Offset   int      `json:"offset"`
	Limit    int      `json:"limit"`
}

// DomainDTO is the minimal domain listing shape (id/name/collected only).
type DomainDTO struct {
	ObjectID  string `json:"objectId"`
	Name      string `json:"name"`
	Collected bool   `json:"collected"`
}

const (
	defaultSearchLimit = 25
	maxSearchLimit     = 100
)

func clampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func clampSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

func hasTag(systemTags, tag string) bool {
	for _, t := range strings.Fields(systemTags) {
		if t == tag {
			return true
		}
	}
	return false
}

func entityFromSearch(r bhservices.SearchResult) Entity {
	e := Entity{
		ObjectID: deref(r.Objectid),
		Name:     deref(r.Name),
		Kind:     deref(r.Type),
	}
	tags := deref(r.SystemTags)
	e.Owned = hasTag(tags, "owned")
	e.TierZero = hasTag(tags, "admin_tier_0")
	return e
}

// SearchEntities returns one page of entity search results.
func (s *Service) SearchEntities(ctx context.Context, query string, offset, limit int) (SearchPage, error) {
	client, err := s.snapshot()
	if err != nil {
		return SearchPage{}, err
	}
	offset = clampOffset(offset)
	limit = clampSearchLimit(limit)
	results, err := client.Community().Search().Query(query).Skip(offset).Limit(limit).Results(ctx)
	if err != nil {
		return SearchPage{}, err
	}
	entities := make([]Entity, 0, len(results))
	for _, r := range results {
		entities = append(entities, entityFromSearch(r))
	}
	return SearchPage{Entities: entities, Offset: offset, Limit: limit}, nil
}

// ListDomains returns the domains available in the BloodHound environment.
func (s *Service) ListDomains(ctx context.Context) ([]DomainDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	selectors, err := client.Community().Search().AvailableDomains(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DomainDTO, 0, len(selectors))
	for _, sel := range selectors {
		d := DomainDTO{
			ObjectID: deref(sel.Id),
			Name:     deref(sel.Name),
		}
		if sel.Collected != nil {
			d.Collected = *sel.Collected
		}
		out = append(out, d)
	}
	return out, nil
}

// Entity resolves an object ID to a typed entity: search gives name/kind/
// owned/tier-zero, then a signed kind-specific fetch fills Properties.
func (s *Service) Entity(ctx context.Context, objectID string) (*Entity, error) {
	client, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	results, err := client.Community().Search().Query(objectID).Limit(5).Results(ctx)
	if err != nil {
		return nil, err
	}
	var ent *Entity
	for _, r := range results {
		if deref(r.Objectid) == objectID {
			e := entityFromSearch(r)
			ent = &e
			break
		}
	}
	if ent == nil {
		return nil, fmt.Errorf("%w: %s", ErrEntityNotFound, objectID)
	}

	if props, err := s.fetchEntityDetail(ctx, ent.Kind, objectID); err == nil && len(props) > 0 {
		ent.Properties = props
	}
	// Detail fetch is best-effort: identity data survives failures.
	return ent, nil
}
