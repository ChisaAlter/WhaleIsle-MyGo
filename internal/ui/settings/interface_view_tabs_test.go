package settings

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestInterfaceViewTabsFallbackStates(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, tc := range []struct {
		name              string
		snapshot          store.ViewState
		fallback, waiting bool
	}{
		{"disconnected", store.ViewState{}, true, false},
		{"pending", store.ViewState{ClientReady: true}, false, true},
		{"loading", store.ViewState{ClientReady: true, Loading: true}, false, true},
		{"load-refused", store.ViewState{ClientReady: true, Err: "fixture load refused"}, false, true},
		{"described-empty", store.ViewState{ClientReady: true, Described: true, Writable: true}, true, false},
		{"described-read-only-empty", store.ViewState{ClientReady: true, Described: true}, true, false},
		{"refreshing-empty", store.ViewState{ClientReady: true, Described: true, Loading: true}, false, true},
		{"failed-empty", store.ViewState{ClientReady: true, Described: true, Err: "fixture retry failed"}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local, archive := true, true
			access := &BodySettingsAccess{Snapshot: tc.snapshot}
			m := Model{Interface: &InterfaceAccess{Host: access, ShowViewTabs: &local, ShowArchivedList: &archive, OnSetViewTabs: func(next bool) (bool, error) { return next, nil }}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { interfacePane(c, m, c.Theme().Text, c.Theme().TextMuted) })
			}, 780, 1000)
			if err := tt.Click("Local preferences"); err != nil {
				t.Fatal(err)
			}
			if got := tt.HasText(i18n.T(i18n.KeySettingsInterfaceViewTabs)); got != tc.fallback {
				t.Fatalf("fallback=%v want=%v", got, tc.fallback)
			}
			if got := tt.HasText("waiting for this Host's settings description"); got != tc.waiting {
				t.Fatalf("waiting=%v want=%v", got, tc.waiting)
			}
			if tc.fallback {
				for range 12 {
					tt.Key(0, ui.KeyTab)
					if tt.Focused(i18n.T(i18n.KeySettingsInterfaceViewTabs)) {
						break
					}
				}
				if !tt.Focused(i18n.T(i18n.KeySettingsInterfaceViewTabs)) {
					t.Fatal("local fallback not keyboard reachable")
				}
				tt.Key(0, ui.KeySpace)
				if local {
					t.Fatal("local fallback did not toggle")
				}
			}
		})
	}
}

func TestInterfaceViewTabsUsesDescribedTypeAndNamespace(t *testing.T) {
	for _, namespace := range []string{"ui-conversation", "conversation", "submissionPolicy", "ui"} {
		for _, kind := range []store.FieldKind{store.FieldBool, store.FieldString} {
			t.Run(fmt.Sprintf("%s/%s", namespace, kind), func(t *testing.T) {
				a := &BodySettingsAccess{Snapshot: store.ViewState{ClientReady: true, Described: true, Namespaces: []store.Namespace{{NS: namespace, Fields: []store.Field{{Key: "viewTabs", Kind: kind}}, Value: json.RawMessage(`{"viewTabs":false}`)}}}}
				if InterfaceViewTabsFallbackAllowed(a) {
					t.Fatal("described field granted local authority")
				}
				on, accepted := InterfaceViewTabsAccepted(a)
				if accepted != (kind == store.FieldBool) || on {
					t.Fatalf("typed accepted=%v value=%v", accepted, on)
				}
			})
		}
	}
	for _, raw := range []string{`{}`, `{"viewTabs":null}`, `{"viewTabs":"false"}`, `{"viewTabs":0}`, `broken`} {
		a := &BodySettingsAccess{Snapshot: store.ViewState{ClientReady: true, Described: true, Namespaces: []store.Namespace{{NS: "ui-conversation", Fields: []store.Field{{Key: "viewTabs", Kind: store.FieldBool}}, Value: json.RawMessage(raw)}}}}
		if InterfaceViewTabsFallbackAllowed(a) {
			t.Fatal("invalid accepted value granted fallback")
		}
		if _, accepted := InterfaceViewTabsAccepted(a); accepted {
			t.Fatalf("manufactured accepted value: %s", raw)
		}
	}
	unrelated := &BodySettingsAccess{Snapshot: store.ViewState{ClientReady: true, Described: true, Namespaces: []store.Namespace{{NS: "unrelated", Fields: []store.Field{{Key: "viewTabs", Kind: store.FieldBool}}, Value: json.RawMessage(`{"viewTabs":false}`)}}}}
	if !InterfaceViewTabsFallbackAllowed(unrelated) {
		t.Fatal("unrelated namespace captured authority")
	}
	if _, accepted := InterfaceViewTabsAccepted(unrelated); accepted {
		t.Fatal("accepted unrelated field")
	}
}

func TestInterfaceViewTabsHostFieldHasOnlyOneControl(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, writable := range []bool{true, false} {
		t.Run(map[bool]string{true: "writable", false: "read-only"}[writable], func(t *testing.T) {
			local, archive := true, true
			writes := 0
			h := &BodySettingsAccess{Owner: 2, Snapshot: store.ViewState{ClientReady: true, Described: true, Writable: writable,
				Namespaces: []store.Namespace{{NS: "ui-conversation", Fields: []store.Field{{Key: "viewTabs", Kind: store.FieldBool}}, Value: json.RawMessage(`{"viewTabs":false}`)}}},
				OnChange: func(string, string, string) { writes++ },
			}
			m := Model{Interface: &InterfaceAccess{Host: h, ShowViewTabs: &local, ShowArchivedList: &archive, OnSave: func() {}}}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { interfacePane(c, m, c.Theme().Text, c.Theme().TextMuted) })
			}, 780, 1000)
			if err := tt.Click("Local preferences"); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText("Conversation and trajectory tabs") {
				t.Fatal("Host control absent")
			}
			if tt.HasText(i18n.T(i18n.KeySettingsInterfaceViewTabs)) {
				t.Fatal("local fallback bypasses the described Host authority")
			}
			if !writable {
				for range 12 {
					tt.Key(0, ui.KeyTab)
					if tt.Focused("Conversation and trajectory tabs") {
						t.Fatal("read-only switch acquired focus")
					}
				}
			}
			if writable {
				for range 12 {
					tt.Key(0, ui.KeyTab)
					if tt.Focused("Conversation and trajectory tabs") {
						break
					}
				}
				if !tt.Focused("Conversation and trajectory tabs") {
					t.Fatal("Host switch is not keyboard reachable")
				}
				tt.Key(0, ui.KeySpace)
			}
			want := 0
			if writable {
				want = 1
			}
			if writes != want || !local {
				t.Fatalf("writes=%d local=%v", writes, local)
			}
		})
	}
}
