package session


import (
	"context"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

// ModelOption retains the exact route and adapter-owned reasoning metadata.
// Labels are presentation only; requests are always built from this object.
type ModelOption struct {
	Name         string
	ProviderName string
	Label        string
	Provider     string
	Model        string
	Reasoning    *api.ModelCatalogGroupsItemModelsItemReasoning
}

type modelRoute struct{ provider, model string }

func modelChoiceKey(provider, model string) modelRoute { return modelRoute{provider, model} }

func cloneModelSelection(v api.ModelCatalogDefault) api.ModelCatalogDefault {
	if v.ReasoningEffort != nil {
		effort := *v.ReasoningEffort
		v.ReasoningEffort = &effort
	}
	return v
}

func flattenModelCatalog(cat api.ModelCatalog) []ModelOption {
	var out []ModelOption
	seen := map[modelRoute]bool{}
	counts := map[string]int{}
	for _, g := range cat.Groups {
		for _, m := range g.Models {
			key := modelChoiceKey(g.Id, m.Id)
			if seen[key] || g.Id == "" || m.Id == "" {
				continue
			}
			seen[key] = true
			label := m.Name
			if label == "" {
				label = m.Id
			}
			name := label
			if g.Name != "" && g.Name != label {
				label = fmt.Sprintf("%s — %s", g.Name, label)
			}
			opt := ModelOption{Label: label, Name: name, ProviderName: g.Name, Provider: g.Id, Model: m.Id}
			if m.Reasoning != nil {
				r := *m.Reasoning
				r.Efforts = append([]api.ModelCatalogGroupsItemModelsItemReasoningEffortsItem(nil), r.Efforts...)
				if r.DefaultEffort != nil {
					effort := *r.DefaultEffort
					r.DefaultEffort = &effort
				}
				opt.Reasoning = &r
			}
			counts[label]++
			out = append(out, opt)
		}
	}
	// SelectBase carries string values: disambiguate human-readable collisions,
	// then ensure even an adversarial catalog caption cannot alias another row.
	used := map[string]bool{}
	for i := range out {
		label := out[i].Label
		if counts[label] > 1 {
			label += fmt.Sprintf(" [%s / %s]", out[i].Provider, out[i].Model)
		}
		candidate := label
		for n := 2; used[candidate]; n++ {
			candidate = fmt.Sprintf("%s (%d)", label, n)
		}
		out[i].Label = candidate
		used[candidate] = true
	}
	return out
}

func modelOptionForSelection(sel api.ModelCatalogDefault, options []ModelOption) (ModelOption, bool) {
	for _, o := range options {
		if o.Provider == sel.Provider && o.Model == sel.Model {
			return o, true
		}
	}
	return ModelOption{}, false
}

func labelForSelection(sel api.ModelCatalogDefault, options []ModelOption) string {
	if o, ok := modelOptionForSelection(sel, options); ok {
		return o.Label
	}
	if sel.Model != "" {
		// The composer's model seat shows the native provider/model pair with no
		// spaces around the slash (installed hero: "deepseek-official/deepseek-flash").
		return sel.Provider + "/" + sel.Model
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

type modelEffortOption struct {
	Label string
	Value *string
}

func modelEffortOptions(opt ModelOption) []modelEffortOption {
	if opt.Reasoning == nil {
		return nil
	}
	var out []modelEffortOption
	if opt.Reasoning.DefaultEffort == nil {
		out = append(out, modelEffortOption{Label: storelocale.T(storelocale.KeyStoreSessionProviderDefault)})
	}
	counts := map[string]int{}
	for _, e := range opt.Reasoning.Efforts {
		counts[e.Name]++
	}
	seen := map[string]bool{}
	labels := map[string]bool{storelocale.T(storelocale.KeyStoreSessionProviderDefault): opt.Reasoning.DefaultEffort == nil}
	for _, e := range opt.Reasoning.Efforts {
		if seen[e.Id] {
			continue
		}
		seen[e.Id] = true
		label := e.Name
		if label == "" {
			label = e.Id
		}
		if counts[e.Name] > 1 || labels[label] {
			label += " [" + e.Id + "]"
		}
		candidate := label
		for n := 2; labels[candidate]; n++ {
			candidate = fmt.Sprintf("%s (%d)", label, n)
		}
		labels[candidate] = true
		id := e.Id
		out = append(out, modelEffortOption{Label: candidate, Value: &id})
	}
	return out
}

func effectiveModelEffort(sel api.ModelCatalogDefault, options []ModelOption) *string {
	if sel.ReasoningEffort != nil {
		return sel.ReasoningEffort
	}
	if o, ok := modelOptionForSelection(sel, options); ok && o.Reasoning != nil {
		return o.Reasoning.DefaultEffort
	}
	return nil
}

func sameModelSelection(a, b api.ModelCatalogDefault, options []ModelOption) bool {
	if a.Provider != b.Provider || a.Model != b.Model {
		return false
	}
	ae, be := effectiveModelEffort(a, options), effectiveModelEffort(b, options)
	return (ae == nil && be == nil) || (ae != nil && be != nil && *ae == *be)
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

// RefreshModelCatalog refreshes Host-level choices, never the active session's
// selection. The default applies only after modelSelection.next is known null.
func (s *Store) RefreshModelCatalog(ctx context.Context) {
	s.mu.Lock()
	client, generation := s.client, s.eventsGeneration
	s.modelState.CatalogRequest++
	request := s.modelState.CatalogRequest
	s.mu.Unlock()
	if client == nil || ctx.Err() != nil {
		return
	}
	go func() {
		var cat api.ModelCatalog
		err := client.Call(ctx, "session/modelCatalog", map[string]any{}, &cat)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation || request != s.modelState.CatalogRequest || ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		s.modelsErr = err
		if err == nil {
			s.modelOptions = flattenModelCatalog(cat)
			s.modelState.Default = cloneModelSelection(cat.Default)
			s.modelState.CatalogReady = true
			var failures []string
			for _, f := range cat.Failures {
				failures = append(failures, f.Name+": "+f.Message)
			}
			s.modelState.CatalogWarning = strings.Join(failures, "; ")
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) SelectModelForSession(ctx context.Context, label string) {
	s.mu.Lock()
	id := s.selectedID
	s.mu.Unlock()
	s.SelectModelForSessionID(ctx, id, label)
}

// SelectModelForSessionID binds a catalog row to the session rendered by the UI.
func (s *Store) SelectModelForSessionID(ctx context.Context, sessionID, label string) {
	s.mu.Lock()
	opt, ok := optionByLabel(label, s.modelOptions)
	current, known := s.currentModelSelectionLocked()
	if !ok {
		s.mu.Unlock()
		return
	}
	selection := api.ModelCatalogDefault{Provider: opt.Provider, Model: opt.Model}
	if known && current.Provider == opt.Provider && current.Model == opt.Model {
		// Re-selecting a route must not silently reset its adapter-owned effort.
		selection.ReasoningEffort = effectiveModelEffort(current, s.modelOptions)
	} else if opt.Reasoning != nil {
		selection.ReasoningEffort = opt.Reasoning.DefaultEffort
	}
	s.mu.Unlock()
	s.SelectModelSelectionForSessionID(ctx, sessionID, selection)
}

func (s *Store) SelectModelEffortForSessionID(ctx context.Context, sessionID, label string) {
	s.mu.Lock()
	selection, known := s.currentModelSelectionLocked()
	opt, ok := modelOptionForSelection(selection, s.modelOptions)
	if !known || !ok || s.selectedID != sessionID {
		s.mu.Unlock()
		return
	}
	var value *string
	found := false
	for _, e := range modelEffortOptions(opt) {
		if e.Label == label {
			value = e.Value
			found = true
			break
		}
	}
	s.mu.Unlock()
	if !found {
		return
	}
	selection.ReasoningEffort = value
	s.SelectModelEffortSelectionForSessionID(ctx, sessionID, selection)
}

// SelectModelSelectionForSessionID sends only the Host's existing request fields.
// The UI stays on committed projection state until the Host confirms the change.
func (s *Store) SelectModelSelectionForSessionID(ctx context.Context, sessionID string, selection api.ModelCatalogDefault) {
	s.selectModelSelectionForSessionID(ctx, sessionID, selection, false)
}

// Effort picks are also bound to the model route shown when the menu was built.
func (s *Store) SelectModelEffortSelectionForSessionID(ctx context.Context, sessionID string, selection api.ModelCatalogDefault) {
	s.selectModelSelectionForSessionID(ctx, sessionID, selection, true)
}

func (s *Store) selectModelSelectionForSessionID(ctx context.Context, sessionID string, selection api.ModelCatalogDefault, effortOnly bool) {
	s.selectModelSelectionWithReceipt(ctx, sessionID, selection, effortOnly, nil, nil)
}

func (s *Store) selectModelSelectionWithReceipt(ctx context.Context, sessionID string, selection api.ModelCatalogDefault, effortOnly bool, owner *NativeCommandOwner, completed func(error)) {
	finish := func(err error) {
		if completed != nil {
			completed(err)
		}
	}
	s.mu.Lock()
	client, generation := s.client, s.followGeneration
	current, known := s.currentModelSelectionLocked()
	opt, available := modelOptionForSelection(selection, s.modelOptions)
	if client == nil || sessionID == "" || s.selectedID != sessionID || ctx.Err() != nil || !known || !available || !s.modelSelectionAvailableLocked() || s.modelState.Pending || (owner != nil && !s.nativeCommandCurrentLocked(*owner, "model")) {
		s.mu.Unlock()
		finish(fmt.Errorf("Model selection unavailable or owner changed"))
		return
	}
	if effortOnly && (current.Provider != selection.Provider || current.Model != selection.Model) {
		s.mu.Unlock()
		finish(fmt.Errorf("Model route changed"))
		return
	}
	if selection.ReasoningEffort != nil {
		valid := false
		if opt.Reasoning != nil {
			for _, e := range opt.Reasoning.Efforts {
				if e.Id == *selection.ReasoningEffort {
					valid = true
					break
				}
			}
		}
		if !valid {
			s.mu.Unlock()
			finish(fmt.Errorf("Reasoning effort is no longer available"))
			return
		}
	}
	if sameModelSelection(current, selection, s.modelOptions) {
		s.mu.Unlock()
		finish(nil)
		return
	}
	selection = cloneModelSelection(selection)
	s.modelState.Operation++
	operation := s.modelState.Operation
	startSeq := s.modelState.Projection.Seq
	s.modelState.Pending = true
	s.modelsErr = nil
	s.mu.Unlock()
	s.fire()
	go func() {
		var err error
		defer func() { finish(err) }()
		s.mu.Lock()
		valid := s.sessionResultCurrentLocked(ctx, client, sessionID, generation) && s.modelState.Operation == operation && (owner == nil || s.nativeCommandCurrentLocked(*owner, "model"))
		s.mu.Unlock()
		var out api.SessionSelectModelValue
		var baseline api.SessionProjectionsValue
		if valid {
			req := api.SessionSelectModelRequest{SessionId: sessionID, Provider: selection.Provider, Model: selection.Model, ReasoningEffort: selection.ReasoningEffort}
			err = client.Call(ctx, "session/selectModel", map[string]any{"request": req}, &out)
			if err == nil && (out.Selected.Provider == "" || out.Selected.Model == "") {
				err = errors.New(storelocale.T(storelocale.KeyStoreSessionTheHostDidNotReturnTheSubmittedModel))
			}
			s.mu.Lock()
			valid = s.sessionResultCurrentLocked(ctx, client, sessionID, generation) && s.modelState.Operation == operation && (owner == nil || s.nativeCommandCurrentLocked(*owner, "model"))
			s.mu.Unlock()
			if err == nil && valid {
				err = client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": sessionID}}, &baseline)
				if err != nil {
					err = errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreSessionModelSelectionSubmittedButReadingTheCu), err))
				}
			}
		}
		s.mu.Lock()
		if s.client != client || s.selectedID != sessionID || s.followGeneration != generation || s.modelState.Operation != operation {
			s.mu.Unlock()
			err = fmt.Errorf("Model selection owner changed")
			return
		}
		s.modelState.Pending = false
		if !valid || ctx.Err() != nil || (owner != nil && !s.nativeCommandCurrentLocked(*owner, "model")) {
			s.mu.Unlock()
			s.fire()
			err = fmt.Errorf("Model selection owner changed")
			return
		}
		if err == nil {
			s.applyModelValuesLocked(baseline.Values, baseline.AsOfSeq)
			actual, known := s.currentModelSelectionLocked()
			if !known || (!sameModelSelection(actual, out.Selected, s.modelOptions) && s.modelState.Projection.Seq <= startSeq) {
				err = errors.New(storelocale.T(storelocale.KeyStoreSessionTheHostHasNotReturnedTheCurrentSessio))
			}
		}
		s.modelsErr = err
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = storelocale.T(storelocale.KeyStoreSessionCurrentSessionModelUpdated)
		}
		s.mu.Unlock()
		s.fire()
	}()
}
