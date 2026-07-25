package actions

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/sobek"

	"sliver-gui/internal/bus"
	"sliver-gui/internal/journal"
)

const (
	scriptHTTPDefaultTimeout = 10 * time.Second
	scriptHTTPMaxResponse    = 1 * 1024 * 1024
	scriptJournalMaxEntries  = 200
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
	return obj.Set("journal", map[string]any{"query": je.scriptJournalQuery})
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

func (je *jsExec) scriptJournalQuery(call sobek.FunctionCall) sobek.Value {
	vm := je.vm
	if je.rc.Deps.Journal == nil {
		panic(vm.NewGoError(fmt.Errorf("sliver.journal.query: journal unavailable")))
	}
	filter := journal.Filter{Limit: scriptJournalMaxEntries}
	if len(call.Arguments) > 0 && !sobek.IsUndefined(call.Argument(0)) && !sobek.IsNull(call.Argument(0)) {
		applyJournalFilter(call.Argument(0).ToObject(vm), &filter)
	}
	entries, total, err := je.rc.Deps.Journal.Query(je.rc.Ctx, filter)
	if err != nil {
		panic(vm.NewGoError(err))
	}
	if len(entries) > scriptJournalMaxEntries {
		entries = entries[:scriptJournalMaxEntries]
	}
	result := vm.NewObject()
	_ = result.Set("entries", entries)
	_ = result.Set("total", total)
	return result
}

func applyJournalFilter(obj *sobek.Object, f *journal.Filter) {
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
	if v := obj.Get("actorKind"); v != nil && !sobek.IsUndefined(v) {
		if s := v.String(); s != "" {
			f.ActorKind = s
		}
	}
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
