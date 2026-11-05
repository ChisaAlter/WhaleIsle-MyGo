package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"whaleisle/internal/dsh/api"
)

// SaveProviderSettings is the settings half of the model editor's single
// settings -> credential submission. It returns the acknowledged namespace,
// including its new revision, before the credential owner continues.
func (b *NamespaceBinding) SaveProviderSettings(ctx context.Context) (api.SettingsNamespaceView, error) {
	if b.providerDraft != nil {
		return b.saveProviderDraft(ctx)
	}
	s := b.store
	s.mu.Lock()
	if !b.currentLocked() || s.client == nil {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsNotReady
	}
	if s.saving || s.loading || s.themeWriteInFlightLocked(b.namespace) {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsBusy
	}
	if !s.writable {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsReadOnly
	}
	ns := *s.findNamespaceLocked(b.namespace)
	currentRevision := ns.Revision
	ns.Revision = s.draftRevisionLocked(ns)
	drafts, resets := copyDrafts(s.drafts[ns.NS]), copyFlags(s.resets[ns.NS])
	patch, ops, submitted, err := draftWrite(ns, drafts, s.dirty[ns.NS], resets)
	if err == nil {
		err = validateServedDraft(ns.NS, drafts, s.dirty[ns.NS], resets)
	}
	if err != nil {
		s.err, s.lastAction = err, err.Error()
		s.mu.Unlock()
		s.fire()
		return api.SettingsNamespaceView{}, err
	}
	if len(patch) == 0 && len(ops) == 0 {
		ns.Revision = currentRevision
		s.acknowledgeDraftsLocked(ns, submitted, resets)
		s.err, s.lastAction = nil, "settings saved"
		out := api.SettingsNamespaceView{Ns: ns.NS, Schema: ns.Schema, Value: ns.Value, Base: ns.Base, User: ns.User, Revision: ns.Revision}
		s.mu.Unlock()
		return out, nil
	}
	s.saving, s.err = true, nil
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
	if !b.currentLocked() {
		s.mu.Unlock()
		return api.SettingsNamespaceView{}, ErrSettingsNotReady
	}
	s.saving = false
	if err == nil && (out.Ns != ns.NS || !namespaceWriteLanded(out, patch, ops)) {
		err = fmt.Errorf("Host did not accept the staged %s values; drafts retained", ns.NS)
	}
	if err != nil {
		s.err, s.lastAction = err, err.Error()
		s.mu.Unlock()
		s.fire()
		return api.SettingsNamespaceView{}, err
	}
	current := s.findNamespaceLocked(ns.NS)
	current.Revision = out.Revision
	current.Value = append(json.RawMessage(nil), out.Value...)
	current.Base = append(json.RawMessage(nil), out.Base...)
	current.User = append(json.RawMessage(nil), out.User...)
	current.HasSecrets = hasSetSecrets(out.Secrets)
	current.SecretPaths = configuredSecretPaths(out.Secrets)
	s.acknowledgeDraftsLocked(*current, submitted, resets)
	s.err, s.lastAction = nil, "settings saved"
	s.mu.Unlock()
	s.fire()
	return out, nil
}

// RefreshProviderSettings reconciles a credential-reference write made after
// the settings half committed, without changing the settings rail selection.
func (b *NamespaceBinding) RefreshProviderSettings(ctx context.Context) {
	s := b.store
	s.mu.Lock()
	current := b.currentLocked()
	s.mu.Unlock()
	if current {
		s.RefreshDescribe(ctx)
	}
}
