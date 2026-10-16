package browserpreview

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"time"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
	desktop "whaleisle/internal/desktop/webview"
)

type browserPictureInPicture struct {
	tab            *Tab
	host           *desktop.Host
	epoch          uint64
	win            *mygo.Window
	ctx            context.Context
	cancel         context.CancelFunc
	bitmap         *ui.Bitmap
	source, status string
}

// Picture-in-picture is a view-only window fed from a pinned WebView, not screen
// recording, media encoding, a second browser navigation or a native JS bridge.
func (b *Controller) openPictureInPicture(t *Tab) {
	if t == nil || b.current() != t || t.Host == nil || !t.Host.Alive() || t.Document != nil || t.HTML != nil || t.Media != nil || b.parent == nil {
		return
	}
	if t.Host.State().Loading {
		return
	}
	b.stopPictureInPicture()
	ctx, cancel := context.WithCancel(context.Background())
	p := &browserPictureInPicture{tab: t, host: t.Host, epoch: t.Host.BrowserPageVersion(), ctx: ctx, cancel: cancel, source: t.Host.State().URL, status: "正在读取来源网页画面…"}
	b.pip = p
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "Browser · 画中画", Width: 480, Height: 320, MinWidth: 240, MinHeight: 180,
		Parent: b.parent, AlwaysOnTop: true, SkipTaskbar: true, DisableMaximize: true, DisableFullScreen: true,
		Page: mygo.PageOptions{DevTools: mygo.DevToolsDisabled},
		Content: ui.View(func(c *ui.Context) {
			fonttheme.ApplyInterface(c)
			ui.Column(c).Fill().MinHeight(0).Padding(8).Gap(6).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "网页画中画 · 最多 4 帧/秒 · 无音频").FontSize(fonttheme.InterfaceSize(11)).Grow(1)
					if ui.Button(c, "关闭").Clicked() {
						b.stopPictureInPicture()
					}
				})
				ui.Text(c, truncate(p.source, 90)).FontSize(fonttheme.InterfaceSize(10)).Tooltip(p.source)
				if p.bitmap != nil {
					ui.Image(c, p.bitmap).Fill().Grow(1).MinHeight(0).Fit(ui.Contain)
				} else {
					ui.Text(c, p.status).FontSize(fonttheme.InterfaceSize(12))
				}
			})
		}),
	})
	p.win = w
	if w == nil || w.IsDestroyed() {
		b.stopPictureInPicture()
		t.Error = "画中画窗口创建失败"
		return
	}
	w.OnClosed(func() {
		p.cancel()
		p.bitmap = nil
		if b.pip == p {
			b.pip = nil
			b.update()
		}
	})
	w.ShowInactive()
	b.update()
	go b.pictureInPictureFrames(p)
}
func (b *Controller) pictureInPictureCurrent(p *browserPictureInPicture) bool {
	if p == nil || b.pip != p || p.tab.Host != p.host || !p.host.Alive() || p.host.BrowserPageVersion() != p.epoch {
		return false
	}
	for _, tab := range b.Tabs {
		if tab == p.tab {
			return true
		}
	}
	return false
}
func (b *Controller) prunePictureInPicture() {
	if b.pip != nil && !b.pictureInPictureCurrent(b.pip) {
		b.stopPictureInPicture()
	}
}
func (b *Controller) stopPictureInPicture() {
	p := b.pip
	if p == nil {
		return
	}
	b.pip = nil
	p.cancel()
	p.bitmap = nil
	if p.win != nil && !p.win.IsDestroyed() {
		p.win.Destroy()
	}
	b.update()
}
func (b *Controller) pictureInPictureFrames(p *browserPictureInPicture) {
	for {
		if p.ctx.Err() != nil {
			return
		}
		data, err := p.host.CaptureBrowserFrame(p.ctx, p.epoch)
		var bitmap *ui.Bitmap
		if err == nil && p.ctx.Err() == nil {
			bitmap, err = decodePictureInPictureFrame(data)
		}
		if p.ctx.Err() != nil {
			return
		}
		applied := make(chan struct{})
		p.win.Update(func() {
			defer close(applied)
			if !b.pictureInPictureCurrent(p) {
				if b.pip == p {
					b.stopPictureInPicture()
				}
				return
			}
			if err != nil {
				p.bitmap = nil
				p.status = "画中画已停止：" + err.Error()
				p.cancel()
				if errors.Is(err, context.Canceled) {
					b.stopPictureInPicture()
				}
				return
			}
			p.bitmap = bitmap
			p.status = ""
		})
		// Update queues its callback asynchronously. Wait for UI consumption so
		// a stalled owner cannot accumulate decoded frames.
		select {
		case <-p.ctx.Done():
			return
		case <-applied:
		}
		if err != nil {
			return
		}
		// No queued frame backlog: capture + decode + UI replacement completes
		// before scheduling another frame. At most one source capture is in flight.
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-p.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func decodePictureInPictureFrame(data []byte) (*ui.Bitmap, error) {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if len(data) > 8*1024*1024 || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 4*1024*1024 {
		return nil, errors.New("画中画帧超过内存上限")
	}
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	width, height := config.Width, config.Height
	if width > 960 {
		height = max(1, height*960/width)
		width = 960
	}
	if height > 720 {
		width = max(1, width*720/height)
		height = 720
	}
	if width == config.Width && height == config.Height {
		return ui.NewBitmap(source), nil
	}
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(target, target.Bounds(), source, source.Bounds(), draw.Src, nil)
	return ui.NewBitmap(target), nil
}
