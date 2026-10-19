package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"whaleisle/internal/dsh"
)

// PatchDescribedField updates one row without changing the active namespace or
// acknowledging unrelated editor drafts. Only the current Host's description
// determines its wire type and enum values.
func (s *Store) PatchDescribedField(ctx context.Context, namespace, key, value string) {
	s.PatchDescribedFields(ctx, namespace, map[string]string{key: value})
}

// PatchDescribedFields commits source-defined coupled preferences atomically.
func (s *Store) PatchDescribedFields(ctx context.Context, namespace string, values map[string]string) {
	s.patchDescribedFieldsOwned(ctx, namespace, values, nil, 0)
}

// BindDescribedPreferences binds callbacks to the Host that rendered them.
func (s *Store) BindDescribedPreferences() (ViewState, uint64, func(context.Context, string, map[string]string)) {
	s.mu.Lock()
	client, generation := s.client, s.generation
	snapshot := s.snapshotLocked()
	s.mu.Unlock()
	return snapshot, generation, func(ctx context.Context, namespace string, values map[string]string) {
		if client != nil {
			if namespace == CustomInstructionsNamespace && len(values) == 1 {
				if text, ok := values[CustomInstructionsField]; ok {
					s.queueInstructions(client, generation, text)
					return
				}
			}
			s.patchDescribedFieldsOwned(ctx, namespace, values, client, generation)
		}
	}
}

func (s *Store) patchDescribedFieldsOwned(ctx context.Context, namespace string, values map[string]string, owner *dsh.Client, ownerGeneration uint64) {
	s.mu.Lock()
	client, generation := s.client, s.generation
	ns := s.findNamespaceLocked(namespace)
	if client == nil || s.describedClient != client || s.loading || s.saving || !s.writable || ns == nil || (owner != nil && (client != owner || generation != ownerGeneration)) {
		s.mu.Unlock()
		return
	}
	if err := validateCustomInstructions(namespace, values); err != nil {
		s.err, s.lastAction = err, err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	ops := []json.RawMessage{}
	for _, key := range slices.Sorted(maps.Keys(values)) {
		value := values[key]
		var field *Field
		for i := range ns.Fields {
			if ns.Fields[i].Key == key {
				field = &ns.Fields[i]
				break
			}
		}
		if field == nil {
			s.mu.Unlock()
			return
		}
		raw, err := PatchValue(field.Kind, value)
		if err == nil && field.Kind == FieldEnum && !slices.Contains(field.EnumOptions, value) {
			err = fmt.Errorf("unsupported value for %s", key)
		}
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		ops = append(ops, fieldOperation("set", []string{key}, raw))
	}
	s.mu.Unlock()
	s.saveMutateOwned(ctx, namespace, ops, client, generation)
}
