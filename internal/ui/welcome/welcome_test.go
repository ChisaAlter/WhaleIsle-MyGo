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

func TestViewWelcomeKeyFormPasswordAutoFocusSubmitAndEscape(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	apiKey := ""
	saved, back := 0, 0
	busy := false
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:      "system",
			Page:         PageKey,
			APIKey:       &apiKey,
			Busy:         busy,
			OnSaveAPIKey: func() { saved++ },
			OnBack:       func() { back++ },
		})
	}, 600, 700)

	// Input is auto-focused on entering PageKey: typing works immediately without clicking.
	tt.Type("sk-secret-12345")
	tt.Frame()
	if apiKey != "sk-secret-12345" {
		t.Fatalf("auto-focused input got %q, want %q", apiKey, "sk-secret-12345")
	}

	// API key is masked by default (plain key not painted in text tree) and reveal toggle is offered.
	if tt.HasText("sk-secret-12345") {
		t.Fatalf("API key leaked in plaintext while Password=true; texts=%q", tt.Texts())
	}
	if !tt.HasText("Show key") {
		t.Fatalf("missing reveal toggle button; texts=%q", tt.Texts())
	}

	// Enter submits the form when not composing.
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if saved != 1 {
		t.Fatalf("Enter submit calls = %d, want 1", saved)
	}

	// Escape returns to sign-in view when not busy.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if back != 1 {
		t.Fatalf("Escape back calls = %d, want 1", back)
	}

	// Escape is ignored while busy.
	busy = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if back != 1 {
		t.Fatalf("Escape fired while busy: back=%d, want 1", back)
	}
}
