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

	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

type hostThemeTestTransport func(*http.Request) (*http.Response, error)

func (f hostThemeTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The transport is entirely in memory; these receipts exercise the actual
// settings writer, debounce/flush, CAS, refresh, and editor adoption path.
func TestHostThemeAsyncReceiptsPreserveDraftAndRetry(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	synctest.Test(t, func(t *testing.T) {
		values := map[string]any{"fontSize": float64(14), "fontSizeCode": float64(13), "fontFamilySans": "", "fontFamilyCode": "", "fontFamilyComposer": "", "fontFamilyTerminal": ""}
		revision := float64(7)
		schema := json.RawMessage(`{"uid":2,"refs":{"1":{"type":"number"},"2":{"type":"object","dict":{"fontSize":1,"fontSizeCode":1}}}}`)
		gate := make(chan struct{})
		fail := true
		requests := []map[string]any{}
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: hostThemeTestTransport(func(r *http.Request) (*http.Response, error) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				return nil, err
			}
			out := func() map[string]any {
				return map[string]any{"ns": "ui-theme", "revision": revision, "schema": schema, "value": values, "base": values, "user": map[string]any{}, "secrets": []any{}}
			}
			var value any
			switch body["method"] {
			case "settings/describe":
				value = map[string]any{"writable": true, "namespaces": []any{out()}}
			case "settings/mutate":
				args := body["payload"].(map[string]any)["args"].(map[string]any)
				requests = append(requests, args)
				<-gate
				if fail {
					return nil, errors.New("synthetic appearance conflict")
				}
				if args["expectedRevision"] != revision {
					t.Errorf("CAS revision=%v want=%v", args["expectedRevision"], revision)
				}
				for _, raw := range args["ops"].([]any) {
					op := raw.(map[string]any)
					path := op["path"].([]any)
					if len(path) != 1 {
						t.Fatal(path)
					}
					values[path[0].(string)] = op["value"]
				}
				revision++
				value = out()
			default:
				value = map[string]any{}
			}
			recorder := httptest.NewRecorder()
			_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": value}})
			return recorder.Result(), nil
		})}}
		s := store.New()
		s.Attach(client, nil)
		s.RefreshDescribe(context.Background())
		synctest.Wait()
		binding := BindHostTheme(s)
		if binding == nil {
			t.Fatal("theme not bound")
		}
		initial := binding.Snapshot()
		if !initial.Ready {
			t.Fatal(initial)
		}
		e := theme.NewEditor(initial.Preferences, binding.SaveTypography)
		e.Pending = binding.Pending
		e.FlushWrites = binding.Flush
		// The old Host declares only numeric fields; a size edit must not migrate
		// empty family strings into locally expanded CSS fallback lists.
		if !e.SaveFontSize("text", 16) {
			t.Fatal(e.Status)
		}
		synctest.Wait()
		if !binding.Pending() || e.Saved.Chat.Size != 14 || e.Draft.Chat.Size != 16 {
			t.Fatalf("pending advanced accepted state: saved=%+v draft=%+v", e.Saved, e.Draft)
		}
		if len(requests) != 1 || len(requests[0]["ops"].([]any)) != 1 {
			t.Fatalf("size-only save serialized unrelated defaults: %v", requests)
		}
		close(gate)
		synctest.Wait()
		failed := binding.Snapshot()
		if failed.Pending || failed.Error == "" || !strings.Contains(failed.Error, "synthetic appearance conflict") {
			t.Fatalf("failure lost across describe refresh: %+v", failed)
		}
		e.AcceptHost(failed.Preferences, false, failed.Error)
		if e.Saved.Chat.Size != 14 || e.Draft.Chat.Size != 16 || !strings.Contains(e.Status, "conflict") {
			t.Fatalf("failed draft erased: saved=%+v draft=%+v status=%s", e.Saved, e.Draft, e.Status)
		}
		fail = false
		e.Save()
		binding.Flush()
		synctest.Wait()
		accepted := binding.Snapshot()
		e.AcceptHost(accepted.Preferences, false, accepted.Error)
		if accepted.Pending || accepted.Error != "" || accepted.Revision != 8 || e.Saved.Chat.Size != 16 || e.Draft.Chat.Size != 16 {
			t.Fatalf("retry did not accept real Host receipt: %+v saved=%+v draft=%+v", accepted, e.Saved, e.Draft)
		}
		if len(requests) != 2 || requests[1]["expectedRevision"] != float64(7) {
			t.Fatalf("retry lost accepted revision: %v", requests)
		}
		for _, family := range []string{e.Saved.Interface.Family, e.Saved.Chat.Family, e.Saved.Code.Family, e.Saved.Composer.Family, e.Saved.Terminal.Family} {
			if family != "" {
				t.Fatal("raw Host family was migrated", family)
			}
		}
	})
}

func TestThemePendingFontEditsAccumulateWithoutSavingCustomDraft(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	p := theme.Defaults()
	var submitted []theme.Preferences
	e := theme.NewEditor(p, func(next theme.Preferences) error { submitted = append(submitted, next); return nil })
	e.Pending = func() bool { return true }
	e.Draft.Custom = []theme.Family{{ID: "unsaved", Name: "Unsaved custom theme"}}
	if !e.SaveFontSize("text", 16) || !e.SaveFontSize("code", 15) {
		t.Fatal(e.Status)
	}
	if len(submitted) != 2 || submitted[1].Chat.Size != 16 || submitted[1].Code.Size != 15 || len(submitted[1].Custom) != 0 {
		t.Fatalf("continuous typography edits lost a submitted field or wrote custom draft: %+v", submitted)
	}
	accepted := submitted[1].Clone()
	e.AcceptHost(accepted, false, "")
	if e.Saved.Chat.Size != 16 || e.Saved.Code.Size != 15 || len(e.Saved.Custom) != 0 || len(e.Draft.Custom) != 1 {
		t.Fatal("receipt acknowledged unsaved custom theme")
	}
	e.AcceptHost(p, true, "")
	if len(e.Draft.Custom) != 0 || e.Draft.Chat.Size != 14 {
		t.Fatal("new Host retained old transient draft")
	}
	e.QueueGeneralFontEdit("text", "Fixture Face")
	if !e.CommitGeneralFontEdit("text") {
		t.Fatal(e.Status)
	}
	e.AcceptHost(p, false, "synthetic family failure")
	if e.GeneralFontFamily("text") != "Fixture Face" || e.Saved.Interface.Family != p.Interface.Family {
		t.Fatal("failed General input was acknowledged or lost")
	}
	if !e.CommitGeneralFontEdit("text") {
		t.Fatal(e.Status)
	}
	e.AcceptHost(submitted[len(submitted)-1], false, "")
	if e.GeneralFontFamily("text") != "Fixture Face" || e.Saved.Interface.Family != "Fixture Face" {
		t.Fatal("accepted General input did not settle")
	}
	e.AcceptHost(p, true, "")
	if e.GeneralFontFamily("text") != p.Interface.Family {
		t.Fatal("Host replacement kept General input")
	}
}
