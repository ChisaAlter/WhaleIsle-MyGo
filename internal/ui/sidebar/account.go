package sidebar

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Account is the sidebar's bottom account seat, mirroring the original
// AccountMenu trigger (24px avatar, name, menu; 36x36 when the rail is
// collapsed). The zero value still renders the row and its menu so the entry
// points exist before the shell publishes a profile: OnOpen then falls back to
// the settings page, which already owns account details and sign-in.
type Account struct {
	Name      string // display name; empty shows the localized account title
	Detail    string // secondary line, e.g. the masked contact
	AvatarURL string // optional profile picture URL or data:image/... URI
	SignedIn  bool
	Busy      bool
	OnOpen    func() // account details; nil falls back to the settings entry
	OnSignOut func() // nil hides the sign-out item (e.g. signed out already)
}

var (
	accountRemoteCache struct {
		mu sync.Mutex
		c  *imgcache.Cache
	}
	accountBitmapCache struct {
		mu      sync.Mutex
		entries map[string]*ui.Bitmap
	}
)

// SetRemoteImageCache installs the shared http(s) image cache for sidebar avatars.
func SetRemoteImageCache(c *imgcache.Cache) {
	accountRemoteCache.mu.Lock()
	accountRemoteCache.c = c
	accountRemoteCache.mu.Unlock()
}

func remoteAvatarCache() *imgcache.Cache {
	accountRemoteCache.mu.Lock()
	defer accountRemoteCache.mu.Unlock()
	if accountRemoteCache.c == nil {
		accountRemoteCache.c = imgcache.New(imgcache.Dir(paths.UserDataDir()))
	}
	return accountRemoteCache.c
}

func decodeAvatarBytes(data []byte) *ui.Bitmap {
	if len(data) == 0 {
		return nil
	}
	if bmp, err := ui.DecodeBitmap(data); err == nil && bmp != nil {
		return bmp
	}
	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil && img != nil {
		return ui.NewBitmap(img)
	}
	return nil
}

func accountAvatarBitmap(c *ui.Context, rawURL string) *ui.Bitmap {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	accountBitmapCache.mu.Lock()
	if accountBitmapCache.entries != nil {
		if bmp, ok := accountBitmapCache.entries[rawURL]; ok {
			accountBitmapCache.mu.Unlock()
			return bmp
		}
	}
	accountBitmapCache.mu.Unlock()

	cacheBmp := func(bmp *ui.Bitmap) *ui.Bitmap {
		accountBitmapCache.mu.Lock()
		if accountBitmapCache.entries == nil {
			accountBitmapCache.entries = make(map[string]*ui.Bitmap)
		}
		accountBitmapCache.entries[rawURL] = bmp
		accountBitmapCache.mu.Unlock()
		return bmp
	}

	if strings.HasPrefix(rawURL, "data:image/") {
		head, encoded, ok := strings.Cut(rawURL, ",")
		if !ok || !strings.HasSuffix(head, ";base64") {
			return cacheBmp(nil)
		}
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return cacheBmp(nil)
		}
		return cacheBmp(decodeAvatarBytes(data))
	}
	if imgcache.IsHTTPURL(rawURL) {
		cache := remoteAvatarCache()
		if cache == nil {
			return nil
		}
		st, local, _ := cache.State(rawURL)
		if st == imgcache.StatusMissing {
			svc := c.Services()
			cache.Ensure(rawURL, func() { svc.Invalidate() })
			return nil
		}
		if st != imgcache.StatusReady || local == "" {
			return nil
		}
		data, err := os.ReadFile(local)
		if err != nil {
			return nil
		}
		return cacheBmp(decodeAvatarBytes(data))
	}
	data, err := os.ReadFile(rawURL)
	if err != nil {
		return nil
	}
	return cacheBmp(decodeAvatarBytes(data))
}

var accountPersonIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"><circle cx="8" cy="5.4" r="2.6"/><path d="M2.9 13.1c.7-2.4 2.7-3.7 5.1-3.7s4.4 1.3 5.1 3.7"/></svg>`))

func accountSignedOut(a Account) bool {
	return !a.SignedIn && strings.TrimSpace(a.Name) == "" && strings.TrimSpace(a.Detail) == ""
}

