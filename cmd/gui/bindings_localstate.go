package gui

import (
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"siren/internal/localstate/casefile"
	"siren/internal/localstate/comments"
	"siren/internal/sliver/casereport"
)

// ---- Entity tags & colors ----

func (a *App) GetEntityTags(entityType, entityID string) []string {
	return a.Tags.GetEntityTags(entityType, entityID)
}

func (a *App) SetEntityTags(entityType, entityID string, tagList []string) error {
	if err := a.Tags.SetEntityTags(entityType, entityID, tagList); err != nil {
		return err
	}
	key := entityType + ":" + entityID
	a.bridge.Emit("entity-tags-updated", key)
	if strings.EqualFold(strings.TrimSpace(entityType), "agent") {
		a.bridge.Emit("agent-tags-updated", entityID)
	}
	return nil
}

func (a *App) GetAllEntityTags() map[string][]string {
	return a.Tags.GetAllEntityTags()
}

func (a *App) GetEntityColor(entityType, entityID string) string {
	return a.Tags.GetEntityColor(entityType, entityID)
}

func (a *App) SetEntityColor(entityType, entityID string, color string) error {
	if err := a.Tags.SetEntityColor(entityType, entityID, color); err != nil {
		return err
	}
	key := entityType + ":" + entityID
	a.bridge.Emit("entity-colors-updated", key)
	if strings.EqualFold(strings.TrimSpace(entityType), "agent") {
		a.bridge.Emit("agent-colors-updated", entityID)
	}
	return nil
}

func (a *App) GetAllEntityColors() map[string]string {
	return a.Tags.GetAllEntityColors()
}

func (a *App) GetAgentTags(agentID string) []string {
	return a.Tags.GetAgentTags(agentID)
}

func (a *App) SetAgentTags(agentID string, tagList []string) error {
	if err := a.Tags.SetAgentTags(agentID, tagList); err != nil {
		return err
	}
	a.bridge.Emit("entity-tags-updated", "agent:"+agentID)
	a.bridge.Emit("agent-tags-updated", agentID)
	return nil
}

func (a *App) GetAllAgentTags() map[string][]string {
	return a.Tags.GetAllTags()
}

func (a *App) ListKnownTags() []string {
	return a.Tags.KnownTags()
}

func (a *App) GetAllAgentColors() map[string]string {
	return a.Tags.GetAllColors()
}

func (a *App) SetAgentColor(agentID string, color string) error {
	if err := a.Tags.SetAgentColor(agentID, color); err != nil {
		return err
	}
	a.bridge.Emit("entity-colors-updated", "agent:"+agentID)
	a.bridge.Emit("agent-colors-updated", agentID)
	return nil
}

// ---- Universal Entity Comments ----

func (a *App) GetEntityComments(entityType, entityID string) []comments.Comment {
	return a.Comments.GetComments(entityType, entityID)
}

func (a *App) GetAllComments() map[string][]comments.Comment {
	return a.Comments.GetAllComments()
}

func (a *App) AddEntityComment(entityType, entityID, author, text string) (comments.Comment, error) {
	c, err := a.Comments.AddComment(entityType, entityID, author, text)
	if err != nil {
		return comments.Comment{}, err
	}
	a.bridge.Emit("comments-updated", entityType+":"+entityID)
	return c, nil
}

func (a *App) DeleteEntityComment(commentID string) error {
	if err := a.Comments.DeleteComment(commentID); err != nil {
		return err
	}
	a.bridge.Emit("comments-updated", "")
	return nil
}

// ---- Case files ----

func (a *App) ListCases() []*casefile.Record {
	return a.Cases.List()
}

func (a *App) GetCase(id string) *casefile.Record {
	return a.Cases.Get(id)
}

func (a *App) CreateCase(name, description string) (*casefile.Record, error) {
	c, err := a.Cases.Create(name, description)
	if err == nil {
		a.bridge.Emit("case-updated", c.ID)
	}
	return c, err
}

func (a *App) UpdateCase(id, name, description, notes string) error {
	if err := a.Cases.Update(id, name, description, notes); err != nil {
		return err
	}
	a.bridge.Emit("case-updated", id)
	return nil
}

func (a *App) DeleteCase(id string) error {
	if err := a.Cases.Delete(id); err != nil {
		return err
	}
	a.bridge.Emit("case-updated", id)
	return nil
}

// AddToCase / RemoveFromCase — collection ∈ {"agent","loot","cred","host","canary"}.
func (a *App) AddToCase(caseID, collection, itemID string) error {
	if err := a.Cases.Add(caseID, casefile.Collection(collection), itemID); err != nil {
		return err
	}
	a.bridge.Emit("case-updated", caseID)
	return nil
}

func (a *App) RemoveFromCase(caseID, collection, itemID string) error {
	if err := a.Cases.Remove(caseID, casefile.Collection(collection), itemID); err != nil {
		return err
	}
	a.bridge.Emit("case-updated", caseID)
	return nil
}

// GenerateCaseReport renders a case as a Markdown document string.
func (a *App) GenerateCaseReport(caseID string) (string, error) {
	return a.Cases.GenerateMarkdown(caseID, casereport.NewReporter(a.Console, a.RPC))
}

func (a *App) ExportCaseReport(caseID string) (string, error) {
	c := a.Cases.Get(caseID)
	if c == nil {
		return "", fmt.Errorf("case %s not found", caseID)
	}
	md, err := a.GenerateCaseReport(caseID)
	if err != nil {
		return "", err
	}
	path, err := a.bridge.SaveFileDialog(&application.SaveFileDialogOptions{
		Title:    "Export Case Report",
		Filename: casefile.ReportFilename(c.Name),
		Filters: []application.FileFilter{{
			DisplayName: "Markdown files (*.md)",
			Pattern:     "*.md",
		}},
	})
	if err != nil || path == "" {
		return path, err
	}
	return path, os.WriteFile(path, []byte(md), 0o600)
}
