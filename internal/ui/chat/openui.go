package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The A2 compatibility workbench for generative UI. The daily chat path
// renders Whale UI natively (openui.go); the web viewer runs only here, in
// its own helper window, after the user clicks "open in compatibility
// workbench" on a card or one of its placeholders (HtmlView, YouTube, a
// component without a native view). One workbench window is alive at a
// time: opening another, closing it, or losing the card's owner destroys
// the helper (ARCHITECTURE.md §5.2, §7).

const openUIStreamCoalesceDelay = 40 * time.Millisecond

type openUISession interface {
	UpdateOpenUI(r openui.Render) error
	SetTheme(appearance string, variables map[string]string) error
	Events() <-chan json.RawMessage
	ReplyOpenUIAPI(response openui.APIResponse) error
	Done() <-chan struct{}
	Close() error
}

// openUIBlock is one workbench window and the conversation block it shows.
type openUIBlock struct {
	mu                   sync.Mutex
	program              string
	appearance           string
	language, locale     string
	variables            map[string]string
	panel                bool
	panelStore           *whalePanelState
	sourceID             string
	sourceOrder          [3]float64
	streaming            bool
	wake                 chan struct{}
	loading              bool
	err                  string
	action               *openui.Event
	handle               openUISession
	cancel               context.CancelFunc
	target               *session.ComposerTarget
	valid                func() bool
	parent               *mygo.Window
	revision             uint64
	instanceID, storeKey string
}

// fail records why the workbench stopped and reports whether the painted
// line changed. The card paints the cause and the next step; the raw error
// goes to the log.
func (b *openUIBlock) fail(err error) bool {
	previous := b.err
	if err != nil {
		log.Printf("openui workbench: %v", err)
	}
	b.err = errcopy.Sentence(err)
	return b.err != previous
}

