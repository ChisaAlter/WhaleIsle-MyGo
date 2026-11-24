package chat

import (
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/webview"
	"whaleisle/internal/openui/card"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	nativeui "whaleisle/internal/ui/openui"
	"whaleisle/internal/ui/theme"
)

// Generative UI in the conversation renders natively (internal/ui/openui):
// each fence and render_ui result is a card.Card kept in nativeCards by its
// block identity, so a card scrolled out of the virtualized transcript keeps
// its parsed program and state. The web viewer runs only in the A2
// workbench (openui.go), after a click.

var (
	nativeCards     = card.NewCache(96)
	nativeResources = openui.NewResourceAPI()
)

// cardHost executes a card's host effects for the conversation that shows
// it. The UI updates its owner every frame; the card calls it from its own
// goroutine after a real click.
type cardHost struct {
	mu         sync.Mutex
	capture    func() (*session.ComposerTarget, func() bool, error)
	panelStore *whalePanelState
	sourceID   string
	order      [3]float64
}

var errCardOwner = errors.New("OpenUI block owner is no longer active")

// Submit sends text through the queue of the conversation that owns the
// card, after checking that the owner is still current.
func (h *cardHost) Submit(ctx context.Context, text string) error {
	h.mu.Lock()
	capture := h.capture
	h.mu.Unlock()
	if capture == nil {
		return errCardOwner
	}
	target, valid, err := capture()
	if err != nil {
		return err
	}
	if valid == nil || !valid() {
		return errCardOwner
	}
	return target.Submit(ctx, text, nil, 0, "")
}

// onMain runs fn on the UI thread and returns its error.
func onMain(fn func() error) error {
	err := errors.New("the application UI loop is unavailable")
	mygo.RunOnMain(func() { err = fn() })
	return err
}

// OpenURL uses the product's HTTP(S)-only external-open route.
func (h *cardHost) OpenURL(u string) error {
	return onMain(func() error { return webview.OpenExternal(u) })
}

// Copy writes the clipboard through MyGo's error-returning API.
func (h *cardHost) Copy(text string) error {
	return onMain(func() error { return mygo.Clipboard.Write(transfer.TextData(text)) })
}

// Panel publishes a whale-ui-panels-v2 operation under the card's own
// conversation position.
func (h *cardHost) Panel(op openui.PanelOperation) error {
	h.mu.Lock()
	ps, source, order := h.panelStore, h.sourceID, h.order
	h.mu.Unlock()
	if ps == nil {
		return nil
	}
	raw, err := json.Marshal(op)
	if err != nil {
		return err
	}
	return ps.Publish(source, order, raw)
}

func seedOf(id string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(id))
	return h.Sum64()
}

func openUIFence(c *ui.Context, m Model, msg session.Message, language, program string, offset int) bool {
	if !openui.ValidLanguage(language) || msg.Role != "assistant" || m.FeedbackStore == nil {
		return false
	}
	messageID := msg.SourceMessageID
	if messageID == "" {
		messageID = msg.ID
	}
	return openUIBlockView(c, m, msg, language, program, offset, messageID, false, nil)
}

type openUIChrome struct{ source bool }

// openUIBlockView renders one generative UI block natively.
func openUIBlockView(c *ui.Context, m Model, msg session.Message, language, program string, offset int, messageID string, panel bool, capture func() (*session.ComposerTarget, func() bool, error)) bool {
	id := openui.Identity(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, offset)
	sourceID := openui.Identity("", m.Snapshot.SelectedID, messageID, offset)
	scope := ""
	if m.FeedbackStore != nil {
		scope = m.FeedbackStore.WhaleUIStorageScope()
	}
	storageID := openui.Identity(scope, m.Snapshot.SelectedID, sourceID, 0)
	if capture == nil && m.FeedbackStore != nil {
		store, binding, sid := m.FeedbackStore, m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID
		capture = func() (*session.ComposerTarget, func() bool, error) {
			return store.CaptureOpenUI(binding, sid, messageID, program)
		}
	}
	invalidate := c.Services().Invalidate
	host := &cardHost{}
	k := nativeCards.Get(id+"/"+storageID, func() *card.Card {
		return card.New(card.Config{
			Locale:     string(i18n.Locale()),
			StateDir:   filepath.Join(paths.UserDataDir(), "openui-state-v1"),
			StateID:    storageID,
			Host:       host,
			Invalidate: invalidate,
			Seed:       seedOf(id),
			Media:      media.Default(),
			Resources:  nativeResources,
		})
	})
	if h, ok := k.Host().(*cardHost); ok {
		host = h
	}
	host.mu.Lock()
	host.capture = capture
	host.panelStore = whalePanelForUI(m, invalidate)
	host.sourceID = sourceID
	host.order = [3]float64{msg.Seq, 0, float64(offset)}
	host.mu.Unlock()
	k.SetSource(language, program, msg.Streaming)

	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	dark := darkSurface(fg)
	box := ui.Column(c.Key("openui:" + id)).FillWidth().MinWidth(0).Gap(design.Sp(2))
	st := ui.Local(box, "chrome", func() openUIChrome { return openUIChrome{} })
	box.Children(func() {
		md := transcriptMarkdownStyle(fg, muted, chrome.Color(colors.Field), chrome.Color(colors.Accent), chrome.Color(colors.Hover), theme.CurrentPreferences().Chat.Size, dark, chrome.Color(colors.Secondary))
		openCompat := func() {
			appearance := "light"
			if dark {
				appearance = "dark"
			}
			openOpenUICompat(openUICompatRequest{
				instanceID: id, storeKey: storageID, language: language, program: program, locale: string(i18n.Locale()),
				appearance: appearance, variables: whaleThemeVariables(m), panel: panel, panelStore: whalePanelForUI(m, invalidate),
				sourceID: sourceID, sourceOrder: [3]float64{msg.Seq, 0, float64(offset)}, parent: m.OpenUIWindow,
				capture: capture, invalidate: invalidate,
			})
		}
		width := float32(0)
		if m.ContentWidth > 0 {
			width = m.ContentWidth
		}
		nativeui.View(c, nativeui.Options{Runtime: k, Markdown: md, OpenURL: func(u string) { _ = webview.OpenExternal(u) }, OpenCompat: openCompat, Width: width})
		ui.Row(c).Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
			if coss.Button(c, i18n.T(i18n.KeyChatIsolatedSource), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Pressed: st.source}).Clicked() {
				st.source = !st.source
			}
			if coss.Button(c, nativeui.OpenCompatLabel(), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Disabled: msg.Streaming}).Clicked() {
				openCompat()
			}
		})
		if st.source {
			ui.Text(c, program).Selectable().FontSize(design.TextXS.Size)
		}
		openUICompatStatus(c, m, id)
	})
	return true
}
