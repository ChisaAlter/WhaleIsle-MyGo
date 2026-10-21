package im

import (
	"testing"

	"github.com/egoist/mygo/ui"

	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/ui/i18n"
)

// TestEmptyStateUsesRealCapability: a connected Host with no IM RPC must not
// tell the user to connect again; only an unready Host uses that sentence.
func TestEmptyStateUsesRealCapability(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })

	ready := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: imstore.New(), ThemeID: "deepseek", HarnessReady: true})
	}, 900, 700)
	if !ready.HasText(i18n.T(keyIMPluginNotReady)) {
		t.Fatalf("missing plugin-not-ready state: %v", ready.Texts())
	}
	if ready.HasText(i18n.T(i18n.KeyIMStubNote)) {
		t.Fatalf("connected Host still asks to connect: %v", ready.Texts())
	}

	offline := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: imstore.New(), ThemeID: "deepseek", HarnessReady: false})
	}, 900, 700)
	if !offline.HasText(i18n.T(i18n.KeyIMStubNote)) {
		t.Fatalf("missing connect state: %v", offline.Texts())
	}
	if offline.HasText(i18n.T(keyIMPluginNotReady)) {
		t.Fatalf("disconnected Host claims a plugin problem: %v", offline.Texts())
	}
}

func TestMessageStateLabelsNameRealDelivery(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })

	cases := map[string]string{
		imstore.MessageStatePending: i18n.T(keyIMSendPending),
		imstore.MessageStateFailed:  i18n.T(keyIMSendFailed),
		imstore.MessageStateLocal:   i18n.T(keyIMSendLocal),
		imstore.MessageStateUnknown: i18n.T(keyIMSendUnknown),
		imstore.MessageStateSent:    "",
		"":                          "",
	}
	for state, want := range cases {
		if got := imMessageStateLabel(state); got != want {
			t.Fatalf("state %q = %q, want %q", state, got, want)
		}
	}
}
