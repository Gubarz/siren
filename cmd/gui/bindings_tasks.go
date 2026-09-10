package gui

import (
	"encoding/json"

	"github.com/gubarz/revils/store"

	"siren/internal/sliver/tasksview"
)

type SessionTaskView struct {
	ID            int64  `json:"id"`
	TS            int64  `json:"ts"`
	Method        string `json:"method"`
	Status        string `json:"status"`
	Evidence      string `json:"evidence"`
	RunID         string `json:"runId"`
	StageID       string `json:"stageId"`
	ChainRef      string `json:"chainRef"`
	RequestBytes  int64  `json:"requestBytes"`
	ResponseBytes int64  `json:"responseBytes"`
	Error         string `json:"error"`
}

type TaskPayloadView struct {
	Direction string `json:"direction"`
	TS        int64  `json:"ts"`
	Preview   string `json:"preview"`
	JSON      string `json:"json"`
	Base64    string `json:"base64"`
}

func (a *App) ListSessionTasks(sessionID string, limit int) ([]SessionTaskView, error) {
	return listSessionTasks(a.CaptureStore, sessionID, limit)
}

func (a *App) GetTaskCallPayloads(chainRef string) ([]TaskPayloadView, error) {
	return getTaskCallPayloads(a.CaptureStore, chainRef)
}

// listSessionTasks joins request envelopes with their completion rows and,
// when the envelope carries a run, the evidence row for that run+stage.
func listSessionTasks(st *store.Store, sessionID string, limit int) ([]SessionTaskView, error) {
	if st == nil {
		return nil, nil
	}
	envelopes, err := st.Query(store.Filter{
		Kind: store.KindCall, Direction: store.DirectionRequest,
		SessionID: sessionID, Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]SessionTaskView, 0, len(envelopes))
	for _, env := range envelopes {
		view := SessionTaskView{
			ID: env.ID, TS: env.TS, Method: env.Method,
			RunID: env.RunID, StageID: env.StageID, ChainRef: env.ChainRef,
		}
		if err := applyCompletion(st, &view); err != nil {
			return nil, err
		}
		if env.RunID != "" {
			evidence, err := evidenceStatus(st, env.RunID, env.StageID)
			if err != nil {
				return nil, err
			}
			view.Evidence = evidence
		}
		out = append(out, view)
	}
	return out, nil
}

func applyCompletion(st *store.Store, view *SessionTaskView) error {
	completions, err := st.Query(store.Filter{
		Kind: store.KindCall, Direction: store.DirectionComplete,
		ChainRef: view.ChainRef, Limit: 1,
	})
	if err != nil {
		return err
	}
	if len(completions) == 0 {
		return nil
	}
	view.Status = completions[0].Status
	var body struct {
		Error         string `json:"error"`
		RequestBytes  int64  `json:"request_bytes"`
		ResponseBytes int64  `json:"response_bytes"`
	}
	if json.Unmarshal([]byte(completions[0].JSON), &body) == nil {
		view.Error = body.Error
		view.RequestBytes = body.RequestBytes
		view.ResponseBytes = body.ResponseBytes
	}
	return nil
}

func evidenceStatus(st *store.Store, runID, stageID string) (string, error) {
	evidence, err := st.Query(store.Filter{
		Kind: store.KindEvidence, RunID: runID, StageID: stageID, Limit: 1,
	})
	if err != nil {
		return "", err
	}
	if len(evidence) == 0 {
		return "", nil
	}
	return evidence[0].Status, nil
}

// getTaskCallPayloads returns the request/response messages of one call in
// capture order, decoded through the registered Sliver descriptors.
func getTaskCallPayloads(st *store.Store, chainRef string) ([]TaskPayloadView, error) {
	if st == nil {
		return nil, nil
	}
	rows, err := st.Query(store.Filter{
		Kind: store.KindMessage, ChainRef: chainRef, Asc: true, Limit: 20,
	})
	if err != nil {
		return nil, err
	}
	out := make([]TaskPayloadView, 0, len(rows))
	for _, row := range rows {
		payload, err := st.GetPayload(row.ID)
		if err != nil {
			return nil, err
		}
		js, b64 := tasksview.Decode(row.Method, row.Direction, payload)
		out = append(out, TaskPayloadView{
			Direction: row.Direction, TS: row.TS, Preview: row.JSON,
			JSON: js, Base64: b64,
		})
	}
	return out, nil
}
