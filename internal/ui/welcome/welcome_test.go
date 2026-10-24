package welcome

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestViewWelcomeActionsAndKeyForm(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	apiKey := ""
	var saved, signedIn, skipped, keyForm, back int
	model := Model{
		ThemeID: "system", Page: PageSignIn, APIKey: &apiKey,
		OnSignIn:      func() { signedIn++ },
		OnShowKeyForm: func() { keyForm++ },
		OnSaveAPIKey:  func() { saved++ },
		OnBack:        func() { back++ },
		OnSkip:        func() { skipped++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, 600, 700)
	for _, label := range []string{
		i18n.T(i18n.KeyWelcomeTagline),
		i18n.T(i18n.KeyWelcomeDescription),
		i18n.T(i18n.KeyWelcomeSignIn),
		i18n.T(i18n.KeyWelcomeAPIKey),
		i18n.T(i18n.KeyWelcomeSkip),
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts = %q", label, tt.Texts())
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeSignIn)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeAPIKey)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeSkip)); err != nil {
		t.Fatal(err)
	}
	if signedIn != 1 || keyForm != 1 || skipped != 1 {
		t.Fatalf("signIn=%d keyForm=%d skip=%d", signedIn, keyForm, skipped)
	}

	// The key form saves only with a non-blank draft and returns with back.
	keyPage := model
	keyPage.Page = PageKey
	saved = 0
	tt = ui.NewTester(func(c *ui.Context) { View(c, keyPage) }, 600, 700)
	for _, label := range []string{
		i18n.T(i18n.KeyWelcomeKeyTitle),
		i18n.T(i18n.KeyWelcomeKeyDescription),
		i18n.T(i18n.KeyWelcomeKeySave),
		i18n.T(i18n.KeyWelcomeKeyBack),
	} {
		if !tt.HasText(label) {
			t.Fatalf("key form missing %q; texts = %q", label, tt.Texts())
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeKeyBack)); err != nil {
		t.Fatal(err)
	}
	if back != 1 {
		t.Fatalf("back = %d", back)
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeAPIKey)); err != nil {
		t.Fatal(err)
	}
	tt.Type("sk-test")
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyWelcomeKeySave)); err != nil {
		t.Fatal(err)
	}
	if saved != 1 {
		t.Fatalf("saved = %d", saved)
	}
}

func TestViewWelcomeNoteAndSignInControls(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var copied, cancelled int
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID: "system", Page: PageSignIn,
			Note: i18n.T(i18n.KeyWelcomeSignInWaiting), SignInActive: true, AuthURL: "https://example.com/auth",
			OnCopyLink: func() { copied++ }, OnCancelSignIn: func() { cancelled++ },
		})
	}, 600, 700)
	if !tt.HasText(i18n.T(i18n.KeyWelcomeSignInWaiting)) {
		t.Fatalf("note missing; texts = %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeCopyLink)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyWelcomeCancel)); err != nil {
		t.Fatal(err)
	}
	if copied != 1 || cancelled != 1 {
		t.Fatalf("copied=%d cancelled=%d", copied, cancelled)
	}
}
