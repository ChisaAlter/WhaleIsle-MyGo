package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// DeepSeekToolsNamespace is the Host settings namespace the DeepSeek application
// API plugin describes while it is mounted. It is absent from every Host whose
// composition does not mount the plugin, which is why the page reports that as
// its own state instead of rendering an empty form.
const DeepSeekToolsNamespace = "dsh-deepseek-tools"

// DeepSeekToolsView is what the native page paints: live stored values, the
// drafts staged on the page, and the Host's own writability.
type DeepSeekToolsView struct {
	Connected, Present, Writable, Loading, Saving bool
	BaseURL, APIKeyEnv, Voice                     string
	TTS, ASR, OCR, FilterMarkdown                 bool
	Drafts                                        map[string]string
	DirtyFields                                   map[string]bool
	Err                                           string
}

// DeepSeekKeyView is the credential reference's state. The key itself is never
// read back; only whether one is stored and whether this Host accepts a write.
type DeepSeekKeyView struct {
	Ref                     string
	Loading, Busy, Writable bool
	Configured              bool
	Err                     string
}

func (s *Store) deepSeekToolsNamespace() *Namespace {
	return s.findNamespaceLocked(DeepSeekToolsNamespace)
}

// SnapshotDeepSeekTools reads one namespace and its drafts. Values come from the
// Host's stored value, except fields staged since; a field without a draft keeps
// showing the stored value.
func (s *Store) SnapshotDeepSeekTools() DeepSeekToolsView {
	if s == nil {
		return DeepSeekToolsView{}
	}
	s.mu.Lock()
	defer s.unlockPublish()
	v := DeepSeekToolsView{
		Connected: s.client != nil, Writable: s.writable,
		Loading: s.loading, Saving: s.saving,
		Drafts:      copyDrafts(s.drafts[DeepSeekToolsNamespace]),
		DirtyFields: copyFlags(s.dirty[DeepSeekToolsNamespace]),
	}
	ns := s.deepSeekToolsNamespace()
	if ns == nil {
		return v
	}
	v.Present = true
	var stored struct {
		BaseURL        string `json:"baseURL"`
		APIKeyEnv      string `json:"apiKeyEnv"`
		Voice          string `json:"voice"`
		TTS            bool   `json:"tts"`
		ASR            bool   `json:"asr"`
		OCR            bool   `json:"ocr"`
		FilterMarkdown *bool  `json:"filterMarkdown"`
	}
	if err := json.Unmarshal(ns.Value, &stored); err != nil {
		v.Err = err.Error()
		v.Writable = false
		return v
	}
	v.BaseURL, v.APIKeyEnv, v.Voice = stored.BaseURL, stored.APIKeyEnv, stored.Voice
	v.TTS, v.ASR, v.OCR = stored.TTS, stored.ASR, stored.OCR
	v.FilterMarkdown = stored.FilterMarkdown == nil || *stored.FilterMarkdown
	return v
}

// DeepSeekToolsBinding stages and saves this one namespace without selecting it
// in the generic editor. Nil while the Host has not described it.
func (s *Store) DeepSeekToolsBinding() *NamespaceBinding {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	present := s.client != nil && s.deepSeekToolsNamespace() != nil
	s.unlockPublish()
	if !present {
		return nil
	}
	return s.BindNamespace(DeepSeekToolsNamespace)
}

// SnapshotDeepSeekKey reports the credential reference's stored state.
func (s *Store) SnapshotDeepSeekKey() DeepSeekKeyView {
	s.mu.Lock()
	defer s.unlockPublish()
	return DeepSeekKeyView{
		Ref: s.deepSeekKeyRef, Loading: s.deepSeekKeyLoading, Busy: s.saving,
		Writable: s.deepSeekKeyWritable, Configured: s.deepSeekKeyConfigured, Err: s.deepSeekKeyErr,
	}
}

// RefreshDeepSeekKey asks the Host whether the reference resolves to a stored
// secret. It never reads the secret itself.
func (s *Store) RefreshDeepSeekKey(ctx context.Context, ref string) {
	ref = strings.TrimSpace(ref)
	s.mu.Lock()
	client, generation := s.client, s.generation
	if ref == "" {
		s.deepSeekKeyRef, s.deepSeekKeyErr, s.deepSeekKeyConfigured, s.deepSeekKeyWritable, s.deepSeekKeyLoading = "", "", false, false, false
		s.unlockPublish()
		return
	}
	if client == nil {
		s.deepSeekKeyRef, s.deepSeekKeyErr, s.deepSeekKeyConfigured, s.deepSeekKeyWritable, s.deepSeekKeyLoading = ref, "", false, false, false
		s.unlockPublish()
		return
	}
	if s.deepSeekKeyLoading && s.deepSeekKeyRef == ref {
		s.unlockPublish()
		return
	}
	s.deepSeekKeyLoading, s.deepSeekKeyRef = true, ref
	s.unlockPublish()
	s.fire()
	go func() {
		callCtx, cancel := withDefaultRPCTimeout(ctx)
		defer cancel()
		described, err := DescribeCredentials(callCtx, client, []string{ref})
		s.mu.Lock()
		if generation != s.generation || s.client != client {
			s.unlockPublish()
			return
		}
		s.deepSeekKeyLoading = false
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				s.deepSeekKeyErr = err.Error()
			}
		} else {
			value := described[ref]
			s.deepSeekKeyErr, s.deepSeekKeyConfigured, s.deepSeekKeyWritable = "", value.Configured, value.Writable
		}
		s.unlockPublish()
		s.fire()
	}()
}

// SaveDeepSeekKey writes a new secret under the reference. The Host stores it;
// the page only ever learns that a secret now exists.
func (s *Store) SaveDeepSeekKey(ctx context.Context, ref, value string) {
	ref = strings.TrimSpace(ref)
	if ref == "" || value == "" {
		return
	}
	s.mu.Lock()
	client, generation := s.client, s.generation
	s.unlockPublish()
	if client == nil {
		return
	}
	if err := SetCredential(ctx, client, ref, value); err != nil {
		s.mu.Lock()
		if generation == s.generation {
			s.deepSeekKeyErr = err.Error()
		}
		s.unlockPublish()
		s.fire()
		return
	}
	s.mu.Lock()
	if generation == s.generation {
		s.deepSeekKeyErr, s.deepSeekKeyConfigured = "", true
	}
	s.unlockPublish()
	s.fire()
}
