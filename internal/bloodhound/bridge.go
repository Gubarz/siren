package bloodhound

import (
	"context"
	"sort"

	bhservices "github.com/Gubarz/bloodhound-sdk-go/services"
)

// KerberoastHashType is the hashcat mode for Kerberos 5 TGS-REP etype 23
// (13100). The frontend hands roasted hashes to the crack service with this
// type; BloodHound itself never carries the hashes — only the SPN accounts.
const KerberoastHashType = 13100

// KerberoastTargets runs the kerberoastable community query and returns the
// typed User entities it references, deduped by object ID and sorted by name.
func (s *Service) KerberoastTargets(ctx context.Context) ([]Entity, error) {
	client, err := s.snapshot()
	if err != nil {
		return nil, err
	}
	query, ok := CommunityCypher(CommunityKerberoastable)
	if !ok {
		return nil, nil
	}
	graph, err := client.Community().Cypher().RunCypher(ctx, query)
	if err != nil {
		if isEmptyCypherResult(err) {
			return []Entity{}, nil // CE: zero matches surface as 404
		}
		return nil, err
	}
	return userEntitiesFromGraph(graph), nil
}

func userEntitiesFromGraph(graph *bhservices.UnifiedGraphGraphWithKeys) []Entity {
	out := []Entity{}
	seen := map[string]bool{}
	if graph == nil || graph.Nodes == nil {
		return out
	}
	for id, n := range *graph.Nodes {
		if deref(n.Kind) != "User" {
			continue
		}
		objectID := deref(n.ObjectId)
		if objectID == "" {
			objectID = id // fall back to the graph node key
		}
		if seen[objectID] {
			continue
		}
		seen[objectID] = true
		label := deref(n.Label)
		if label == "" {
			label = objectID
		}
		out = append(out, Entity{
			ObjectID: objectID,
			Name:     label,
			Kind:     "User",
			Owned:    derefBool(n.IsOwnedObject),
			TierZero: derefBool(n.IsTierZero),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
