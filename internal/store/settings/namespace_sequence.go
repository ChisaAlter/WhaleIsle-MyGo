package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"whaleisle/internal/dsh/api"
)

// SaveSequence settles provider forms before the web provider selection. A
// refusal retains that form's draft and leaves unattempted forms untouched.
func (b *NamespaceBinding) SaveSequence(ctx context.Context, following ...*NamespaceBinding) {
	b.saveSequence(ctx, nil, following...)
}

func (b *NamespaceBinding) saveSequence(ctx context.Context, credential *WebSearchCredential, following ...*NamespaceBinding) {
	s := b.store
	bindings := append([]*NamespaceBinding{b}, following...)
	type write struct {
		ns        Namespace
		patch     map[string]json.RawMessage
		ops       []json.RawMessage
		submitted map[string]string
		resets    map[string]bool
	}
	s.mu.Lock()
	if !b.currentLocked() || s.saving || s.loading || !s.writable || s.client == nil {
		s.mu.Unlock()
		return
	}
	writes := []write{}
	ref, literal := "", ""
	if credential != nil {
		if credential.store != s || !credential.currentLocked() {
			s.mu.Unlock()
			return
		}
		ref, literal = s.webSearchReferenceLocked(), credential.state.Draft
		if literal != "" && (!credential.state.Known || !credential.state.Writable || credential.state.Loading) {
			s.mu.Unlock()
			return
		}
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		if binding == nil || binding.store != s || !binding.currentLocked() || binding.client != b.client || binding.generation != b.generation {
			s.mu.Unlock()
			return
		}
		if seen[binding.namespace] {
			continue
		}
		seen[binding.namespace] = true
		ns := *s.findNamespaceLocked(binding.namespace)
		if ns.NS == "subagent-model-selection-settings" && s.draftRevisionLocked(ns) != ns.Revision {
			s.err = fmt.Errorf("Subagent model selection changed on the Host; discard this draft before editing again")
			s.lastAction = s.err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		ns.Revision = s.draftRevisionLocked(ns)
		drafts, resets := copyDrafts(s.drafts[ns.NS]), copyFlags(s.resets[ns.NS])
		patch, ops, submitted, err := draftWrite(ns, drafts, s.dirty[ns.NS], resets)
		if err == nil && !resets[CustomInstructionsField] {
			err = validateCustomInstructions(ns.NS, drafts)
		}
		if err == nil {
			err = validateServedDraft(ns.NS, drafts, s.dirty[ns.NS], resets)
		}
		if err != nil {
			s.err, s.lastAction = err, err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		if len(patch) == 0 && len(ops) == 0 {
			s.acknowledgeDraftsLocked(ns, submitted, resets)
			continue
		}
		writes = append(writes, write{ns, patch, ops, submitted, resets})
	}
	if len(writes) == 0 && literal == "" {
		s.mu.Unlock()
		s.fire()
		return
	}
	s.saving, s.err = true, nil
	s.mu.Unlock()
	s.fire()
	go func() {
		keySaved := literal == ""
		saveKey := func() error {
			if keySaved {
				return nil
			}
			if err := credential.saveLiteral(ctx, ref, literal); err != nil {
				return err
			}
			keySaved = true
			return nil
		}
		fail := func(err error) {
			s.mu.Lock()
			if s.client == b.client && s.generation == b.generation {
				s.saving, s.err, s.lastAction = false, err, err.Error()
			}
			s.mu.Unlock()
			s.fire()
		}
		for _, w := range writes {
			s.mu.Lock()
			if s.client != b.client || s.generation != b.generation {
				s.mu.Unlock()
				return
			}
			current := s.findNamespaceLocked(w.ns.NS)
			if current == nil {
				s.saving = false
				s.err = fmt.Errorf("Settings section %s is no longer available; drafts retained", w.ns.NS)
				s.lastAction = s.err.Error()
				s.mu.Unlock()
				s.fire()
				return
			}
			s.mu.Unlock()
			if len(w.patch) == 0 && len(w.ops) == 0 {
				continue
			}
			method := "settings/update"
			args := map[string]any{"ns": w.ns.NS, "patch": w.patch, "expectedRevision": w.ns.Revision}
			enables := visualReplyEnabled(w.patch)
			if len(w.ops) > 0 {
				method = "settings/mutate"
				delete(args, "patch")
				args["ops"] = w.ops
				enables = visualReplyMutationEnables(w.ops)
			}
			err := s.verifyVisualReplyEnable(ctx, w.ns.NS, enables, b.client.Endpoint.Origin, b.generation)
			var out api.SettingsNamespaceView
			if err == nil {
				err = b.client.Call(ctx, method, args, &out)
			}
			s.mu.Lock()
			if s.client != b.client || s.generation != b.generation {
				s.mu.Unlock()
				return
			}
			current = s.findNamespaceLocked(w.ns.NS)
			if err == nil && current == nil {
				err = fmt.Errorf("Settings section %s is no longer available; drafts retained", w.ns.NS)
			}
			if err != nil {
				s.saving, s.err, s.lastAction = false, err, err.Error()
				s.mu.Unlock()
				s.fire()
				return
			}
			current.Revision = out.Revision
			current.Value = append(json.RawMessage(nil), out.Value...)
			current.Base = append(json.RawMessage(nil), out.Base...)
			current.User = append(json.RawMessage(nil), out.User...)
			current.HasSecrets = hasSetSecrets(out.Secrets)
			current.SecretPaths = configuredSecretPaths(out.Secrets)
			if !namespaceWriteLanded(out, w.patch, w.ops) {
				s.saving = false
				s.err = fmt.Errorf("Host did not accept the staged %s values; drafts retained", w.ns.NS)
				s.lastAction = s.err.Error()
				s.mu.Unlock()
				s.fire()
				return
			}
			s.acknowledgeDraftsLocked(*current, w.submitted, w.resets)
			s.mu.Unlock()
		}
		if err := saveKey(); err != nil {
			fail(err)
			return
		}
		s.mu.Lock()
		if s.client != b.client || s.generation != b.generation {
			s.mu.Unlock()
			return
		}
		s.saving, s.err, s.lastAction = false, nil, "settings saved"
		s.mu.Unlock()
		s.fire()
	}()
}
