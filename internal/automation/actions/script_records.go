package actions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/sobek"
	"github.com/gubarz/revils/store"
)

const (
	scriptRecordsMaxEntries = 200
	scriptRecordsFetchLimit = 500
)

type recordFilter struct {
	Verb     string
	TargetID string
	Status   string
	Since    int64
	Until    int64
	Limit    int
}

// recordEntry's JSON keys are a stable contract for existing scripts.
type recordEntry struct {
	ID            int64  `json:"id"`
	Time          int64  `json:"time"`
	ConnectionID  string `json:"connectionID"`
	ActorKind     string `json:"actorKind"`
	RuleID        string `json:"ruleID"`
	RuleName      string `json:"ruleName"`
	Verb          string `json:"verb"`
	CommandLine   string `json:"commandLine"`
	TargetID      string `json:"targetID"`
	TargetKind    string `json:"targetKind"`
	Hostname      string `json:"hostname"`
	Panel         string `json:"panel"`
	Status        string `json:"status"`
	Err           string `json:"err"`
	DurationMs    int64  `json:"durationMs"`
	CorrelationID string `json:"correlationID"`
}

func (je *jsExec) scriptRecordsQuery(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	if je.rc.Deps.Records == nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.journal.query: record store unavailable")))
	}
	filter := recordFilter{Limit: scriptRecordsMaxEntries}
	if len(call.Arguments) > 0 && !sobek.IsUndefined(call.Argument(0)) && !sobek.IsNull(call.Argument(0)) {
		applyRecordFilter(call.Argument(0).ToObject(vm), &filter)
	}
	rows, err := je.rc.Deps.Records.Query(store.Filter{
		Kind:      store.KindCall,
		Direction: store.DirectionComplete,
		Limit:     scriptRecordsFetchLimit,
	})
	if err != nil {
		panic(vm.NewGoError(err))
	}
	entries, total := mapRecordRows(rows, filter)
	result := vm.NewObject()
	_ = result.Set("entries", entries)
	_ = result.Set("total", total)
	return result
}

func applyRecordFilter(obj *sobek.Object, f *recordFilter) {
	if v := obj.Get("verb"); v != nil && !sobek.IsUndefined(v) {
		if s := v.String(); s != "" {
			f.Verb = s
		}
	}
	if v := obj.Get("targetID"); v != nil && !sobek.IsUndefined(v) {
		if s := v.String(); s != "" {
			f.TargetID = s
		}
	}
	if v := obj.Get("status"); v != nil && !sobek.IsUndefined(v) {
		if s := v.String(); s != "" {
			f.Status = s
		}
	}
	if v := obj.Get("since"); v != nil && !sobek.IsUndefined(v) {
		f.Since = v.ToInteger()
	}
	if v := obj.Get("until"); v != nil && !sobek.IsUndefined(v) {
		f.Until = v.ToInteger()
	}
	if v := obj.Get("limit"); v != nil && !sobek.IsUndefined(v) {
		f.Limit = int(v.ToInteger())
	}
}

// mapRecordRows maps completion rows to the historical entry shape. Method
// suffix, target, time, and status filters run here because the store filter
// cannot express suffix, session-or-beacon matching, or status normalization.
// Known limitation: since/until and verb/target filters apply after the
// 500-row fetch, and actorKind is unsupported.
func mapRecordRows(rows []store.RecordRow, f recordFilter) ([]recordEntry, int) {
	if f.Status != "" {
		f.Status = normalizeRecordStatus(f.Status)
	}
	entries := make([]recordEntry, 0, len(rows))
	for _, row := range rows {
		if !recordRowMatches(row, f) {
			continue
		}
		entries = append(entries, recordEntryFromRow(row))
	}
	total := len(entries)
	if limit := recordRowLimit(f.Limit); len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, total
}

func recordRowMatches(row store.RecordRow, f recordFilter) bool {
	if f.Verb != "" && shortRecordMethod(row.Method) != f.Verb {
		return false
	}
	if f.TargetID != "" && row.SessionID != f.TargetID && row.BeaconID != f.TargetID {
		return false
	}
	if f.Status != "" && normalizeRecordStatus(row.Status) != f.Status {
		return false
	}
	if f.Since != 0 && row.TS < f.Since {
		return false
	}
	if f.Until != 0 && row.TS > f.Until {
		return false
	}
	return true
}

func recordEntryFromRow(row store.RecordRow) recordEntry {
	targetID, targetKind := row.SessionID, ""
	if targetID != "" {
		targetKind = "session"
	} else if row.BeaconID != "" {
		targetID, targetKind = row.BeaconID, "beacon"
	}
	return recordEntry{
		ID:            row.ID,
		Time:          row.TS,
		Verb:          shortRecordMethod(row.Method),
		TargetID:      targetID,
		TargetKind:    targetKind,
		Status:        normalizeRecordStatus(row.Status),
		Err:           completionError(row.JSON),
		CorrelationID: row.RunID,
	}
}

func recordRowLimit(limit int) int {
	if limit <= 0 || limit > scriptRecordsMaxEntries {
		return scriptRecordsMaxEntries
	}
	return limit
}

func shortRecordMethod(method string) string {
	if i := strings.LastIndexByte(method, '/'); i >= 0 {
		return method[i+1:]
	}
	return method
}

// normalizeRecordStatus collapses the vocabulary mismatch between envelope
// rows ("attempted") and completion rows (gRPC code strings such as "OK" or
// "Canceled") into the three script-facing statuses.
func normalizeRecordStatus(status string) string {
	switch status {
	case "OK", "ok":
		return "ok"
	case "", "attempted":
		return "attempted"
	default:
		return "error"
	}
}

func completionError(preview string) string {
	if preview == "" {
		return ""
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(preview), &body); err != nil {
		return ""
	}
	return body.Error
}
