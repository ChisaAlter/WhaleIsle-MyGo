package browserpreview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/desktop/webview"
)

// ActionTarget is captured on the UI thread at the user's gesture. Valid and
// Append are called only on that thread, never from a capture goroutine.
type ActionTarget struct {
	Valid            func() bool
	Append           func(string) bool
	AppendAnnotation func(text, screenshotPath string) bool
}

type previewOperation struct {
	pageCurrent    func() bool
	tab            *Tab
	host           *desktop.Host
	target         ActionTarget
	ctx            context.Context
	cancel         context.CancelFunc
	kind           string
	annotation     string
	attachmentPath string
	ready          bool
}

// SetActionTarget wires the owning session's draft without giving web pages
// access to it. Each operation captures a fresh target before any async work.
func (b *Controller) SetActionTarget(target func() ActionTarget) {
	b.actionTarget = target
}

// BeginFrame/EndFrame bracket the owner's actual paint. A docked page must not
// be hidden and re-shown every frame; missing/occluded/narrow panes are hidden.
func (b *Controller) BeginFrame() {
	if !b.developerToolsEnabled() {
		b.cancelInteractivePreviews()
	}
	b.rendered = false
	if b.mini != nil {
		b.mini.rendered = false
	}
	b.PruneActions()
}
func (b *Controller) EndFrame() {
	if p := b.mini; p != nil && p.rendered && p.context.HasForegroundModal() {
		p.host.SetVisible(false)
	}
	if !b.rendered {
		if b.mini != nil && b.mini.rendered {
			for _, t := range b.Tabs {
				if t.Host != nil && t.Host != b.mini.host {
					t.Host.SetVisible(false)
				}
			}
		} else {
			b.Hide()
		}
	}
}

// CancelAction is also called by Hide, navigation, tab changes and Close.
func (b *Controller) CancelAction() {
	b.cancelBrowserRecording(browserChromeLabel("录制已取消，未导出", "Recording cancelled; nothing exported"))
	b.cancelBrowserTools()
	if op := b.operation; op != nil {
		b.operation = nil
		op.cancel()
		if op.attachmentPath != "" {
			_ = os.Remove(op.attachmentPath)
		}
		op.tab.ActionMessage = ""
	}
}

// PruneActions is called each owning-window frame, including session/Host
// changes when this browser's chrome is not being painted.
func (b *Controller) PruneActions() {
	if b.mini != nil && !b.miniPlayerCurrent() {
		b.CloseMiniPlayer()
	}
	b.prunePictureInPicture()
	b.pruneBrowserRecording()
	b.pruneBrowserTools()
	if op := b.operation; op != nil && !b.actionCurrent(op) {
		b.CancelAction()
	}
}

func (b *Controller) actionCurrent(op *previewOperation) bool {
	return op != nil && b.operation == op && b.current() == op.tab &&
		op.tab.Host == op.host && op.host.Alive() && op.ctx.Err() == nil &&
		op.target.Valid != nil && op.target.Valid() && (op.pageCurrent == nil || op.pageCurrent())
}

