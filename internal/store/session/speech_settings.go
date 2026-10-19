package session

import (
	"context"
	"encoding/json"
	"fmt"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type SpeechSettingsView struct {
	Catalog         *api.SpeechCatalog
	Connected       bool
	Error           string
	ConnectionError string
	Saving          bool
	Submitting      map[string]bool
}
type speechReadinessState struct {
	client           *dsh.Client
	mux              *dsh.Mux
	ctx              context.Context
	generation, next uint64
	view             SpeechSettingsView
	operations       map[string]uint64
}
type SpeechSettingsOwner struct {
	store      *Store
	client     *dsh.Client
	generation uint64
	pageKey    string
}

func cloneSpeechCatalog(catalog *api.SpeechCatalog) *api.SpeechCatalog {
	if catalog == nil {
		return nil
	}
	copy := *catalog
	copy.Providers = append([]api.SpeechCatalogProvidersItem(nil), catalog.Providers...)
	for i := range copy.Providers {
		provider := &copy.Providers[i]
		provider.Languages = append([]string(nil), provider.Languages...)
		provider.DownloadSources = append([]string(nil), provider.DownloadSources...)
		provider.Preparation = append(json.RawMessage(nil), provider.Preparation...)
		if provider.SetupEstimate != nil {
			estimate := *provider.SetupEstimate
			provider.SetupEstimate = &estimate
		}
	}
	return &copy
}

// BindSpeechSettings freezes the Host readiness generation for every page
// action. Session navigation does not own speech preferences or preparation.
func (s *Store) BindSpeechSettings() (*SpeechSettingsOwner, SpeechSettingsView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &s.speechReadiness
	v := r.view
	v.Catalog = cloneSpeechCatalog(v.Catalog)
	v.Submitting = map[string]bool{}
	for key := range r.operations {
		if key == "@selection" {
			v.Saving = true
		} else {
			v.Submitting[key] = true
		}
	}
	if r.client != s.client || r.mux != s.mux {
		v = SpeechSettingsView{}
	}
	return &SpeechSettingsOwner{s, s.client, r.generation, fmt.Sprintf("%p:%d", s.client, s.composerHostGeneration)}, v
}
func (o *SpeechSettingsOwner) currentLocked() bool {
	return o != nil && o.client != nil && o.store.client == o.client && o.store.speechReadiness.client == o.client && o.store.speechReadiness.mux == o.store.mux && o.store.speechReadiness.generation == o.generation
}
func (o *SpeechSettingsOwner) run(key, endpoint string, args map[string]any) {
	s := o.store
	s.mu.Lock()
	if !o.currentLocked() || !s.speechReadiness.view.Connected || s.speechReadiness.operations[key] != 0 {
		s.mu.Unlock()
		return
	}
	r := &s.speechReadiness
	r.next++
	token := r.next
	if r.operations == nil {
		r.operations = map[string]uint64{}
	}
	r.operations[key] = token
	r.view.Error = ""
	ctx := r.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		var result json.RawMessage
		err := o.client.Call(ctx, endpoint, args, &result)
		s.mu.Lock()
		if !o.currentLocked() || s.speechReadiness.operations[key] != token {
			s.mu.Unlock()
			return
		}
		delete(s.speechReadiness.operations, key)
		if err != nil {
			s.speechReadiness.view.Error = err.Error()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (o *SpeechSettingsOwner) Configure(provider, language string) {
	patch := api.SpeechSelectionPatch{}
	if provider != "" {
		patch.ProviderId = &provider
	}
	if language != "" {
		patch.Language = &language
	}
	if patch.ProviderId == nil && patch.Language == nil {
		return
	}
	o.run("@selection", "speech/configure", map[string]any{"patch": patch})
}
func (o *SpeechSettingsOwner) Prepare(provider, source string) {
	args := map[string]any{"providerId": provider}
	if source != "" {
		args["options"] = api.SpeechPreparationOptions{DownloadSource: &source}
	}
	o.run(provider, "speech/prepare", args)
}
func (o *SpeechSettingsOwner) CancelPreparation(provider string) {
	o.run(provider, "speech/cancelPreparation", map[string]any{"providerId": provider})
}
func (o *SpeechSettingsOwner) Retry() {
	o.store.watchSpeechCatalog(context.Background(), o)
}

// Key retires page-local source choices and dismissed feedback on Host changes.
func (o *SpeechSettingsOwner) Key() string { return o.pageKey }

func (s *Store) speechReadinessFlagsLocked() (known, ready, installation bool) {
	r := &s.speechReadiness
	known = r.client != nil && r.client == s.client && r.mux == s.mux
	if !known || r.view.Catalog == nil {
		return
	}
	for _, provider := range r.view.Catalog.Providers {
		if provider.Id != r.view.Catalog.Selection.ProviderId {
			continue
		}
		var state struct{ Phase string }
		_ = json.Unmarshal(provider.Preparation, &state)
		ready = r.view.Connected && (state.Phase == "ready" || state.Phase == "standby" || state.Phase == "waking")
		installation = r.view.Connected && provider.Location == "host-local" && state.Phase == "unprepared"
		break
	}
	return
}
func (o *SpeechSettingsOwner) DismissError() {
	s := o.store
	s.mu.Lock()
	if !o.currentLocked() {
		s.mu.Unlock()
		return
	}
	s.speechReadiness.view.Error = ""
	s.mu.Unlock()
	s.fire()
}

func (s *Store) failSpeechReadiness(ctx context.Context, mux *dsh.Mux, generation uint64, err error) {
	s.mu.Lock()
	r := &s.speechReadiness
	if ctx.Err() != nil || s.mux != mux || r.client != s.client || r.generation != generation {
		s.mu.Unlock()
		return
	}
	r.view.Connected = false
	if err != nil {
		r.view.ConnectionError = fmt.Sprint(err)
	}
	s.mu.Unlock()
	s.fire()
}
