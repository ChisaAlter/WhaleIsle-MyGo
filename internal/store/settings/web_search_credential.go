package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"whaleisle/internal/dsh"
)

type WebSearchCredentialView struct {
	Ref, Draft, Error                    string
	Known, Configured, Writable, Loading bool
}

// WebSearchCredential is the original card's write-only credential field. Its
// only targets are the served provider's apiKeyEnv or the provider's declared
// DEEPSEEK_API_KEY default. It never reads key contents or desktop credentials.
type WebSearchCredential struct {
	store      *Store
	client     *dsh.Client
	generation uint64
	state      WebSearchCredentialView
	read       uint64
}

func (s *Store) BindWebSearchCredential() *WebSearchCredential {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.findNamespaceLocked("web-search-deepseek") == nil {
		return nil
	}
	if s.webSearchKey == nil || s.webSearchKey.client != s.client || s.webSearchKey.generation != s.generation {
		s.webSearchKey = &WebSearchCredential{store: s, client: s.client, generation: s.generation}
	}
	return s.webSearchKey
}

func (s *Store) webSearchReferenceLocked() string {
	name := "web-search-deepseek"
	if s.findNamespaceLocked("web-deepseek-api") != nil && s.findNamespaceLocked("web") != nil {
		s.ensureDraftsLocked("web")
		if s.drafts["web"]["searchProvider"] == "deepseek-search-api" {
			name = "web-deepseek-api"
		}
	}
	ns := s.findNamespaceLocked(name)
	if ns == nil {
		return ""
	}
	var value struct {
		Ref string `json:"apiKeyEnv"`
	}
	if json.Unmarshal(ns.Value, &value) != nil {
		return ""
	}
	if value.Ref == "" {
		return "DEEPSEEK_API_KEY"
	}
	return value.Ref
}

func (k *WebSearchCredential) currentLocked() bool {
	return k.store.client == k.client && k.store.generation == k.generation && k.store.findNamespaceLocked("web-search-deepseek") != nil
}

func (k *WebSearchCredential) Snapshot() WebSearchCredentialView {
	s := k.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !k.currentLocked() {
		return WebSearchCredentialView{}
	}
	ref := s.webSearchReferenceLocked()
	if ref != k.state.Ref {
		k.read++
		k.state.Ref = ref
		k.state.Known = false
		k.state.Configured = false
		k.state.Writable = false
		k.state.Loading = false
	}
	return k.state
}

func (k *WebSearchCredential) Edit(text string) {
	s := k.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !k.currentLocked() || s.saving {
		return
	}
	k.state.Draft, k.state.Error = text, ""
}
func (k *WebSearchCredential) Discard() { k.Edit("") }

func (k *WebSearchCredential) Refresh(ctx context.Context) {
	s := k.store
	s.mu.Lock()
	if !k.currentLocked() || k.state.Loading {
		s.mu.Unlock()
		return
	}
	ref := s.webSearchReferenceLocked()
	if ref == "" {
		s.mu.Unlock()
		return
	}
	k.read++
	read := k.read
	k.state.Ref, k.state.Loading, k.state.Error = ref, true, ""
	s.mu.Unlock()
	s.fire()
	go func() {
		states, err := DescribeCredentials(ctx, k.client, []string{ref})
		s.mu.Lock()
		if !k.currentLocked() || k.read != read || s.webSearchReferenceLocked() != ref {
			s.mu.Unlock()
			return
		}
		k.state.Loading = false
		state, known := states[ref]
		if err != nil || !known {
			k.state.Known = false
			k.state.Error = "Unable to read Web Search key status; refresh to retry"
		} else {
			k.state.Known = true
			k.state.Configured = state.Configured
			k.state.Writable = state.Writable
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (k *WebSearchCredential) Save(ctx context.Context, first *NamespaceBinding, following ...*NamespaceBinding) {
	first.saveSequence(ctx, k, following...)
}

// saveLiteral is called after provider settings settle and before the provider
// selector mutation. A read-only or changed reference refuses the key write.
func (k *WebSearchCredential) saveLiteral(ctx context.Context, ref, literal string) error {
	s := k.store
	s.mu.Lock()
	current := k.currentLocked() && s.webSearchReferenceLocked() == ref
	s.mu.Unlock()
	if !current {
		return fmt.Errorf("Web Search credential reference changed; draft retained")
	}
	states, err := DescribeCredentials(ctx, k.client, []string{ref})
	state, known := states[ref]
	if err != nil || !known || !state.Writable {
		return fmt.Errorf("Web Search key is unavailable or read-only; draft retained")
	}
	s.mu.Lock()
	current = k.currentLocked() && s.webSearchReferenceLocked() == ref
	s.mu.Unlock()
	if !current {
		return fmt.Errorf("Web Search credential reference changed; draft retained")
	}
	if err = SetCredential(ctx, k.client, ref, literal); err != nil {
		return fmt.Errorf("Unable to save Web Search key; draft retained")
	}
	states, err = DescribeCredentials(ctx, k.client, []string{ref})
	state, known = states[ref]
	s.mu.Lock()
	defer s.mu.Unlock()
	if !k.currentLocked() || s.webSearchReferenceLocked() != ref {
		return fmt.Errorf("Web Search credential reference changed")
	}
	if err != nil || !known || !state.Configured {
		return fmt.Errorf("Web Search key acceptance is unconfirmed; draft retained")
	}
	k.state.Known, k.state.Configured, k.state.Writable = true, state.Configured, state.Writable
	if k.state.Draft == literal {
		k.state.Draft = ""
	}
	return nil
}
