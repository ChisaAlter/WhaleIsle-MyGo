package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const (
	defaultOpenUIOffscreenReclaimTTL = 3 * time.Second
	openUIStreamCoalesceDelay        = 40 * time.Millisecond
)

type openUISession interface {
	ContentHeight() int
	RenderPixels(c *ui.Context, height float32, appearance string, variables map[string]string) bool
	PresentSurface(parent uintptr, x, y, w, h, clipX, clipY, clipW, clipH int) error
	UpdateOpenUI(r openui.Render) error
	SetTheme(appearance string, variables map[string]string) error
	Events() <-chan json.RawMessage
	ReplyOpenUIAPI(response openui.APIResponse) error
	Done() <-chan struct{}
	Close() error
}

var (
	chatViewSeq              atomic.Uint64
	activeInlineOpenUIMu     sync.Mutex
	activeInlineOpenUIBlocks = map[*openUIBlock]struct{}{}
)

func currentChatViewSeq() uint64 {
	return chatViewSeq.Load()
}

func registerInlineOpenUIBlock(b *openUIBlock) {
	if b == nil {
		return
	}
	activeInlineOpenUIMu.Lock()
	activeInlineOpenUIBlocks[b] = struct{}{}
	activeInlineOpenUIMu.Unlock()
}

func unregisterInlineOpenUIBlock(b *openUIBlock) {
	if b == nil {
		return
	}
	activeInlineOpenUIMu.Lock()
	delete(activeInlineOpenUIBlocks, b)
	activeInlineOpenUIMu.Unlock()
}

func checkOpenUIListEvictions(seq uint64) {
	activeInlineOpenUIMu.Lock()
	if len(activeInlineOpenUIBlocks) == 0 {
		activeInlineOpenUIMu.Unlock()
		return
	}
	blocks := make([]*openUIBlock, 0, len(activeInlineOpenUIBlocks))
	for b := range activeInlineOpenUIBlocks {
		blocks = append(blocks, b)
	}
	activeInlineOpenUIMu.Unlock()

	now := time.Now()
	for _, b := range blocks {
		b.mu.Lock()
		evictNow := b.inline && b.handle != nil && b.lastViewSeq != 0 && b.lastViewSeq != seq && !b.evicted
		if evictNow {
			b.evicted = true
			if b.offscreenSince.IsZero() {
				b.offscreenSince = now
			}
		}
		b.mu.Unlock()
		if evictNow {
			b.notifyWake()
		}
	}
}

type openUIBlock struct {
	mu                           sync.Mutex
	program                      string
	appearance                   string
	language, locale             string
	variables                    map[string]string
	panel                        bool
	panelStore                   *whalePanelState
	sourceID                     string
	sourceOrder                  [3]float64
	capture                      func() (*session.ComposerTarget, func() bool, error)
	streaming, inline, attempted bool
	visible, evicted             bool
	lastPaint                    time.Time
	offscreenSince               time.Time
	lastViewSeq                  uint64
	offscreenTTL                 time.Duration
	wake                         chan struct{}
	loading                      bool
	err                          string
	action                       *openui.Event
	handle                       openUISession
	cancel                       context.CancelFunc
	target                       *session.ComposerTarget
	valid                        func() bool
	parent                       *mygo.Window
	revision                     uint64
	media                        *openUIMedia
}

