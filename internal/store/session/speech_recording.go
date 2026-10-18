package session

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"
	"whaleisle/internal/desktop/voicecapture"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// SpeechRecordingPlan freezes the existing selected provider/language and Host
// before touching a microphone. It never installs/prepares a model implicitly.
type SpeechRecordingPlan struct {
	Limits                          voicecapture.Limits
	ProviderID, Language, SessionID string
	owner                           *Store
	client                          *dsh.Client
	rpc                             *dsh.Client
}

func (s *Store) PrepareSpeechRecording(ctx context.Context) (SpeechRecordingPlan, error) {
	s.mu.Lock()
	client, id := s.client, s.selectedID
	s.mu.Unlock()
	if client == nil {
		return SpeechRecordingPlan{}, errors.New("语音 Host 未连接 / Speech Host offline")
	}
	// Audio and this Host's cookie may not be redirected to another endpoint.
	rpc := *client
	hc := http.Client{}
	if client.HTTP != nil {
		hc = *client.HTTP
	}
	hc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	rpc.HTTP = &hc
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var catalog api.SpeechCatalog
	if err := rpc.Call(ctx, "speech/catalog", map[string]any{}, &catalog); err != nil {
		return SpeechRecordingPlan{}, errors.New("语音服务不可用，请检查语音插件 / Speech unavailable")
	}
	provider := strings.TrimSpace(catalog.Selection.ProviderId)
	ready := false
	for _, p := range catalog.Providers {
		if p.Id == provider {
			var prep struct {
				Phase string `json:"phase"`
			}
			_ = json.Unmarshal(p.Preparation, &prep)
			ready = prep.Phase == "ready" || prep.Phase == "standby" || prep.Phase == "waking"
			break
		}
	}
	if provider == "" || !ready {
		return SpeechRecordingPlan{}, errors.New("请先在语音输入插件设置准备所选提供方 / Prepare the selected speech provider")
	}
	bytes, seconds := catalog.MaxAudioBytes, catalog.MaxDurationSeconds
	if math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes < 46 || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return SpeechRecordingPlan{}, errors.New("语音服务限额无效 / Invalid speech limits")
	}
	maxBytes := int(math.Min(math.Floor(bytes), 8<<20))
	duration := time.Duration(math.Min(seconds, 120) * float64(time.Second))
	// PCM capacity, timer and Host bounds must agree even for a tiny byte limit.
	duration = min(duration, time.Duration((maxBytes-44)/2)*time.Second/16000)
	if duration <= 0 {
		return SpeechRecordingPlan{}, errors.New("语音服务限额过小 / Speech limit too small")
	}
	plan := SpeechRecordingPlan{Limits: voicecapture.Limits{MaxBytes: maxBytes, MaxDuration: duration}, ProviderID: provider, Language: catalog.Selection.Language, SessionID: id, owner: s, client: client, rpc: &rpc}
	if !plan.Current() {
		return SpeechRecordingPlan{}, errors.New("语音会话已变化 / Speech session changed")
	}
	return plan, nil
}
func (p SpeechRecordingPlan) Current() bool {
	if p.owner == nil {
		return false
	}
	p.owner.mu.Lock()
	defer p.owner.mu.Unlock()
	return p.owner.client == p.client && p.owner.selectedID == p.SessionID
}
func (p SpeechRecordingPlan) Transcribe(ctx context.Context, raw []byte) (string, error) {
	if !p.Current() {
		return "", errors.New("speech session changed")
	}
	if err := validateRecording(raw, p.Limits); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	request := api.TranscriptionRequest{AudioBase64: base64.StdEncoding.EncodeToString(raw), ProviderId: &p.ProviderID, Language: &p.Language}
	var result api.Transcript
	if err := p.rpc.Call(ctx, "speech/transcribe", map[string]any{"request": request}, &result); err != nil {
		return "", errors.New("转写失败，原草稿未改动 / Transcription failed")
	}
	if !p.Current() || ctx.Err() != nil {
		return "", errors.New("speech session changed")
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		return "", errors.New("未识别到文字 / No speech recognized")
	}
	if len(text) > 1<<20 {
		return "", errors.New("转写文字过长 / Transcript too large")
	}
	return text, nil
}
func validateRecording(b []byte, limit voicecapture.Limits) error {
	bad := errors.New("无有效录音或超出限额 / Invalid recording")
	if len(b) < 46 || len(b) > limit.MaxBytes || string(b[:4]) != "RIFF" || string(b[8:16]) != "WAVEfmt " || binary.LittleEndian.Uint32(b[16:20]) != 16 || binary.LittleEndian.Uint16(b[20:22]) != 1 || binary.LittleEndian.Uint16(b[22:24]) != 1 || binary.LittleEndian.Uint32(b[24:28]) != 16000 || binary.LittleEndian.Uint32(b[28:32]) != 32000 || binary.LittleEndian.Uint16(b[32:34]) != 2 || binary.LittleEndian.Uint16(b[34:36]) != 16 || string(b[36:40]) != "data" || int(binary.LittleEndian.Uint32(b[4:8])) != len(b)-8 || int(binary.LittleEndian.Uint32(b[40:44])) != len(b)-44 || (len(b)-44)%2 != 0 {
		return bad
	}
	if time.Duration((len(b)-44)/2)*time.Second/16000 > limit.MaxDuration {
		return bad
	}
	return nil
}
