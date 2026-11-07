// Package welcome draws the first-run welcome window: API key entry, account
// sign-in and skip. It ports the Electron welcome window semantics
// (reference/electron/src/main/welcome-window.js + renderer/welcome.html +
// welcome-backend.js) onto the MyGo native UI: the key form validates the key
// shape before saving, sign-in reuses the desktop account login RPC, and skip
// enters the ordinary launcher/boot flow without a credential.
//
// Geometry comes from reference/electron/src/renderer/welcome-layout.css, which
// the page's grid is the authority for: #root is a 6/16 padded flex column with
// an 8 gap, the .titlebar is an empty 36px drag strip, .welcome is a
// 40px / 1fr / 104px grid with 24 48 76 padding, .actions is a 240px column with
// 16 gaps of 44px buttons, and .actions .back is the 28px text action 16 below
// them. See docs/review-20261022/round153-welcome-metric-parity.md.
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

// welcome-layout.css metrics, in DIP.
const (
	rootPadV       = 6 // #root padding 6px 16px
	rootPadH       = 16
	rootGap        = 8  // #root gap 8px
	titleBarHeight = 36 // .titlebar flex 0 0 36px
	brandHeight    = 40 // .welcome grid row 1 / .brand height 40px
	gridPadTop     = 24 // .welcome padding 24px 48px 76px
	gridPadSide    = 48
	gridPadBottom  = 76
	actionsHeight  = 104 // .welcome grid row 3
	actionsWidth   = 240 // .actions width 240px
	actionsGap     = 16  // .actions gap 16px
	buttonHeight   = 44  // .actions button height 44px
	buttonRadius   = 10  // .actions button border-radius 10px
	backHeight     = 28  // .actions .back height 28px
	backMinWidth   = 58  // .actions .back min-width 58px
	backTop        = 16  // .actions .back top calc(100% + 16px)
	keyFormWidth   = 440 // .key-form width 440px
	keyFormGap     = 40  // .key-form gap 40px
	keyHeadGap     = 8   // .key-heading gap 8px
	inputHeight    = 48  // .key-field input height 48px
	inputRadius    = 12  // .key-field input border-radius 12px
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

	ui.Column(c).Fill().Background(bg).
		Padding(rootPadV, rootPadH, rootPadV, rootPadH).Gap(rootGap).Children(func() {
		// #root > .titlebar: an empty drag strip. The system caption buttons draw
		// the close button; the logo and the product name belong to .brand below.
		ui.Row(c).FillWidth().Height(titleBarHeight).DragWindow()

		ui.Column(c).Grow(1).FillWidth().
			Padding(gridPadTop, gridPadSide, gridPadBottom, gridPadSide).Children(func() {
			// Row 1: .brand, a 40px block (welcome-layout.css .brand).
			ui.Row(c).FillWidth().Height(brandHeight).AlignItems(ui.Center).Gap(8).Children(func() {
				if logo := chrome.FishLogo(); logo != nil {
					ui.Image(c, logo).Height(20).Width(27).Fit(ui.Contain)
				}
				ui.Text(c, i18n.T(i18n.KeyAppName)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			})

			// Row 2: the page content, centred in the free row.
			content := func() {
				if page == PageKey {
					keyForm(c, m, fg, muted, border, failure)
					return
				}
				// .tagline: the sentence inherits 24/40 weight 300 secondary; the
				// product name is the 26/40 weight 500 primary `em`.
				ui.Column(c).Gap(0).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyWelcomeTagline)).FontSize(theme.InterfaceSize(26)).
						FixedLineHeight(theme.InterfaceSize(40)).FontWeight(500).TextColor(fg)
					ui.Text(c, i18n.T(i18n.KeyWelcomeDescription)).FontSize(theme.InterfaceSize(24)).
						FixedLineHeight(theme.InterfaceSize(40)).FontWeight(300).TextColor(muted)
				})
			}
			ui.Column(c).Grow(1).FillWidth().Center().Children(content)

			// Row 3: .actions, fixed 104 tall: the 240px button column, then the
			// text action 16 below it.
			ui.Column(c).FillWidth().Height(actionsHeight).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Width(actionsWidth).MaxWidthPercent(100).Gap(actionsGap).Children(func() {
					switch page {
					case PageKey:
						saveLabel := i18n.T(i18n.KeyWelcomeKeySave)
						if m.Busy {
							saveLabel = i18n.T(i18n.KeyWelcomeKeySaving)
						}
						save := ui.PrimaryButton(c, saveLabel).FillWidth().Height(buttonHeight).Radius(buttonRadius).
							Disabled(m.Busy || m.OnSaveAPIKey == nil || keyBlank(m.APIKey))
						if save.Clicked() && m.OnSaveAPIKey != nil {
							m.OnSaveAPIKey()
						}
					default:
						if note := noteLine(m); note != "" {
							ui.Text(c, note).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(noteColor(m, muted, failure))
						}
						signIn := secondaryButton(c, i18n.T(i18n.KeyWelcomeSignIn), fg, border).
							Disabled(m.OnSignIn == nil)
						if signIn.Clicked() && m.OnSignIn != nil {
							m.OnSignIn()
						}
						keyBtn := secondaryButton(c, i18n.T(i18n.KeyWelcomeAPIKey), fg, border).
							Disabled(m.OnShowKeyForm == nil)
						if keyBtn.Clicked() && m.OnShowKeyForm != nil {
							m.OnShowKeyForm()
						}
					}
					if !m.SignInActive {
						// Placed last so the note reads above the actions, as the
						// reference page orders its status and controls.
					}
				})
				// .actions .back: a centred 28px text action under the buttons. A
				// running sign-in swaps it for its copy/cancel pair, keeping every
				// control inside the fixed 104px actions row.
				ui.Row(c).Height(backHeight).Margin(backTop, 0, 0, 0).Gap(8).Children(func() {
					if m.SignInActive {
						if m.AuthURL != "" && m.OnCopyLink != nil && textAction(c, i18n.T(i18n.KeyWelcomeCopyLink), muted).Clicked() {
							m.OnCopyLink()
						}
						if m.OnCancelSignIn != nil && textAction(c, i18n.T(i18n.KeyWelcomeCancel), muted).Clicked() {
							m.OnCancelSignIn()
						}
						return
					}
					label := i18n.T(i18n.KeyWelcomeSkip)
					onTap := m.OnSkip
					if page == PageKey {
						label = i18n.T(i18n.KeyWelcomeKeyBack)
						onTap = m.OnBack
					}
					if textAction(c, label, muted).Disabled(onTap == nil).Clicked() && onTap != nil {
						onTap()
					}
				})
			})
		})
	})
}

