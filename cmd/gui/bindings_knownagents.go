package gui

import knownagents "siren/internal/localstate/agents"

type KnownAgentView struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Hostname      string `json:"hostname"`
	Username      string `json:"username"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	RemoteAddress string `json:"remoteAddress"`
	Transport     string `json:"transport"`
	Status        string `json:"status"`
	FirstSeen     int64  `json:"firstSeen"`
	LastSeen      int64  `json:"lastSeen"`
	LostAt        int64  `json:"lostAt"`
}

func (a *App) ListKnownAgents() ([]KnownAgentView, error) {
	return listKnownAgents(a.KnownAgents)
}

func (a *App) RemoveKnownAgent(id string) error {
	return removeKnownAgent(a.KnownAgents, id)
}

func listKnownAgents(svc *knownagents.Service) ([]KnownAgentView, error) {
	if svc == nil {
		return []KnownAgentView{}, nil
	}
	records := svc.List()
	out := make([]KnownAgentView, 0, len(records))
	for _, r := range records {
		out = append(out, KnownAgentView{
			ID:            r.ID,
			Kind:          r.Kind,
			Name:          r.Name,
			Hostname:      r.Hostname,
			Username:      r.Username,
			OS:            r.OS,
			Arch:          r.Arch,
			RemoteAddress: r.RemoteAddress,
			Transport:     r.Transport,
			Status:        r.Status,
			FirstSeen:     r.FirstSeen,
			LastSeen:      r.LastSeen,
			LostAt:        r.LostAt,
		})
	}
	return out, nil
}

func removeKnownAgent(svc *knownagents.Service, id string) error {
	if svc == nil {
		return nil
	}
	return svc.Remove(id)
}
