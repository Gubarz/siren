package bloodhound

import (
	"context"
	"fmt"
)

func (s *Service) EntityAttackPaths(ctx context.Context, objectID string, maxPaths int) (GraphDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return GraphDTO{}, err
	}
	query, err := AttackPathsCypher(objectID, maxPaths)
	if err != nil {
		return GraphDTO{}, err
	}
	graph, err := client.Community().Cypher().RunCypher(ctx, query)
	if err != nil {
		if isEmptyCypherResult(err) {
			return GraphDTO{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}, nil
		}
		return GraphDTO{}, err
	}
	return GraphFromUnified(graph), nil
}

func (s *Service) CommunityQuery(ctx context.Context, kind CommunityKind) (GraphDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return GraphDTO{}, err
	}
	query, ok := CommunityCypher(kind)
	if !ok {
		return GraphDTO{}, fmt.Errorf("unknown community query kind %q", kind)
	}
	graph, err := client.Community().Cypher().RunCypher(ctx, query)
	if err != nil {
		if isEmptyCypherResult(err) {
			return GraphDTO{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}, nil
		}
		return GraphDTO{}, err
	}
	return GraphFromUnified(graph), nil
}

func (s *Service) EntityLocalAdmins(ctx context.Context, objectID, entityKind string) (GraphDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return GraphDTO{}, err
	}
	query, err := LocalAdminsCypher(objectID, entityKind)
	if err != nil {
		return GraphDTO{}, err
	}
	graph, err := client.Community().Cypher().RunCypher(ctx, query)
	if err != nil {
		if isEmptyCypherResult(err) {
			return GraphDTO{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}, nil
		}
		return GraphDTO{}, err
	}
	return GraphFromUnified(graph), nil
}

func (s *Service) EntitySessions(ctx context.Context, objectID, entityKind string) (GraphDTO, error) {
	client, err := s.snapshot()
	if err != nil {
		return GraphDTO{}, err
	}
	query, err := SessionsCypher(objectID, entityKind)
	if err != nil {
		return GraphDTO{}, err
	}
	graph, err := client.Community().Cypher().RunCypher(ctx, query)
	if err != nil {
		if isEmptyCypherResult(err) {
			return GraphDTO{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}, nil
		}
		return GraphDTO{}, err
	}
	return GraphFromUnified(graph), nil
}