func (b *openUIBlock) effectiveOffscreenTTL() time.Duration {
	if b != nil && b.offscreenTTL > 0 {
		return b.offscreenTTL
	}
	return defaultOpenUIOffscreenReclaimTTL
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

func (b *openUIBlock) isOffscreen() bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	off := b.inline && (!b.visible || b.evicted)
	b.mu.Unlock()
	return off
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

func openUIBlockView(c *ui.Context, m Model, msg session.Message, language, program string, offset int, messageID string, panel bool, capture func() (*session.ComposerTarget, func() bool, error)) bool {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	appearance := "light"
	th := theme.Resolve(m.ThemeID)
	if th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) {
		appearance = "dark"
	}
	id := openui.Identity(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, offset)
	box := ui.Column(c.Key("openui:" + id)).FillWidth().Padding(8).Gap(6)
	b := *ui.Local(box, "instance", func() *openUIBlock {
		return &openUIBlock{
			program:   program,
			streaming: msg.Streaming,
			revision:  1,
			visible:   true,
			wake:      make(chan struct{}, 1),
		}
	})
	b.mu.Lock()
	if b.wake == nil {
		b.wake = make(chan struct{}, 1)
	}
	nextVariables := whaleThemeVariables(m)
	sourceChanged := b.program != program || b.streaming != msg.Streaming
	appearanceChanged := b.language != language || b.locale != string(i18n.Locale()) || !equalWhaleVariables(b.variables, nextVariables) || b.appearance != appearance
	if sourceChanged {
		b.action = nil
		b.revision++
	}
	if appearanceChanged && !sourceChanged {
		b.revision++
	}
	wasEvicted := b.evicted
	b.evicted = false
	b.lastViewSeq = currentChatViewSeq()
	if wasEvicted && b.visible {
		b.offscreenSince = time.Time{}
	}
	b.program = program
	b.streaming = msg.Streaming
	b.appearance = appearance
	b.parent = m.OpenUIWindow
	b.language = language
	b.locale = string(i18n.Locale())
	b.variables = nextVariables
	b.panel = panel
	b.capture = capture
	b.panelStore = whalePanelForUI(m, c.Services().Invalidate)
	b.sourceID = openui.Identity("", m.Snapshot.SelectedID, messageID, offset)
	b.sourceOrder = [3]float64{msg.Seq, 0, float64(offset)}
	media, revision := b.media, b.revision
	loading, errText, action, handle, inline := b.loading, b.err, b.action, b.handle, b.inline
	b.mu.Unlock()
	if handle != nil && (sourceChanged || appearanceChanged || wasEvicted) {
		b.notifyWake()
	}
	if sourceChanged && media != nil {
		media.Retire(revision)
	}
	invalidate := c.Services().Invalidate
	start := func(inline bool) {
		b.mu.Lock()
		if b.loading || b.handle != nil {
			b.mu.Unlock()
			return
		}
		if b.wake == nil {
			b.wake = make(chan struct{}, 1)
		}
		b.attempted = true
		b.mu.Unlock()
		var target *session.ComposerTarget
		var valid func() bool
		var err error
		if capture != nil {
			target, valid, err = capture()
		} else {
			target, valid, err = m.FeedbackStore.CaptureOpenUI(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, program)
		}
		if err != nil {
			b.mu.Lock()
			b.err = err.Error()
			b.mu.Unlock()
			invalidate()
			return
		}
		ctx, cancel := newOpenUIContext()
		b.mu.Lock()
		b.loading = true
		b.err = ""
		b.cancel = cancel
		b.target = target
		b.valid = valid
		b.inline = inline
		b.visible = true
		b.evicted = false
		b.offscreenSince = time.Time{}
		b.lastViewSeq = currentChatViewSeq()
		b.lastPaint = time.Now()
		b.mu.Unlock()
		go runOpenUI(ctx, b, id, program, appearance, invalidate)
	}
	box.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyChatIsolated)).FontSize(12)
		if stdButton(c, palette, i18n.T(i18n.KeyChatOpenUIRetry), buttonGhost, loading || handle != nil).Clicked() {
			start(true)
		}
		if stdButton(c, palette, i18n.T(i18n.KeyChatIsolatedOpen), buttonPrimary, msg.Streaming || loading || handle != nil).Clicked() {
			start(false)
		}
		if handle != nil && stdButton(c, palette, i18n.T(i18n.KeyChatOpenUIClose), buttonGhost, false).Clicked() {
			b.mu.Lock()
			if b.cancel != nil {
				b.cancel()
			}
			b.mu.Unlock()
		}
		if errText != "" {
			ui.Text(c, errText).Selectable().FontSize(11)
		}
		height := float32(360)
		if handle != nil {
			height = float32(max(80, min(2000, handle.ContentHeight())))
		}
		if handle == nil || inline {
			seat := ui.Column(c).FillWidth().Height(height)
			seat.DrawOver(func(p *ui.Painter, r ui.Rect) {
				clip := p.NativeSurfaceClip()
				b.mu.Lock()
				media := b.media
				b.mu.Unlock()
				if media != nil {
					picking := false
					if n := inspectorChatNavigation(m); n != nil {
						picking = n.Picking
					}
					media.Paint(r, clip, m.OpenUIMediaOcclusions, picking)
				}
				visible := childImageVisible(r, clip)
				b.mu.Lock()
				wasVisible := b.visible
				if visible {
					b.lastPaint = time.Now()
					b.visible = true
					b.evicted = false
					b.offscreenSince = time.Time{}
				} else if b.visible {
					b.visible = false
					if b.offscreenSince.IsZero() {
						b.offscreenSince = time.Now()
					}
				}
				attempted := b.attempted
				b.mu.Unlock()
				if !visible {
					if wasVisible {
						b.notifyWake()
					}
					return
				}
				if !wasVisible {
					b.notifyWake()
				}
				if !attempted && strings.TrimSpace(program) != "" && m.OpenUIParent != 0 {
					start(true)
				}
			})
			seat.Children(func() {
				if handle != nil {
					if !handle.RenderPixels(c, height, appearance, whalePreviewVariables(nextVariables)) {
						ui.Box(c).FillWidth().Height(height).DrawOver(func(p *ui.Painter, r ui.Rect) {
							clip := p.NativeSurfaceClip()
							if n := inspectorChatNavigation(m); n != nil && n.Picking {
								clip = ui.Rect{}
							}
							if !childImageVisible(r, clip) {
								return
							}
							if err := handle.PresentSurface(m.OpenUIParent, int(r.X), int(r.Y), int(r.W), int(r.H), int(clip.X), int(clip.Y), int(clip.W), int(clip.H)); err != nil {
								b.mu.Lock()
								b.err = err.Error()
								b.mu.Unlock()
							}
						})
					}
				} else {
					ui.Text(c, i18n.T(i18n.KeyChatIsolatedVisible)).FontSize(11)
				}
			})
		}
		if action != nil {
			ui.Text(c, i18n.T(i18n.KeyChatIsolatedGesture)).FontSize(11)
			ui.Text(c, action.ActionText()).Selectable().FontSize(12)
			if stdButton(c, palette, i18n.T(i18n.KeyChatIsolatedSend), buttonPrimary, false).Clicked() {
				b.mu.Lock()
				current := b.action
				target, valid := b.target, b.valid
				b.action = nil
				b.mu.Unlock()
				if current != nil && target != nil && valid != nil && valid() {
					go func() {
						err := target.Submit(context.Background(), current.ActionText(), nil, 0, "")
						if current.RequestID != "" {
							_ = handle.ReplyOpenUIAPI(whaleActionResponse(*current, err))
						}
						if err != nil {
							b.mu.Lock()
							b.err = err.Error()
							b.mu.Unlock()
						}
						invalidate()
					}()
				}
			}
			if stdButton(c, palette, i18n.T(i18n.KeyChatIsolatedIgnore), buttonGhost, false).Clicked() {
				b.mu.Lock()
				current := b.action
				b.action = nil
				b.mu.Unlock()
				if current != nil && current.RequestID != "" && handle != nil {
					go func() {
						_ = handle.ReplyOpenUIAPI(whaleActionResponse(*current, fmt.Errorf("The interaction was dismissed")))
					}()
				}
			}
		}
		show := ui.Local(box, "source", func() bool { return false })
		if stdButton(c, palette, i18n.T(i18n.KeyChatIsolatedSource), buttonGhost, false).Clicked() {
			*show = !*show
		}
		if *show {
			ui.Text(c, program).Selectable().FontSize(12)
		}
		showNotices := ui.Local(box, "licenses", func() bool { return false })
		if stdButton(c, palette, whaleLabel("许可", "Licenses"), buttonGhost, false).Clicked() {
			*showNotices = !*showNotices
		}
		if *showNotices {
			ui.Text(c, openui.WhaleNotices()).Selectable().FontSize(11)
		}
	})
	return true
}

