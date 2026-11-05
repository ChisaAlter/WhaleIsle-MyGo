package accountlogin

import (
	"testing"
	"time"

	"whaleisle/internal/ui/i18n"
)

func TestCallbackOriginUsesExistingHostLoopback(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:42713", "http://[::1]:42713", "http://localhost:42713/"} {
		got, err := CallbackOrigin(origin)
		if err != nil || got == "" || got[len(got)-1] == '/' {
			t.Fatalf("%s: %q, %v", origin, got, err)
		}
	}
	for _, origin := range []string{"whaleisle://auth", "https://localhost:42713", "http://localhost", "http://127.0.0.1:0", "http://example.com:80", "http://localhost:123/x", "http://localhost:123?x=1", "http://user@localhost:123"} {
		if _, err := CallbackOrigin(origin); err == nil {
			t.Errorf("accepted non-Host callback: %s", origin)
		}
	}
}

func TestAccountMetadataUsesAcceptedLocaleAndBuild(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	SetClientVersion("0.3.13-mygo-test")
	for locale, expected := range map[string]string{"zh": "zh-CN", "en": "en-US"} {
		i18n.SetLocale(locale)
		got := ClientMetadata()
		_, offset := time.Now().Zone()
		if got.Version != "0.3.13-mygo-test" || got.Locale != expected || got.TimezoneOffsetSeconds != float64(offset) {
			t.Fatalf("metadata: %+v", got)
		}
	}
}
