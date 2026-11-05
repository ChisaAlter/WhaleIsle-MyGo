package settings

import (
	"context"
	"encoding/json"
	"fmt"

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
		delete(s.preferenceErrors, ns+"."+key)
	}
	q := s.preferenceWrites[ns]
	start := q == nil || q.client != client || q.generation != generation
	if start {
		q = &preferenceWriteQueue{client: client, generation: generation, revision: s.findNamespaceLocked(ns).Revision}
		s.preferenceWrites[ns] = q
	}
	q.pending = append(q.pending, write)
	s.snapCache.valid = false
	if start {
		go s.flushPreferenceQueue(ns, q)
	}
}

func (s *Store) flushPreferenceQueue(name string, q *preferenceWriteQueue) {
	for {
		s.mu.Lock()
		if s.client != q.client || s.generation != q.generation || s.preferenceWrites[name] != q {
			s.mu.Unlock()
			return
		}
		write := q.pending[0]
		ns := s.findNamespaceLocked(name)
		conflict := ns == nil || ns.Revision != q.revision
		rev := q.revision
		s.mu.Unlock()
		var out api.SettingsNamespaceView
		var err error
		if conflict {
			err = fmt.Errorf("settings changed while this preference was queued; retry the change")
		} else {
			err = s.verifyVisualReplyEnable(write.ctx, name, visualReplyMutationEnables(write.ops), q.client.Endpoint.Origin, q.generation)
			if err == nil {
				err = q.client.Call(write.ctx, "settings/mutate", map[string]any{"ns": name, "ops": write.ops, "expectedRevision": rev}, &out)
			}
			if err == nil && !namespaceWriteLanded(out, nil, write.ops) {
				err = fmt.Errorf("preference was not accepted by Host")
			}
		}
		s.mu.Lock()
		if s.client != q.client || s.generation != q.generation || s.preferenceWrites[name] != q {
			s.mu.Unlock()
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
			s.err = err
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
		s.snapCache.valid = false
		done := err != nil || len(q.pending) == 0
		s.mu.Unlock()
		s.fire()
		if err != nil {
			s.RefreshDescribe(context.Background())
		}
		if done {
			return
		}
	}
}
