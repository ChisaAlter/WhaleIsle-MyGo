package settings

import (
	"reflect"
	"strings"
	"testing"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestRequestReplaceRespectsConfirm(t *testing.T) {
	st := store.New()
	asked := false
	RequestReplace(Model{
		Store: st,
		ConfirmReplace: func() bool {
			asked = true
			return false
		},
	})
	if !asked {
		t.Fatal("ConfirmReplace should be invoked")
	}
	if snap := st.Snapshot(); snap.LastAction != "" || snap.Err != "" {
		t.Fatalf("cancelled replace must not mutate store, last=%q err=%q", snap.LastAction, snap.Err)
	}
}

func TestRequestReplaceNilStoreNoPanic(t *testing.T) {
	RequestReplace(Model{})
}

func TestReplaceI18nKeys(t *testing.T) {
	keys := []string{
		i18n.KeySettingsReplace,
		i18n.KeySettingsReplaceHint,
		i18n.KeySettingsReplaceConfirmTitle,
		i18n.KeySettingsReplaceConfirmMessage,
		i18n.KeySettingsReplaceCancel,
	}
	i18n.SetLocale(i18n.LocaleZH)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("zh missing %q got %q", k, got)
		}
	}
	if !strings.Contains(i18n.T(i18n.KeySettingsReplace), "替换") {
		t.Fatalf("zh replace label = %q", i18n.T(i18n.KeySettingsReplace))
	}
	i18n.SetLocale(i18n.LocaleEN)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("en missing %q got %q", k, got)
		}
	}
	if got := i18n.T(i18n.KeySettingsReplace); got != "Replace document" {
		t.Fatalf("en replace label = %q", got)
	}
	i18n.SetLocale(i18n.DefaultLocale)
}

func TestFilterNamespaces(t *testing.T) {
	names := []string{"general", "plugins", "permission-foo", "mcp"}
	if got := FilterNamespaces(names, ""); !reflect.DeepEqual(got, names) {
		t.Fatalf("empty query = %v, want original", got)
	}
	if got := FilterNamespaces(names, "  "); !reflect.DeepEqual(got, names) {
		t.Fatalf("whitespace query = %v, want original", got)
	}
	got := FilterNamespaces(names, "PLUG")
	want := []string{"plugins"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plug = %v, want %v", got, want)
	}
	got = FilterNamespaces(names, "perm")
	want = []string{"permission-foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("perm = %v, want %v", got, want)
	}
	got = FilterNamespaces(names, "other")
	want = []string{"permission-foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("section other = %v, want %v", got, want)
	}
	if got := FilterNamespaces(names, "zzz"); len(got) != 0 {
		t.Fatalf("zzz = %v, want empty", got)
	}
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	got = FilterNamespaces(names, "插件")
	want = []string{"plugins"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("zh section label = %v, want %v", got, want)
	}
}

func TestSecretLikeKey(t *testing.T) {
	yes := []string{"apiKey", "API_KEY", "githubToken", "secret", "client-secret", "provider.apiKey", "password"}
	for _, k := range yes {
		if !SecretLikeKey(k) {
			t.Errorf("SecretLikeKey(%q) = false", k)
		}
	}
	no := []string{"", "workspace", "port", "model", "cacheKeyName", "label"}
	for _, k := range no {
		if SecretLikeKey(k) {
			t.Errorf("SecretLikeKey(%q) = true", k)
		}
	}
}

func TestMultilineStringField(t *testing.T) {
	if MultilineStringField(store.Field{Key: "name", Kind: store.FieldString}) {
		t.Fatal("plain string should stay single-line")
	}
	if !MultilineStringField(store.Field{Key: "systemPrompt", Kind: store.FieldString}) {
		t.Fatal("systemPrompt should be multiline")
	}
	if !MultilineStringField(store.Field{Key: "bio", Kind: store.FieldString, Description: strings.Repeat("x", 80)}) {
		t.Fatal("long description should be multiline")
	}
	if MultilineStringField(store.Field{Key: "systemPrompt", Kind: store.FieldBool}) {
		t.Fatal("non-string kinds must not use the multiline heuristic")
	}
}

func TestGenericFormI18nKeys(t *testing.T) {
	keys := []string{
		i18n.KeySettingsFilter,
		i18n.KeySettingsFilterEmpty,
		i18n.KeySettingsWritable,
		i18n.KeySettingsRevision,
		i18n.KeySettingsRevealSecrets,
	}
	i18n.SetLocale(i18n.LocaleZH)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("zh missing %q got %q", k, got)
		}
	}
	i18n.SetLocale(i18n.LocaleEN)
	for _, k := range keys {
		if got := i18n.T(k); got == "" || got == k {
			t.Fatalf("en missing %q got %q", k, got)
		}
	}
	if got := i18n.T(i18n.KeySettingsFilter); got != "Filter namespaces…" {
		t.Fatalf("en filter = %q", got)
	}
	i18n.SetLocale(i18n.DefaultLocale)
}