func (b *Controller) startAction(t *Tab, kind string) {
	if b.operation != nil || b.parent == nil || t == nil || t.Host == nil {
		return
	}
	target := ActionTarget{}
	if b.actionTarget != nil {
		target = b.actionTarget()
	}
	if target.Valid == nil || !target.Valid() {
		t.Error = browserChromeLabel("当前窗口或会话已变化，请重试", "The window or session changed; try again")
		return
	}
	if kind == "pick" && target.Append == nil {
		t.Error = browserChromeLabel("请先选择会话，再把网页元素加入该会话草稿", "Select a session before adding webpage elements to its draft")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	op := &previewOperation{tab: t, host: t.Host, target: target, ctx: ctx, cancel: cancel, kind: kind}
	b.operation = op
	t.Error = ""
	t.ActionMessage = browserChromeLabel("正在截取当前网页可见区域…", "Capturing the visible webpage…")
	if kind == "pick" {
		t.ActionMessage = browserChromeLabel("点击网页元素加入当前会话草稿；Esc 或「取消」退出", "Click a webpage element to add it to the current draft; press Esc or Cancel to exit")
	}
	b.update()
	go func() {
		var annotation, path string
		var err error
		if kind == "pick" {
			var picked desktop.PickedElement
			picked, err = op.host.PickElement(ctx)
			if err == nil {
				annotation = picked.Markdown()
			}
		} else {
			var data []byte
			data, err = op.host.CaptureBrowserScreenshot(ctx)
			if err == nil && ctx.Err() == nil {
				path, err = saveBrowserScreenshot(ctx, data)
			}
		}
		b.parent.Update(func() {
			if !b.actionCurrent(op) {
				if path != "" {
					_ = os.Remove(path)
				}
				// Do not clear an operation started after this result.
				if b.operation == op {
					b.CancelAction()
				}
				return
			}
			b.operation = nil
			op.cancel()
			t.ActionMessage = ""
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				t.Error = err.Error()
				return
			}
			if kind == "pick" {
				if !target.Append(annotation) {
					t.Error = browserChromeLabel("草稿已发送、进入编辑或所属会话变化；本次选取未写入", "The draft was sent, entered edit mode or changed session; the selection was not added")
					return
				}
				t.ActionMessage = browserChromeLabel("网页元素已加入发起会话草稿，请返回工作区检查后发送", "Webpage element added to the initiating session’s draft; review it in the workspace before sending")
				return
			}
			if path != "" {
				t.ArtifactPath = path
				t.ActionMessage = browserChromeLabel("截图已保存：", "Screenshot saved: ") + path
				mygo.Shell.ShowItemInFolder(path)
			}
		})
	}()
}

func (b *Controller) actionControls(c *ui.Context, t *Tab, state desktop.State) {
	if t.Document != nil || t.HTML != nil || t.Media != nil {
		return
	}
	busy := b.operation != nil
	disabled := t.Host == nil || state.Loading || busy || b.recording != nil
	ui.ScrollHorizontal(c).Height(28).FillWidth().Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			b.recordingControls(c, t, disabled)
			if ui.Button(c, browserChromeLabel("可视批注", "Visual annotation")).Disabled(disabled).Clicked() {
				b.startAnnotation(t)
			}
			if op := b.operation; op != nil && op.ready {
				if ui.Button(c, browserChromeLabel("查看批注截图", "View annotated screenshot")).Clicked() {
					mygo.Shell.ShowItemInFolder(op.attachmentPath)
				}
				if ui.Button(c, browserChromeLabel("复制批注", "Copy annotation")).Clicked() {
					mygo.Clipboard.WriteText(op.annotation)
				}
				if ui.Button(c, browserChromeLabel("批注和截图加入草稿", "Add annotation and screenshot to draft")).Tooltip(truncate(op.annotation, 2000)).Clicked() {
					b.appendAnnotation(op)
				}
			}
			if ui.Button(c, browserChromeLabel("选择元素", "Select element")).Disabled(disabled).Clicked() {
				b.startAction(t, "pick")
			}
			if ui.Button(c, browserChromeLabel("网页截图", "Screenshot")).Disabled(disabled).Clicked() {
				b.startAction(t, "screenshot")
			}
			if busy && ui.Button(c, browserChromeLabel("取消", "Cancel")).Clicked() {
				b.CancelAction()
			}
			if t.ArtifactPath != "" && ui.Button(c, browserChromeLabel("显示截图", "Show screenshot")).Clicked() {
				if info, err := os.Stat(t.ArtifactPath); err == nil && info.Mode().IsRegular() {
					mygo.Shell.ShowItemInFolder(t.ArtifactPath)
				} else {
					t.Error = browserChromeLabel("截图文件已移除，请重新截图", "Screenshot file was removed; take another screenshot")
				}
			}
		})
	})
}

// This folder and filename are chosen locally, never by inspected page content.
// No upload, attachment, prompt submission or workspace modification occurs.
func saveBrowserScreenshot(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > desktop.MaxBrowserScreenshotBytes {
		return "", errors.New("browser: invalid screenshot size")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "WhaleIsle", "browser-screenshots")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(dir, "browser-"+time.Now().Format("20060102-150405")+"-*.png")
	if err != nil {
		return "", err
	}
	path := file.Name()
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if n, err := file.Write(data); err != nil || n != len(data) {
		if err == nil {
			err = fmt.Errorf("short screenshot write: %d of %d bytes", n, len(data))
		}
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("browser: screenshot path unavailable")
	}
	ok = true
	return path, nil
}