// A visible form remains alive until its owner, explicit close or offscreen
// reclamation cancels it. Elapsed wall time alone does not revoke its owner.
func newOpenUIContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func runOpenUI(ctx context.Context, b *openUIBlock, id, program, appearance string, invalidate func()) {
	defer invalidate()
	b.mu.Lock()
	stop := b.cancel
	b.mu.Unlock()
	if stop != nil {
		defer stop()
	}
	var handle *previewprocess.Session
	finish := func(err error) {
		b.mu.Lock()
		b.loading = false
		b.handle = nil
		b.media = nil
		if err != nil {
			b.err = err.Error()
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
	storageSource := b.sourceID
	b.mu.Unlock()
	storageID := openui.Identity(b.target.StorageScope(), b.target.SessionID, storageSource, 0)
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
	program = b.program
	streaming, inline := b.streaming, b.inline
	parent, revision := b.parent, b.revision
	language, locale, variables, panel := b.language, b.locale, b.variables, b.panel
	b.mu.Unlock()
	if valid == nil || !valid() {
		finish(fmt.Errorf("OpenUI owner changed"))
		return
	}
	hostActions := explicitViewer == "" && openui.IsOwnedViewer(viewer) && (runtime.GOOS == "windows" || runtime.GOOS == "darwin")
	mediaEnabled := hostActions && inline && parent != nil
	handle, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: viewer, Width: 900, Appearance: appearance, Inline: inline, OpenUI: &openui.Render{InstanceID: id, Revision: revision, Program: program, Language: language, Locale: locale, Variables: variables, Panel: panel, Streaming: streaming, State: state, HostActions: hostActions, Media: mediaEnabled}})
	if err != nil {
		finish(err)
		return
	}
	defer handle.Close()
	b.mu.Lock()
	b.handle = handle
	b.loading = false
	b.mu.Unlock()
	var media *openUIMedia
	if mediaSession, ok := any(handle).(*previewprocess.Session); ok && mediaEnabled {
		media = newOpenUIMedia(ctx, b, mediaSession, parent, valid, revision, invalidate)
		b.mu.Lock()
		b.media = media
		b.mu.Unlock()
		defer media.Close()
	}
	invalidate()
	runOpenUILoop(ctx, b, handle, id, viewer, root, storageID, program, appearance, cloneWhaleVariables(variables), panel, streaming, inline, media, valid, invalidate, finish)
}

