package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestInterfaceI18nKeys(t *testing.T) {
	keys := []string{
		i18n.KeySettingsInterfaceNav,
		i18n.KeySettingsInterfaceTitle,
		i18n.KeySettingsInterfaceHint,
		i18n.KeySettingsDesktopLocale,
		i18n.KeySettingsDesktopCloseToTray,
		i18n.KeySettingsDesktopOpenAtLogin,
		i18n.KeySettingsInterfaceAutoStart,
		i18n.KeySettingsInterfaceAutoStartDesc,
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
	i18n.SetLocale(i18n.DefaultLocale)
}

func TestInterfaceAccessFieldsPresent(t *testing.T) {
	locale := "zh"
	closeToTray := true
	openAtLogin := false
	saved := false
	a := InterfaceAccess{
		Locale:      &locale,
		CloseToTray: &closeToTray,
		OpenAtLogin: &openAtLogin,
		OnSave:      func() { saved = true },
		LastAction:  "ok",
		Err:         "",
	}
	if a.Locale == nil || *a.Locale != "zh" || a.CloseToTray == nil || !*a.CloseToTray {
		t.Fatal("expected locale/closeToTray fields to round-trip")
	}
	if a.OpenAtLogin == nil || *a.OpenAtLogin {
		t.Fatal("expected openAtLogin false")
	}
	a.OnSave()
	if !saved {
		t.Fatal("OnSave should run")
	}
}
