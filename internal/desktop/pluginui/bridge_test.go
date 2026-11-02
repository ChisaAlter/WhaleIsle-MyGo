package pluginui

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"whaleisle/internal/desktop/pluginslots"
)

func TestBridgeActionsRequireCurrentPageRevisionAndSubject(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("host")) }))
	defer host.Close()
	o := &Owner{selected: Subject{Kind: "bundle", Name: "fixture"}}
	gateway, err := pluginslots.NewGatewayWithHandler(host.URL, "host-secret", o.serve)
	if err != nil {
		t.Fatal(err)
	}
	o.gateway = gateway
	defer o.Close()
	entry, _ := gateway.EntryURL(host.URL + "/")
	jar, _ := cookiejar.New(nil)
	browser := &http.Client{Jar: jar}
	response, err := browser.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	browserURL := response.Request.URL
	connect := func() *websocket.Conn {
		t.Helper()
		u := *browserURL
		u.Scheme = "ws"
		u.Path = bridgePath
		u.RawQuery = ""
		headers := http.Header{"Origin": {browserURL.Scheme + "://" + browserURL.Host}}
		for _, c := range jar.Cookies(browserURL) {
			headers.Add("Cookie", c.Name+"="+c.Value)
		}
		conn, _, e := websocket.Dial(context.Background(), u.String(), &websocket.DialOptions{HTTPHeader: headers, Subprotocols: []string{bridgeProtocol}})
		if e != nil {
			t.Fatal(e)
		}
		var selectMessage map[string]any
		if e = wsjson.Read(context.Background(), conn, &selectMessage); e != nil {
			t.Fatal(e)
		}
		return conn
	}
	first := connect()
	defer first.CloseNow()
	publish := func(conn *websocket.Conn, revision uint64, subject Subject) {
		t.Helper()
		e := wsjson.Write(context.Background(), conn, map[string]any{"type": "state", "revision": revision, "locale": "en", "entries": []Entry{{Slot: "plugins.bundle.config", Key: "fixture"}}, "subject": subject, "actions": []Action{{Handle: "current", Label: "Fixture action"}}, "badges": []string{"Ready"}, "height": 0})
		if e != nil {
			t.Fatal(e)
		}
	}
	wait := func(ready func(Snapshot) bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if ready(o.Snapshot()) {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("bridge did not settle")
	}
	publish(first, 1, o.selected)
	wait(func(s Snapshot) bool { return s.Connected && len(s.Actions) == 1 })
	s := o.Snapshot()
	if err = o.Click(s.Page, s.Revision, *s.Subject, "current"); err != nil {
		t.Fatal(err)
	}
	var command map[string]any
	if err = wsjson.Read(context.Background(), first, &command); err != nil || command["type"] != "click" {
		t.Fatalf("control handoff %v %v", command, err)
	}
	if err = o.Click(s.Page, s.Revision+1, *s.Subject, "current"); err == nil {
		t.Fatal("stale revision accepted")
	}
	if err = o.Click(s.Page, s.Revision, Subject{Kind: "bundle", Name: "other"}, "current"); err == nil {
		t.Fatal("foreign subject accepted")
	}
	second := connect()
	defer second.CloseNow()
	publish(second, 1, o.selected)
	wait(func(v Snapshot) bool { return v.Connected && v.Page != s.Page })
	if err = o.Click(s.Page, s.Revision, *s.Subject, "current"); err == nil {
		t.Fatal("previous page accepted")
	}
	o.Select(Subject{Kind: "item", ItemID: "changed"})
	if len(o.Snapshot().Actions) != 0 {
		t.Fatal("navigation retained action authority")
	}
	publish(second, 2, Subject{Kind: "bundle", Name: "fixture"})
	wait(func(v Snapshot) bool { return v.Revision == 2 })
	if len(o.Snapshot().Actions) != 0 {
		t.Fatal("late old subject regained action authority")
	}
	o.Close()
	if o.Snapshot().Connected {
		t.Fatal("closed bridge retained metadata")
	}
	u, _ := url.Parse(entry)
	if _, err = browser.Get(u.String()); err == nil {
		t.Fatal("closed browser capability survived")
	}
}
