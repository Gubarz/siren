package actions

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/sobek"
	"github.com/gubarz/revils/store"

	"siren/internal/bus"
)

const (
	scriptHTTPDefaultTimeout = 10 * time.Second
	scriptHTTPMaxResponse    = 1 * 1024 * 1024
	scriptRecordsMaxEntries  = 200
	scriptRecordsFetchLimit  = 500
)

func (je *jsExec) extendAPI(vm *sobek.Runtime, sliver sobek.Value) error {
	obj := sliver.ToObject(vm)
	if err := obj.Set("http", je.scriptHTTP); err != nil {
		return err
	}
	if err := obj.Set("loot", map[string]any{"add": je.scriptLootAdd, "list": je.scriptLootList}); err != nil {
		return err
	}
	if err := obj.Set("case", map[string]any{"add": je.scriptCaseAdd}); err != nil {
		return err
	}
	if err := obj.Set("events", map[string]any{"emit": je.scriptEventsEmit}); err != nil {
		return err
	}
	return obj.Set("journal", map[string]any{"query": je.scriptRecordsQuery})
}

func (je *jsExec) scriptHTTP(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	url := call.Argument(0).String()
	if url == "" {
		panic(vm.NewGoError(fmt.Errorf("sliver.http: url required")))
	}
	if je.rc.Deps.HTTP == nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.http: HTTP client unavailable")))
	}
	var opts *sobek.Object
	if len(call.Arguments) > 1 && !sobek.IsUndefined(call.Argument(1)) && !sobek.IsNull(call.Argument(1)) {
		opts = call.Argument(1).ToObject(vm)
	}
	method := optString(opts, "method", http.MethodGet)
	body := optString(opts, "body", "")
	timeoutMs := optInt(opts, "timeoutMs", int64(scriptHTTPDefaultTimeout/time.Millisecond))

	ctx, cancel := context.WithTimeout(je.rc.Ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	if err != nil {
		panic(vm.NewGoError(err))
	}
	if opts != nil {
		if headers := opts.Get("headers"); headers != nil && !sobek.IsUndefined(headers) && !sobek.IsNull(headers) {
			for _, key := range headers.ToObject(vm).Keys() {
				req.Header.Set(key, headers.ToObject(vm).Get(key).String())
			}
		}
	}
	resp, err := je.rc.Deps.HTTP.Do(req)
	if err != nil {
		panic(vm.NewGoError(err))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, scriptHTTPMaxResponse))
	if err != nil {
		panic(vm.NewGoError(err))
	}
	result := vm.NewObject()
	_ = result.Set("status", resp.StatusCode)
	_ = result.Set("body", string(data))
	return result
}

func (je *jsExec) scriptLootAdd(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	if je.rc.Deps.Loot == nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.loot.add: loot unavailable")))
	}
	name := call.Argument(0).String()
	lootType := call.Argument(1).String()
	dataB64 := call.Argument(2).String()
	data, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.loot.add: base64: %w", err)))
	}
	if err := je.rc.Deps.Loot.Add(je.rc.Ctx, name, lootType, data); err != nil {
		panic(vm.NewGoError(err))
	}
	return sobek.Undefined()
}

func (je *jsExec) scriptLootList(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	if je.rc.Deps.Loot == nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.loot.list: loot unavailable")))
	}
	items, err := je.rc.Deps.Loot.List(je.rc.Ctx)
	if err != nil {
		panic(vm.NewGoError(err))
	}
	return vm.ToValue(items)
}

func (je *jsExec) scriptCaseAdd(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	if je.rc.Deps.Cases == nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.case.add: cases unavailable")))
	}
	caseRef := call.Argument(0).String()
	itemType := call.Argument(1).String()
	payload := call.Argument(2).String()
	note := fmt.Sprintf("### Script note — %s\n\n- Type: `%s`\n\n```\n%s\n```\n", je.rc.Rule.Name, itemType, payload)
	if err := je.rc.Deps.Cases.AppendNote(je.rc.Ctx, caseRef, note); err != nil {
		panic(vm.NewGoError(err))
	}
	return sobek.Undefined()
}

func (je *jsExec) scriptEventsEmit(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	eventType := call.Argument(0).String()
	if eventType == "" {
		panic(vm.NewGoError(fmt.Errorf("sliver.events.emit: type required")))
	}
	if !strings.HasPrefix(eventType, "automation.") {
		eventType = "automation." + eventType
	}
	var payload any
	if len(call.Arguments) > 1 {
		payload = call.Argument(1).Export()
	}
	if je.rc.Deps.Bus != nil {
		je.rc.Deps.Bus.Publish(bus.Event{
			Type:    eventType,
			Source:  "automation",
			Payload: payload,
		})
	}
	return sobek.Undefined()
}

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
		verb := shortRecordMethod(row.Method)
		if f.Verb != "" && verb != f.Verb {
			continue
		}
		if f.TargetID != "" && row.SessionID != f.TargetID && row.BeaconID != f.TargetID {
			continue
		}
		status := normalizeRecordStatus(row.Status)
		if f.Status != "" && status != f.Status {
			continue
		}
		if f.Since != 0 && row.TS < f.Since {
			continue
		}
		if f.Until != 0 && row.TS > f.Until {
			continue
		}
		targetID, targetKind := row.SessionID, ""
		if targetID != "" {
			targetKind = "session"
		} else if row.BeaconID != "" {
			targetID, targetKind = row.BeaconID, "beacon"
		}
		entries = append(entries, recordEntry{
			ID:            row.ID,
			Time:          row.TS,
			Verb:          verb,
			TargetID:      targetID,
			TargetKind:    targetKind,
			Status:        status,
			Err:           completionError(row.JSON),
			CorrelationID: row.RunID,
		})
	}
	total := len(entries)
	limit := f.Limit
	if limit <= 0 || limit > scriptRecordsMaxEntries {
		limit = scriptRecordsMaxEntries
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, total
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

func optString(opts *sobek.Object, key, fallback string) string {
	if opts == nil {
		return fallback
	}
	v := opts.Get(key)
	if v == nil || sobek.IsUndefined(v) || sobek.IsNull(v) {
		return fallback
	}
	return v.String()
}

func optInt(opts *sobek.Object, key string, fallback int64) int64 {
	if opts == nil {
		return fallback
	}
	v := opts.Get(key)
	if v == nil || sobek.IsUndefined(v) || sobek.IsNull(v) {
		return fallback
	}
	return v.ToInteger()
}
