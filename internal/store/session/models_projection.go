package session

import (
	"bytes"
	"encoding/json"

	"whaleisle/internal/dsh/api"
)

// ModelCatalogChoice carries display metadata only; Label still resolves through ModelChoices.
type ModelCatalogChoice struct{ Label, Name, Provider, ProviderName string }

type modelProjectionState struct {
	SessionID  string
	Generation uint64
	Seq        float64
	Present    bool
	Next       *api.ModelCatalogDefault
}

type modelSelectionState struct {
	Default        api.ModelCatalogDefault
	CatalogReady   bool
	CatalogRequest uint64
	CatalogWarning string
	Projection     modelProjectionState
	Pending        bool
	Operation      uint64
}

func (s *Store) resetModelSelectionLocked() {
	s.modelState.Projection = modelProjectionState{}
	s.modelState.Pending = false
	s.modelState.Operation++
	s.modelsErr = nil
}

func (s *Store) resetModelCatalogLocked() {
	s.modelOptions = nil
	s.modelState.Default = api.ModelCatalogDefault{}
	s.modelState.CatalogReady = false
	s.modelState.CatalogWarning = ""
	s.modelState.CatalogRequest++
	s.resetModelSelectionLocked()
}

func (s *Store) modelSelectionAvailableLocked() bool {
	for _, item := range s.items {
		if item.SessionID == s.selectedID && item.Origin == "subagent" {
			return false
		}
	}
	return s.selectedID != ""
}

func (s *Store) currentModelSelectionLocked() (api.ModelCatalogDefault, bool) {
	p := s.modelState.Projection
	if s.selectedID == "" || p.SessionID != s.selectedID || p.Generation != s.followGeneration || !p.Present {
		return api.ModelCatalogDefault{}, false
	}
	if p.Next != nil {
		return cloneModelSelection(*p.Next), true
	}
	if s.modelState.CatalogReady && s.modelState.Default.Provider != "" && s.modelState.Default.Model != "" {
		return cloneModelSelection(s.modelState.Default), true
	}
	return api.ModelCatalogDefault{}, false
}

func (s *Store) snapshotModelsLocked(st *ViewState) {
	selection, known := s.currentModelSelectionLocked()
	st.ModelSelectionKnown = known
	st.ModelSelectionBusy = s.modelState.Pending
	if s.modelsErr != nil {
		st.ModelsErr = s.modelsErr.Error()
	} else {
		st.ModelsErr = s.modelState.CatalogWarning
	}
	if !known {
		return
	}
	st.SelectedModel = labelForSelection(selection, s.modelOptions)
	st.SelectedProvider = selection.Provider
	copy := cloneModelSelection(selection)
	st.ModelSelection = &copy
	effort := effectiveModelEffort(selection, s.modelOptions)
	if effort != nil {
		st.SelectedModelEffort = *effort
	}
	if !s.modelSelectionAvailableLocked() {
		return
	}
	st.ModelChoices = make(map[string]api.ModelCatalogDefault, len(s.modelOptions))
	for _, o := range s.modelOptions {
		st.ModelLabels = append(st.ModelLabels, o.Label)
		st.ModelCatalogChoices = append(st.ModelCatalogChoices, ModelCatalogChoice{Label: o.Label, Name: o.Name, Provider: o.Provider, ProviderName: o.ProviderName})
		choice := api.ModelCatalogDefault{Provider: o.Provider, Model: o.Model}
		if o.Provider == selection.Provider && o.Model == selection.Model {
			choice.ReasoningEffort = effectiveModelEffort(selection, s.modelOptions)
		} else if o.Reasoning != nil {
			choice.ReasoningEffort = o.Reasoning.DefaultEffort
		}
		st.ModelChoices[o.Label] = cloneModelSelection(choice)
	}
	if opt, ok := modelOptionForSelection(selection, s.modelOptions); ok {
		st.ModelEffortChoices = make(map[string]api.ModelCatalogDefault)
		for _, e := range modelEffortOptions(opt) {
			st.ModelEffortLabels = append(st.ModelEffortLabels, e.Label)
			choice := cloneModelSelection(selection)
			choice.ReasoningEffort = e.Value
			st.ModelEffortChoices[e.Label] = cloneModelSelection(choice)
			if (effort == nil && e.Value == nil) || (effort != nil && e.Value != nil && *effort == *e.Value) {
				st.SelectedModelEffort = e.Label
			}
		}
	}
}

func (s *Store) applyModelValuesLocked(values json.RawMessage, seq float64) {
	var all map[string]json.RawMessage
	if json.Unmarshal(values, &all) != nil {
		return
	}
	s.applyModelProjectionLocked(all["modelSelection"], seq)
}

func (s *Store) applyModelProjectionLocked(raw json.RawMessage, seq float64) {
	if s.selectedID == "" {
		return
	}
	p := &s.modelState.Projection
	if p.SessionID == s.selectedID && p.Generation == s.followGeneration && seq < p.Seq {
		return
	}
	next := modelProjectionState{SessionID: s.selectedID, Generation: s.followGeneration, Seq: seq}
	// Missing/null is an absent capability. Only an explicit {next:null}
	// establishes that the catalog default is this session's effective route.
	raw = bytes.TrimSpace(raw)
	if len(raw) != 0 && string(raw) != "null" {
		var value struct {
			Next json.RawMessage `json:"next"`
		}
		if json.Unmarshal(raw, &value) != nil || len(value.Next) == 0 {
			return
		}
		next.Present = true
		if string(bytes.TrimSpace(value.Next)) != "null" {
			var selection api.ModelCatalogDefault
			if json.Unmarshal(value.Next, &selection) != nil || selection.Provider == "" || selection.Model == "" {
				return
			}
			copy := cloneModelSelection(selection)
			next.Next = &copy
		}
	}
	*p = next
}

func (s *Store) applyModelControlLocked(raw json.RawMessage) {
	var frame struct {
		Type      string          `json:"type"`
		SessionID string          `json:"sessionId"`
		Key       string          `json:"key"`
		Seq       float64         `json:"seq"`
		Value     json.RawMessage `json:"value"`
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Type != "projection" || frame.SessionID != s.selectedID || frame.Key != "modelSelection" {
		return
	}
	s.applyModelProjectionLocked(frame.Value, frame.Seq)
}

func (s *Store) applyModelSelectionEventLocked(raw json.RawMessage) bool {
	var event struct {
		Seq  float64                 `json:"seq"`
		Data api.ModelCatalogDefault `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil || event.Data.Provider == "" || event.Data.Model == "" {
		return false
	}
	value, _ := json.Marshal(struct {
		Next api.ModelCatalogDefault `json:"next"`
	}{event.Data})
	s.applyModelProjectionLocked(value, event.Seq)
	return true
}
