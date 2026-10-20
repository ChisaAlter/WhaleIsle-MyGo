package i18n

import "testing"

func TestRegisterAddsBothCatalogsAndKeepsDefaults(t *testing.T) {
	const key = "test.register.only"
	const zh = "仅测试"
	const enText = "Test only"

	if Registered(key) {
		t.Fatalf("%q already registered", key)
	}
	Register(key, zh, enText)
	if !Registered(key) {
		t.Fatal("Register did not populate both catalogs")
	}
	if got := Lookup(LocaleZH, key); got != zh {
		t.Fatalf("zh lookup = %q, want %q", got, zh)
	}
	if got := Lookup(LocaleEN, key); got != enText {
		t.Fatalf("en lookup = %q, want %q", got, enText)
	}

	// A second registration must not overwrite the first one.
	Register(key, "改掉", "overwritten")
	if got := Lookup(LocaleZH, key); got != zh {
		t.Fatalf("second Register overwrote zh: %q", got)
	}
	if got := Lookup(LocaleEN, key); got != enText {
		t.Fatalf("second Register overwrote en: %q", got)
	}
}

func TestRegisterIgnoresEmptyKey(t *testing.T) {
	Register("", "空", "empty")
	if Registered("") {
		t.Fatal("empty key reported as registered")
	}
}

func TestRegisterDoesNotOverwriteSeedKeys(t *testing.T) {
	before := Lookup(LocaleZH, KeyAppName)
	Register(KeyAppName, "另一个名字", "Another name")
	if got := Lookup(LocaleZH, KeyAppName); got != before {
		t.Fatalf("seed key overwritten: %q -> %q", before, got)
	}
}
