package i18n

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":      LocaleZH,
		"zh":    LocaleZH,
		"zh-CN": LocaleZH,
		"zh_CN": LocaleZH,
		"ZH":    LocaleZH,
		"en":    LocaleEN,
		"en-US": LocaleEN,
		"en_GB": LocaleEN,
		"fr":    LocaleZH, // unknown → default
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTChineseAndEnglish(t *testing.T) {
	SetLocale("zh")
	if Locale() != LocaleZH {
		t.Fatalf("Locale() = %q after SetLocale(zh)", Locale())
	}
	if got := T(KeyTrayOpen); got != "打开主窗口" {
		t.Fatalf("zh tray.open = %q", got)
	}
	if got := T(KeyTrayQuit); got != "退出" {
		t.Fatalf("zh tray.quit = %q", got)
	}
	if got := T(KeyAppName); got != "鲸屿 MyGo" {
		t.Fatalf("app.name = %q", got)
	}

	SetLocale("en")
	if Locale() != LocaleEN {
		t.Fatalf("Locale() = %q after SetLocale(en)", Locale())
	}
	if got := T(KeyAppName); got != "Whale Isle MyGo" {
		t.Fatalf("en app.name = %q", got)
	}
	if got := T(KeyTrayOpen); got != "Open main window" {
		t.Fatalf("en tray.open = %q", got)
	}
	if got := T(KeyStatusPlaceholder); got != "Skeleton: window and tray ready" {
		t.Fatalf("en status.placeholder = %q", got)
	}

	// Restore default so other tests / packages see zh-CN.
	SetLocale(DefaultLocale)
}

func TestFallbackMissingKey(t *testing.T) {
	const missing = "shell.no.such.key"
	SetLocale(LocaleZH)
	if got := T(missing); got != missing {
		t.Fatalf("missing key should return key, got %q", got)
	}
	if got := Lookup(LocaleEN, missing); got != missing {
		t.Fatalf("Lookup missing = %q", got)
	}
}

func TestFallbackToEnglishWhenZHMissing(t *testing.T) {
	// Inject a key only present in English to exercise zh → en fallback.
	const key = "test.en.only"
	en[key] = "English only"
	defer delete(en, key)

	if got := Lookup(LocaleZH, key); got != "English only" {
		t.Fatalf("zh missing should fall back to en, got %q", got)
	}
	if got := Lookup(LocaleEN, key); got != "English only" {
		t.Fatalf("en lookup = %q", got)
	}
}

func TestSettingsMCPPluginsSkillsKeys(t *testing.T) {
	keys := []string{
		KeySettingsMCPNav, KeySettingsMCPTitle, KeySettingsMCPHint, KeySettingsMCPEmpty,
		KeySettingsMCPStubHint, KeySettingsMCPSchemaHint, KeySettingsMCPEnabled, KeySettingsMCPDisabled,
		KeySettingsPluginsNav, KeySettingsPluginsTitle, KeySettingsPluginsHint, KeySettingsPluginsEmpty,
		KeySettingsPluginsNoRPC, KeySettingsPluginsMarketHint, KeySettingsPluginsOpenMarket,
		KeySettingsSkillsNav, KeySettingsSkillsTitle, KeySettingsSkillsHint, KeySettingsSkillsEmpty,
		KeySettingsSkillsSession, KeySettingsSkillsNoSession, KeySettingsSkillsModelInvocable, KeySettingsSkillsUserOnly,
		KeySettingsAgentPresetsNav, KeySettingsAgentPresetsTitle, KeySettingsAgentPresetsHint,
		KeySettingsAgentPresetsPicker, KeySettingsAgentPresetsEmpty, KeySettingsAgentPresetsSession,
		KeySettingsAgentPresetsNoSession,
		KeySettingsAccountNav, KeySettingsAccountTitle, KeySettingsAccountHint, KeySettingsAccountStub,
		KeySettingsAccountCredHint, KeySettingsAccountStatus, KeySettingsAccountEmail,
		KeySettingsAccountName, KeySettingsAccountPlan, KeySettingsAccountUsageURL,
		KeySettingsAccountTopUpURL, KeySettingsAccountBalance, KeySettingsAccountAttempt,
		KeySettingsAccountAuthorize,
	}
	SetLocale(LocaleZH)
	for _, k := range keys {
		if got := T(k); got == "" || got == k {
			t.Fatalf("zh missing %q got %q", k, got)
		}
	}
	SetLocale(LocaleEN)
	for _, k := range keys {
		if got := T(k); got == "" || got == k {
			t.Fatalf("en missing %q got %q", k, got)
		}
	}
	SetLocale(DefaultLocale)
}

func TestTitlebarKeys(t *testing.T) {
	SetLocale(LocaleZH)
	for _, key := range []string{KeyTitlebarClose, KeyTitlebarMinimize, KeyTitlebarMaximize} {
		if T(key) == key {
			t.Fatalf("expected translation for %q", key)
		}
	}
	SetLocale(DefaultLocale)
}
