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
	Valid  func() bool
	Append func(string) bool
}

type previewOperation struct {
	tab    *Tab
	host   *desktop.Host
	target ActionTarget
	ctx    context.Context
	cancel context.CancelFunc
	kind   string
}

// SetActionTarget wires the owning session's draft without giving web pages
// access to it. Each operation captures a fresh target before any async work.
func (b *Controller) SetActionTarget(target func() ActionTarget) {
	b.actionTarget = target
}

// BeginFrame/EndFrame bracket the owner's actual paint. A docked page must not
// be hidden and re-shown every frame; missing/occluded/narrow panes are hidden.
func (b *Controller) BeginFrame() {
	b.rendered = false
	b.PruneActions()
}
func (b *Controller) EndFrame() {
	if !b.rendered {
		b.Hide()
	}
}

// CancelAction is also called by Hide, navigation, tab changes and Close.
func (b *Controller) CancelAction() {
	b.cancelBrowserTools()
	if op := b.operation; op != nil {
		b.operation = nil
		op.cancel()
		op.tab.ActionMessage = ""
	}
}

// PruneActions is called each owning-window frame, including session/Host
// changes when this browser's chrome is not being painted.
func (b *Controller) PruneActions() {
	b.prunePictureInPicture()
	b.pruneBrowserTools()
	if op := b.operation; op != nil && !b.actionCurrent(op) {
		b.CancelAction()
	}
}

func (b *Controller) actionCurrent(op *previewOperation) bool {
	return op != nil && b.operation == op && b.current() == op.tab &&
		op.tab.Host == op.host && op.host.Alive() && op.ctx.Err() == nil &&
		op.target.Valid != nil && op.target.Valid()
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
		t.Error = "当前窗口或会话已变化，请重试"
		return
	}
	if kind == "pick" && target.Append == nil {
		t.Error = "请先选择会话，再把网页元素加入该会话草稿"
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	op := &previewOperation{tab: t, host: t.Host, target: target, ctx: ctx, cancel: cancel, kind: kind}
	b.operation = op
	t.Error = ""
	t.ActionMessage = "正在截取当前网页可见区域…"
	if kind == "pick" {
		t.ActionMessage = "点击网页元素加入当前会话草稿；Esc 或「取消」退出"
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
					t.Error = "草稿已发送、进入编辑或所属会话变化；本次选取未写入"
					return
				}
				t.ActionMessage = "网页元素已加入发起会话草稿，请返回工作区检查后发送"
				return
			}
			if path != "" {
				t.ArtifactPath = path
				t.ActionMessage = "截图已保存：" + path
				mygo.Shell.ShowItemInFolder(path)
			}
		})
	}()
}

func (b *Controller) actionControls(c *ui.Context, t *Tab, state desktop.State) {
	if t.Document != nil || t.HTML != nil {
		return
	}
	busy := b.operation != nil
	disabled := t.Host == nil || state.Loading || busy
	ui.ScrollHorizontal(c).Height(28).FillWidth().Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "选择元素").Disabled(disabled).Clicked() {
				b.startAction(t, "pick")
			}
			if ui.Button(c, "网页截图").Disabled(disabled).Clicked() {
				b.startAction(t, "screenshot")
			}
			if busy && ui.Button(c, "取消").Clicked() {
				b.CancelAction()
			}
			if t.ArtifactPath != "" && ui.Button(c, "显示截图").Clicked() {
				if info, err := os.Stat(t.ArtifactPath); err == nil && info.Mode().IsRegular() {
					mygo.Shell.ShowItemInFolder(t.ArtifactPath)
				} else {
					t.Error = "截图文件已移除，请重新截图"
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