// keyForm is .key-form: a 440px centred column, 40 between the heading block and
// the field, with the 20/32 heading over the 16/22 line and a 48px 12-radius
// input; the form's save button lives in the .actions row, as the reference
// page does.
func keyForm(c *ui.Context, m Model, fg, muted, border ui.Color, failure ui.Color) {
	ui.Column(c).Width(keyFormWidth).MaxWidthPercent(100).Gap(keyFormGap).Children(func() {
		ui.Column(c).FillWidth().Gap(keyHeadGap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyWelcomeKeyTitle)).FontSize(theme.InterfaceSize(20)).
				FixedLineHeight(theme.InterfaceSize(32)).TextColor(fg)
			ui.Text(c, i18n.T(i18n.KeyWelcomeKeyDescription)).FontSize(theme.InterfaceSize(16)).
				FixedLineHeight(theme.InterfaceSize(22)).FontWeight(300).LetterSpacing(0.32).TextColor(muted).
				MaxLines(3)
		})
		if m.APIKey != nil {
			ui.TextInput(c, m.APIKey).FillWidth().Height(inputHeight).Radius(inputRadius).Border(1, border).
				Placeholder(i18n.T(i18n.KeyWelcomeKeyPlaceholder)).Label(i18n.T(i18n.KeyWelcomeAPIKey))
		}
		if note := noteLine(m); note != "" {
			ui.Text(c, note).FillWidth().FontSize(theme.InterfaceSize(14)).
				FixedLineHeight(theme.InterfaceSize(22)).TextColor(noteColor(m, muted, failure))
		}
		if m.Hint != "" {
			ui.Text(c, m.Hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
	})
}

func noteLine(m Model) string {
	return welcomeNoticeCopy(m.Note)
}

var welcomeCatalogZH = map[string]string{
	"welcomeTitle":           "使用你自己的 API Key",
	"welcomeBody":            "当前尚未设置默认模型。请选择一个内置服务商并填写对应的 API Key，应用会用它校验并初始化默认模型。\n你也可以先跳过，直接进入应用后再前往设置配置。",
	"welcomeProviderLabel":   "选择服务商",
	"welcomeKeyHint":         "应用不会展示已保存的密钥原文，保存前会先做一次可用性校验。",
	"welcomeSave":            "保存并进入应用",
	"welcomeSaving":          "校验并保存中…",
	"welcomeSkip":            "稍后配置，直接进入",
	"welcomeErrorEmptyKey":   "请先输入 API Key。",
	"welcomeErrorNoProvider": "未找到可用的内置服务商。",
	"welcomeSaved":           "已保存 API Key，正在进入应用…",
}

var welcomeCatalogEN = map[string]string{
	"welcomeTitle":           "Use your own API key",
	"welcomeBody":            "No default model is configured yet. Choose a built-in provider and enter an API key to validate it and set up your default model.\nYou can also skip this for now and configure it later in Settings.",
	"welcomeProviderLabel":   "Choose a provider",
	"welcomeKeyHint":         "Saved API keys are never shown in plain text. The key will be validated before it is saved.",
	"welcomeSave":            "Save & continue",
	"welcomeSaving":          "Validating and saving…",
	"welcomeSkip":            "Set up later",
	"welcomeErrorEmptyKey":   "Enter an API key first.",
	"welcomeErrorNoProvider": "No built-in provider is available.",
	"welcomeSaved":           "API key saved. Opening the app…",
}

func welcomeNoticeCopy(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	dict := welcomeCatalogZH
	if i18n.Locale() == i18n.LocaleEN {
		dict = welcomeCatalogEN
	}
	if text, ok := dict[trimmed]; ok {
		return text
	}
	return raw
}

func noteColor(m Model, muted, failure ui.Color) ui.Color {
	if m.NoteError {
		return failure
	}
	return muted
}

// secondaryButton is .actions button + .secondary: 44px tall, 10 radius, the l2
// border and the elevated fill.
func secondaryButton(c *ui.Context, label string, fg, border ui.Color) ui.Element {
	return ui.Button(c, label).FillWidth().Height(buttonHeight).Radius(buttonRadius).
		Border(1, border).Background(ui.Transparent).TextColor(fg)
}

// textAction is .actions .back: a borderless text button, 28 tall.
func textAction(c *ui.Context, label string, muted ui.Color) ui.Element {
	return ui.Button(c, label).MinWidth(backMinWidth).Height(backHeight).Padding(4, 10, 4, 10).
		Radius(24).BorderWidth(0).Background(ui.Transparent).TextColor(muted).
		FontSize(theme.InterfaceSize(12))
}

func keyBlank(key *string) bool {
	return key == nil || strings.TrimSpace(*key) == ""
}
