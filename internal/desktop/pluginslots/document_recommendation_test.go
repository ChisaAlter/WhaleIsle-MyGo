package pluginslots

import "testing"

func TestB25RecommendationPinnedToResource(t *testing.T) {
	gateway := &Gateway{}
	message := &DocumentRecommendation{Session: "session", Path: "a.md", Extension: true}
	if gateway.modules.acceptsDocument(message) {
		t.Fatal("ordinary Host page accepted document recommendation")
	}
	gateway.ConfigureDocumentViewer("session", "a.md")
	if !gateway.modules.acceptsDocument(message) {
		t.Fatal("same document rejected")
	}
	for _, wrong := range []*DocumentRecommendation{nil, {Session: "other", Path: "a.md", Extension: true}, {Session: "session", Path: "b.md", Extension: true}} {
		if gateway.modules.acceptsDocument(wrong) {
			t.Fatal("recommendation crossed resource owner")
		}
	}
	gateway.modules.state.Document = message
	copy := gateway.ClientModules()
	copy.Document.Extension = false
	if !gateway.ClientModules().Document.Extension {
		t.Fatal("snapshot aliased mutable recommendation")
	}
	gateway.DisconnectClientModules()
	if gateway.ClientModules().Document != nil {
		t.Fatal("navigation retained obsolete recommendation")
	}
}
