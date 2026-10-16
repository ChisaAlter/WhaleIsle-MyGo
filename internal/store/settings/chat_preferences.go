package settings

import (
	"context"
	"encoding/json"
)

// ChatPreferences are accepted Host values, independent of settings form drafts.
type ChatPreferences struct {
	LinkOpening      string `json:"linkOpening"`
	TranscriptView   string `json:"transcriptView"`
	PerformanceUsage string `json:"performanceUsage"`
}

type ConversationPreferences struct {
	ViewTabs             bool            `json:"viewTabs"`
	ComposerResize       bool            `json:"composerResize"`
	ComposerResizeHeight float32         `json:"composerResizeHeight"`
	ComposerResizeWidth  float32         `json:"composerResizeWidth"`
	NextPrompt           bool            `json:"nextPrompt"`
	ComposerBeam         bool            `json:"composerBeam"`
	ComposerBeamStyle    json.RawMessage `json:"composerBeamStyle"`
	TypingFx             bool            `json:"typingFx"`
	TypingFxStyle        json.RawMessage `json:"typingFxStyle"`
}

func ChatPreferencesFrom(v ViewState) ChatPreferences {
	p := ChatPreferences{LinkOpening: "sidebar", TranscriptView: "standard", PerformanceUsage: "detailed"}
	for _, ns := range v.Namespaces {
		if ns.NS == "ui-chat" {
			_ = json.Unmarshal(ns.Value, &p)
			break
		}
	}
	switch p.TranscriptView {
	case "normal", "expanded":
		p.TranscriptView = "detailed"
	case "compact", "standard", "detailed", "verbose":
	default:
		p.TranscriptView = "standard"
	}
	return p
}

func ConversationPreferencesFrom(v ViewState) ConversationPreferences {
	p := ConversationPreferences{ViewTabs: true, NextPrompt: true, ComposerBeam: true}
	for _, ns := range v.Namespaces {
		if ns.NS == "ui-conversation" {
			_ = json.Unmarshal(ns.Value, &p)
			break
		}
	}
	return p
}

// SaveComposerSize uses the existing Host mutation path and writes only the axis
// released by the user. It leaves unsubmitted settings drafts untouched.
func (s *Store) SaveComposerSize(ctx context.Context, axis string, value float32) {
	field := "composerResizeHeight"
	if axis == "width" {
		field = "composerResizeWidth"
	}
	raw, _ := json.Marshal(value)
	s.saveMutate(ctx, "ui-conversation", []json.RawMessage{fieldOperation("set", []string{field}, raw)})
}
