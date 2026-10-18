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
	"whaleisle/internal/store/session"
)

type openUIBlock struct {
	mu                           sync.Mutex
	program                      string
	appearance                   string
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
	if language != "openui" && language != "openui-lang" {
		return false
	}
	if msg.Role != "assistant" || m.FeedbackStore == nil {
		return false
	}
	messageID := msg.SourceMessageID
	if messageID == "" {
		messageID = msg.ID
	}
	id := openui.Identity(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, offset)
	box := ui.Column(c.Key("openui:" + id)).FillWidth().Padding(8).Gap(6)
	b := *ui.Local(box, "instance", func() *openUIBlock { return &openUIBlock{program: program, streaming: msg.Streaming, revision: 1} })
	b.mu.Lock()
	sourceChanged := b.program != program || b.streaming != msg.Streaming
	if sourceChanged {
		b.action = nil
		b.revision++
	}
	b.program = program
	b.streaming = msg.Streaming
	b.parent = m.OpenUIWindow
	media, revision := b.media, b.revision
	loading, errText, action, handle, inline := b.loading, b.err, b.action, b.handle, b.inline
	b.mu.Unlock()
	if sourceChanged && media != nil {
		media.Retire(revision)
	}
	invalidate := c.Services().Invalidate
	appearance := "light"
	if m.SystemDark {
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
		target, valid, err := m.FeedbackStore.CaptureOpenUI(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, program)
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
		ui.Text(c, "OpenUI · 隔离交互 / Isolated interaction").FontSize(12)
		if ui.Button(c, "重试内联 UI / Retry inline UI").Disabled(loading || handle != nil).Clicked() {
			start(true)
		}
		if ui.Button(c, "打开独立窗口 / Open separate window").Disabled(msg.Streaming || loading || handle != nil).Clicked() {
			start(false)
		}
		if handle != nil && ui.Button(c, "关闭交互 UI / Close interactive UI").Clicked() {
			b.mu.Lock()
			if b.cancel != nil {
				b.cancel()
			}
			b.mu.Unlock()
		}
		if loading {
			ui.Text(c, "正在启动隔离渲染器 / Starting isolated renderer").FontSize(11)
		}
		if errText != "" {
			ui.Text(c, errText).Selectable().FontSize(11)
		}
		height := float32(360)
		if handle != nil {
			height = float32(max(80, min(700, handle.ContentHeight())))
		}
		if handle == nil || inline {
			seat := ui.Column(c).FillWidth().Height(height)
			seat.DrawOver(func(p *ui.Painter, r ui.Rect) {
				clip := p.NativeSurfaceClip()
				b.mu.Lock()
				media := b.media
				b.mu.Unlock()
				if media != nil {
					media.Paint(r, clip, m.OpenUIMediaOcclusions)
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
					if !handle.RenderPixels(c, height, appearance, nil) {
						ui.Box(c).FillWidth().Height(height).DrawOver(func(p *ui.Painter, r ui.Rect) {
							clip := p.NativeSurfaceClip()
							if err := handle.PresentSurface(m.OpenUIParent, int(r.X), int(r.Y), int(r.W), int(r.H), int(clip.X), int(clip.Y), int(clip.W), int(clip.H)); err != nil {
								b.mu.Lock()
								b.err = err.Error()
								b.mu.Unlock()
							}
						})
					}
				} else {
					ui.Text(c, "当前可见时加载真实运行资源 / Loads the real runtime when visible").FontSize(11)
				}
			})
		}
		if action != nil {
			ui.Text(c, "交互请求尚未发送：渲染消息不含可信用户手势证明 / Confirm this action: renderer messages do not attest a user gesture.").FontSize(11)
			ui.Text(c, action.ActionText()).Selectable().FontSize(12)
			if ui.Button(c, "发送交互结果 / Send interaction result").Clicked() {
				b.mu.Lock()
				current := b.action
				target, valid := b.target, b.valid
				b.action = nil
				b.mu.Unlock()
				if current != nil && target != nil && valid != nil && valid() {
					go func() {
						err := target.Submit(context.Background(), current.ActionText(), nil, 0, "")
						if err != nil {
							b.mu.Lock()
							b.err = err.Error()
							b.mu.Unlock()
						}
						invalidate()
					}()
				}
			}
			if ui.Button(c, "忽略交互请求 / Dismiss interaction request").Clicked() {
				b.mu.Lock()
				b.action = nil
				b.mu.Unlock()
			}
		}
		show := ui.Local(box, "source", func() bool { return false })
		if ui.Button(c, "源码 / Source").Clicked() {
			*show = !*show
		}
		if *show {
			ui.Text(c, program).Selectable().FontSize(12)
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
	config, err := os.UserConfigDir()
	if err != nil {
		finish(err)
		return
	}
	root := filepath.Join(config, "WhaleIsle-MyGo", "openui-state-v1")
	state, err := openui.ReadState(root, id)
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
	b.mu.Unlock()
	if valid == nil || !valid() {
		finish(fmt.Errorf("OpenUI owner changed"))
		return
	}
	hostActions := explicitViewer == "" && openui.IsOwnedViewer(viewer) && (runtime.GOOS == "windows" || runtime.GOOS == "darwin")
	mediaEnabled := hostActions && inline && parent != nil
	handle, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: viewer, Width: 900, Appearance: appearance, Inline: inline, OpenUI: &openui.Render{InstanceID: id, Revision: revision, Program: program, Streaming: streaming, State: state, HostActions: hostActions, Media: mediaEnabled}})
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
				if e := updater.UpdateOpenUI(openui.Render{InstanceID: id, Revision: currentRevision, Program: latest, Streaming: live, State: json.RawMessage(`{}`)}); e != nil {
					finish(e)
					return
				}
				program, streaming, revision = latest, live, currentRevision
			}
			if e := handle.SetTheme(currentAppearance, nil); e != nil {
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
			switch event.Type {
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
			case "clipboard", "open-url":
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
				if e = openui.SaveState(root, id, event.State); e != nil {
					b.mu.Lock()
					b.err = e.Error()
					b.mu.Unlock()
				}
			case "assistant":
				b.mu.Lock()
				b.action = &event
				b.mu.Unlock()
			case "error":
				b.mu.Lock()
				b.err = event.Message
				b.mu.Unlock()
			case "resolved":
				b.mu.Lock()
				if event.Program == b.program {
					b.err = ""
				}
				b.mu.Unlock()
			}
			invalidate()
		}
	}
}