// accountLabel is the row's visible title. A profile-less seat is the merged
// settings/account trigger, whose installed label is launcherLabel
// (设置与账号 / Settings & account).
func accountLabel(a Account) string {
	if name := strings.TrimSpace(a.Name); name != "" {
		return name
	}
	if detail := strings.TrimSpace(a.Detail); detail != "" {
		return detail
	}
	return i18n.T(i18n.KeySettingsAccountSeat)
}

// accountInitial is the avatar monogram: the first non-space rune of the name.
func accountInitial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsSpace(r) {
			continue
		}
		return strings.ToUpper(string(r))
	}
	return ""
}

func accountMenu(a Account, open func()) func(*ui.Menu) {
	return func(menu *ui.Menu) {
		if open != nil && menu.Item(i18n.T(i18n.KeySettingsAccountTitle)).Chosen() {
			open()
		}
		if a.OnSignOut != nil && menu.Item(i18n.T(i18n.KeySettingsAccountSignOut)).Chosen() {
			a.OnSignOut()
		}
	}
}

// accountRow paints the account seat. compact is the collapsed rail's 36x36
// control, which keeps the same menu.
func accountRow(c *ui.Context, m Model, colors theme.Colors, muted ui.Color, compact bool) {
	a := m.Account
	open := a.OnOpen
	if open == nil {
		open = m.OnOpenSettings
	}
	if open == nil && a.OnSignOut == nil {
		return
	}
	label := accountLabel(a)
	signedOut := accountSignedOut(a)
	// The native trigger's accessible name stays 账号菜单 even when the signed-out
	// label reads launcherLabel.
	menuLabel := label
	if signedOut {
		menuLabel = i18n.T(i18n.KeySettingsAccountMenu)
	}
	height := float32(44) // AccountMenu.module.css .trigger
	if compact {
		height = 36 // AccountMenu.module.css [data-collapsed='true']
	} else if signedOut {
		// Installed AccountMenu trigger[data-signed-out='true']: 36 DIP tall with
		// 7/8 padding, replacing the avatar with a 16 DIP settings glyph whose
		// label is launcherLabel (设置与账号 / Settings & account).
		height = 36
	}
	// Installed AccountMenu retains the same trigger/ref when wide changes.
	// Keep its identity so auto-collapse preserves keyboard focus and menu state.
	row := ui.Button(c.Key("sidebar-account"), "").FillWidth().Height(height).Radius(12).
		BorderWidth(0).Background(ui.Transparent).Padding(6).
		Label(menuLabel).Tooltip(menuLabel).Disabled(a.Busy)
	if !compact {
		// The footer seat sits on the page's 14-DIP inset with the native 252-DIP
		// width, as the 账号菜单 button does in the installed page.
		maxW := float32(252)
		if sidebarUpdatePhase(m) != "" {
			maxW = 212
		}
		row.Margin(0, 2, 0, 2).MaxWidth(maxW)
	}
	if signedOut && !compact {
		// Installed AccountMenu trigger[data-signed-out='true']: 7/8 padding.
		row.Padding(7, 8, 7, 8)
	}
	if row.Hovered() {
		row.Background(chrome.Color(colors.Hover))
	}
	row.Menu(accountMenu(a, open))
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.Start).Children(func() {
			if signedOut {
				// The trigger swaps the avatar for a 16 DIP settings glyph.
				ui.Icon(c, settingsGear).Size(16, 16).TextColor(chrome.Color(colors.FG))
				if compact {
					return
				}
				ui.Text(c, label).Grow(1).FontSize(theme.InterfaceSize(14)).TextColor(chrome.Color(colors.FG)).SingleLine()
				return
			}
			avatar := ui.Box(c).Size(24, 24).Radius(12).Clip().Center()
			avatar.Background(chrome.Color(colors.Accent).Alpha(.12))
			avatar.Children(func() {
				if bmp := accountAvatarBitmap(c, a.AvatarURL); bmp != nil {
					ui.Image(c, bmp).Size(24, 24).Fit(ui.Cover).Radius(12)
					return
				}
				if initial := accountInitial(a.Name); initial != "" {
					ui.Text(c, initial).FontSize(theme.InterfaceSize(11)).FontWeight(600).TextColor(chrome.Color(colors.Accent))
					return
				}
				ui.Icon(c, accountPersonIcon).Size(14, 14).TextColor(muted)
			})
			if compact {
				return
			}
			ui.Text(c, label).MinWidth(0).Grow(1).
				FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).
				FontWeight(400).TextColor(chrome.Color(colors.FG)).SingleLine()
		})
	})
}
