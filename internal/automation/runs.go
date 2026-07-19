package automation

import (
	"context"
	"log"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/google/uuid"
)

// dispatchTrigger fans a matching-trigger event out to every enabled rule.
// Callers hold no locks; the fan-out takes the read lock only for the
// duration of the rule snapshot.
func (e *Engine) dispatchTrigger(trigger string, target automationTarget) {
	e.mu.RLock()
	rules := append([]AutomationRule(nil), e.rules...)
	e.mu.RUnlock()
	for _, rule := range rules {
		if rule.Enabled && rule.Trigger == trigger {
			e.dispatchRule(rule, trigger, &target)
		}
	}
}

func (e *Engine) dispatchRule(rule AutomationRule, trigger string, target *automationTarget) {
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

func (e *Engine) currentTargets() []automationTarget {
	if !e.rpc.Connected() {
		return nil
	}
	ctx := context.Background()
	sessions, sessionErr := e.rpc.RPC.GetSessions(ctx, &commonpb.Empty{})
	beaconsResp, beaconErr := e.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{})
	if sessionErr != nil && beaconErr != nil {
		return nil
	}
	var targets []automationTarget
	if sessionErr == nil {
		for _, session := range sessions.Sessions {
			targets = append(targets, targetFromSession(session))
		}
	}
	if beaconErr == nil {
		for _, beacon := range beaconsResp.Beacons {
			targets = append(targets, targetFromBeacon(beacon))
		}
	}
	return targets
}

func (e *Engine) queueRun(rule AutomationRule, trigger string, target automationTarget) {
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

func (e *Engine) execute(rule AutomationRule, trigger string, target automationTarget, key string) {
	run := AutomationRun{
		ID: uuid.NewString(), RuleID: rule.ID, RuleName: rule.Name,
		Trigger: trigger, TargetID: target.ID,
		TargetName: displayTargetName(target), TargetKind: target.Kind,
		Status: "running", StartedAt: time.Now().UnixMilli(),
	}
	e.storeRun(run)

	var output string
	var runErr error
	if automationExecutionMode(rule) == ExecutionModeJavaScript {
		output, run.Commands, runErr = e.executeJavaScript(rule, trigger, target)
	} else {
		output, run.Commands, runErr = e.executeCommands(rule, target)
	}

	run.Output = output
	run.FinishedAt = time.Now().UnixMilli()
	if runErr != nil {
		run.Status = "failed"
		run.Error = runErr.Error()
	} else {
		run.Status = "completed"
	}
	e.finalizeRun(run, rule.ID, key)
	e.emit("automation-run", run)
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
