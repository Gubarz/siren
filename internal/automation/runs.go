package automation

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"siren/internal/journal"
)

func (e *Engine) dispatchRule(rule AutomationRule, trigger string, target *Target) {
	if rule.MaxRuns > 0 && rule.RunCount >= rule.MaxRuns {
		return
	}
	if target != nil {
		if matchesAutomationRule(rule, *target) {
			e.queueRun(rule, trigger, *target)
		}
		return
	}
	for _, candidate := range e.currentTargets() {
		if matchesAutomationRule(rule, candidate) {
			e.queueRun(rule, trigger, candidate)
		}
	}
}

func (e *Engine) currentTargets() []Target {
	if !e.targets.Connected() {
		return nil
	}
	ctx := context.Background()
	sessions, sessionErr := e.targets.GetSessions(ctx)
	beacons, beaconErr := e.targets.GetBeacons(ctx)
	if sessionErr != nil && beaconErr != nil {
		return nil
	}
	var targets []Target
	if sessionErr == nil {
		targets = append(targets, sessions...)
	}
	if beaconErr == nil {
		targets = append(targets, beacons...)
	}
	return targets
}

func (e *Engine) queueRun(rule AutomationRule, trigger string, target Target) {
	key := rule.ID + ":" + target.ID
	now := time.Now()
	e.mu.Lock()
	if e.running[key] {
		e.mu.Unlock()
		return
	}
	if cooldown := time.Duration(rule.CooldownSeconds) * time.Second; cooldown > 0 && now.Sub(e.lastRun[key]) < cooldown {
		e.mu.Unlock()
		return
	}
	current := e.ruleByIDLocked(rule.ID)
	if current == nil || (current.MaxRuns > 0 && current.RunCount+e.activeByRule[rule.ID] >= current.MaxRuns) {
		e.mu.Unlock()
		return
	}
	e.running[key] = true
	e.activeByRule[rule.ID]++
	e.lastRun[key] = now
	e.mu.Unlock()
	go e.execute(rule, trigger, target, key)
}

func (e *Engine) execute(rule AutomationRule, trigger string, target Target, key string) {
	run := AutomationRun{
		ID: uuid.NewString(), RuleID: rule.ID, RuleName: rule.Name,
		Trigger: trigger, TargetID: target.ID,
		TargetName: displayTargetName(target), TargetKind: target.Kind,
		Status: "running", StartedAt: time.Now().UnixMilli(),
	}
	e.storeRun(run)
	parent := e.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctxWithOverlay := journal.WithContext(parent, journal.Overlay{
		ActorKind: "automation", RuleID: rule.ID, RuleName: rule.Name, CorrelationID: run.ID,
	})
	var output strings.Builder
	var commands []string
	rc := &RunContext{
		Ctx:      ctxWithOverlay,
		Rule:     rule,
		Trigger:  trigger,
		Target:   target,
		RunID:    run.ID,
		Commands: &commands,
		Deps:     e.actionDeps(),
	}
	rc.Log = func(args ...any) {
		line := fmt.Sprint(args...)
		if output.Len() > 0 {
			output.WriteByte('\n')
		}
		output.WriteString(line)
	}
	rc.OutputSoFar = output.String
	e.executeActionList(rc, &run, &output)
	run.Commands = commands
	run.Output = output.String()
	run.FinishedAt = time.Now().UnixMilli()
	if err := firstActionError(run.ActionResults); err != nil {
		run.Status = "failed"
		run.Error = err.Error
	} else {
		run.Status = "completed"
	}
	e.finalizeRun(run, rule.ID, key)
	e.emit("automation-run", run)
}

func (e *Engine) executeActionList(rc *RunContext, run *AutomationRun, output *strings.Builder) {
	for _, spec := range rc.Rule.Actions {
		rc.Action = spec
		result := e.executeAction(rc, spec)
		run.ActionResults = append(run.ActionResults, result)
		if result.Output != "" {
			if output.Len() > 0 {
				output.WriteString("\n\n")
			}
			output.WriteString(result.Output)
		}
		if result.Status == "error" && !rc.Rule.ContinueOnError {
			break
		}
	}
}

func firstActionError(results []ActionResult) *ActionResult {
	for _, r := range results {
		if r.Status == "error" {
			return &r
		}
	}
	return nil
}

func (e *Engine) finalizeRun(run AutomationRun, ruleID, key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.running, key)
	if e.activeByRule[ruleID] > 0 {
		e.activeByRule[ruleID]--
	}
	if current := e.ruleByIDLocked(ruleID); current != nil {
		current.RunCount++
		current.UpdatedAt = time.Now().UnixMilli()
	}
	e.replaceRunLocked(run)
	if err := e.persistLocked(); err != nil {
		log.Printf("automation: persist run: %v", err)
	}
}

func (e *Engine) storeRun(run AutomationRun) {
	e.mu.Lock()
	e.history = append([]AutomationRun{run}, e.history...)
	if len(e.history) > automationHistoryLimit {
		e.history = e.history[:automationHistoryLimit]
	}
	if err := e.persistLocked(); err != nil {
		log.Printf("automation: persist started run: %v", err)
	}
	e.mu.Unlock()
	e.emit("automation-run", run)
}

func (e *Engine) replaceRunLocked(run AutomationRun) {
	for index := range e.history {
		if e.history[index].ID == run.ID {
			e.history[index] = run
			return
		}
	}
	e.history = append([]AutomationRun{run}, e.history...)
}
