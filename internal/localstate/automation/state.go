package automation

import (
	"context"

	"siren/internal/automation"
	"siren/internal/localstate/jsonstore"
)

type JSONStore struct {
	store *jsonstore.ScopedStore[automation.State]
}

const statePrefix = "gui-automation"

func New(rootDir string) *JSONStore {
	return &JSONStore{
		store: jsonstore.New[automation.State](rootDir, statePrefix),
	}
}

func (s *JSONStore) Load(_ context.Context) (*automation.State, error) {
	state, ok, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if !ok {
		return &automation.State{}, nil
	}
	return &state, nil
}

func (s *JSONStore) Save(_ context.Context, state *automation.State) error {
	if state == nil {
		return s.store.Save(automation.State{})
	}
	return s.store.Save(*state)
}

func (s *JSONStore) SetServer(host string, port uint32) {
	s.store.SetServer(host, port)
}
