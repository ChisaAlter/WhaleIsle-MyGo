package remote

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/ui/i18n"
)

func TestRemoteSettingsGroupingAndDraftOwnership(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	active := "gateway"
	e := &GatewayEditor{}
	m := Model{Embedded: true, SettingsPage: "remote", GatewayProtocol: "chisacode-v2", ActiveTab: &active, GatewayEditor: e, RemoteMode: "relay", RemotePort: 3180, RemoteBindAddress: "127.0.0.1", OnSaveGateway: func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil }, ChannelsContent: func(c *ui.Context) { ui.Text(c, "Fixture platform settings") }}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).FillWidth().Gap(12).Children(func() { View(c, m) }) }, 780, 900)
	text := strings.Join(tt.Texts(), "\n")
	if strings.Contains(text, "SSH tunnel") || strings.Contains(text, "Remote machines") || strings.Contains(text, "Relay access token") {
		t.Fatal(text)
	}
	if err := tt.Click("Gateway port"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("43180")
	tt.Frame()
	if err := tt.Click("Channels"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "Fixture platform settings") {
		t.Fatal("channels contribution missing")
	}
	if err := tt.Click("Gateway"); err != nil {
		t.Fatal(err)
	}
	if e.Draft.Port != "43180" || !e.Open {
		t.Fatal("tab switch discarded gateway draft")
	}
}

func remoteFixtureStore(t *testing.T, state string) *remotestore.Store {
	t.Helper()
	gate := make(chan struct{})
	t.Cleanup(func() { close(gate) })
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if state == "loading" {
			<-gate
		}
		if r.Method != "GET" {
			t.Errorf("unexpected fixture mutation %s", r.Method)
			w.WriteHeader(405)
			return
		}
		if state == "error" {
			http.Error(w, "fixture read failed", 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/dsh-remote/machines":
			rows := []map[string]any{}
			if state == "data" {
				rows = append(rows, map[string]any{"id": "fixture", "name": strings.Repeat("Fixture workstation metadata ", 3), "host": "host.fixture.invalid", "port": 22, "username": "developer", "workspace": "/fixture/workspace/with/a/long/name"})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"machines": rows, "currentId": ""})
		case "/api/dsh-remote/status":
			_, _ = w.Write([]byte(`{"connected":false}`))
		default:
			_, _ = w.Write([]byte(`{"entries":[],"forwards":[],"audit":[]}`))
		}
	})

	s := remotestore.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://remote.fixture.invalid"}, HTTP: &http.Client{Transport: remoteFixtureTransport(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})}}, nil)
	s.Refresh()
	if state == "loading" {
		return s
	}
	deadline := time.Now().Add(3 * time.Second)
	for s.Snapshot().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().Busy {
		t.Fatal("fixture read timed out")
	}
	return s
}

func TestRemoteSettingsSoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set SECONDARY_SNAPSHOT_DIR for body renders")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, state := range []string{"data", "empty", "loading", "error", "no-permission"} {
		for _, page := range []string{"remote", "workspace"} {
			m := Model{Embedded: true, SettingsPage: page, GatewayProtocol: "chisacode-v2", RemoteMode: "relay", RemotePort: 3180, RemoteBindAddress: "127.0.0.1", GatewayEditor: &GatewayEditor{}, OnSaveGateway: func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil }, HarnessReady: true, RemoteWorkspaceEnabled: true, MachineEditor: &MachineEditor{}}
			if page == "workspace" {
				m.Machines = remoteFixtureStore(t, state)
			}
			if state == "no-permission" {
				m.HarnessReady = false
				m.OnSaveGateway = nil
			}
			if state == "loading" {
				m.PhoneRuntime.Status = "starting"
			}
			if state == "error" {
				m.PhoneActionMsg = "Fixture gateway unavailable"
			}
			for _, width := range []int{360, 780} {
				tt := ui.NewTester(func(c *ui.Context) { ui.Scroll(c).Fill().Padding(20).Children(func() { View(c, m) }) }, width, 760)
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("settings-%s-%s-%d.png", page, state, width)))
				if err != nil {
					t.Fatal(err)
				}
				if err = png.Encode(f, tt.Image()); err != nil {
					t.Fatal(err)
				}
				if err = f.Close(); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

type remoteFixtureTransport func(*http.Request) (*http.Response, error)

func (f remoteFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
