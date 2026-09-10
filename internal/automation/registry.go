package automation

import (
	"fmt"
	"sync"
)

type schemaProvider interface {
	ConfigSchema() []FieldSpec
}

func collectSchemas[T schemaProvider](entries map[string]T) map[string][]FieldSpec {
	out := make(map[string][]FieldSpec, len(entries))
	for typ, entry := range entries {
		out[typ] = entry.ConfigSchema()
	}
	return out
}

func registerIn[T any](mu *sync.RWMutex, items map[string]T, kind, typ string, item T) error {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := items[typ]; exists {
		return fmt.Errorf("%s already registered: %s", kind, typ)
	}
	items[typ] = item
	return nil
}

func lookupIn[T any](mu *sync.RWMutex, items map[string]T, typ string) (T, bool) {
	mu.RLock()
	defer mu.RUnlock()
	item, ok := items[typ]
	return item, ok
}
