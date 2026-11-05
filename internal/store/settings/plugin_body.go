package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/dsh"
)

// NamespaceBinding captures one namespace and Host generation without changing
// the settings rail selection. All field and Save callbacks retain this owner.
type NamespaceBinding struct {
	store         *Store
	namespace     string
	generation    uint64
	client        *dsh.Client
	owner         string
	providerDraft *providerDraft
}

// Owner is an ephemeral UI cache identity. It contains no endpoint credentials.
func (b *NamespaceBinding) Owner() string {
	if b.owner != "" {
		return b.owner
	}
	return fmt.Sprintf("%p/%d/%s", b.client, b.generation, b.namespace)
}

// SetForm stores transient structured-editor input without marking settings
// dirty. The binding keeps a late UI callback on its original Host/namespace.
func (b *NamespaceBinding) SetForm(key, value string) {
	if b.providerDraft != nil {
		b.setProviderForm(key, value)
		return
	}
	if !strings.HasPrefix(key, "@") {
		return
	}
	s := b.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !b.currentLocked() {
		return
	}
	s.ensureDraftsLocked(b.namespace)
	s.drafts[b.namespace][key] = value
}

func (s *Store) BindNamespace(namespace string) *NamespaceBinding {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findNamespaceLocked(namespace) == nil {
		return nil
	}
	s.ensureDraftsLocked(namespace)
	return &NamespaceBinding{store: s, namespace: namespace, generation: s.generation, client: s.client, owner: fmt.Sprintf("%p/%d/%s", s.client, s.generation, namespace)}
}
func (b *NamespaceBinding) currentLocked() bool {
	return b.store.generation == b.generation && b.store.client == b.client && b.store.findNamespaceLocked(b.namespace) != nil && (b.providerDraft == nil || b.providerDraft.page == providerPage(b.store))
}
func (b *NamespaceBinding) Snapshot() ViewState {
	if b.providerDraft != nil {
		return b.providerSnapshot()
	}
	s := b.store
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{SelectedNS: b.namespace, Drafts: map[string]string{}}
	if !b.currentLocked() {
		return st
	}
	st.ClientReady = s.client != nil
	st.Loading = s.loading
	st.Saving = s.saving || s.themeWriteInFlightLocked(b.namespace)
	st.Writable = s.writable
	st.LastAction = s.lastAction
	st.Resets = copyFlags(s.resets[b.namespace])
	st.Dirty = len(s.dirty[b.namespace]) > 0
	if s.err != nil {
		st.Err = s.err.Error()
	}
	ns := *s.findNamespaceLocked(b.namespace)
	ns.DraftRevision, ns.HasDraftRevision = s.draftRevisions[b.namespace]
	ns.Fields = append([]Field{}, ns.Fields...)
	ns.Value = append(json.RawMessage{}, ns.Value...)
	ns.Base = append(json.RawMessage{}, ns.Base...)
	ns.User = append(json.RawMessage{}, ns.User...)
	ns.Schema = append(json.RawMessage{}, ns.Schema...)
	ns.SecretPaths = cloneSecretPaths(ns.SecretPaths)
	st.Namespaces = []Namespace{ns}
	st.Drafts = copyDrafts(s.drafts[b.namespace])
	return st
}
func (b *NamespaceBinding) SetDraft(key, value string) {
	if b.providerDraft != nil {
		b.setProviderDraft(key, value, false)
		return
	}
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || !s.writable || s.loading || s.saving || s.themeWriteInFlightLocked(b.namespace) {
		s.mu.Unlock()
		return
	}
	ns := s.findNamespaceLocked(b.namespace)
	known := false
	for _, f := range ns.Fields {
		if f.Key == key {
			known = true
			break
		}
	}
	if !known {
		s.mu.Unlock()
		return
	}
	s.beginDraftLocked(b.namespace)
	s.ensureDraftsLocked(b.namespace)
	if s.dirty[b.namespace] == nil {
		s.dirty[b.namespace] = map[string]bool{}
	}
	s.drafts[b.namespace][key] = value
	s.dirty[b.namespace][key] = true
	delete(s.resets[b.namespace], key)
	s.err, s.lastAction = nil, ""
	s.mu.Unlock()
	s.fire()
}
func (b *NamespaceBinding) Reset(key string) {
	if b.providerDraft != nil {
		b.setProviderDraft(key, "", true)
		return
	}
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || !s.writable || s.loading || s.saving || s.themeWriteInFlightLocked(b.namespace) {
		s.mu.Unlock()
		return
	}
	ns := s.findNamespaceLocked(b.namespace)
	for _, field := range ns.Fields {
		if field.Key == key {
			s.beginDraftLocked(b.namespace)
			s.ensureDraftsLocked(b.namespace)
			if s.resets == nil {
				s.resets = map[string]map[string]bool{}
			}
			if s.resets[b.namespace] == nil {
				s.resets[b.namespace] = map[string]bool{}
			}
			if s.dirty[b.namespace] == nil {
				s.dirty[b.namespace] = map[string]bool{}
			}
			s.resets[b.namespace][key] = true
			s.dirty[b.namespace][key] = true
			s.drafts[b.namespace][key] = ValueDraft(ns.Base, key, field.Kind)
			s.err, s.lastAction = nil, ""
			break
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (b *NamespaceBinding) Discard() {
	if b.providerDraft != nil {
		b.discardProviderDraft()
		return
	}
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || s.saving || s.themeWriteInFlightLocked(b.namespace) {
		s.mu.Unlock()
		return
	}
	delete(s.drafts, b.namespace)
	delete(s.dirty, b.namespace)
	delete(s.resets, b.namespace)
	delete(s.draftRevisions, b.namespace)
	s.ensureDraftsLocked(b.namespace)
	s.mu.Unlock()
	s.fire()
}

// Save uses the existing field diff, secret-preservation guard and namespace
// revision. Its reply acknowledges only the drafts that were actually sent.
func (b *NamespaceBinding) Save(ctx context.Context) {
	if b.providerDraft != nil {
		go func() {
			ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()
			_, _ = b.SaveProviderSettings(ctx)
		}()
		return
	}
	b.SaveSequence(ctx)
}
