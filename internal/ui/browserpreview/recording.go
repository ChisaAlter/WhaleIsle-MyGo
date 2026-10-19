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
	"whaleisle/internal/paths"
)

// UI-thread state for one explicitly clicked Browser tab. Worker inputs are
// immutable channels/context/Host/epoch; workers never read Controller or Tab.
type browserRecording struct {
	tab      *Tab
	host     *desktop.Host
	epoch    uint64
	ctx      context.Context
	cancel   context.CancelFunc
	stop     chan struct{}
	stopOnce sync.Once
	phase    string
	frames   int
	elapsed  time.Duration
	artifact *browserrecording.Artifact
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
		b.cancelBrowserRecording(browserChromeLabel("来源网页或窗口已变化；录制已取消，未导出", "Source page or window changed; recording cancelled without export"))
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
		if r.phase == "saving" {
			message = browserChromeLabel("录制已结束；已完成保存的文件会保留。", "Recording ended; any completed saved file is retained.")
		}
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
	t.ActionMessage = browserChromeLabel("准备视频编码器；仅录制当前网页，无音频…", "Preparing video encoder; recording only this webpage, without audio…")
	b.update()
	// Freeze references before leaving the UI thread. No live Host/model RPC,
	// display capture, guest script execution or authentication material is used.
	owner, host, epoch := b.parent, r.host, r.epoch
	go func() {
		// Original recording has no duration cutoff; cancellation and per-operation
		// deadlines still bind source capture, encoding and owner callbacks.
		workCtx := ctx
		current := func() bool {
			return workCtx.Err() == nil && recordingSourceCurrent(host, epoch)
		}
		result, err := browserrecording.Capture(workCtx, func(ctx context.Context) ([]byte, error) { return host.CaptureBrowserRecordingFrame(ctx, epoch) }, current, r.stop, func(p browserrecording.Progress) error {
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
						r.tab.ActionMessage = browserChromeLabel("正在录制当前网页（无音频）；停止后自动保存视频", "Recording this webpage without audio; stop to save the video")
					}
				}
				r.frames = p.Frames
				r.elapsed = p.Elapsed
				if p.Phase == "stopping" {
					r.phase = "saving"
					r.tab.ActionMessage = browserChromeLabel("停止取帧，正在编码并保存视频…", "Stopped capturing frames; encoding and saving video…")
				}
				accepted <- true
			})
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return errors.New(browserChromeLabel("录制窗口未响应，已取消", "Recording window did not respond; cancelled"))
			case ok := <-accepted:
				if !ok {
					return context.Canceled
				}
				return nil
			}
		})
		var savedPath string
		var saveErr error
		if err == nil && current() {
			savedPath, saveErr = result.Save(workCtx, filepath.Join(paths.UserDataDir(), "preview-recordings"))
		}
		owner.Update(func() {
			if !b.recordingCurrent(r) {
				if b.recording == r {
					b.cancelBrowserRecording(browserChromeLabel("来源网页已变化；录制已取消，未导出", "Source page changed; recording cancelled without export"))
				}
				return
			}
			if err != nil {
				b.cancelBrowserRecording("")
				if errors.Is(err, context.Canceled) {
					t.ActionMessage = browserChromeLabel("录制已取消，未导出", "Recording cancelled; nothing exported")
				} else {
					t.Error = browserChromeLabel("录制失败，未导出：", "Recording failed; nothing exported: ") + err.Error()
				}
				return
			}
			r.artifact = &result
			r.phase = "ready"
			if saveErr != nil {
				t.Error = browserChromeLabel("视频保存失败：", "Recording could not be saved: ") + saveErr.Error()
				t.ActionMessage = fmt.Sprintf(browserChromeLabel("视频已编码（%.1f MiB），可导出到其他位置。", "Video encoded (%.1f MiB); export it to another location."), float64(len(result.Data))/(1024*1024))
				return
			}
			b.recording = nil
			r.artifact = nil
			r.cancel()
			t.RecordingPath = savedPath
			t.ActionMessage = browserChromeLabel("视频已保存：", "Recording saved: ") + savedPath
		})
	}()
}
func (b *Controller) stopBrowserRecording(r *browserRecording) {
	if !b.recordingCurrent(r) || r.phase != "recording" {
		return
	}
	r.phase = "stopping"
	r.stopOnce.Do(func() { close(r.stop) })
	r.tab.ActionMessage = browserChromeLabel("停止取帧，正在完成视频编码…", "Stopped capturing frames; finishing video encoding…")
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
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: owner, Title: browserChromeLabel("导出网页视频（无音频）", "Export webpage video (no audio)"), DefaultPath: "browser-" + time.Now().Format("20060102-150405") + artifact.Extension(), Filters: []mygo.FileFilter{{Name: "Video", Extensions: []string{artifact.Extension()[1:]}}}})
		owner.Update(func() {
			if !b.recordingCurrent(r) {
				if b.recording == r {
					b.cancelBrowserRecording(browserChromeLabel("来源网页已变化；视频未导出", "Source page changed; video was not exported"))
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
				path += artifact.Extension()
			}
			r.tab.Error = ""
			r.tab.ActionMessage = browserChromeLabel("正在导出视频…", "Exporting video…")
			go func() {
				err := artifact.Export(r.ctx, path)
				owner.Update(func() {
					if b.recording != r || r.ctx.Err() != nil {
						return
					}
					if err != nil {
						r.phase = "ready"
						r.tab.Error = browserChromeLabel("视频未导出：", "Video was not exported: ") + err.Error()
						return
					}
					b.recording = nil
					r.artifact = nil
					r.cancel()
					r.tab.RecordingPath = path
					r.tab.ActionMessage = browserChromeLabel("视频已导出：", "Video exported: ") + path
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
			ui.Text(c, browserChromeLabel("准备编码器…", "Preparing encoder…")).FontSize(fonttheme.InterfaceSize(11))
		case "recording":
			ui.Text(c, fmt.Sprintf(browserChromeLabel("● 录制 %02d:%02d · %d 帧", "● Recording %02d:%02d · %d frames"), int(r.elapsed.Seconds())/60, int(r.elapsed.Seconds())%60, r.frames)).FontSize(fonttheme.InterfaceSize(11))
			if ui.Button(c, browserChromeLabel("停止录制", "Stop recording")).Clicked() {
				b.stopBrowserRecording(r)
			}
		case "stopping", "saving":
			ui.Text(c, browserChromeLabel("完成视频编码…", "Finishing video encoding…")).FontSize(fonttheme.InterfaceSize(11))
		case "ready":
			if ui.Button(c, browserChromeLabel("导出视频…", "Export video…")).Clicked() {
				b.exportBrowserRecording(r)
			}
		case "exporting":
			ui.Text(c, browserChromeLabel("导出视频…", "Export video…")).FontSize(fonttheme.InterfaceSize(11))
		}
		label := browserChromeLabel("取消录制", "Cancel recording")
		if r.phase == "ready" {
			label = browserChromeLabel("丢弃视频", "Discard video")
		}
		if ui.Button(c, label).Clicked() {
			b.cancelBrowserRecording(browserChromeLabel("录制已取消，未导出", "Recording cancelled; nothing exported"))
			b.update()
		}
		return
	}
	if ui.Button(c, browserChromeLabel("录制视频", "Record video")).Disabled(disabled || t.tools.operation != nil || !browserrecording.PlatformAvailable()).Tooltip(browserChromeLabel("仅当前网页，无音频；约 12 帧/秒上限，保留首帧分辨率，无固定时长上限；编码文件最多 512 MiB，超限会丢弃。支持最大边 3840、面积 3840×2160 的视口；保留原生 HiDPI 像素，单帧 PNG 最多 64 MiB。切换标签、导航或关闭将取消；停止后自动保存到应用录制目录。启动时检测系统 WebView 的 WebM/MP4 编码支持；可用格式取决于系统 WebView 与媒体插件；WebM 不可用时尝试 MP4。", "This webpage only, without audio; up to about 12 frames/second, using the first frame’s resolution, with no fixed duration limit. Encoded files are limited to 512 MiB and discarded if exceeded. Maximum viewport edge: 3840; area: 3840×2160. Native HiDPI pixels are retained; each PNG frame is limited to 64 MiB. Switching tabs, navigating or closing cancels recording; Stop saves to the application recordings folder. WebM/MP4 encoder support is detected at startup and depends on the system WebView and media plugins; MP4 is tried if WebM is unavailable.")).Clicked() {
		b.startBrowserRecording(t)
	}
	if t.RecordingPath != "" && ui.Button(c, browserChromeLabel("显示视频", "Show video")).Clicked() {
		if info, err := os.Stat(t.RecordingPath); err == nil && info.Mode().IsRegular() {
			mygo.Shell.ShowItemInFolder(t.RecordingPath)
		} else {
			t.Error = browserChromeLabel("视频文件已移除，请重新录制", "Video file was removed; record again")
		}
	}
}
