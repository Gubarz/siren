package bloodhound

import (
	"testing"

	bhservices "github.com/Gubarz/bloodhound-sdk-go/services"
)

func strp(s string) *string { return &s }
func boolp(b bool) *bool    { return &b }

func TestEntityFromSearchTags(t *testing.T) {
	ent := entityFromSearch(bhservices.SearchResult{
		Name: strp("DA@corp.local"), Objectid: strp("S-1-5-999"),
		Type: strp("Group"), SystemTags: strp("owned admin_tier_0"),
	})
	if !ent.Owned || !ent.TierZero {
		t.Fatalf("entity = %+v, want owned+tierzero", ent)
	}
	if ent.Kind != "Group" || ent.Name != "DA@corp.local" {
		t.Fatalf("entity = %+v", ent)
	}
	plain := entityFromSearch(bhservices.SearchResult{})
	if plain.Owned || plain.TierZero || plain.Kind != "" {
		t.Fatalf("empty search result should map to zero flags: %+v", plain)
	}
}

func TestGraphFromUnified(t *testing.T) {
	graph := &bhservices.UnifiedGraphGraphWithKeys{
		Nodes: &map[string]bhservices.UnifiedGraphNode{
			"n1": {Label: strp("JADMIN@CORP.LOCAL"), Kind: strp("User"), IsTierZero: boolp(true)},
			"n2": {Kind: strp("Group")},
			"n3": {},
		},
		Edges: &[]bhservices.UnifiedGraphEdge{
			{Source: strp("n1"), Target: strp("n2"), Kind: strp("MemberOf")},
			{Source: strp("n2"), Kind: strp("Owns")}, // missing target dropped
		},
	}
	dto := GraphFromUnified(graph)
	if len(dto.Nodes) != 3 || len(dto.Edges) != 1 {
		t.Fatalf("nodes=%d edges=%d, want 3/1", len(dto.Nodes), len(dto.Edges))
	}
	n1 := dto.Nodes[0]
	if n1.ID != "n1" || n1.Label != "JADMIN@CORP.LOCAL" || n1.Kind != "User" || !n1.TierZero {
		t.Fatalf("n1 = %+v", n1)
	}
	n3 := dto.Nodes[2]
	if n3.Label != "n3" || n3.TierZero {
		t.Fatalf("node without label should fall back to id and not be tier zero: %+v", n3)
	}
	e := dto.Edges[0]
	if e.Source != "n1" || e.Target != "n2" || e.Label != "MemberOf" {
		t.Fatalf("edge = %+v", e)
	}
}

func TestGraphFromUnifiedNilSafe(t *testing.T) {
	dto := GraphFromUnified(nil)
	if dto.Nodes == nil || dto.Edges == nil || len(dto.Nodes) != 0 || len(dto.Edges) != 0 {
		t.Fatalf("nil graph should yield empty non-nil GraphDTO, got %+v", dto)
	}
	empty := &bhservices.UnifiedGraphGraphWithKeys{}
	if got := GraphFromUnified(empty); got.Nodes == nil || got.Edges == nil {
		t.Fatalf("empty graph should yield non-nil slices, got %+v", got)
	}
}
