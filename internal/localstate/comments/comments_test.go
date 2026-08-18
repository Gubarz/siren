package comments

import (
	"testing"
)

func TestCommentsService(t *testing.T) {
	dir := t.TempDir()
	svc := New(dir)

	c1, err := svc.AddComment("agent", "session-1", "Alice", "First agent note")
	if err != nil {
		t.Fatalf("AddComment failed: %v", err)
	}
	if c1.ID == "" || c1.Text != "First agent note" {
		t.Errorf("Unexpected comment: %+v", c1)
	}

	c2, err := svc.AddComment("agent", "session-1", "Bob", "Second note")
	if err != nil {
		t.Fatalf("AddComment 2 failed: %v", err)
	}

	comments := svc.GetComments("agent", "session-1")
	if len(comments) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != c1.ID || comments[1].ID != c2.ID {
		t.Errorf("Unexpected order or content: %+v", comments)
	}

	all := svc.GetAllComments()
	if len(all["agent:session-1"]) != 2 {
		t.Errorf("Expected 2 in GetAllComments, got %d", len(all["agent:session-1"]))
	}

	if err := svc.DeleteComment(c1.ID); err != nil {
		t.Fatalf("DeleteComment failed: %v", err)
	}

	comments = svc.GetComments("agent", "session-1")
	if len(comments) != 1 || comments[0].ID != c2.ID {
		t.Errorf("Expected only c2 remaining, got %+v", comments)
	}
}
