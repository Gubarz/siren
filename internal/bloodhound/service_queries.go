package bloodhound

import (
	"context"
	"fmt"
)

func (s *Service) EntityAttackPaths(ctx context.Context, objectID string, maxPaths int) (GraphDTO, error) {
	return s.runCypherQuery(ctx, func() (string, error) {
		return AttackPathsCypher(objectID, maxPaths)
	})
}

func (s *Service) CommunityQuery(ctx context.Context, kind CommunityKind) (GraphDTO, error) {
	return s.runCypherQuery(ctx, func() (string, error) {
		query, ok := CommunityCypher(kind)
		if !ok {
			return "", fmt.Errorf("unknown community query kind %q", kind)
		}
		return query, nil
	})
}

func (s *Service) EntityLocalAdmins(ctx context.Context, objectID, entityKind string) (GraphDTO, error) {
	return s.runCypherQuery(ctx, func() (string, error) {
		return LocalAdminsCypher(objectID, entityKind)
	})
}

func (s *Service) EntitySessions(ctx context.Context, objectID, entityKind string) (GraphDTO, error) {
	return s.runCypherQuery(ctx, func() (string, error) {
		return SessionsCypher(objectID, entityKind)
	})
}

// runCypherQuery snapshots the client, resolves the Cypher text, and maps an
// empty result to an empty graph. Query errors surface before any RPC.
func (s *Service) runCypherQuery(ctx context.Context, query func() (string, error)) (GraphDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return GraphDTO{}, err
	}
	cypher, err := query()
	if err != nil {
		return GraphDTO{}, err
	}
	graph, err := client.Community().Cypher().RunCypher(ctx, cypher)
	if err != nil {
		if isEmptyCypherResult(err) {
			return GraphDTO{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}, nil
		}
		return GraphDTO{}, err
	}
	return GraphFromUnified(graph), nil
}
