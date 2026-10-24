// Package welcome draws the first-run welcome window: API key entry, account
// sign-in and skip. It ports the Electron welcome window semantics
// (reference/electron/src/main/welcome-window.js + renderer/welcome.html +
// welcome-backend.js) onto the MyGo native UI: the key form validates the key
// shape before saving, sign-in reuses the desktop account login RPC, and skip
// enters the ordinary launcher/boot flow without a credential.
package welcome

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Welcome pages, mirroring the Electron renderer's tagline / key-form views.
const (
	PageSignIn = "signin"
	PageKey    = "key"
)

// Model is the welcome window's observable state, filled by the shell each
// frame. Callbacks run on the UI thread from button clicks.
type Model struct {
	ThemeID    string
	SystemDark bool
	// Page selects the tagline view (sign-in) or the API key form.
	Page string
	// APIKey is the editable key draft owned by the shell.
	APIKey *string
	// Busy marks a save in flight.
	Busy bool
	// SignInActive marks a running account sign-in attempt.
	SignInActive bool
	// AuthURL is the authorize URL of the running attempt, when known.
	AuthURL string
	// Note is a status line; NoteError paints it as a failure.
	Note      string
	NoteError bool
	// Hint is an optional secondary line under Note (sign-in guidance).
	Hint string
	// OnSignIn opens the account sign-in flow.
	OnSignIn func()
	// OnShowKeyForm switches to the API key form.
	OnShowKeyForm func()
	// OnBack returns from the API key form to the sign-in view.
	OnBack func()
	// OnSaveAPIKey validates and stores the draft key.
	OnSaveAPIKey func()
	// OnCopyLink copies the authorize URL to the clipboard.
	OnCopyLink func()
	// OnCancelSignIn abandons the running attempt.
	OnCancelSignIn func()
	// OnSkip enters the shell without a credential.
	OnSkip func()
}

// View paints the welcome window body. The shell owns the window, the drafts
// and every backend call; this package only renders.
func View(c *ui.Context, m Model) {
	theme.ApplyInterface(c)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	failure := c.Theme().Danger
	page := m.Page
	if page != PageKey {
		page = PageSignIn
	}

	ui.Column(c).Fill().Background(bg).Padding(16, 24, 20, 24).Gap(12).Children(func() {
		// Drag strip standing in for the Electron 36px app-region title bar;
		// the system caption button closes the window.
		bar := c.TitleBar()
		ui.Row(c).FillWidth().Height(24).AlignItems(ui.Center).Gap(8).DragWindow().Children(func() {
			if bar.Left > 0 {
				ui.Box(c).Width(bar.Left)
			}
			if logo := chrome.FishLogo(); logo != nil {
				ui.Image(c, logo).Height(20).Width(27).Fit(ui.Contain)
			}
			ui.Text(c, i18n.T(i18n.KeyAppName)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			ui.Box(c).Grow(1)
			if bar.Right > 0 {
				ui.Box(c).Width(bar.Right)
			}
		})
		ui.Column(c).Grow(1).FillWidth().Center().Gap(6).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWelcomeTagline)).FontSize(theme.InterfaceSize(26)).FontWeight(500).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyWelcomeDescription)).FontSize(theme.InterfaceSize(13)).TextColor(muted)
		})
		ui.Column(c).FillWidth().Center().Gap(10).Children(func() {
			// The action column mirrors the Electron .actions block: one
			// primary path, secondary sign-in, and text-only skip/back.
			ui.Column(c).Width(280).MaxWidthPercent(100).Gap(8).Children(func() {
				switch page {
				case PageKey:
					ui.Text(c, i18n.T(i18n.KeyWelcomeKeyTitle)).FontSize(theme.InterfaceSize(15)).FontWeight(500).TextColor(fg)
					ui.Text(c, i18n.T(i18n.KeyWelcomeKeyDescription)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					if m.APIKey != nil {
						ui.TextInput(c, m.APIKey).FillWidth().Height(40).Radius(10).Border(1, border).
							Placeholder(i18n.T(i18n.KeyWelcomeKeyPlaceholder)).Label(i18n.T(i18n.KeyWelcomeAPIKey))
					}
					saveLabel := i18n.T(i18n.KeyWelcomeKeySave)
					if m.Busy {
						saveLabel = i18n.T(i18n.KeyWelcomeKeySaving)
					}
					save := ui.PrimaryButton(c, saveLabel).FillWidth().Height(40).Radius(10).Disabled(m.Busy || m.OnSaveAPIKey == nil || keyBlank(m.APIKey))
					if save.Clicked() && m.OnSaveAPIKey != nil {
						m.OnSaveAPIKey()
					}
					if ui.Button(c, i18n.T(i18n.KeyWelcomeKeyLater)).FillWidth().Height(32).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(12)).Clicked() && m.OnSkip != nil {
						m.OnSkip()
					}
				default:
					signIn := ui.Button(c, i18n.T(i18n.KeyWelcomeSignIn)).FillWidth().Height(40).Radius(10).Border(1, border).Background(ui.Transparent).TextColor(fg).Disabled(m.OnSignIn == nil)
					if signIn.Clicked() && m.OnSignIn != nil {
						m.OnSignIn()
					}
					if ui.Button(c, i18n.T(i18n.KeyWelcomeAPIKey)).FillWidth().Height(40).Radius(10).Border(1, border).Background(ui.Transparent).TextColor(fg).Disabled(m.OnShowKeyForm == nil).Clicked() && m.OnShowKeyForm != nil {
						m.OnShowKeyForm()
					}
					if ui.Button(c, i18n.T(i18n.KeyWelcomeSkip)).FillWidth().Height(32).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(12)).Clicked() && m.OnSkip != nil {
						m.OnSkip()
					}
				}
				if m.Note != "" {
					note := muted
					if m.NoteError {
						note = failure
					}
					ui.Text(c, m.Note).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(note)
				}
				if m.Hint != "" {
					ui.Text(c, m.Hint).FillWidth().FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
				if m.SignInActive {
					if m.AuthURL != "" && ui.Button(c, i18n.T(i18n.KeyWelcomeCopyLink)).FillWidth().Height(28).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(12)).Clicked() && m.OnCopyLink != nil {
						m.OnCopyLink()
					}
					if ui.Button(c, i18n.T(i18n.KeyWelcomeCancel)).FillWidth().Height(28).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(12)).Disabled(m.OnCancelSignIn == nil).Clicked() && m.OnCancelSignIn != nil {
						m.OnCancelSignIn()
					}
				}
				if page == PageKey && ui.Button(c, i18n.T(i18n.KeyWelcomeKeyBack)).FillWidth().Height(28).BorderWidth(0).Background(ui.Transparent).TextColor(muted).FontSize(theme.InterfaceSize(12)).Clicked() && m.OnBack != nil {
					m.OnBack()
				}
			})
		})
	})
}

func keyBlank(key *string) bool {
	return key == nil || strings.TrimSpace(*key) == ""
}
