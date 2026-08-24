package bloodhound

import (
	"errors"
	"net/http"
	"sort"

	bhservices "github.com/Gubarz/bloodhound-sdk-go/services"
)

// isEmptyCypherResult reports whether err is the BloodHound CE signal for a
// cypher query that matched zero rows. CE returns HTTP 404 in that case
// (verified against a live 6.x instance); real endpoint errors never get
// masked because they carry different statuses.
func isEmptyCypherResult(err error) bool {
	var apiErr *bhservices.APIError
	return errors.As(err, &apiErr) && apiErr.HTTPStatus == http.StatusNotFound
}

type NodeDTO struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	TierZero bool   `json:"tierZero"`
	Owned    bool   `json:"owned"`
}

type EdgeDTO struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// GraphDTO is the transfer shape for all graph results (attack paths, quick
// queries). Node IDs match edge Source/Target values.
type GraphDTO struct {
	Nodes []NodeDTO `json:"nodes"`
	Edges []EdgeDTO `json:"edges"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	return b != nil && *b
}

func GraphFromUnified(graph *bhservices.UnifiedGraphGraphWithKeys) GraphDTO {
	dto := GraphDTO{Nodes: []NodeDTO{}, Edges: []EdgeDTO{}}
	if graph == nil {
		return dto
	}
	if graph.Nodes != nil {
		for id, n := range *graph.Nodes {
			label := deref(n.Label)
			if label == "" {
				label = id
			}
			dto.Nodes = append(dto.Nodes, NodeDTO{
				ID:       id,
				Label:    label,
				Kind:     deref(n.Kind),
				TierZero: derefBool(n.IsTierZero),
				Owned:    derefBool(n.IsOwnedObject),
			})
		}
		sort.Slice(dto.Nodes, func(i, j int) bool { return dto.Nodes[i].ID < dto.Nodes[j].ID })
	}
	if graph.Edges != nil {
		for _, e := range *graph.Edges {
			source, target := deref(e.Source), deref(e.Target)
			if source == "" || target == "" {
				continue
			}
			label := deref(e.Kind)
			if label == "" {
				label = deref(e.Label)
			}
			dto.Edges = append(dto.Edges, EdgeDTO{Source: source, Target: target, Label: label})
		}
	}
	return dto
}
