package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// TranscribeAudioFile reads path, base64-encodes it, and calls speech/transcribe.
// Prefers speech/catalog selection (provider/language) and MaxAudioBytes when available.
func (s *Store) TranscribeAudioFile(ctx context.Context, absPath string) (string, error) {
	absPath = strings.TrimSpace(absPath)
	if absPath == "" {
		return "", fmt.Errorf("empty path")
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return "", fmt.Errorf("harness not ready")
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("empty audio file")
	}
	maxAudio := 25 << 20 // soft cap before hitting harness MaxAudioBytes
	var providerID, language *string
	var catalog api.SpeechCatalog
	if err := client.Call(ctx, "speech/catalog", map[string]any{}, &catalog); err == nil {
		if catalog.MaxAudioBytes > 0 && int(catalog.MaxAudioBytes) < maxAudio {
			maxAudio = int(catalog.MaxAudioBytes)
		}
		if id := strings.TrimSpace(catalog.Selection.ProviderId); id != "" {
			providerID = &id
			// Best-effort prepare so local models are ready before transcribe.
			var prep api.PrepareResult
			_ = client.Call(ctx, "speech/prepare", map[string]any{
				"providerId": id,
			}, &prep)
		}
		if lang := strings.TrimSpace(catalog.Selection.Language); lang != "" {
			language = &lang
		}
	}
	if len(raw) > maxAudio {
		return "", fmt.Errorf("audio too large (%d bytes; max %d)", len(raw), maxAudio)
	}
	req := api.TranscriptionRequest{
		AudioBase64: base64.StdEncoding.EncodeToString(raw),
		ProviderId:  providerID,
		Language:    language,
	}
	var out api.Transcript
	if err := client.Call(ctx, "speech/transcribe", map[string]any{"request": req}, &out); err != nil {
		return "", err
	}
	text := strings.TrimSpace(out.Text)
	if text == "" {
		return "", fmt.Errorf("empty transcript")
	}
	return text, nil
}

// ConfigureSpeech calls speech/configure with an optional provider/language patch.
func (s *Store) ConfigureSpeech(ctx context.Context, providerID, language string) error {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return fmt.Errorf("harness not ready")
	}
	patch := api.SpeechSelectionPatch{}
	if id := strings.TrimSpace(providerID); id != "" {
		patch.ProviderId = &id
	}
	if lang := strings.TrimSpace(language); lang != "" {
		patch.Language = &lang
	}
	var out api.ConfigureResult
	return client.Call(ctx, "speech/configure", map[string]any{"patch": patch}, &out)
}

// CycleSpeechProvider advances speech/catalog selection via speech/configure.
// Cancels preparation on the previous provider when it changes.
func (s *Store) CycleSpeechProvider(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var catalog api.SpeechCatalog
		if err := client.Call(ctx, "speech/catalog", map[string]any{}, &catalog); err != nil {
			s.mu.Lock()
			s.lastAction = "speech catalog: " + err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		if len(catalog.Providers) == 0 {
			s.mu.Lock()
			s.lastAction = "speech: no providers"
			s.mu.Unlock()
			s.fire()
			return
		}
		cur := strings.TrimSpace(catalog.Selection.ProviderId)
		idx := 0
		for i, p := range catalog.Providers {
			if strings.TrimSpace(p.Id) == cur {
				idx = (i + 1) % len(catalog.Providers)
				break
			}
		}
		next := catalog.Providers[idx]
		nextID := strings.TrimSpace(next.Id)
		if nextID == "" {
			return
		}
		lang := ""
		if len(next.Languages) > 0 {
			lang = strings.TrimSpace(next.Languages[0])
		}
		if cur != "" && cur != nextID {
			_ = s.CancelSpeechPreparation(ctx, cur)
		}
		if err := s.ConfigureSpeech(ctx, nextID, lang); err != nil {
			s.mu.Lock()
			s.lastAction = "speech configure: " + err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		sel := nextID
		if lang != "" {
			sel += "/" + lang
		}
		s.mu.Lock()
		s.speechSelection = sel
		s.lastAction = "speech: " + sel
		s.mu.Unlock()
		s.fire()
	}()
}

// CancelSpeechPreparation calls speech/cancelPreparation for the given provider.
func (s *Store) CancelSpeechPreparation(ctx context.Context, providerID string) error {
	providerID = strings.TrimSpace(providerID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || providerID == "" {
		return fmt.Errorf("harness not ready")
	}
	var out api.CancelPreparationResult
	return client.Call(ctx, "speech/cancelPreparation", map[string]any{
		"providerId": providerID,
	}, &out)
}

// WatchSpeechCatalog opens speech/follow and stores selection summary in LastAction.
func (s *Store) WatchSpeechCatalog(ctx context.Context) {
	s.mu.Lock()
	mux := s.mux
	if s.speechFollowCancel != nil {
		s.speechFollowCancel()
		s.speechFollowCancel = nil
	}
	if mux == nil {
		s.mu.Unlock()
		return
	}
	fctx, cancel := context.WithCancel(ctx)
	s.speechFollowCancel = cancel
	s.mu.Unlock()
	go s.runSpeechFollow(fctx, mux)
}

func (s *Store) runSpeechFollow(ctx context.Context, mux *dsh.Mux) {
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "speech/follow", map[string]any{"args": map[string]any{}})
	if err != nil {
		return
	}
	defer stream.Cancel()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			return
		}
		var cat api.SpeechCatalog
		if json.Unmarshal(raw, &cat) != nil {
			continue
		}
		sel := strings.TrimSpace(cat.Selection.ProviderId)
		if lang := strings.TrimSpace(cat.Selection.Language); lang != "" {
			if sel != "" {
				sel += "/" + lang
			} else {
				sel = lang
			}
		}
		if sel == "" {
			continue
		}
		s.mu.Lock()
		s.speechSelection = sel
		s.mu.Unlock()
		s.fire()
	}
}
