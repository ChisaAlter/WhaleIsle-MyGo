//go:build functionalacceptance

package pluginui

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"
)

func TestIncompatibleSessionCannotRetainControls(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "same", Epoch: "3:4"}
	o := &Owner{selected: subject, state: Snapshot{Connected: true, Page: 1, Revision: 1, Subject: &subject, Surface: "conversation.session.header.actions", Outlets: []SessionOutlet{{Handle: "entry", Slot: "conversation.session.header.actions", Available: true}}, SessionControls: []SessionControl{{Handle: "control", Outlet: "entry", Slot: "conversation.session.header.actions", Label: "Original"}}}}
	source := []byte(`window.__ModuleLoader__.load({id: "@deepseek-ai/dsh-client-ui-settings-models", factory(){return futureUnverifiedWelcome}})`)
	r := &http.Response{StatusCode: 200, Request: &http.Request{URL: &url.URL{Path: "/plugins/"}}, Header: http.Header{"Content-Type": []string{"text/javascript"}}, Body: io.NopCloser(bytes.NewReader(source))}
	if err := o.sessionResponse(r); err != nil {
		t.Fatal(err)
	}
	s := o.Snapshot()
	if s.Unavailable != "session-version-incompatible" || s.Subject != nil || len(s.SessionControls) != 0 || s.Surface != "" {
		t.Fatal("unrecognized Models policy retained a visible owner")
	}
	if o.ShowSessionPage(1, 1, subject) == nil || o.ClickSessionControl(1, 1, subject, "control") == nil || o.ShowSessionOutlet(1, 1, subject, "entry") == nil {
		t.Fatal("incompatible owner retained authority")
	}
}
