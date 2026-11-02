package previewprocess

import (
	"encoding/json"
	"testing"
	"time"
	"whaleisle/internal/desktop/openui"
)

func TestWhaleUIDownloadClaimAndFullRenderUpdate(t *testing.T) {
	viewer, err := openui.LoadWhaleViewer()
	if err != nil {
		t.Fatal(err)
	}
	doc := openUITestDocument()
	doc.HTML = viewer
	doc.OpenUI.HostActions = true
	doc.OpenUI.Revision = 1
	doc.OpenUI.Language = "whale-ui"
	doc.OpenUI.Locale = "en"
	s := &Session{done: make(chan struct{}), closeFn: func() error { return nil }}
	s.initOpenUIEvents(doc)
	defer s.Close()
	s.enableOpenUIAPI(true, openui.MaxAPIResponseBytes)
	s.enableOpenUIUpdates(true)
	e := openui.Event{Channel: "whale-openui", Version: 1, InstanceID: doc.OpenUI.InstanceID, Type: "download", Filename: "report.html", MIME: "text/html", RequestID: "native-action-1", Sequence: 1, UserGesture: true, Parts: 1}
	raw, _ := json.Marshal(e)
	if err := s.acceptOpenUIEvent(raw); err != nil {
		t.Fatal(err)
	}
	<-s.Events()
	forged := e
	forged.Filename = "other.html"
	if s.ClaimOpenUINativeAction(forged) == nil {
		t.Fatal("changed export filename claimed")
	}
	e.Content = "assembled content"
	if err := s.ClaimOpenUINativeAction(e); err != nil {
		t.Fatal("assembled native export rejected", err)
	}
	if s.ClaimOpenUINativeAction(e) == nil {
		t.Fatal("replayed export claimed")
	}
	r := *doc.OpenUI
	r.Revision = 2
	r.Locale = "zh"
	r.Variables = map[string]string{"--accent": "#123abc"}
	r.Panel = true
	if err := s.UpdateOpenUI(r); err != nil {
		t.Fatal(err)
	}
	wire := s.takeOpenUIUpdate(time.Now())
	u, err := validateRevisionedOpenUIUpdate(wire)
	if err != nil || u.Locale != "zh" || u.Language != "whale-ui" || u.Variables["--accent"] != "#123abc" || !u.Panel {
		t.Fatal("lost full v2 update", err, string(wire))
	}
	r.Revision = 1
	if s.UpdateOpenUI(r) == nil {
		t.Fatal("old revision admitted")
	}
	r.Revision = 3
	r.Variables = map[string]string{"bad": "x"}
	if s.UpdateOpenUI(r) == nil {
		t.Fatal("invalid theme token admitted")
	}
	r.Variables = nil
	r.Locale = "invalid"
	if s.UpdateOpenUI(r) == nil {
		t.Fatal("invalid locale admitted")
	}
}
