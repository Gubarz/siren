package casefile

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) AppendNote(_ context.Context, caseRef, markdown string) error {
	caseRef = strings.TrimSpace(caseRef)
	if caseRef == "" {
		return fmt.Errorf("case reference required")
	}
	target := s.Get(caseRef)
	if target == nil {
		for _, c := range s.List() {
			if strings.EqualFold(c.Name, caseRef) {
				target = c
				break
			}
		}
	}
	if target == nil {
		var err error
		target, err = s.Create(caseRef, "created by automation case-add action")
		if err != nil {
			return err
		}
	}
	stamp := time.Now().Format("2006-01-02 15:04")
	notes := target.Notes
	if notes != "" {
		notes += "\n\n"
	}
	notes += fmt.Sprintf("**%s**\n\n%s", stamp, markdown)
	return s.Update(target.ID, target.Name, target.Description, notes)
}