func (b *openUIBlock) notifyWake() {
	if b == nil {
		return
	}
	b.mu.Lock()
	ch := b.wake
	b.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

func cloneWhaleVariables(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// openUICompat is the single live workbench window.
var openUICompat struct {
	sync.Mutex
	block *openUIBlock
}

// openUICompatFor returns the live workbench of a card, if it is open.
func openUICompatFor(instanceID string) *openUIBlock {
	openUICompat.Lock()
	defer openUICompat.Unlock()
	if b := openUICompat.block; b != nil && b.instanceID == instanceID {
		return b
	}
	return nil
}

// openUICompatRequest is everything a workbench needs from the card that
// asked for it.
type openUICompatRequest struct {
	instanceID, storeKey      string
	language, program, locale string
	appearance                string
	variables                 map[string]string
	panel                     bool
	panelStore                *whalePanelState
	sourceID                  string
	sourceOrder               [3]float64
	parent                    *mygo.Window
	capture                   func() (*session.ComposerTarget, func() bool, error)
	invalidate                func()
}

// openOpenUICompat opens the workbench for a card after a click. It
// replaces any other workbench window; a second click on the same card
// keeps the window it has.
func openOpenUICompat(req openUICompatRequest) {
	openUICompat.Lock()
	if b := openUICompat.block; b != nil {
		if b.instanceID == req.instanceID {
			b.mu.Lock()
			alive := b.loading || b.handle != nil
			b.mu.Unlock()
			if alive {
				openUICompat.Unlock()
				return
			}
		}
		b.mu.Lock()
		if b.cancel != nil {
			b.cancel()
		}
		b.mu.Unlock()
	}
	b := &openUIBlock{instanceID: req.instanceID, storeKey: req.storeKey, program: req.program, language: req.language, locale: req.locale,
		appearance: req.appearance, variables: req.variables, panel: req.panel, panelStore: req.panelStore, sourceID: req.sourceID,
		sourceOrder: req.sourceOrder, parent: req.parent, revision: 1, wake: make(chan struct{}, 1), loading: true}
	openUICompat.block = b
	openUICompat.Unlock()

	target, valid, err := req.capture()
	if err != nil {
		b.mu.Lock()
		b.loading = false
		b.fail(err)
		b.mu.Unlock()
		req.invalidate()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.cancel, b.target, b.valid = cancel, target, valid
	b.mu.Unlock()
	go runOpenUI(ctx, b, req.instanceID, req.appearance, req.invalidate)
}

// closeOpenUICompat closes the workbench of a card.
func closeOpenUICompat(instanceID string) {
	if b := openUICompatFor(instanceID); b != nil {
		b.mu.Lock()
		if b.cancel != nil {
			b.cancel()
		}
		b.mu.Unlock()
	}
}

func runOpenUI(ctx context.Context, b *openUIBlock, id, appearance string, invalidate func()) {
	defer invalidate()
	b.mu.Lock()
	stop := b.cancel
	b.mu.Unlock()
	if stop != nil {
		defer stop()
	}
	finish := func(err error) {
		b.mu.Lock()
		b.loading = false
		b.handle = nil
		if err != nil {
			b.fail(err)
		}
		b.mu.Unlock()
	}
	explicitViewer := os.Getenv("WHALEISLE_OPENUI_VIEWER")
	viewer, err := openui.LoadViewer(explicitViewer)
	if err != nil {
		finish(err)
		return
	}
	root := filepath.Join(paths.UserDataDir(), "openui-state-v1")
	b.mu.Lock()
	storageID := b.storeKey
	b.mu.Unlock()
	state, err := openui.ReadState(root, storageID)
	if err != nil {
		finish(err)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		finish(err)
		return
	}
	b.mu.Lock()
	valid := b.valid
	program := b.program
	streaming := b.streaming
	revision := b.revision
	language, locale, variables, panel := b.language, b.locale, b.variables, b.panel
	b.mu.Unlock()
	if valid == nil || !valid() {
		finish(fmt.Errorf("OpenUI owner changed"))
		return
	}
	hostActions := explicitViewer == "" && openui.IsOwnedViewer(viewer) && (runtime.GOOS == "windows" || runtime.GOOS == "darwin")
	handle, err := previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: viewer, Width: 900, Appearance: appearance, Inline: false, OpenUI: &openui.Render{InstanceID: id, Revision: revision, Program: program, Language: language, Locale: locale, Variables: variables, Panel: panel, Streaming: streaming, State: state, HostActions: hostActions}})
	if err != nil {
		finish(err)
		return
	}
	defer handle.Close()
	b.mu.Lock()
	b.handle = handle
	b.loading = false
	b.mu.Unlock()
	invalidate()
	runOpenUILoop(ctx, b, handle, id, viewer, root, storageID, program, appearance, cloneWhaleVariables(variables), panel, streaming, valid, invalidate, finish)
}

func runOpenUILoop(
	ctx context.Context,
	b *openUIBlock,
	handle openUISession,
	id, viewer, root, storageID, program, appearance string,
	variables map[string]string,
	panel, streaming bool,
	valid func() bool,
	invalidate func(),
	finish func(error),
) {
	if b.wake == nil {
		b.mu.Lock()
		if b.wake == nil {
			b.wake = make(chan struct{}, 1)
		}
		b.mu.Unlock()
	}
	downloads := newWhaleDownloads()
	resources := openui.NewResourceAPI()
	b.mu.Lock()
	revision := b.revision
	lastVariables := cloneWhaleVariables(b.variables)
	if len(lastVariables) == 0 {
		lastVariables = cloneWhaleVariables(variables)
	}
	lastAppearance := appearance
	b.mu.Unlock()

	var (
		streamTimer    *time.Timer
		lastStreamPush time.Time
	)
	stopTimer := func(tp **time.Timer) {
		if *tp == nil {
			return
		}
		if !(*tp).Stop() {
			select {
			case <-(*tp).C:
			default:
			}
		}
		*tp = nil
	}
	defer stopTimer(&streamTimer)

	sync := func() bool {
		if valid != nil && !valid() {
			finish(fmt.Errorf("OpenUI owner changed; renderer closed"))
			return false
		}
		b.mu.Lock()
		latest, live, currentAppearance, currentRevision := b.program, b.streaming, b.appearance, b.revision
		currentVariables, currentLanguage, currentLocale := b.variables, b.language, b.locale
		b.mu.Unlock()
		if latest != program || live != streaming || currentRevision != revision {
			if live && !lastStreamPush.IsZero() && time.Since(lastStreamPush) < openUIStreamCoalesceDelay {
				if streamTimer == nil {
					streamTimer = time.NewTimer(openUIStreamCoalesceDelay - time.Since(lastStreamPush))
				}
			} else {
				stopTimer(&streamTimer)
				if e := handle.UpdateOpenUI(openui.Render{InstanceID: id, Revision: currentRevision, Program: latest, Language: currentLanguage, Locale: currentLocale, Variables: currentVariables, Panel: panel, Streaming: live, State: json.RawMessage(`{}`)}); e != nil {
					finish(e)
					return false
				}
				downloads = newWhaleDownloads()
				program, streaming, revision = latest, live, currentRevision
				if live {
					lastStreamPush = time.Now()
				} else {
					lastStreamPush = time.Time{}
				}
			}
		}
		if currentAppearance != lastAppearance || !equalWhaleVariables(currentVariables, lastVariables) {
			if e := handle.SetTheme(currentAppearance, whalePreviewVariables(currentVariables)); e != nil {
				finish(e)
				return false
			}
			lastAppearance = currentAppearance
			lastVariables = cloneWhaleVariables(currentVariables)
		}
		return true
	}

	if !sync() {
		return
	}

	for {
		var streamC <-chan time.Time
		if streamTimer != nil {
			streamC = streamTimer.C
		}
		select {
		case <-ctx.Done():
			finish(nil)
			return
		case <-handle.Done():
			finish(nil)
			return
		case <-b.wake:
			if !sync() {
				return
			}
		case <-streamC:
			streamTimer = nil
			if !sync() {
				return
			}
		case raw, ok := <-handle.Events():
			if !ok {
				finish(nil)
				return
			}
			if valid != nil && !valid() {
				finish(fmt.Errorf("OpenUI owner changed"))
				return
			}
			event, e := openui.ValidateEvent(json.RawMessage(raw), id)
			if e != nil {
				finish(e)
				return
			}
			if event.Revision > 0 && event.Type != "state" {
				b.mu.Lock()
				currentRevision := b.revision
				b.mu.Unlock()
				if event.Revision != currentRevision {
					continue
				}
			}
			switch event.Type {
			case "download-part":
				if err := downloads.Add(event); err != nil {
					_ = handle.ReplyOpenUIAPI(whaleActionResponse(event, err))
					downloads = newWhaleDownloads()
				}
			case "panel":
				b.mu.Lock()
				ps, source, order := b.panelStore, b.sourceID, b.sourceOrder
				b.mu.Unlock()
				if ps != nil {
					if e := ps.Publish(source, order, event.Operation); e != nil {
						b.mu.Lock()
						b.fail(e)
						b.mu.Unlock()
					}
				}
			case "title":
				// Child title does not replace the native owner or source identity.
			case "media":
				// The workbench window shows no private native media seat.
				_ = handle.ReplyOpenUIAPI(whaleActionResponse(event, fmt.Errorf("OpenUI private media is unavailable for this surface")))
			case "clipboard", "open-url", "download":
				if event.Type == "download" {
					if e := downloads.Complete(&event); e != nil {
						_ = handle.ReplyOpenUIAPI(whaleActionResponse(event, e))
						continue
					}
					downloads = newWhaleDownloads()
				}
				if ps, ok := any(handle).(*previewprocess.Session); ok {
					response := executeOpenUINativeAction(ctx, b, ps, valid, event)
					if (valid == nil || valid()) && ctx.Err() == nil {
						if replyErr := handle.ReplyOpenUIAPI(response); replyErr != nil {
							b.mu.Lock()
							b.fail(replyErr)
							b.mu.Unlock()
						}
					}
				}
			case "api-request":
				go func(request openui.Event) {
					if (valid != nil && !valid()) || ctx.Err() != nil {
						return
					}
					response := resources.Handle(ctx, request)
					if (valid != nil && !valid()) || ctx.Err() != nil {
						return
					}
					if replyErr := handle.ReplyOpenUIAPI(response); replyErr != nil && ctx.Err() == nil {
						b.mu.Lock()
						b.fail(replyErr)
						b.mu.Unlock()
						invalidate()
					}
				}(event)
			case "state":
				if e = openui.SaveState(root, storageID, event.State); e != nil {
					b.mu.Lock()
					b.fail(e)
					b.mu.Unlock()
				}
			case "assistant":
				if openui.IsWhaleViewer(viewer) && event.RequestID != "" {
					b.mu.Lock()
					target := b.target
					sourceRevision := b.revision
					b.mu.Unlock()
					go func(action openui.Event) {
						b.mu.Lock()
						current := b.revision == sourceRevision
						b.mu.Unlock()
						var err error
						if target == nil || !current || (valid != nil && !valid()) || ctx.Err() != nil {
							err = fmt.Errorf("Whale UI action owner changed")
						} else {
							err = target.Submit(ctx, action.ActionText(), nil, 0, "")
						}
						if (valid == nil || valid()) && ctx.Err() == nil {
							if e := handle.ReplyOpenUIAPI(whaleActionResponse(action, err)); e != nil {
								b.mu.Lock()
								b.fail(e)
								b.mu.Unlock()
							}
						}
						invalidate()
					}(event)
				} else {
					b.mu.Lock()
					b.action = &event
					b.mu.Unlock()
				}
			case "error":
				b.mu.Lock()
				b.fail(fmt.Errorf("%s", event.Message))
				b.mu.Unlock()
			case "resolved":
				b.mu.Lock()
				if whaleResolvedCurrent(event, b.program, b.revision) {
					b.err = ""
				}
				b.mu.Unlock()
			}
			invalidate()
		}
	}
}

// openUICompatStatus paints the workbench state of a card: its failure,
// a pending action from an unowned viewer, and a close button while open.
func openUICompatStatus(c *ui.Context, m Model, instanceID string) {
	b := openUICompatFor(instanceID)
	if b == nil {
		return
	}
	b.mu.Lock()
	errText, action, open := b.err, b.action, b.loading || b.handle != nil
	b.mu.Unlock()
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if errText != "" {
		ui.Text(c, errText).Selectable().FontSize(11)
	}
	if open && stdButton(c, palette, i18n.T(i18n.KeyChatOpenUIClose), buttonGhost, false).Clicked() {
		closeOpenUICompat(instanceID)
	}
	if action == nil {
		return
	}
	ui.Text(c, i18n.T(i18n.KeyChatIsolatedGesture)).FontSize(11)
	ui.Text(c, action.ActionText()).Selectable().FontSize(12)
	if stdButton(c, palette, i18n.T(i18n.KeyChatIsolatedSend), buttonPrimary, false).Clicked() {
		b.mu.Lock()
		current := b.action
		target, valid, handle := b.target, b.valid, b.handle
		b.action = nil
		b.mu.Unlock()
		if current != nil && target != nil && valid != nil && valid() {
			invalidate := c.Services().Invalidate
			go func() {
				err := target.Submit(context.Background(), current.ActionText(), nil, 0, "")
				if current.RequestID != "" && handle != nil {
					_ = handle.ReplyOpenUIAPI(whaleActionResponse(*current, err))
				}
				if err != nil {
					b.mu.Lock()
					b.fail(err)
					b.mu.Unlock()
				}
				invalidate()
			}()
		}
	}
	if stdButton(c, palette, i18n.T(i18n.KeyChatIsolatedIgnore), buttonGhost, false).Clicked() {
		b.mu.Lock()
		current, handle := b.action, b.handle
		b.action = nil
		b.mu.Unlock()
		if current != nil && current.RequestID != "" && handle != nil {
			go func() {
				_ = handle.ReplyOpenUIAPI(whaleActionResponse(*current, fmt.Errorf("The interaction was dismissed")))
			}()
		}
	}
}
