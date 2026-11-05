package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"whaleisle/internal/dsh/api"
)

// A provider editor owns its drafts, not the namespace browser. The UI holds
// this binding only while its card is mounted. Leaving settings retires every
// old callback, including callbacks from a modal that was open on that page.
type providerDraft struct {
	page          uint64
	ns            Namespace
	drafts        map[string]string
	dirty, resets map[string]bool
	busy          bool
	err           string
	path          []string
	paths         map[string][]string
}

var providerPages sync.Map // *Store -> *atomic.Uint64
func providerPage(s *Store) uint64 {
	p, _ := providerPages.LoadOrStore(s, &atomic.Uint64{})
	return p.(*atomic.Uint64).Load()
}
func (s *Store) LeaveProviderForms() {
	p, _ := providerPages.LoadOrStore(s, &atomic.Uint64{})
	p.(*atomic.Uint64).Add(1)
}

func (s *Store) BindProviderEditor(namespace, provider string, path []string) *NamespaceBinding {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := s.findNamespaceLocked(namespace)
	if ns == nil {
		return nil
	}
	d := &providerDraft{page: providerPage(s), ns: *ns, drafts: map[string]string{}, dirty: map[string]bool{}, resets: map[string]bool{}, path: append([]string(nil), path...), paths: map[string][]string{}}
	for _, f := range ns.Fields {
		d.drafts[f.Key] = ValueDraft(ns.Value, f.Key, f.Kind)
	}
	return &NamespaceBinding{store: s, namespace: namespace, generation: s.generation, client: s.client, owner: fmt.Sprintf("%p/%d/%d/%s/%s", s.client, s.generation, d.page, namespace, provider), providerDraft: d}
}
func (b *NamespaceBinding) providerSnapshot() ViewState {
	s := b.store
	s.mu.Lock()
	defer s.mu.Unlock()
	d := b.providerDraft
	st := ViewState{SelectedNS: b.namespace, Drafts: map[string]string{}}
	if !b.currentLocked() {
		return st
	}
	ns := d.ns
	ns.DraftRevision, ns.HasDraftRevision = ns.Revision, len(d.dirty) > 0
	st.ClientReady = s.client != nil
	st.Loading = s.loading
	st.Writable = s.writable
	st.Saving = d.busy
	st.Err = d.err
	st.Dirty = len(d.dirty) > 0
	st.Resets = copyFlags(d.resets)
	st.Drafts = copyDrafts(d.drafts)
	st.Namespaces = []Namespace{ns}
	return st
}
func (b *NamespaceBinding) setProviderForm(key, value string) {
	if !strings.HasPrefix(key, "@") {
		return
	}
	s := b.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.currentLocked() {
		b.providerDraft.drafts[key] = value
	}
}
func (b *NamespaceBinding) setProviderDraft(key, value string, reset bool) {
	s := b.store
	s.mu.Lock()
	d := b.providerDraft
	if !b.currentLocked() || !s.writable || s.loading || d.busy {
		s.mu.Unlock()
		return
	}
	if len(d.path) > 0 && key != d.path[0] {
		s.mu.Unlock()
		return
	}
	for _, f := range d.ns.Fields {
		if f.Key == key {
			changed := []json.RawMessage{}
			if f.Kind == FieldJSON && !reset {
				diffVisibleJSON(json.RawMessage(d.drafts[key]), json.RawMessage(value), []string{key}, &changed)
			} else {
				changed = append(changed, fieldOperation("set", []string{key}, nil))
			}
			for _, op := range changed {
				var v struct {
					Path []string `json:"path"`
				}
				_ = json.Unmarshal(op, &v)
				if len(d.path) > 0 && providerContains(v.Path, d.path) {
					v.Path = append([]string(nil), d.path...)
				}
				if len(d.path) == 0 || providerContains(d.path, v.Path) {
					encoded, _ := json.Marshal(v.Path)
					d.paths[string(encoded)] = v.Path
				}
			}
			if reset {
				value = ValueDraft(d.ns.Base, key, f.Kind)
				d.resets[key] = true
			} else {
				delete(d.resets, key)
			}
			d.drafts[key] = value
			d.dirty[key] = true
			d.err = ""
			break
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (b *NamespaceBinding) discardProviderDraft() {
	s := b.store
	s.mu.Lock()
	d := b.providerDraft
	if !b.currentLocked() || d.busy {
		s.mu.Unlock()
		return
	}
	ns := s.findNamespaceLocked(b.namespace)
	d.ns = *ns
	d.drafts = map[string]string{}
	d.dirty = map[string]bool{}
	d.resets = map[string]bool{}
	d.paths = map[string][]string{}
	d.err = ""
	for _, f := range ns.Fields {
		d.drafts[f.Key] = ValueDraft(ns.Value, f.Key, f.Kind)
	}
	s.mu.Unlock()
	s.fire()
}

func (b *NamespaceBinding) saveProviderDraft(ctx context.Context) (api.SettingsNamespaceView, error) {
	s := b.store
	s.mu.Lock()
	d := b.providerDraft
	if !b.currentLocked() || s.client == nil {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsNotReady
	}
	if d.busy || s.loading {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsBusy
	}
	if !s.writable {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsReadOnly
	}
	ns := d.ns
	drafts, resets := copyDrafts(d.drafts), copyFlags(d.resets)
	_, _, submitted, err := draftWrite(ns, drafts, d.dirty, resets)
	if err == nil {
		err = validateServedDraft(ns.NS, drafts, d.dirty, resets)
	}
	patch := map[string]json.RawMessage{}
	var ops []json.RawMessage
	// Diff only leaves actually edited in this card against its USER layer.
	// Inherited fields stay inherited; typing a base value can explicitly pin it.
	if err == nil {
		root := map[string]json.RawMessage{}
		for _, f := range ns.Fields {
			raw, e := PatchValue(f.Kind, drafts[f.Key])
			if e != nil {
				continue
			}
			root[f.Key] = raw
		}
		afterRoot, _ := json.Marshal(root)
		paths := [][]string{}
		for _, path := range d.paths {
			paths = append(paths, path)
		}
		sort.Slice(paths, func(i, j int) bool { return len(paths[i]) < len(paths[j]) })
		kept := [][]string{}
		for _, path := range paths {
			covered := false
			for _, parent := range kept {
				covered = covered || providerContains(parent, path)
			}
			if covered {
				continue
			}
			kept = append(kept, path)
			before, after := providerRawPath(ns.User, path), providerRawPath(afterRoot, path)
			if resets[path[0]] || len(after) == 0 {
				if len(before) > 0 {
					ops = append(ops, fieldOperation("unset", path, nil))
				}
			} else {
				diffVisibleJSON(before, after, path, &ops)
			}
		}
		err = validateJSONSecretPaths(ns, ops)
	}
	if err != nil {
		d.err = err.Error()
		s.mu.Unlock()
		s.fire()
		return api.SettingsNamespaceView{}, err
	}
	if len(patch) == 0 && len(ops) == 0 {
		d.dirty = map[string]bool{}
		d.resets = map[string]bool{}
		d.paths = map[string][]string{}
		s.mu.Unlock()
		return api.SettingsNamespaceView{Ns: ns.NS, Schema: ns.Schema, Value: ns.Value, Base: ns.Base, User: ns.User, Revision: ns.Revision}, nil
	}
	d.busy = true
	d.err = ""
	s.mu.Unlock()
	s.fire()
	method := "settings/update"
	args := map[string]any{"ns": ns.NS, "patch": patch, "expectedRevision": ns.Revision}
	if len(ops) > 0 {
		method = "settings/mutate"
		delete(args, "patch")
		args["ops"] = ops
	}
	var out api.SettingsNamespaceView
	err = b.client.Call(ctx, method, args, &out)
	s.mu.Lock()
	if s.client != b.client || s.generation != b.generation {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsNotReady
	}
	d.busy = false
	if err == nil && (out.Ns != ns.NS || !namespaceWriteLanded(out, patch, ops)) {
		err = fmt.Errorf("Host did not accept the staged %s values; drafts retained", ns.NS)
	}
	if err != nil {
		d.err = err.Error()
	} else {
		if current := s.findNamespaceLocked(ns.NS); current != nil {
			current.Revision = out.Revision
			current.Value = append(json.RawMessage(nil), out.Value...)
			current.Base = append(json.RawMessage(nil), out.Base...)
			current.User = append(json.RawMessage(nil), out.User...)
			current.HasSecrets = hasSetSecrets(out.Secrets)
			current.SecretPaths = configuredSecretPaths(out.Secrets)
		}
		d.ns.Revision = out.Revision
		d.ns.Value = append(json.RawMessage(nil), out.Value...)
		d.ns.Base = append(json.RawMessage(nil), out.Base...)
		d.ns.User = append(json.RawMessage(nil), out.User...)
		d.ns.HasSecrets = hasSetSecrets(out.Secrets)
		d.ns.SecretPaths = configuredSecretPaths(out.Secrets)
		for key, value := range submitted {
			if d.drafts[key] == value {
				delete(d.dirty, key)
				delete(d.resets, key)
			}
		}
		d.paths = map[string][]string{}
	}
	s.mu.Unlock()
	s.fire()
	return out, err
}
func providerRawPath(raw json.RawMessage, path []string) json.RawMessage {
	for _, part := range path {
		var object map[string]json.RawMessage
		if json.Unmarshal(raw, &object) != nil {
			return nil
		}
		raw = object[part]
	}
	return raw
}

func providerContains(parent, path []string) bool {
	if len(path) < len(parent) {
		return false
	}
	for i, p := range parent {
		if path[i] != p {
			return false
		}
	}
	return true
}

// ProviderInput separates an inherited placeholder from a user-owned value.
func (b *NamespaceBinding) ProviderInput(path []string, effective any) (any, bool) {
	if b.providerDraft == nil {
		return effective, false
	}
	s := b.store
	s.mu.Lock()
	defer s.mu.Unlock()
	d := b.providerDraft
	for _, changed := range d.paths {
		if providerContains(changed, path) || providerContains(path, changed) {
			return effective, false
		}
	}
	raw := providerRawPath(d.ns.User, path)
	if len(raw) == 0 {
		return nil, true
	}
	var value any
	_ = json.Unmarshal(raw, &value)
	return value, false
}
