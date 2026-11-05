package pluginui

import (
	"context"
	"errors"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type composerWireFixture struct {
	mu                 sync.Mutex
	draft              ComposerDraft
	applied, submitted int
}

func (f *composerWireFixture) Read(s Subject) (ComposerDraft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.draft.Owner.Subject.equal(s) {
		return ComposerDraft{}, errors.New("owner changed")
	}
	return f.draft, nil
}
func (f *composerWireFixture) Apply(e ComposerEdit) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e.Owner != f.draft.Owner {
		return errors.New("draft changed")
	}
	f.applied++
	if e.Text != nil {
		f.draft.Text = *e.Text
	}
	f.draft.Owner.Revision++
	return nil
}
func (f *composerWireFixture) Submit(g ComposerGesture) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if g.Owner != f.draft.Owner {
		return errors.New("draft changed")
	}
	f.submitted++
	return nil
}
func TestComposerWireRequiresCapturedNativeOwnerAndRevision(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "2:4"}
	fixture := &composerWireFixture{draft: ComposerDraft{Owner: ComposerBinding{Subject: subject, Binding: "native-target", Revision: 3, TargetID: "s", EditSeq: -1}, Text: "native"}}
	o := &Owner{selected: subject, composer: fixture}
	defer o.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !o.serve(w, r) {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+bridgePath, &websocket.DialOptions{Subprotocols: []string{bridgeProtocol}, HTTPHeader: http.Header{"Origin": {server.URL}}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	var reply struct {
		Type, Request, Error string
		Draft                ComposerDraft
	}
	if err = wsjson.Read(ctx, conn, &reply); err != nil || reply.Type != "select" {
		t.Fatal(err, reply)
	}
	old := fixture.draft.Owner
	text := "changed through native"
	request := ComposerRequest{Request: "one", Op: "edit", Subject: subject, Edit: ComposerEdit{Owner: old, Text: &text}}
	if err = wsjson.Write(ctx, conn, map[string]any{"type": "composer", "composer": request}); err != nil {
		t.Fatal(err)
	}
	reply = struct {
		Type, Request, Error string
		Draft                ComposerDraft
	}{}
	if err = wsjson.Read(ctx, conn, &reply); err != nil || reply.Error != "" || reply.Draft.Text != text || reply.Draft.Owner.Revision != 4 {
		t.Fatal("native apply did not settle", err, reply)
	}
	request.Request = "stale"
	if err = wsjson.Write(ctx, conn, map[string]any{"type": "composer", "composer": request}); err != nil {
		t.Fatal(err)
	}
	reply = struct {
		Type, Request, Error string
		Draft                ComposerDraft
	}{}
	if err = wsjson.Read(ctx, conn, &reply); err != nil || reply.Error == "" {
		t.Fatal("stale draft accepted", err, reply)
	}
	fixture.mu.Lock()
	applied := fixture.applied
	fixture.mu.Unlock()
	if applied != 1 {
		t.Fatal("stale native write ran", applied)
	}
}
