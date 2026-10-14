package session

import (
	"context"
	"fmt"

	"whaleisle/internal/dsh/api"
)

// ModelOption is one row in the composer model picker.
type ModelOption struct {
	Label    string
	Provider string
	Model    string
}

func modelChoiceKey(provider, model string) string {
	return provider + "/" + model
}

func flattenModelCatalog(cat api.ModelCatalog) []ModelOption {
	var out []ModelOption
	seen := map[string]struct{}{}
	for _, g := range cat.Groups {
		for _, m := range g.Models {
			key := modelChoiceKey(g.Id, m.Id)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			label := m.Name
			if g.Name != "" && g.Name != m.Name {
				label = fmt.Sprintf("%s — %s", g.Name, m.Name)
			}
			out = append(out, ModelOption{
				Label:    label,
				Provider: g.Id,
				Model:    m.Id,
			})
		}
	}
	return out
}

func labelForSelection(sel api.ModelCatalogDefault, options []ModelOption) string {
	key := modelChoiceKey(sel.Provider, sel.Model)
	for _, o := range options {
		if modelChoiceKey(o.Provider, o.Model) == key {
			return o.Label
		}
	}
	if sel.Model != "" {
		return sel.Provider + " / " + sel.Model
	}
	return ""
}

func optionByLabel(label string, options []ModelOption) (ModelOption, bool) {
	for _, o := range options {
		if o.Label == label {
			return o, true
		}
	}
	return ModelOption{}, false
}

// DiscoverLLMProviders calls llm/listConfigurableProviders then llm/discoverModels
// for each settings namespace (settings Models page / Electron provider discovery).
func (s *Store) DiscoverLLMProviders(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var providers api.ListConfigurableProvidersResult
		err := client.Call(ctx, "llm/listConfigurableProviders", map[string]any{}, &providers)
		if err != nil {
			s.mu.Lock()
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		var listed api.ListProvidersResult
		_ = client.Call(ctx, "llm/listProviders", map[string]any{}, &listed)

		discovered := 0
		for _, p := range providers {
			ns := p.SettingsNs
			if ns == "" {
				continue
			}
			var models api.DiscoverModelsResult
			err := client.Call(ctx, "llm/discoverModels", map[string]any{
				"settingsNs": ns,
				"request":    api.LlmModelDiscoveryRequest{},
			}, &models)
			if err != nil {
				continue
			}
			discovered += len(models)
		}
		s.mu.Lock()
		s.lastAction = fmt.Sprintf("llm: %d configurable, %d listed, %d models", len(providers), len(listed), discovered)
		s.mu.Unlock()
		s.fire()
		s.RefreshModelCatalog(ctx)
	}()
}

// InitializeDefaultModel calls session/initializeDefaultModel (Electron first-run model seed).
func (s *Store) InitializeDefaultModel(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out api.InitializeDefaultModelResult
		_ = client.Call(ctx, "session/initializeDefaultModel", map[string]any{}, &out)
		s.RefreshModelCatalog(ctx)
	}()
}

// RefreshModelCatalog loads session/modelCatalog for the composer picker.
func (s *Store) RefreshModelCatalog(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var cat api.ModelCatalog
		err := client.Call(ctx, "session/modelCatalog", map[string]any{}, &cat)
		s.mu.Lock()
		if err != nil {
			s.modelsErr = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.modelsErr = nil
		s.modelOptions = flattenModelCatalog(cat)
		if lbl := labelForSelection(cat.Default, s.modelOptions); lbl != "" {
			s.selectedModelLabel = lbl
		} else if len(s.modelOptions) > 0 {
			s.selectedModelLabel = s.modelOptions[0].Label
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SelectModelForSession applies session/selectModel for the chosen catalog row.
func (s *Store) SelectModelForSession(ctx context.Context, label string) {
	s.mu.Lock()
	opt, ok := optionByLabel(label, s.modelOptions)
	client := s.client
	sessionID := s.selectedID
	prev := s.selectedModelLabel
	s.mu.Unlock()
	if !ok || client == nil || sessionID == "" || label == prev {
		return
	}
	go func() {
		req := api.SessionSelectModelRequest{
			SessionId: sessionID,
			Provider:  opt.Provider,
			Model:     opt.Model,
		}
		var out api.SessionSelectModelValue
		err := client.Call(ctx, "session/selectModel", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		if lbl := labelForSelection(out.Selected, s.modelOptions); lbl != "" {
			s.selectedModelLabel = lbl
		} else {
			s.selectedModelLabel = label
		}
		s.lastAction = "model selected"
		s.mu.Unlock()
		s.fire()
	}()
}
