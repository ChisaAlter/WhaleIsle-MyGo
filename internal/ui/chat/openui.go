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
	"time"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

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
	lastPaint                    time.Time
	loading                      bool
	err                          string
	action                       *openui.Event
	handle                       *previewprocess.Session
	cancel                       context.CancelFunc
	target                       *session.ComposerTarget
	valid                        func() bool
	parent                       *mygo.Window
	revision                     uint64
	media                        *openUIMedia
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
	id := openui.Identity(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, offset)
	box := ui.Column(c.Key("openui:" + id)).FillWidth().Padding(8).Gap(6)
	b := *ui.Local(box, "instance", func() *openUIBlock { return &openUIBlock{program: program, streaming: msg.Streaming, revision: 1} })
	b.mu.Lock()
	nextVariables := whaleThemeVariables(m)
	sourceChanged := b.program != program || b.streaming != msg.Streaming
	appearanceChanged := b.language != language || b.locale != string(i18n.Locale()) || !equalWhaleVariables(b.variables, nextVariables)
	if sourceChanged {
		b.action = nil
		b.revision++
	}
	if appearanceChanged && !sourceChanged {
		b.revision++
	}
	b.program = program
	b.streaming = msg.Streaming
	b.parent = m.OpenUIWindow
	b.language = language
	b.locale = string(i18n.Locale())
	b.variables = nextVariables
	b.panel = panel
	b.capture = capture
	b.panelStore = whalePanelFor(m)
	b.sourceID = openui.Identity("", m.Snapshot.SelectedID, messageID, offset)
	b.sourceOrder = [3]float64{msg.Seq, 0, float64(offset)}
	media, revision := b.media, b.revision
	loading, errText, action, handle, inline := b.loading, b.err, b.action, b.handle, b.inline
	b.mu.Unlock()
	if sourceChanged && media != nil {
		media.Retire(revision)
	}
	invalidate := c.Services().Invalidate
	appearance := "light"
	th := theme.Resolve(m.ThemeID)
	if th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) {
		appearance = "dark"
	}
	b.mu.Lock()
	b.appearance = appearance
	b.mu.Unlock()
	start := func(inline bool) {
		b.mu.Lock()
		if b.loading || b.handle != nil {
			b.mu.Unlock()
			return
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
		if loading {
			ui.Text(c, i18n.T(i18n.KeyChatIsolatedStart)).FontSize(11)
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
				if !childImageVisible(r, clip) {
					return
				}
				b.mu.Lock()
				b.lastPaint = time.Now()
				attempted := b.attempted
				b.mu.Unlock()
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
	if handle != nil || loading {
		c.After(100 * time.Millisecond)
	}
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
	if mediaEnabled {
		media = newOpenUIMedia(ctx, b, handle, parent, valid, revision, invalidate)
		b.mu.Lock()
		b.media = media
		b.mu.Unlock()
		defer media.Close()
	}
	invalidate()
	downloads := newWhaleDownloads()
	resources := openui.NewResourceAPI()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			finish(nil)
			return
		case <-handle.Done():
			finish(nil)
			return
		case <-tick.C:
			b.mu.Lock()
			latest, live, lastPaint, currentAppearance, currentRevision := b.program, b.streaming, b.lastPaint, b.appearance, b.revision
			currentVariables, currentLanguage, currentLocale := b.variables, b.language, b.locale
			b.mu.Unlock()
			if media != nil {
				media.Retire(currentRevision)
				media.ExpirePaint()
			}
			if inline && time.Since(lastPaint) > 300*time.Millisecond && runtime.GOOS == "windows" {
				_ = handle.PresentSurface(0, 0, 0, 0, 0, 0, 0, 0, 0)
			}
			if inline && time.Since(lastPaint) > 30*time.Second {
				b.mu.Lock()
				b.attempted = false
				b.mu.Unlock()
				finish(nil)
				return
			}
			if latest != program || live != streaming || currentRevision != revision {
				updater, ok := any(handle).(interface{ UpdateOpenUI(openui.Render) error })
				if !ok {
					finish(fmt.Errorf("OpenUI streaming helper is unavailable"))
					return
				}
				if e := updater.UpdateOpenUI(openui.Render{InstanceID: id, Revision: currentRevision, Program: latest, Language: currentLanguage, Locale: currentLocale, Variables: currentVariables, Panel: panel, Streaming: live, State: json.RawMessage(`{}`)}); e != nil {
					finish(e)
					return
				}
				downloads = newWhaleDownloads()
				program, streaming, revision = latest, live, currentRevision
			}
			if e := handle.SetTheme(currentAppearance, whalePreviewVariables(currentVariables)); e != nil {
				finish(e)
				return
			}
			if !valid() {
				finish(fmt.Errorf("OpenUI owner changed; renderer closed"))
				return
			}
		case raw, ok := <-handle.Events():
			if !ok {
				finish(nil)
				return
			}
			if !valid() {
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
				if valid() && ctx.Err() == nil {
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
				response := executeOpenUINativeAction(ctx, b, handle, valid, event)
				if valid() && ctx.Err() == nil {
					if replyErr := handle.ReplyOpenUIAPI(response); replyErr != nil {
						b.mu.Lock()
						b.err = replyErr.Error()
						b.mu.Unlock()
					}
				}
			case "api-request":
				go func(request openui.Event) {
					if !valid() || ctx.Err() != nil {
						return
					}
					response := resources.Handle(ctx, request)
					if !valid() || ctx.Err() != nil {
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
						if target == nil || !current || !valid() || ctx.Err() != nil {
							err = fmt.Errorf("Whale UI action owner changed")
						} else {
							err = target.Submit(ctx, action.ActionText(), nil, 0, "")
						}
						if valid() && ctx.Err() == nil {
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
