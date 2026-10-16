package browserpreview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/previewprocess"
)

type interactivePreview struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// SetDeveloperTools binds the current accepted Host preference. It never grants
// permission to run a preview: the explicit gesture and sandbox checks remain.
func (b *Controller) SetDeveloperTools(enabled func() bool) { b.developerTools = enabled }
func (b *Controller) developerToolsEnabled() bool {
	return b.developerTools == nil || b.developerTools()
}

func (b *Controller) interactiveHTMLControls(c *ui.Context, t *Tab, dark bool) {
	if !b.developerToolsEnabled() {
		b.stopInteractiveHTML(t)
		return
	}
	if t.interactive != nil {
		if ui.Button(c, "停止交互预览").Clicked() {
			b.stopInteractiveHTML(t)
			t.ActionMessage = "交互预览已停止；当前静态快照保留"
		}
		return
	}
	supported := runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")
	if ui.Button(c, "运行脚本（隔离实验）").Disabled(!supported).
		Tooltip("仅本次在独立 AppContainer 窗口运行已保存 HTML 的脚本；不开放网络或应用权限。任何失败都保留当前静态预览。").Clicked() {
		b.startInteractiveHTML(t, dark)
	}
}

func (b *Controller) stopInteractiveHTML(t *Tab) {
	if t == nil || t.interactive == nil {
		return
	}
	op := t.interactive
	t.interactive = nil
	t.ActionMessage = "交互预览已停止；当前静态快照保留"
	op.cancel() // Job cleanup is asynchronous; never block the UI on process exit.
}

func (b *Controller) cancelInteractivePreviews() {
	for _, t := range b.Tabs {
		b.stopInteractiveHTML(t)
	}
}

func (b *Controller) ownsInteractiveHTML(t *Tab, op *interactivePreview) bool {
	if t == nil || t.interactive != op || op.ctx.Err() != nil || t.HTML == nil || !t.HTML.Valid() {
		return false
	}
	for _, candidate := range b.Tabs {
		if candidate == t {
			return true
		}
	}
	return false
}

func installedPreviewHelper() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(executable), "whaleisle-preview.exe")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("隔离 helper 未随应用安装，保留静态预览")
	}
	return path, nil
}

// This is the only product call site enabling the helper. No startup path,
// environment variable or persistent preference automatically selects it.
func (b *Controller) startInteractiveHTML(t *Tab, dark bool) {
	if !b.developerToolsEnabled() || b.parent == nil || t == nil || t.HTML == nil || t.interactive != nil || !t.HTML.Valid() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	op := &interactivePreview{ctx: ctx, cancel: cancel}
	t.interactive = op
	t.ActionMessage = "正在检查隔离条件并准备独立窗口；静态快照保持打开…"
	doc := t.HTML
	appearance := "light"
	if dark {
		appearance = "dark"
	}
	// Lease checks continue even if the owning window is not painting. A
	// session/Host/cwd change closes Document, which cancels the helper job.
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-doc.Done():
				cancel()
				return
			case <-ticker.C:
				if !doc.Valid() {
					cancel()
					return
				}
			}
		}
	}()
	go func() {
		defer cancel()
		helper, err := installedPreviewHelper()
		var prepared, resources string
		if err == nil && ctx.Err() == nil {
			var report htmldocument.ResourceReport
			prepared, report, err = doc.InteractiveSnapshot(ctx)
			resources = strings.Replace(report.Summary(), "静态资源", "资源", 1)
		}
		var session *previewprocess.Session
		if err == nil && ctx.Err() == nil {
			session, err = previewprocess.OpenExperimental(ctx, previewprocess.Options{Enabled: true, HelperPath: helper},
				previewprocess.Document{HTML: prepared, Width: 900, Appearance: appearance})
		}
		if ctx.Err() != nil && err == nil {
			err = ctx.Err()
		}
		b.parent.Update(func() {
			if err != nil {
				// Update may run after this worker returns and cancels ctx.
				// A completed failure still belongs to this tab/document;
				// Hide/Close already clears op, so genuinely stale results lose.
				if t.interactive != op || t.HTML != doc || !doc.Valid() {
					return
				}
				t.interactive = nil
				t.ActionMessage = "交互预览未启动，静态快照已保留：" + err.Error()
				return
			}
			if !b.ownsInteractiveHTML(t, op) {
				if t.interactive == op {
					t.interactive = nil
				}
				return
			}
			t.ActionMessage = "交互预览已在独立隔离窗口打开；" + resources + "；网络/运行时文件依赖禁用，关闭/切换标签会停止脚本"
		})
		if session == nil {
			return
		}
		defer session.Close()
		select {
		case <-ctx.Done():
		case <-doc.Done():
		case <-session.Done():
		}
		b.parent.Update(func() {
			if t.interactive != op {
				return
			}
			t.interactive = nil
			t.ActionMessage = "交互预览已结束；当前静态快照保留"
		})
	}()
}
