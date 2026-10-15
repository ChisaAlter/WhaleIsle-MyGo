package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
	"whaleisle/internal/petdirect"
)

// Construction performs no reads or network operations. Only the native card's
// explicit Send calls Load. Reuse the desktop's existing credentials.json
// resolver, including safeStorage; never copy a credential into page state.
func newPetDirectChat(cfg *config.Store, userData, home string) *petdirect.Client {
	return &petdirect.Client{Load: func() (petdirect.Config, error) {
		creds, err := credentials.Load(userData)
		if err != nil {
			return petdirect.Config{}, petdirect.ErrCredentials
		}
		persona := petdirect.Persona{}
		// This is the original assistant settings file, not a session/history scan.
		// As upstream, a missing/unreadable persona uses the ordinary default.
		if f, err := os.Open(filepath.Join(home, "data", "whale", "settings.json")); err == nil {
			raw, readErr := io.ReadAll(io.LimitReader(f, (64<<10)+1))
			f.Close()
			if readErr == nil && len(raw) <= 64<<10 {
				var next petdirect.Persona
				if json.Unmarshal(raw, &next) == nil {
					persona = next
				}
			}
		}
		personality, _ := cfg.SavedPetPersonality()
		return petdirect.Config{APIKey: creds.APIKey, BaseURL: creds.BaseURL, Prompt: petdirect.Prompt(personality, persona)}, nil
	}}
}
