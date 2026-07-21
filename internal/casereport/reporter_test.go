package casereport

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func TestFormatLootShowsAgentNotePreview(t *testing.T) {
	r := &reporter{}
	loot := &clientpb.Loot{
		ID:       "0192c3ef-0000-4000-8000-000000000000",
		Name:     "note-bc5bd3c8-97e0-4997-b731-444a104c3aaf",
		FileType: clientpb.FileType_TEXT,
		File: &commonpb.File{
			Data: []byte("first line\nsecond line"),
		},
	}

	got := r.formatLoot(loot)
	for _, want := range []string{
		"**Agent note BC5BD3C8** `0192C3EF`",
		"text",
		"preview=`first line second line`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatLoot() = %q, want it to contain %q", got, want)
		}
	}
}

func TestFormatLootMarksMissingTextPreview(t *testing.T) {
	r := &reporter{}
	loot := &clientpb.Loot{
		ID:       "9e5ad33d-0000-4000-8000-000000000000",
		Name:     "operator-note",
		FileType: clientpb.FileType_TEXT,
	}

	got := r.formatLoot(loot)
	if !strings.Contains(got, "text preview unavailable") {
		t.Fatalf("formatLoot() = %q, want missing preview marker", got)
	}
}
