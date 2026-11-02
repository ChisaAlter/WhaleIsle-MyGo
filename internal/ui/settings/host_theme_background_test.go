package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
)

const hostBackgroundSchema = `{"type":"object","dict":{"backgroundEffect":{"type":"union","list":[{"type":"const","value":"none"},{"type":"const","value":"gradient"}]},"glassOpacity":{"type":"number"}}}`

type hostBackgroundReceipt struct {
	value    any
	schema   string
	writable bool
	fail     bool
	revision float64
	writes   []map[string]any
}

func (f *hostBackgroundReceipt) client() *dsh.Client {
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://background.fixture.invalid"}, HTTP: &http.Client{Transport: hostThemeTestTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return nil, err
		}
		if f.fail {
			return nil, errors.New("synthetic background describe failure")
		}
		namespace := func() map[string]any {
			return map[string]any{"ns": "ui-theme", "revision": f.revision, "schema": json.RawMessage(f.schema), "value": f.value, "base": map[string]any{}, "user": map[string]any{}, "secrets": []any{}}
		}
		var result any
		switch request["method"] {
		case "settings/describe":
			result = map[string]any{"writable": f.writable, "namespaces": []any{namespace()}}
		case "settings/mutate":
			args := request["payload"].(map[string]any)["args"].(map[string]any)
			f.writes = append(f.writes, args)
			values := f.value.(map[string]any)
			for _, raw := range args["ops"].([]any) {
				op := raw.(map[string]any)
				values[op["path"].([]any)[0].(string)] = op["value"]
			}
			f.revision++
			result = namespace()
		default:
			return nil, errors.New("unexpected background RPC")
		}
		response := httptest.NewRecorder()
		if err := json.NewEncoder(response).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": result}}); err != nil {
			return nil, err
		}
		return response.Result(), nil
	})}}
}

func TestHostBackgroundDescribeAuthority(t *testing.T) {
	for _, tc := range []struct {
		name       string
		value      map[string]any
		schema     string
		writable   bool
		wantEffect string
		explicit   bool
	}{
		{"served default gradient", map[string]any{}, hostBackgroundSchema, true, "gradient", true},
		{"explicit gradient", map[string]any{"backgroundEffect": "gradient"}, hostBackgroundSchema, true, "gradient", true},
		{"explicit none", map[string]any{"backgroundEffect": "none"}, hostBackgroundSchema, true, "none", true},
		{"read only default gradient", map[string]any{}, hostBackgroundSchema, false, "gradient", true},
		{"older Host without effect field", map[string]any{}, `{"type":"object","dict":{"glassOpacity":{"type":"number"}}}`, true, "none", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				fixture := &hostBackgroundReceipt{value: tc.value, schema: tc.schema, writable: tc.writable, revision: 7}
				s := store.New()
				s.Attach(fixture.client(), nil)
				s.RefreshDescribe(context.Background())
				synctest.Wait()
				binding := BindHostTheme(s)
				if binding == nil {
					t.Fatal("accepted ui-theme was not bound")
				}
				got := binding.Snapshot()
				if !got.Ready || got.Writable != tc.writable || got.Background.Effect != tc.wantEffect || got.Background.EffectExplicit != tc.explicit || wallpapers.NormalizeEffect("", got.Background).Effect != tc.wantEffect {
					t.Fatalf("served backdrop = %+v", got)
				}
				if len(fixture.writes) != 0 {
					t.Fatal("reading appearance wrote defaults")
				}
			})
		})
	}
}

func TestHostBackgroundTransitionsAndRetiredBinding(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fixture := &hostBackgroundReceipt{value: map[string]any{}, schema: hostBackgroundSchema, writable: true, revision: 7}
		s := store.New()
		s.Attach(fixture.client(), nil)
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		binding := BindHostTheme(s)
		owner := binding.Snapshot().Owner
		for _, effect := range []string{"gradient", "none", "gradient"} {
			fixture.value = map[string]any{"backgroundEffect": effect}
			fixture.revision++
			s.RefreshDescribe(context.Background())
			synctest.Wait()
			got := binding.Snapshot()
			if !got.Ready || got.Owner != owner || got.Background.Effect != effect || !got.Background.EffectExplicit {
				t.Fatalf("effect %s not adopted: %+v", effect, got)
			}
		}
		s.Detach()
		if got := binding.Snapshot(); got.Ready || got.Background.EffectExplicit {
			t.Fatalf("disconnected binding supplied a backdrop: %+v", got)
		}
		replacement := &hostBackgroundReceipt{value: map[string]any{"backgroundEffect": "none"}, schema: hostBackgroundSchema, writable: true, revision: 1}
		s.Attach(replacement.client(), nil)
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		if got := binding.Snapshot(); got.Ready {
			t.Fatal("retired Host retained appearance authority")
		}
		got := BindHostTheme(s).Snapshot()
		if !got.Ready || got.Owner == owner || got.Background.Effect != "none" {
			t.Fatalf("replacement Host none was not adopted: %+v", got)
		}
	})
}

func TestHostBackgroundFailedReadDoesNotSeed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fixture := &hostBackgroundReceipt{value: map[string]any{}, schema: hostBackgroundSchema, writable: true, fail: true}
		s := store.New()
		s.Attach(fixture.client(), nil)
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		if BindHostTheme(s) != nil {
			t.Fatal("failed first describe supplied appearance defaults")
		}
		fixture.fail = false
		fixture.value = map[string]any{"backgroundEffect": "none"}
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		binding := BindHostTheme(s)
		fixture.fail = true
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		got := binding.Snapshot()
		if got.Background.Effect != "none" || !strings.Contains(got.Error, "synthetic background describe failure") {
			t.Fatalf("failed refresh replaced accepted none: %+v", got)
		}
		fixture.fail = false
		fixture.value = "invalid ui-theme section"
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		got = binding.Snapshot()
		if got.Ready || got.Background.EffectExplicit || got.Error == "" {
			t.Fatalf("invalid section supplied appearance defaults: %+v", got)
		}
	})
}

func TestHostBackgroundSaveProjectsServedDefault(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		fixture := &hostBackgroundReceipt{value: map[string]any{}, schema: hostBackgroundSchema, writable: true, revision: 7}
		s := store.New()
		s.Attach(fixture.client(), nil)
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		binding := BindHostTheme(s)
		prefs := binding.Snapshot().Background
		if err := binding.SaveBackground(prefs); err != nil || binding.Pending() {
			t.Fatalf("unchanged default queued a write: %v", err)
		}
		prefs.Glass = 65
		if err := binding.SaveBackground(prefs); err != nil {
			t.Fatal(err)
		}
		binding.Flush()
		synctest.Wait()
		if len(fixture.writes) != 1 {
			t.Fatalf("slider write count = %d", len(fixture.writes))
		}
		ops := fixture.writes[0]["ops"].([]any)
		if len(ops) != 1 || ops[0].(map[string]any)["path"].([]any)[0] != "glassOpacity" {
			t.Fatalf("slider write serialized background defaults: %v", ops)
		}
		prefs.Effect = "none"
		if err := binding.SaveBackground(prefs); err != nil {
			t.Fatal(err)
		}
		binding.Flush()
		synctest.Wait()
		got := binding.Snapshot()
		if got.Pending || got.Background.Effect != "none" || got.Revision != 9 {
			t.Fatalf("explicit none was not accepted: %+v", got)
		}
		s.Detach()
		if err := binding.SaveBackground(prefs); err == nil {
			t.Fatal("disconnected binding accepted a background save")
		}
	})
}
