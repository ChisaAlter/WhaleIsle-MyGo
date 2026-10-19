package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// NamespaceBinding captures one namespace and Host generation without changing
// the settings rail selection. All field and Save callbacks retain this owner.
type NamespaceBinding struct {
	store      *Store
	namespace  string
	generation uint64
	client     *dsh.Client
}

// Owner is an ephemeral UI cache identity. It contains no endpoint credentials.
func (b *NamespaceBinding) Owner() string {
	return fmt.Sprintf("%p/%d/%s", b.client, b.generation, b.namespace)
}

// SetForm stores transient structured-editor input without marking settings
// dirty. The binding keeps a late UI callback on its original Host/namespace.
func (b *NamespaceBinding) SetForm(key, value string) {
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
	return &NamespaceBinding{s, namespace, s.generation, s.client}
}
func (b *NamespaceBinding) currentLocked() bool {
	return b.store.generation == b.generation && b.store.client == b.client && b.store.findNamespaceLocked(b.namespace) != nil
}
func (b *NamespaceBinding) Snapshot() ViewState {
	s := b.store
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{SelectedNS: b.namespace, Drafts: map[string]string{}}
	if !b.currentLocked() {
		return st
	}
	st.ClientReady = s.client != nil
	st.Loading = s.loading
	st.Saving = s.saving
	st.Writable = s.writable
	st.LastAction = s.lastAction
	st.Resets = copyFlags(s.resets[b.namespace])
	st.Dirty = len(s.dirty[b.namespace]) > 0
	if s.err != nil {
		st.Err = s.err.Error()
	}
	ns := *s.findNamespaceLocked(b.namespace)
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
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || !s.writable || s.loading || s.saving {
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
	s.ensureDraftsLocked(b.namespace)
	if s.dirty[b.namespace] == nil {
		s.dirty[b.namespace] = map[string]bool{}
	}
	s.drafts[b.namespace][key] = value
	s.dirty[b.namespace][key] = true
	delete(s.resets[b.namespace], key)
	s.mu.Unlock()
	s.fire()
}
func (b *NamespaceBinding) Reset(key string) {
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || !s.writable || s.loading || s.saving {
		s.mu.Unlock()
		return
	}
	ns := s.findNamespaceLocked(b.namespace)
	for _, field := range ns.Fields {
		if field.Key == key {
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
			break
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (b *NamespaceBinding) Discard() {
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || s.saving {
		s.mu.Unlock()
		return
	}
	delete(s.drafts, b.namespace)
	delete(s.dirty, b.namespace)
	delete(s.resets, b.namespace)
	s.ensureDraftsLocked(b.namespace)
	s.mu.Unlock()
	s.fire()
}

// Save uses the existing field diff, secret-preservation guard and namespace
// revision. Its reply acknowledges only the drafts that were actually sent.
func (b *NamespaceBinding) Save(ctx context.Context) {
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || s.saving || s.loading || !s.writable || s.client == nil {
		s.mu.Unlock()
		return
	}
	ns := *s.findNamespaceLocked(b.namespace)
	drafts := copyDrafts(s.drafts[b.namespace])
	resets := copyFlags(s.resets[b.namespace])
	patch, ops, submitted, err := draftWrite(ns, drafts, s.dirty[b.namespace], resets)
	if err == nil && !resets[CustomInstructionsField] {
		err = validateCustomInstructions(ns.NS, drafts)
	}
	if err != nil {
		s.err = err
		s.lastAction = err.Error()
		s.mu.Unlock()
		s.fire()
		return
	}
	if len(patch) == 0 && len(ops) == 0 {
		s.acknowledgeDraftsLocked(ns, submitted, resets)
		s.mu.Unlock()
		s.fire()
		return
	}
	s.saving = true
	s.err = nil
	s.mu.Unlock()
	s.fire()
	go func() {
		method := "settings/update"
		args := map[string]any{"ns": ns.NS, "patch": patch, "expectedRevision": ns.Revision}
		enables := visualReplyEnabled(patch)
		if len(ops) > 0 {
			method = "settings/mutate"
			delete(args, "patch")
			args["ops"] = ops
			enables = visualReplyMutationEnables(ops)
		}
		err := s.verifyVisualReplyEnable(ctx, ns.NS, enables, b.client.Endpoint.Origin, b.generation)
		var out api.SettingsNamespaceView
		if err == nil {
			err = b.client.Call(ctx, method, args, &out)
		}
		s.mu.Lock()
		if !b.currentLocked() {
			s.mu.Unlock()
			return
		}
		s.saving = false
		if err != nil {
			s.err = err
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.err = nil
		s.lastAction = "settings saved"
		current := s.findNamespaceLocked(ns.NS)
		current.Revision = out.Revision
		current.Value = append(json.RawMessage{}, out.Value...)
		current.Base = append(json.RawMessage{}, out.Base...)
		current.User = append(json.RawMessage{}, out.User...)
		current.HasSecrets = hasSetSecrets(out.Secrets)
		current.SecretPaths = configuredSecretPaths(out.Secrets)
		s.acknowledgeDraftsLocked(*current, submitted, resets)
		s.mu.Unlock()
		s.fire()
	}()
}
