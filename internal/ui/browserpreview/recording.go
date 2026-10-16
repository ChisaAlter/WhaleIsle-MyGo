package browserpreview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/browserrecording"
	desktop "whaleisle/internal/desktop/webview"
)

// UI-thread state for one explicitly clicked Browser tab. Worker inputs are
// immutable channels/context/Host/epoch; workers never read Controller or Tab.
type browserRecording struct {
	tab          *Tab
	host         *desktop.Host
	epoch        uint64
	ctx          context.Context
	cancel       context.CancelFunc
	stop         chan struct{}
	stopOnce     sync.Once
	phase        string
	frames       int
	elapsed      time.Duration
	limitReached bool
	artifact     *browserrecording.Artifact
}

func (b *Controller) recordingCurrent(r *browserRecording) bool {
	return r != nil && b.recording == r && b.current() == r.tab && r.tab.Host == r.host &&
		r.tab.Document == nil && r.tab.HTML == nil && r.tab.Media == nil && r.ctx.Err() == nil &&
		recordingSourceCurrent(r.host, r.epoch)
}
func recordingSourceCurrent(host *desktop.Host, epoch uint64) bool {
	if host == nil || !host.Alive() || host.BrowserPageVersion() != epoch {
		return false
	}
	state := host.State()
	return !state.Loading && state.Error == ""
}
func (b *Controller) pruneBrowserRecording() {
	if r := b.recording; r != nil && !b.recordingCurrent(r) {
		b.cancelBrowserRecording("来源网页或窗口已变化；录制已取消，未导出")
	}
}
func (b *Controller) cancelBrowserRecording(message string) {
	r := b.recording
	if r == nil {
		return
	}
	b.recording = nil
	r.cancel()
	r.artifact = nil
	if message != "" {
		r.tab.ActionMessage = message
	}
}
func (b *Controller) startBrowserRecording(t *Tab) {
	if b.recording != nil || b.parent == nil || b.parent.IsDestroyed() || t == nil || b.current() != t || t.Host == nil ||
		t.Document != nil || t.HTML != nil || t.Media != nil || t.tools.operation != nil || !recordingSourceCurrent(t.Host, t.Host.BrowserPageVersion()) {
		return
	}
	b.CancelAction()
	b.stopPictureInPicture()
	ctx, cancel := context.WithCancel(context.Background())
	r := &browserRecording{tab: t, host: t.Host, epoch: t.Host.BrowserPageVersion(), ctx: ctx, cancel: cancel, stop: make(chan struct{}), phase: "starting"}
	b.recording = r
	t.Error = ""
	t.ActionMessage = "准备 WebM 编码器；仅录制当前网页，无音频…"
	b.update()
	// Freeze references before leaving the UI thread. No live Host/model RPC,
	// display capture, guest script execution or authentication material is used.
	owner, host, epoch := b.parent, r.host, r.epoch
	go func() {
		workCtx, workCancel := context.WithTimeout(ctx, 3*time.Minute)
		defer workCancel()
		current := func() bool {
			return workCtx.Err() == nil && recordingSourceCurrent(host, epoch)
		}
		result, err := browserrecording.Capture(workCtx, func(ctx context.Context) ([]byte, error) { return host.CaptureBrowserFrame(ctx, epoch) }, current, r.stop, func(p browserrecording.Progress) error {
			accepted := make(chan bool, 1)
			owner.Update(func() {
				if !b.recordingCurrent(r) {
					accepted <- false
					return
				}
				// A Stop click must not be overwritten by an already queued frame update.
				if r.phase != "stopping" {
					r.phase = p.Phase
					if p.Phase == "recording" {
						r.tab.ActionMessage = "正在录制当前网页（无音频）；停止后可导出 WebM"
					}
				}
				r.frames = p.Frames
				r.elapsed = p.Elapsed
				r.limitReached = p.LimitReached
				if p.Phase == "stopping" {
					r.tab.ActionMessage = "停止取帧，正在完成 WebM 编码…"
				}
				accepted <- true
			})
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return errors.New("录制窗口未响应，已取消")
			case ok := <-accepted:
				if !ok {
					return context.Canceled
				}
				return nil
			}
		})
		owner.Update(func() {
			if !b.recordingCurrent(r) {
				if b.recording == r {
					b.cancelBrowserRecording("来源网页已变化；录制已取消，未导出")
				}
				return
			}
			if err != nil {
				b.cancelBrowserRecording("")
				if errors.Is(err, context.Canceled) {
					t.ActionMessage = "录制已取消，未导出"
				} else {
					t.Error = "录制失败，未导出：" + err.Error()
				}
				return
			}
			r.artifact = &result
			r.phase = "ready"
			t.ActionMessage = fmt.Sprintf("WebM 已编码（%.1f MiB）；点击「导出视频」保存，或丢弃", float64(len(result.Data))/(1024*1024))
		})
	}()
}
func (b *Controller) stopBrowserRecording(r *browserRecording) {
	if !b.recordingCurrent(r) || r.phase != "recording" {
		return
	}
	r.phase = "stopping"
	r.stopOnce.Do(func() { close(r.stop) })
	r.tab.ActionMessage = "停止取帧，正在完成 WebM 编码…"
	b.update()
}
func (b *Controller) exportBrowserRecording(r *browserRecording) {
	if !b.recordingCurrent(r) || r.phase != "ready" || r.artifact == nil {
		return
	}
	r.phase = "exporting"
	b.update()
	owner, artifact := b.parent, *r.artifact
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: owner, Title: "导出网页视频（WebM，无音频）", DefaultPath: "browser-" + time.Now().Format("20060102-150405") + ".webm", Filters: []mygo.FileFilter{{Name: "WebM video", Extensions: []string{"webm"}}}})
		owner.Update(func() {
			if !b.recordingCurrent(r) {
				if b.recording == r {
					b.cancelBrowserRecording("来源网页已变化；视频未导出")
				}
				return
			}
			if err != nil {
				r.phase = "ready"
				r.tab.Error = err.Error()
				return
			}
			if path == "" {
				r.phase = "ready"
				return
			}
			if filepath.Ext(path) == "" {
				path += ".webm"
			}
			r.tab.Error = ""
			r.tab.ActionMessage = "正在导出 WebM…"
			go func() {
				err := artifact.Export(r.ctx, path)
				owner.Update(func() {
					if b.recording != r || r.ctx.Err() != nil {
						return
					}
					if err != nil {
						r.phase = "ready"
						r.tab.Error = "视频未导出：" + err.Error()
						return
					}
					b.recording = nil
					r.artifact = nil
					r.cancel()
					r.tab.RecordingPath = path
					r.tab.ActionMessage = "视频已导出：" + path
				})
			}()
		})
	}()
}
func (b *Controller) recordingControls(c *ui.Context, t *Tab, disabled bool) {
	if r := b.recording; r != nil {
		if r.tab != t {
			return
		}
		switch r.phase {
		case "starting":
			ui.Text(c, "准备编码器…").FontSize(fonttheme.InterfaceSize(11))
		case "recording":
			ui.Text(c, fmt.Sprintf("● 录制 %02d:%02d · %d 帧", int(r.elapsed.Seconds())/60, int(r.elapsed.Seconds())%60, r.frames)).FontSize(fonttheme.InterfaceSize(11))
			if ui.Button(c, "停止录制").Clicked() {
				b.stopBrowserRecording(r)
			}
		case "stopping":
			ui.Text(c, "完成 WebM 编码…").FontSize(fonttheme.InterfaceSize(11))
		case "ready":
			if r.limitReached {
				ui.Text(c, "已达 2 分钟上限").FontSize(fonttheme.InterfaceSize(11))
			}
			if ui.Button(c, "导出视频…").Clicked() {
				b.exportBrowserRecording(r)
			}
		case "exporting":
			ui.Text(c, "导出视频…").FontSize(fonttheme.InterfaceSize(11))
		}
		label := "取消录制"
		if r.phase == "ready" {
			label = "丢弃视频"
		}
		if ui.Button(c, label).Clicked() {
			b.cancelBrowserRecording("录制已取消，未导出")
			b.update()
		}
		return
	}
	if ui.Button(c, "录制 WebM").Disabled(disabled || t.tools.operation != nil || !browserrecording.PlatformAvailable()).Tooltip("仅当前网页，无音频；最多 8 帧/秒、1280×720、2 分钟、64 MiB。切换标签、导航或关闭将取消；停止后需手动导出。启动时检测系统 WebView 的 WebM 编码支持；macOS 需要 Safari/WebKit 18.4 或更新，Linux 取决于 WebKitGTK 与媒体插件。").Clicked() {
		b.startBrowserRecording(t)
	}
	if t.RecordingPath != "" && ui.Button(c, "显示视频").Clicked() {
		if info, err := os.Stat(t.RecordingPath); err == nil && info.Mode().IsRegular() {
			mygo.Shell.ShowItemInFolder(t.RecordingPath)
		} else {
			t.Error = "视频文件已移除，请重新录制"
		}
	}
}