func runOpenUILoop(
	ctx context.Context,
	b *openUIBlock,
	handle openUISession,
	id, viewer, root, storageID, program, appearance string,
	variables map[string]string,
	panel, streaming, inline bool,
	media *openUIMedia,
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
	if inline {
		registerInlineOpenUIBlock(b)
		defer unregisterInlineOpenUIBlock(b)
	}
	defer func() {
		b.mu.Lock()
		b.evicted = false
		b.offscreenSince = time.Time{}
		b.mu.Unlock()
	}()
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
		reclaimTimer   *time.Timer
		streamTimer    *time.Timer
		lastStreamPush time.Time
		surfaceHidden  bool
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
	defer stopTimer(&reclaimTimer)
	defer stopTimer(&streamTimer)

	syncStateAndVisibility := func() bool {
		if valid != nil && !valid() {
			finish(fmt.Errorf("OpenUI owner changed; renderer closed"))
			return false
		}
		b.mu.Lock()
		latest, live, currentAppearance, currentRevision := b.program, b.streaming, b.appearance, b.revision
		currentVariables, currentLanguage, currentLocale := b.variables, b.language, b.locale
		offscreen := b.inline && (!b.visible || b.evicted)
		offscreenSince := b.offscreenSince
		b.mu.Unlock()

		if media != nil {
			media.Retire(currentRevision)
			if offscreen {
				media.ExpirePaint()
			}
		}
		if offscreen {
			if !surfaceHidden {
				surfaceHidden = true
				_ = handle.PresentSurface(0, 0, 0, 0, 0, 0, 0, 0, 0)
			}
			if reclaimTimer == nil {
				wait := b.effectiveOffscreenTTL()
				if !offscreenSince.IsZero() {
					if elapsed := time.Since(offscreenSince); elapsed >= wait {
						wait = time.Millisecond
					} else {
						wait -= elapsed
					}
				}
				reclaimTimer = time.NewTimer(wait)
			}
		} else {
			if surfaceHidden {
				surfaceHidden = false
				if invalidate != nil {
					invalidate()
				}
			}
			stopTimer(&reclaimTimer)
		}

		if latest != program || live != streaming || currentRevision != revision {
			if live && !lastStreamPush.IsZero() && time.Since(lastStreamPush) < openUIStreamCoalesceDelay {
				if streamTimer == nil {
					streamTimer = time.NewTimer(openUIStreamCoalesceDelay - time.Since(lastStreamPush))
				}
			} else {
				stopTimer(&streamTimer)
				updater, ok := any(handle).(interface{ UpdateOpenUI(openui.Render) error })
				if !ok {
					finish(fmt.Errorf("OpenUI streaming helper is unavailable"))
					return false
				}
				if e := updater.UpdateOpenUI(openui.Render{InstanceID: id, Revision: currentRevision, Program: latest, Language: currentLanguage, Locale: currentLocale, Variables: currentVariables, Panel: panel, Streaming: live, State: json.RawMessage(`{}`)}); e != nil {
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

	if !syncStateAndVisibility() {
		return
	}

	for {
		var reclaimC <-chan time.Time
		if reclaimTimer != nil {
			reclaimC = reclaimTimer.C
		}
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
			if !syncStateAndVisibility() {
				return
			}
		case <-streamC:
			streamTimer = nil
			if !syncStateAndVisibility() {
				return
			}
		case <-reclaimC:
			reclaimTimer = nil
			if valid != nil && !valid() {
				finish(fmt.Errorf("OpenUI owner changed; renderer closed"))
				return
			}
			if b.isOffscreen() {
				b.mu.Lock()
				b.attempted = false
				b.mu.Unlock()
				finish(nil)
				return
			}
			surfaceHidden = false
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
						b.err = e.Error()
						b.mu.Unlock()
					}
				}
			case "title":
				// Child title does not replace the native owner or source identity.
			case "media":
				if media == nil {
					finish(fmt.Errorf("OpenUI private media is unavailable for this surface"))
					return
				}
				response := media.Handle(event)
				if (valid == nil || valid()) && ctx.Err() == nil {
					if replyErr := handle.ReplyOpenUIAPI(response); replyErr != nil {
						b.mu.Lock()
						b.err = replyErr.Error()
						b.mu.Unlock()
					}
				}
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
							b.err = replyErr.Error()
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
					replier, ok := any(handle).(interface {
						ReplyOpenUIAPI(openui.APIResponse) error
					})
					var replyErr error
					if !ok {
						replyErr = fmt.Errorf("OpenUI resource reply helper is unavailable")
					} else {
						replyErr = replier.ReplyOpenUIAPI(response)
					}
					if replyErr != nil && ctx.Err() == nil {
						b.mu.Lock()
						b.err = replyErr.Error()
						b.mu.Unlock()
						invalidate()
					}
				}(event)
			case "state":
				if e = openui.SaveState(root, storageID, event.State); e != nil {
					b.mu.Lock()
					b.err = e.Error()
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
								b.err = e.Error()
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
				b.err = event.Message
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
