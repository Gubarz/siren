package casefile

import (
	"fmt"
	"sort"
)

// Collection = the field name of one of the ID lists on Record. Kept as a
// typed string so the frontend can hand us "agent" / "loot" / etc.
// without a per-collection method matrix.
type Collection string

const (
	CollectionAgents Collection = "agent"
	CollectionLoot   Collection = "loot"
	CollectionCreds  Collection = "cred"
	CollectionHosts  Collection = "host"
	CollectionCanary Collection = "canary"
)

// Add appends an id to a case's collection if it's not already there,
// then persists. Idempotent.
func (s *Service) Add(caseID string, collection Collection, id string) error {
	return s.mutateCollection(caseID, collection, func(list []string) []string {
		for _, existing := range list {
			if existing == id {
				return list
			}
		}
		return append(list, id)
	})
}

// Remove drops an id if present. Idempotent.
func (s *Service) Remove(caseID string, collection Collection, id string) error {
	return s.mutateCollection(caseID, collection, func(list []string) []string {
		out := make([]string, 0, len(list))
		for _, existing := range list {
			if existing != id {
				out = append(out, existing)
			}
		}
		return out
	})
}

func (s *Service) mutateCollection(caseID string, collection Collection, mutate func([]string) []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cases[caseID]
	if !ok {
		return fmt.Errorf("case %s not found", caseID)
	}
	list, err := collectionField(c, collection)
	if err != nil {
		return err
	}
	next := mutate(list)
	sort.Strings(next)
	setCollectionField(c, collection, next)
	return s.persistLocked(c)
}

func collectionField(c *Record, collection Collection) ([]string, error) {
	switch collection {
	case CollectionAgents:
		return c.AgentIDs, nil
	case CollectionLoot:
		return c.LootIDs, nil
	case CollectionCreds:
		return c.CredIDs, nil
	case CollectionHosts:
		return c.HostIDs, nil
	case CollectionCanary:
		return c.CanaryIDs, nil
	default:
		return nil, fmt.Errorf("unknown collection %q", collection)
	}
}

func setCollectionField(c *Record, collection Collection, list []string) {
	switch collection {
	case CollectionAgents:
		c.AgentIDs = list
	case CollectionLoot:
		c.LootIDs = list
	case CollectionCreds:
		c.CredIDs = list
	case CollectionHosts:
		c.HostIDs = list
	case CollectionCanary:
		c.CanaryIDs = list
	}
}
