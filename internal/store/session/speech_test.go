package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestTranscribeAudioFile(t *testing.T) {
	dir := t.TempDir()
	audio := filepath.Join(dir, "a.wav")
	if err := os.WriteFile(audio, []byte("RIFF...."), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		switch r.URL.Path {
		case "/api/speech/catalog":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": env.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"maxAudioBytes":      float64(1 << 20),
						"maxDurationSeconds": 60,
						"providers":          []any{},
						"selection":          map[string]any{"providerId": "local", "language": "zh"},
					},
				},
			})
		case "/api/speech/prepare":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": env.RPCID,
				"result": map[string]any{
					"ok":    true,
					"value": map[string]any{},
				},
			})
		case "/api/speech/transcribe":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": env.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"text":             "hello world",
						"audioSeconds":     1.0,
						"inferenceSeconds": 0.2,
					},
				},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	text, err := st.TranscribeAudioFile(context.Background(), audio)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello world" {
		t.Fatalf("%q", text)
	}
}

func TestCycleSpeechProvider(t *testing.T) {
	var configured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args map[string]any `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		switch r.URL.Path {
		case "/api/speech/catalog":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": env.RPCID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"maxAudioBytes": float64(1 << 20),
						"providers": []any{
							map[string]any{"id": "a", "name": "A", "languages": []any{"en"}},
							map[string]any{"id": "b", "name": "B", "languages": []any{"zh"}},
						},
						"selection": map[string]any{"providerId": "a", "language": "en"},
					},
				},
			})
		case "/api/speech/cancelPreparation":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": env.RPCID,
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		case "/api/speech/configure":
			if patch, ok := env.Payload.Args["patch"].(map[string]any); ok {
				if id, _ := patch["providerId"].(string); id != "" {
					configured = id
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": env.RPCID,
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		default:
			t.Fatalf("path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.CycleSpeechProvider(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snap := st.Snapshot(); snap.SpeechSelection == "b/zh" && configured == "b" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("configured=%q selection=%q", configured, st.Snapshot().SpeechSelection)
}
