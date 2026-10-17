package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"sync"
	"time"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/store/session"
)

var openUISlots = make(chan struct{}, 4)

type openUIBlock struct {
	mu      sync.Mutex
	program string
	loading bool
	err     string
	action  *openui.Event
	handle  *previewprocess.Session
	cancel  context.CancelFunc
	target  *session.ComposerTarget
	valid   func() bool
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
	block := ui.Local(box, "instance", func() *openUIBlock { return &openUIBlock{program: program} })
	b := *block
	b.mu.Lock()
	changed := b.program != program
	if changed && b.cancel != nil {
		b.cancel()
	}
	b.mu.Unlock()
	if changed {
		b = &openUIBlock{program: program}
		*block = b
	}
	b.mu.Lock()
	loading, errText, action, opened := b.loading, b.err, b.action, b.handle != nil
	b.mu.Unlock()
	box.Children(func() {
		ui.Text(c, "OpenUI · 隔离交互窗口 / Isolated interactive window").FontSize(12)
		if msg.Streaming {
			ui.Text(c, "生成完成后可打开交互窗口 / Available when generation completes").FontSize(11)
		}
		if ui.Button(c, "打开交互 UI / Open interactive UI").Disabled(msg.Streaming || loading || opened).Clicked() {
			target, valid, err := m.FeedbackStore.CaptureOpenUI(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, messageID, program)
			if err != nil {
				b.mu.Lock()
				b.err = err.Error()
				b.mu.Unlock()
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				b.mu.Lock()
				b.loading = true
				b.err = ""
				b.cancel = cancel
				b.target = target
				b.valid = valid
				b.mu.Unlock()
				appearance := "light"
				if m.SystemDark {
					appearance = "dark"
				}
				go runOpenUI(ctx, b, id, program, appearance, c.Services().Invalidate)
			}
		}
		if opened && ui.Button(c, "关闭交互 UI / Close interactive UI").Clicked() {
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
						c.Services().Invalidate()
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
	return true
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
		if err != nil {
			b.err = err.Error()
		}
		b.mu.Unlock()
	}
	select {
	case openUISlots <- struct{}{}:
		defer func() { <-openUISlots }()
	default:
		finish(fmt.Errorf("At most four OpenUI windows may be open"))
		return
	}
	viewer, err := openui.LoadViewer(os.Getenv("WHALEISLE_OPENUI_VIEWER"))
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
	b.mu.Unlock()
	if valid == nil || !valid() {
		finish(fmt.Errorf("OpenUI owner changed"))
		return
	}
	handle, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: previewprocess.HelperPath(exe)}, previewprocess.Document{HTML: viewer, Width: 900, Appearance: appearance, OpenUI: &openui.Render{InstanceID: id, Program: program, State: state}})
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
				b.err = ""
				b.mu.Unlock()
			}
			invalidate()
		}
	}
}
