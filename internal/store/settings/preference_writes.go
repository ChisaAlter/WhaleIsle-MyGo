package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type preferenceMutation struct {
	ctx  context.Context
	keys []string
	ops  []json.RawMessage
}

// A namespace is one optimistic revision stream. Other namespaces and provider
// cards remain usable; an error never replays an uncertain mutation.
type preferenceWriteQueue struct {
	client     *dsh.Client
	generation uint64
	revision   float64
	pending    []preferenceMutation
}

func (s *Store) preferencePendingLocked() map[string]bool {
	pending := map[string]bool{}
	for ns, q := range s.preferenceWrites {
		if q.client != s.client || q.generation != s.generation {
			continue
		}
		for _, write := range q.pending {
			for _, key := range write.keys {
				pending[ns+"."+key] = true
			}
		}
	}
	return pending
}

func (s *Store) enqueuePreferenceLocked(ctx context.Context, ns string, values map[string]string, ops []json.RawMessage, client *dsh.Client, generation uint64) {
	if len(ops) == 0 {
		return
	}
	if s.preferenceWrites == nil {
		s.preferenceWrites = map[string]*preferenceWriteQueue{}
	}
	if s.preferenceErrors == nil {
		s.preferenceErrors = map[string]string{}
	}
	write := preferenceMutation{ctx: ctx, ops: ops}
	for key := range values {
		write.keys = append(write.keys, key)
		if prevErr, ok := s.preferenceErrors[ns+"."+key]; ok {
			if s.err != nil && s.err.Error() == prevErr {
				s.err = nil
			}
			delete(s.preferenceErrors, ns+"."+key)
		}
	}
	q := s.preferenceWrites[ns]
	start := q == nil || q.client != client || q.generation != generation
	if start {
		q = &preferenceWriteQueue{client: client, generation: generation, revision: s.findNamespaceLocked(ns).Revision}
		s.preferenceWrites[ns] = q
	}
	q.pending = append(q.pending, write)
	s.viewStale = true
	if start {
		go s.flushPreferenceQueue(ns, q)
	}
}

func isSettingsConflictError(err error) bool {
	if err == nil {
		return false
	}
	var rpcErr *dsh.Error
	if errors.As(err, &rpcErr) {
		code := strings.ToLower(strings.TrimSpace(rpcErr.Code))
		msg := strings.ToLower(strings.TrimSpace(rpcErr.Message))
		return strings.Contains(code, "conflict") ||
			strings.Contains(msg, "settings_conflict") ||
			strings.Contains(msg, "settings/conflict") ||
			strings.Contains(msg, "settingsconflicterror") ||
			(strings.Contains(msg, "expected revision") && strings.Contains(msg, "conflict"))
	}
	return false
}

func (s *Store) refreshNamespaceRevisionForRetry(ctx context.Context, client *dsh.Client, generation uint64, name string) (float64, bool) {
	var desc api.SettingsDescribeValue
	if err := client.Call(ctx, "settings/describe", nil, &desc); err != nil {
		return 0, false
	}
	for _, item := range desc.Namespaces {
		if item.Ns != name {
			continue
		}
		fields, fErr := BuildFields(item.Schema)
		if fErr != nil {
			return 0, false
		}
		s.mu.Lock()
		if s.client != client || s.generation != generation {
			s.unlockPublish()
			return 0, false
		}
		if ns := s.findNamespaceLocked(name); ns != nil && ns.Revision <= item.Revision {
			ns.Revision = item.Revision
			ns.Fields = fields
			ns.Value = append(json.RawMessage(nil), item.Value...)
			ns.Schema = append(json.RawMessage(nil), item.Schema...)
			ns.Base = append(json.RawMessage(nil), item.Base...)
			ns.User = append(json.RawMessage(nil), item.User...)
			ns.HasSecrets = hasSetSecrets(item.Secrets)
			ns.SecretPaths = configuredSecretPaths(item.Secrets)
			s.seedDraftsLocked(*ns)
			s.viewStale = true
		}
		s.unlockPublish()
		return item.Revision, true
	}
	return 0, false
}

func (s *Store) flushPreferenceQueue(name string, q *preferenceWriteQueue) {
	for {
		s.mu.Lock()
		if s.client != q.client || s.generation != q.generation || s.preferenceWrites[name] != q {
			s.unlockPublish()
			return
		}
		write := q.pending[0]
		ns := s.findNamespaceLocked(name)
		conflict := ns == nil || ns.Revision < q.revision
		if !conflict && ns.Revision > q.revision {
			q.revision = ns.Revision
		}
		rev := q.revision
		s.unlockPublish()
		var out api.SettingsNamespaceView
		var err error
		if conflict {
			err = fmt.Errorf("settings changed while this preference was queued; retry the change")
		} else {
			err = s.verifyVisualReplyEnable(write.ctx, name, visualReplyMutationEnables(write.ops), q.client.Endpoint.Origin, q.generation)
			if err == nil {
				err = q.client.Call(write.ctx, "settings/mutate", map[string]any{"ns": name, "ops": write.ops, "expectedRevision": rev}, &out)
				if err != nil && isSettingsConflictError(err) {
					// Host rejected expectedRevision before applying any mutation;
					// refresh the namespace revision once and retry cleanly.
					if freshRev, ok := s.refreshNamespaceRevisionForRetry(write.ctx, q.client, q.generation, name); ok && freshRev != rev {
						out = api.SettingsNamespaceView{}
						err = q.client.Call(write.ctx, "settings/mutate", map[string]any{"ns": name, "ops": write.ops, "expectedRevision": freshRev}, &out)
					}
				}
			}
			if err == nil && !namespaceWriteLanded(out, nil, write.ops) {
				err = fmt.Errorf("preference was not accepted by Host")
			}
		}
		s.mu.Lock()
		if s.client != q.client || s.generation != q.generation || s.preferenceWrites[name] != q {
			s.unlockPublish()
			return
		}
		if name == "session-log-deepseek" {
			s.uploadNoticeSequence++
			s.uploadNotice = "saved"
			if err != nil {
				s.uploadNotice = "failed"
			}
		}
		if err != nil {
			// Field-scoped immediate preference failures stay in PreferenceErrors
			// instead of setting global s.err, which would paint an unrelated red
			// banner across every other settings page.
			s.lastAction = err.Error()
			for _, pending := range q.pending {
				for _, key := range pending.keys {
					s.preferenceErrors[name+"."+key] = err.Error()
				}
			}
			delete(s.preferenceWrites, name)
		} else {
			if ns = s.findNamespaceLocked(name); ns != nil && ns.Revision <= out.Revision {
				ns.Revision = out.Revision
				ns.Value = append(json.RawMessage(nil), out.Value...)
				ns.Base = append(json.RawMessage(nil), out.Base...)
				ns.User = append(json.RawMessage(nil), out.User...)
				ns.HasSecrets = hasSetSecrets(out.Secrets)
				ns.SecretPaths = configuredSecretPaths(out.Secrets)
				s.acknowledgeDraftsLocked(*ns, map[string]string{})
			}
			q.revision = out.Revision
			q.pending = q.pending[1:]
			s.err = nil
			s.lastAction = "settings mutated"
			if len(q.pending) == 0 {
				delete(s.preferenceWrites, name)
			}
		}
		s.viewStale = true
		done := err != nil || len(q.pending) == 0
		s.unlockPublish()
		s.fire()
		if err != nil {
			s.refreshDescribe(context.Background(), true)
		}
		if done {
			return
		}
	}
}
