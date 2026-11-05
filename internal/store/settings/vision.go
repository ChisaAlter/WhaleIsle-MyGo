package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"whaleisle/internal/dsh/api"
)

const VisionNamespace = "llm-vision-fallback"

type VisionRoute struct{ Provider, Model, Label string }
type VisionView struct {
	Present, Writable, Loading, Saving           bool
	Provider, Model, Err, CatalogErr, LastAction string
	Routes                                       []VisionRoute
}

// VisionSnapshot reads the resolved live entry, never legacy backups or session defaults.
func (s *Store) VisionSnapshot() VisionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := VisionView{Writable: s.client != nil && s.writable, Loading: s.loading || s.visionLoading, Saving: s.saving, CatalogErr: s.visionErr, LastAction: s.lastAction, Routes: append([]VisionRoute(nil), s.visionRoutes...)}
	if s.err != nil {
		v.Err = s.err.Error()
	}
	if ns := s.findNamespaceLocked(VisionNamespace); ns != nil {
		v.Present = true
		var value struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		}
		if err := json.Unmarshal(ns.Value, &value); err != nil {
			v.Err = err.Error()
			v.Writable = false
		} else {
			v.Provider, v.Model = value.Provider, value.Model
		}
	}
	return v
}

// RefreshVision uses the same global catalog contract as remote.session.modelCatalog.
func (s *Store) RefreshVision(ctx context.Context) {
	s.RefreshDescribe(ctx)
	s.RefreshImageCatalog(ctx)
}

// RefreshImageCatalog reads only session/modelCatalog, without changing or
// depending on the global vision fallback configuration.
func (s *Store) RefreshImageCatalog(ctx context.Context) {
	s.mu.Lock()
	client, generation := s.client, s.generation
	if client == nil || s.visionLoading {
		s.mu.Unlock()
		return
	}
	s.visionLoading, s.visionErr = true, ""
	s.mu.Unlock()
	s.fire()
	go func() {
		var catalog api.ModelCatalog
		err := client.Call(ctx, "session/modelCatalog", map[string]any{}, &catalog)
		var routes []VisionRoute
		if err == nil {
			for _, group := range catalog.Groups {
				for _, model := range group.Models {
					if slices.Contains(model.InputModalities, "image") {
						routes = append(routes, VisionRoute{Provider: group.Id, Model: model.Id, Label: fmt.Sprintf("%s / %s", group.Name, model.Name)})
					}
				}
			}
		}
		s.mu.Lock()
		if generation != s.generation {
			s.mu.Unlock()
			return
		}
		s.visionLoading = false
		if err != nil {
			s.visionErr = err.Error()
		} else {
			s.visionRoutes = routes
			for _, failure := range catalog.Failures {
				s.visionErr += failure.Name + ": " + failure.Message + "\n"
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SaveVision changes only the two live route fields. Off removes the user
// overrides, preserving the installed editor's base-layer semantics.
func (s *Store) SaveVision(ctx context.Context, provider, model string) {
	if (provider == "") != (model == "") {
		return
	}
	ops := make([]json.RawMessage, 0, 2)
	for _, field := range []struct{ key, value string }{{"provider", provider}, {"model", model}} {
		operation := map[string]any{"op": "set", "path": []string{field.key}, "value": field.value}
		if provider == "" {
			operation = map[string]any{"op": "unset", "path": []string{field.key}}
		}
		raw, _ := json.Marshal(operation)
		ops = append(ops, raw)
	}
	s.saveMutate(ctx, VisionNamespace, ops)
}
